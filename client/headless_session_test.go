package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestHeadlessSessionHelper is the fake serve.mjs: it is run as a subprocess.
func TestHeadlessSessionHelper(t *testing.T) {
	if os.Getenv("ALZA_SESSION_HELPER") != "1" {
		t.Skip("helper process")
	}
	sc := bufio.NewScanner(os.Stdin)
	var mu sync.Mutex
	var wg sync.WaitGroup
	for sc.Scan() {
		var req sessionRequest
		if err := json.Unmarshal(sc.Bytes(), &req); err != nil {
			continue
		}
		switch req.URL {
		case "die":
			fmt.Fprintln(os.Stderr, "browser crashed")
			os.Exit(1)
		case "hang":
			continue
		}
		wg.Add(1)
		go func(req sessionRequest) {
			defer wg.Done()
			// Answer later requests first to exercise id routing.
			time.Sleep(time.Duration(10-req.ID) * 5 * time.Millisecond)
			var line []byte
			if req.URL == "fail" {
				line, _ = json.Marshal(map[string]any{"id": req.ID, "error": map[string]string{"code": "headless_error", "message": "boom"}})
			} else {
				line, _ = json.Marshal(map[string]any{"id": req.ID, "data": map[string]string{"url": req.URL, "script": req.Script}})
			}
			mu.Lock()
			fmt.Println(string(line))
			mu.Unlock()
		}(req)
	}
	wg.Wait()
	os.Exit(0)
}

func helperSession() *HeadlessSession {
	return &HeadlessSession{command: func() *exec.Cmd {
		cmd := exec.Command(os.Args[0], "-test.run=TestHeadlessSessionHelper") //nolint:gosec
		cmd.Env = append(os.Environ(), "ALZA_SESSION_HELPER=1")
		return cmd
	}}
}

func TestHeadlessSessionRoutesConcurrentRequests(t *testing.T) {
	s := helperSession()
	defer s.Close()

	var wg sync.WaitGroup
	errs := make([]error, 6)
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			u := fmt.Sprintf("https://example.com/%d", i)
			data, err := s.Extract(context.Background(), u, "return 1;")
			if err != nil {
				errs[i] = err
				return
			}
			var got struct{ URL string }
			if err := json.Unmarshal(data, &got); err != nil || got.URL != u {
				errs[i] = fmt.Errorf("reply %s for %s (%v)", data, u, err)
			}
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Errorf("request %d: %v", i, err)
		}
	}

	if _, err := s.Extract(context.Background(), "fail", ""); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("fail: err = %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := s.Extract(ctx, "hang", ""); err == nil || !strings.Contains(err.Error(), "deadline") {
		t.Errorf("hang: err = %v", err)
	}
}

func TestHeadlessSessionProcessDeath(t *testing.T) {
	s := helperSession()
	defer s.Close()
	if _, err := s.Extract(context.Background(), "die", ""); err == nil || !strings.Contains(err.Error(), "browser crashed") {
		t.Fatalf("die: err = %v", err)
	}
	if _, err := s.Extract(context.Background(), "https://example.com/", ""); err == nil || !strings.Contains(err.Error(), "exited") {
		t.Errorf("after death: err = %v", err)
	}
}

func TestHeadlessSessionCloseUnstarted(t *testing.T) {
	if err := helperSession().Close(); err != nil {
		t.Error(err)
	}
}
