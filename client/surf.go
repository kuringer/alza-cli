package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SurfBinEnv overrides the surf CLI binary (default "surf" on PATH).
const SurfBinEnv = "ALZA_SURF_BIN"

// ErrSurfMissing means the surf CLI is not installed or not on PATH.
var ErrSurfMissing = errors.New("surf CLI not found")

// SurfRunner opens a URL in a real browser, runs a page script and returns
// the JSON the script returned. Heureka blocks non-browser HTTP clients, so
// market lookups go through the user's browser via surf.
type SurfRunner interface {
	Extract(ctx context.Context, url, script string) (json.RawMessage, error)
}

// ExecSurf runs `surf extract` as a subprocess.
type ExecSurf struct {
	Bin string
}

// NewExecSurf resolves the surf binary from ALZA_SURF_BIN or PATH.
func NewExecSurf() (*ExecSurf, error) {
	bin := strings.TrimSpace(os.Getenv(SurfBinEnv))
	if bin == "" {
		bin = "surf"
	}
	path, err := exec.LookPath(bin)
	if err != nil {
		return nil, fmt.Errorf("%w (%q): install surf (browser automation CLI) or set %s; use --no-heureka to skip the market check", ErrSurfMissing, bin, SurfBinEnv)
	}
	return &ExecSurf{Bin: path}, nil
}

type surfExtractOutput struct {
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// Extract implements SurfRunner.
func (s *ExecSurf) Extract(ctx context.Context, url, script string) (json.RawMessage, error) {
	cmd := exec.CommandContext(ctx, s.Bin, "extract", url, "--code", script, "--json", "--allow-empty")
	// surf is a node script (#!/usr/bin/env node); keep its bin dir on PATH so
	// the matching node is found in clean shells (nvm installs).
	cmd.Env = append(os.Environ(), "PATH="+filepath.Dir(s.Bin)+string(os.PathListSeparator)+os.Getenv("PATH"))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()

	data, err := parseSurfOutput(stdout.Bytes())
	if err == nil {
		return data, nil
	}
	if ctx.Err() != nil {
		return nil, fmt.Errorf("surf extract %s: %w", url, ctx.Err())
	}
	if runErr != nil && stdout.Len() == 0 {
		return nil, fmt.Errorf("surf extract %s: %v: %s", url, runErr, snippet(stderr.Bytes(), 300))
	}
	return nil, fmt.Errorf("surf extract %s: %w", url, err)
}

func parseSurfOutput(out []byte) (json.RawMessage, error) {
	if len(bytes.TrimSpace(out)) == 0 {
		return nil, errors.New("empty output")
	}
	var parsed surfExtractOutput
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil, fmt.Errorf("unexpected output: %s", snippet(out, 200))
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("%s: %s", parsed.Error.Code, parsed.Error.Message)
	}
	if len(parsed.Data) == 0 {
		return nil, errors.New("no data in output")
	}
	return parsed.Data, nil
}
