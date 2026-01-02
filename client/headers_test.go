package client

import (
	"runtime"
	"strings"
	"testing"
)

func TestBaseHeaders(t *testing.T) {
	headers := baseHeaders()

	requiredHeaders := []string{
		"User-Agent",
		"Accept",
		"Accept-Language",
		"Referer",
		"Origin",
		"Sec-Fetch-Dest",
		"Sec-Fetch-Mode",
		"Sec-Fetch-Site",
		"Sec-Ch-Ua",
		"Sec-Ch-Ua-Mobile",
		"Sec-Ch-Ua-Platform",
	}

	for _, header := range requiredHeaders {
		if headers.Get(header) == "" {
			t.Errorf("baseHeaders() missing header %q", header)
		}
	}
}

func TestBaseHeadersAccept(t *testing.T) {
	headers := baseHeaders()
	got := headers.Get("Accept")
	if got != acceptHeader {
		t.Errorf("Accept header = %q, want %q", got, acceptHeader)
	}
}

func TestBaseHeadersReferer(t *testing.T) {
	headers := baseHeaders()
	got := headers.Get("Referer")
	if got != "https://www.alza.sk/" {
		t.Errorf("Referer header = %q, want https://www.alza.sk/", got)
	}
}

func TestBaseHeadersOrigin(t *testing.T) {
	headers := baseHeaders()
	got := headers.Get("Origin")
	if got != "https://www.alza.sk" {
		t.Errorf("Origin header = %q, want https://www.alza.sk", got)
	}
}

func TestUserAgent(t *testing.T) {
	ua := userAgent()

	if !strings.Contains(ua, "Mozilla/5.0") {
		t.Errorf("userAgent() = %q, expected to contain Mozilla/5.0", ua)
	}

	if !strings.Contains(ua, "Chrome/120") {
		t.Errorf("userAgent() = %q, expected to contain Chrome/120", ua)
	}

	// Check OS-specific
	switch runtime.GOOS {
	case "linux":
		if !strings.Contains(ua, "Linux") {
			t.Errorf("userAgent() on Linux = %q, expected to contain Linux", ua)
		}
	case "windows":
		if !strings.Contains(ua, "Windows") {
			t.Errorf("userAgent() on Windows = %q, expected to contain Windows", ua)
		}
	case "darwin":
		if !strings.Contains(ua, "Macintosh") {
			t.Errorf("userAgent() on macOS = %q, expected to contain Macintosh", ua)
		}
	}
}

func TestSecCHPlatform(t *testing.T) {
	platform := secCHPlatform()

	switch runtime.GOOS {
	case "linux":
		if platform != `"Linux"` {
			t.Errorf("secCHPlatform() on Linux = %q, want \"Linux\"", platform)
		}
	case "windows":
		if platform != `"Windows"` {
			t.Errorf("secCHPlatform() on Windows = %q, want \"Windows\"", platform)
		}
	case "darwin":
		if platform != `"macOS"` {
			t.Errorf("secCHPlatform() on macOS = %q, want \"macOS\"", platform)
		}
	}
}

func TestSecCHUA(t *testing.T) {
	if secCHUA != `"Chromium";v="120", "Not A(Brand";v="24"` {
		t.Errorf("secCHUA = %q, expected Chrome 120 value", secCHUA)
	}
}
