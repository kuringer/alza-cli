package chromecookies

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadCookieHeaderMissingTargetURL(t *testing.T) {
	t.Parallel()

	_, err := LoadCookieHeader(context.Background(), Options{CacheDir: t.TempDir()})
	if err == nil {
		t.Fatal("expected error for missing target URL")
	}
}

func TestLoadCookieHeaderMissingCacheDir(t *testing.T) {
	t.Parallel()

	_, err := LoadCookieHeader(context.Background(), Options{TargetURL: "https://www.alza.sk/"})
	if err == nil {
		t.Fatal("expected error for missing cache dir")
	}
}

func TestLoadCookieHeaderSuccess(t *testing.T) {
	cacheDir := setupCacheDir(t)

	orig := runScript
	runScript = func(ctx context.Context, cacheDir, scriptPath, outPath string, input []byte, logWriter io.Writer, timeout time.Duration) (scriptOutput, error) {
		return scriptOutput{CookieHeader: "a=1; b=2", CookieCount: 2}, nil
	}
	defer func() { runScript = orig }()

	res, err := LoadCookieHeader(context.Background(), Options{
		TargetURL: "https://www.alza.sk/",
		CacheDir:  cacheDir,
		Timeout:   5 * time.Second,
	})
	if err != nil {
		t.Fatalf("LoadCookieHeader error: %v", err)
	}
	if res.CookieHeader != "a=1; b=2" || res.CookieCount != 2 {
		t.Fatalf("unexpected result: %#v", res)
	}
}

func TestLoadCookieHeaderScriptError(t *testing.T) {
	cacheDir := setupCacheDir(t)

	orig := runScript
	runScript = func(ctx context.Context, cacheDir, scriptPath, outPath string, input []byte, logWriter io.Writer, timeout time.Duration) (scriptOutput, error) {
		return scriptOutput{Error: "boom"}, nil
	}
	defer func() { runScript = orig }()

	_, err := LoadCookieHeader(context.Background(), Options{
		TargetURL: "https://www.alza.sk/",
		CacheDir:  cacheDir,
	})
	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected script error, got: %v", err)
	}
}

func setupCacheDir(t *testing.T) string {
	t.Helper()

	cacheDir := t.TempDir()
	modDir := filepath.Join(cacheDir, "node_modules", "chrome-cookies-secure")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "package.json"), []byte("{}"), 0o600); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	return cacheDir
}
