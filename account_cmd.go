package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kuringer/alza-cli/client"
	"github.com/kuringer/alza-cli/internal/chromecookies"
)

func extractBearerToken(output string) (string, error) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Bearer ") && len(line) > 40 {
			return line, nil
		}
	}
	return "", fmt.Errorf("no Bearer token found in SSH output")
}

func expandHomePath(input string) (string, error) {
	if input == "" {
		return "", nil
	}
	if strings.HasPrefix(input, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, input[2:]), nil
	}
	return input, nil
}

func defaultChromeProfile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	customProfile := filepath.Join(home, ".config", "alza", "pw-profile")
	if info, err := os.Stat(customProfile); err == nil && info.IsDir() {
		defaultDir := filepath.Join(customProfile, "Default")
		if info, err := os.Stat(defaultDir); err == nil && info.IsDir() {
			return defaultDir, nil
		}
		return customProfile, nil
	}
	return "Default", nil
}

// === TOKEN ===

type TokenCmd struct {
	Refresh TokenRefreshCmd `cmd:"" help:"Refresh auth token from Chrome cookies"`
	Pull    TokenPullCmd    `cmd:"" help:"Pull auth token from a remote host via SSH"`
}

type TokenRefreshCmd struct {
	ChromeProfile string        `help:"Chrome profile name or path (auto-detect if empty)"`
	CookiePath    string        `help:"Explicit path to Chrome Cookies DB" type:"path"`
	CacheDir      string        `help:"Cache dir for chrome-cookies-secure" default:"~/.cache/alza/chromecookies" type:"path"`
	Timeout       time.Duration `help:"Timeout for cookie read" default:"15s"`
	URL           string        `help:"Target URL to match cookies" default:"https://www.alza.sk/"`
}

func (c *TokenRefreshCmd) Run(g *Globals) error {
	cacheDir, err := expandHomePath(c.CacheDir)
	if err != nil {
		return err
	}
	if cacheDir == "" {
		return fmt.Errorf("cache dir is empty")
	}

	profile := strings.TrimSpace(c.ChromeProfile)
	if profile == "" {
		profile, err = defaultChromeProfile()
		if err != nil {
			return err
		}
	}

	targetURL := strings.TrimSpace(c.URL)
	if targetURL == "" {
		targetURL = client.BaseURL
	}

	opts := chromecookies.Options{
		TargetURL:          targetURL,
		ChromeProfile:      profile,
		ExplicitCookiePath: c.CookiePath,
		CacheDir:           cacheDir,
		Timeout:            c.Timeout,
	}
	if g.Debug {
		opts.LogWriter = os.Stderr
	}

	res, err := chromecookies.LoadCookieHeader(context.Background(), opts)
	if err != nil {
		return formatTokenRefreshError(err, profile, c.CookiePath)
	}
	if strings.TrimSpace(res.CookieHeader) == "" {
		return formatTokenRefreshError(fmt.Errorf("no cookies found for %s (are you logged in in Chrome?)", targetURL), profile, c.CookiePath)
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()

	token, err := client.RefreshTokenWithCookies(ctx, res.CookieHeader, g.Debug)
	if err != nil {
		return formatTokenRefreshError(err, profile, c.CookiePath)
	}
	if err := client.SaveToken(token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	fmt.Printf("✓ Token refreshed from Chrome cookies (%d cookies)\n", res.CookieCount)
	return nil
}

func formatTokenRefreshError(err error, profile, cookiePath string) error {
	msg := err.Error()
	if !needsLoginGuidance(msg) {
		return err
	}

	lines := []string{
		"Login required to refresh token.",
		"",
		"Local (Mac/Linux desktop):",
		"1) Open Chrome/Chromium and sign in to https://www.alza.sk/",
		"2) Run: alza token refresh",
		"",
		"Headless server:",
		"1) Run: ./scripts/remote-login.sh",
		"2) Connect via VNC and sign in",
		"3) Run: alza token refresh --chrome-profile ~/.config/alza/pw-profile",
	}

	if profile != "" {
		lines = append(lines, "", "Detected profile:", profile)
	}
	if cookiePath != "" {
		lines = append(lines, "", "Explicit cookie path:", cookiePath)
	}

	lines = append(lines, "", "Original error:", msg)
	return fmt.Errorf("%s", strings.Join(lines, "\n"))
}

func needsLoginGuidance(msg string) bool {
	needles := []string{
		"No Cookies DB found",
		"no cookies found",
		"cookie header missing",
		"token endpoint returned HTML",
		"missing accessToken",
		"session expired",
	}
	for _, needle := range needles {
		if strings.Contains(strings.ToLower(msg), strings.ToLower(needle)) {
			return true
		}
	}
	return false
}

type TokenPullCmd struct {
	From       string        `help:"SSH host (from ~/.ssh/config or user@host)"`
	RemotePath string        `help:"Remote auth_token.txt path" default:"~/.config/alza/auth_token.txt"`
	Timeout    time.Duration `help:"SSH timeout" default:"15s"`
}

func (c *TokenPullCmd) Run(g *Globals) error {
	if c.From == "" {
		return fmt.Errorf("missing --from (SSH host)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()

	var stdout bytes.Buffer
	cmd := exec.CommandContext(ctx, "ssh", c.From, "cat", "--", c.RemotePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("ssh timed out after %s", c.Timeout)
		}
		return fmt.Errorf("ssh failed: %w", err)
	}

	token, err := extractBearerToken(stdout.String())
	if err != nil {
		return err
	}

	if err := client.SaveToken(token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	fmt.Println("✓ Token pulled and saved to ~/.config/alza/auth_token.txt")
	return nil
}

// === WHOAMI ===

type WhoamiCmd struct{}

func (c *WhoamiCmd) Run(g *Globals) error {
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}

	status, err := cl.GetUserStatus()
	if err != nil {
		return err
	}

	if g.Format == "json" {
		outputJSON(status)
		return nil
	}

	fmt.Printf("User: %s (ID: %d)\n", status.UserName, status.UserID)
	fmt.Printf("Basket ID: %d\n", status.BasketID)
	fmt.Printf("Cart items: %d\n", status.BasketCnt)
	fmt.Printf("Orders: %d\n", status.OrdersCnt)
	if status.IsPremium {
		fmt.Println("Status: AlzaPlus+ member ✓")
	}

	return nil
}
