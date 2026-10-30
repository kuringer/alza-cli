package client

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"
)

//go:embed headless-serve.mjs
var headlessServeScript []byte

// sessionCloseWait bounds how long Close waits for the browser to exit.
const sessionCloseWait = 10 * time.Second

// HeadlessSession keeps one headless Chromium running and serves many page
// scripts from it, concurrently (one page per request). It starts on first
// use; call Close when done.
type HeadlessSession struct {
	runner *HeadlessRunner
	// command starts the server process (tests replace it).
	command func() *exec.Cmd

	mu      sync.Mutex
	started bool
	dead    error
	stdin   io.WriteCloser
	cmd     *exec.Cmd
	nextID  int64
	pending map[int64]chan sessionReply
	stderr  bytes.Buffer
	done    chan struct{}
}

type sessionReply struct {
	data json.RawMessage
	err  error
}

type sessionRequest struct {
	ID     int64  `json:"id"`
	URL    string `json:"url,omitempty"`
	Ready  string `json:"ready,omitempty"`
	Script string `json:"script,omitempty"`
	// TimeoutMs is the server-side budget for navigation + readiness + script.
	TimeoutMs int64 `json:"timeoutMs,omitempty"`
	// Cancel aborts the running request ID (closes its browser context).
	Cancel bool `json:"cancel,omitempty"`
}

// sessionDeadlineMargin: the server gives up this much before the caller's
// deadline, so its own error (not a bare timeout) reaches the caller.
const sessionDeadlineMargin = 2 * time.Second

// serverTimeoutMs is the server budget for ctx: its deadline minus the margin
// (at least 1 s), or 0 (server default) without a deadline.
func serverTimeoutMs(ctx context.Context) int64 {
	d, ok := ctx.Deadline()
	if !ok {
		return 0
	}
	ms := (time.Until(d) - sessionDeadlineMargin).Milliseconds()
	if ms < 1000 {
		ms = 1000
	}
	return ms
}

type sessionResponse struct {
	ID    int64           `json:"id"`
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// Session returns a shared-browser session backed by this runner.
func (h *HeadlessRunner) Session() *HeadlessSession {
	s := &HeadlessSession{runner: h}
	s.command = func() *exec.Cmd {
		cmd := exec.Command("node", "serve.mjs") //nolint:gosec
		cmd.Dir = h.Dir
		return cmd
	}
	return s
}

// Prepare installs the browser (first use) and starts the server process.
func (s *HeadlessSession) Prepare(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return s.dead
	}
	if s.runner != nil {
		if err := s.runner.Prepare(ctx); err != nil {
			return err
		}
	}
	cmd := s.command()
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = &lockedWriter{mu: &s.mu, buf: &s.stderr}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start headless browser: %w", err)
	}
	s.started = true
	s.cmd = cmd
	s.stdin = stdin
	s.pending = map[int64]chan sessionReply{}
	s.done = make(chan struct{})
	go s.readLoop(stdout)
	return nil
}

func (s *HeadlessSession) readLoop(stdout io.Reader) {
	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	var fatal string
	for sc.Scan() {
		var r sessionResponse
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			continue
		}
		rep := sessionReply{data: r.Data}
		switch {
		case r.Error != nil:
			rep = sessionReply{err: fmt.Errorf("%s: %s", r.Error.Code, r.Error.Message)}
		case len(r.Data) == 0 || string(r.Data) == "null":
			rep = sessionReply{err: errors.New("no data in output")}
		}
		s.mu.Lock()
		ch, ok := s.pending[r.ID]
		delete(s.pending, r.ID)
		s.mu.Unlock()
		if ok {
			ch <- rep
		} else if r.ID == 0 && rep.err != nil {
			fatal = rep.err.Error()
		}
	}
	waitErr := s.cmd.Wait()
	s.mu.Lock()
	msg := fatal
	if msg == "" {
		msg = snippet(s.stderr.Bytes(), 300)
	}
	if msg == "" && waitErr != nil {
		msg = waitErr.Error()
	}
	s.dead = fmt.Errorf("headless browser exited: %s", msg)
	for id, ch := range s.pending {
		ch <- sessionReply{err: s.dead}
		delete(s.pending, id)
	}
	s.mu.Unlock()
	close(s.done)
}

// Extract implements SurfRunner.
func (s *HeadlessSession) Extract(ctx context.Context, pageURL, script string) (json.RawMessage, error) {
	if err := s.Prepare(ctx); err != nil {
		return nil, err
	}
	s.mu.Lock()
	if s.dead != nil {
		s.mu.Unlock()
		return nil, s.dead
	}
	s.nextID++
	id := s.nextID
	ch := make(chan sessionReply, 1)
	s.pending[id] = ch
	line, err := json.Marshal(sessionRequest{ID: id, URL: pageURL, Ready: heurekaReadySelector, Script: script, TimeoutMs: serverTimeoutMs(ctx)})
	if err == nil {
		_, err = s.stdin.Write(append(line, '\n'))
	}
	if err != nil {
		delete(s.pending, id)
		s.mu.Unlock()
		return nil, fmt.Errorf("headless %s: %w", pageURL, err)
	}
	s.mu.Unlock()

	select {
	case r := <-ch:
		if r.err != nil {
			return nil, fmt.Errorf("headless %s: %w", pageURL, r.err)
		}
		return r.data, nil
	case <-ctx.Done():
		s.mu.Lock()
		delete(s.pending, id)
		// Tell the server to close the page: an abandoned page must not keep running.
		if s.dead == nil {
			if cancel, err := json.Marshal(sessionRequest{ID: id, Cancel: true}); err == nil {
				_, _ = s.stdin.Write(append(cancel, '\n'))
			}
		}
		s.mu.Unlock()
		return nil, fmt.Errorf("headless %s: %w", pageURL, ctx.Err())
	}
}

// Close ends the session: the browser finishes open pages and exits.
func (s *HeadlessSession) Close() error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return nil
	}
	_ = s.stdin.Close()
	s.mu.Unlock()
	select {
	case <-s.done:
	case <-time.After(sessionCloseWait):
		_ = s.cmd.Process.Kill()
		<-s.done
	}
	return nil
}

// lockedWriter guards the stderr buffer shared with readLoop.
type lockedWriter struct {
	mu  *sync.Mutex
	buf *bytes.Buffer
}

func (w *lockedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.buf.Len() < 4096 {
		w.buf.Write(p)
	}
	return len(p), nil
}
