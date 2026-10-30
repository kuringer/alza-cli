package client

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// BrowserRunnerEnv selects the market browser runner: "headless" (default) or "surf".
const BrowserRunnerEnv = "ALZA_BROWSER_RUNNER"

// playwrightVersion pins the Playwright package installed into the cache dir.
const playwrightVersion = "1.64.1"

// headlessBootstrapTimeout bounds the first-use npm + Chromium install. It is
// separate from (and much longer than) the per-page deadline.
const headlessBootstrapTimeout = 5 * time.Minute

// heurekaReadySelector: page scripts need one of these server-rendered nodes.
const heurekaReadySelector = "#__NEXT_DATA__, #category-search-state"

//go:embed headless.mjs
var headlessScript []byte

// HeadlessRunner runs page scripts in headless Chromium via Playwright (Node).
// No browser window opens. First use installs Playwright and the Chromium
// headless shell into ~/.cache/alza/headless (~100 MB, ~1 min).
type HeadlessRunner struct {
	Dir      string
	prepared bool
}

// NewHeadlessRunner checks node/npm and returns a runner using the cache dir.
func NewHeadlessRunner() (*HeadlessRunner, error) {
	for _, bin := range []string{"node", "npm", "npx"} {
		if _, err := exec.LookPath(bin); err != nil {
			return nil, fmt.Errorf("%s not found: the Heureka check needs Node.js (or use --no-heureka)", bin)
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return &HeadlessRunner{Dir: filepath.Join(home, ".cache", "alza", "headless")}, nil
}

// Prepare installs Playwright + Chromium on first use. It ignores the caller's
// deadline (a page timeout must not kill a cold install) and applies
// headlessBootstrapTimeout instead; cancellation still propagates.
func (h *HeadlessRunner) Prepare(ctx context.Context) error {
	if h.prepared {
		return nil
	}
	bctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), headlessBootstrapTimeout)
	defer cancel()
	stop := context.AfterFunc(ctx, func() {
		if errors.Is(ctx.Err(), context.Canceled) {
			cancel()
		}
	})
	defer stop()
	if err := h.install(bctx); err != nil {
		return fmt.Errorf("headless browser setup: %w", err)
	}
	h.prepared = true
	return nil
}

func (h *HeadlessRunner) install(ctx context.Context) error {
	marker := filepath.Join(h.Dir, ".ready-"+playwrightVersion)
	if _, err := os.Stat(marker); err == nil {
		return h.writeScripts()
	}
	if err := os.MkdirAll(h.Dir, 0o700); err != nil {
		return err
	}
	pkg := fmt.Sprintf("{\"private\":true,\"type\":\"module\",\"dependencies\":{\"playwright\":%q}}\n", playwrightVersion)
	if err := os.WriteFile(filepath.Join(h.Dir, "package.json"), []byte(pkg), 0o600); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "installing headless Chromium, ~1 min")
	steps := [][]string{
		{"npm", "install", "--silent", "--no-progress", "--no-fund", "--no-audit"},
		{"npx", "playwright", "install", "chromium-headless-shell"},
	}
	for _, s := range steps {
		cmd := exec.CommandContext(ctx, s[0], s[1:]...) //nolint:gosec
		cmd.Dir = h.Dir
		cmd.Env = append(os.Environ(), "npm_config_loglevel=error")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("%s: %w: %s", strings.Join(s, " "), err, snippet(out, 300))
		}
	}
	if err := h.writeScripts(); err != nil {
		return err
	}
	return os.WriteFile(marker, nil, 0o600)
}

// writeScripts refreshes the embedded page runners in the cache dir.
func (h *HeadlessRunner) writeScripts() error {
	if err := os.WriteFile(filepath.Join(h.Dir, "run.mjs"), headlessScript, 0o600); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(h.Dir, "serve.mjs"), headlessServeScript, 0o600)
}

// Extract implements SurfRunner.
func (h *HeadlessRunner) Extract(ctx context.Context, pageURL, script string) (json.RawMessage, error) {
	if err := h.Prepare(ctx); err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, "node", "run.mjs", pageURL, heurekaReadySelector) //nolint:gosec
	cmd.Dir = h.Dir
	cmd.Stdin = strings.NewReader(script)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	data, err := parseSurfOutput(stdout.Bytes())
	if err == nil {
		return data, nil
	}
	if ctx.Err() != nil {
		return nil, fmt.Errorf("headless %s: %w", pageURL, ctx.Err())
	}
	if runErr != nil && stdout.Len() == 0 {
		return nil, fmt.Errorf("headless %s: %v: %s", pageURL, runErr, snippet(stderr.Bytes(), 300))
	}
	return nil, fmt.Errorf("headless %s: %w", pageURL, err)
}

// NewBrowserRunner returns the runner chosen by ALZA_BROWSER_RUNNER.
func NewBrowserRunner() (SurfRunner, error) {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(BrowserRunnerEnv))) {
	case "", "headless":
		return NewHeadlessRunner()
	case "surf":
		return NewExecSurf()
	default:
		return nil, errors.New(BrowserRunnerEnv + " must be headless or surf")
	}
}
