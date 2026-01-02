package client

import (
	"runtime"

	http "github.com/bogdanfinn/fhttp"
)

const (
	secCHUA      = `"Chromium";v="120", "Not A(Brand";v="24"`
	acceptHeader = "application/json, text/plain, */*"
)

func baseHeaders() http.Header {
	return http.Header{
		"User-Agent":         {userAgent()},
		"Accept":             {acceptHeader},
		"Accept-Language":    {"sk-SK"},
		"Referer":            {"https://www.alza.sk/"},
		"Origin":             {"https://www.alza.sk"},
		"Sec-Fetch-Dest":     {"empty"},
		"Sec-Fetch-Mode":     {"cors"},
		"Sec-Fetch-Site":     {"same-origin"},
		"Sec-Ch-Ua":          {secCHUA},
		"Sec-Ch-Ua-Mobile":   {"?0"},
		"Sec-Ch-Ua-Platform": {secCHPlatform()},
	}
}

func userAgent() string {
	switch runtime.GOOS {
	case "linux":
		return "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	case "windows":
		return "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	default:
		return "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	}
}

func secCHPlatform() string {
	switch runtime.GOOS {
	case "linux":
		return `"Linux"`
	case "windows":
		return `"Windows"`
	default:
		return `"macOS"`
	}
}
