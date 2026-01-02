package client

import (
	"testing"
)

func TestExtractProductID(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want int
	}{
		{
			name: "standard URL with -d suffix",
			url:  "https://www.alza.sk/apple-iphone-15-pro-max-256gb-d7630088.htm",
			want: 7630088,
		},
		{
			name: "URL with /d prefix",
			url:  "https://www.alza.sk/EN/apple-macbook-air-m2-2022/d7300001.htm",
			want: 7300001,
		},
		{
			name: "URL with uppercase -D",
			url:  "https://www.alza.sk/product-D12345.htm",
			want: 12345,
		},
		{
			name: "empty string",
			url:  "",
			want: 0,
		},
		{
			name: "no product ID",
			url:  "https://www.alza.sk/category/phones",
			want: 0,
		},
		{
			name: "malformed URL",
			url:  "not-a-url",
			want: 0,
		},
		{
			name: "relative URL",
			url:  "/apple-iphone-d123456.htm",
			want: 123456,
		},
		{
			name: "large product ID",
			url:  "/product-d999999999.htm",
			want: 999999999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractProductID(tt.url)
			if got != tt.want {
				t.Errorf("extractProductID(%q) = %d, want %d", tt.url, got, tt.want)
			}
		})
	}
}

func TestParsePrice(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want float64
	}{
		{
			name: "price with comma",
			raw:  "1 299,90 €",
			want: 1299.90,
		},
		{
			name: "price with dot",
			raw:  "1299.90",
			want: 1299.90,
		},
		{
			name: "price without decimals",
			raw:  "1299 €",
			want: 1299,
		},
		{
			name: "price with currency symbol (European format)",
			raw:  "€ 1299,90",
			want: 1299.90,
		},
		{
			name: "empty string",
			raw:  "",
			want: 0,
		},
		{
			name: "no digits",
			raw:  "Price unknown",
			want: 0,
		},
		{
			name: "simple number",
			raw:  "99",
			want: 99,
		},
		{
			name: "decimal only",
			raw:  "0,99",
			want: 0.99,
		},
		{
			name: "thousands separator and decimal",
			raw:  "12 345,67",
			want: 12345.67,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePrice(tt.raw)
			if got != tt.want {
				t.Errorf("parsePrice(%q) = %f, want %f", tt.raw, got, tt.want)
			}
		})
	}
}

func TestSnippet(t *testing.T) {
	tests := []struct {
		name  string
		body  string
		limit int
		want  string
	}{
		{
			name:  "within limit",
			body:  "short text",
			limit: 100,
			want:  "short text",
		},
		{
			name:  "exceeds limit",
			body:  "this is a long text that should be truncated",
			limit: 20,
			want:  "this is a long text ...",
		},
		{
			name:  "exact limit",
			body:  "exact",
			limit: 5,
			want:  "exact",
		},
		{
			name:  "empty body",
			body:  "",
			limit: 100,
			want:  "",
		},
		{
			name:  "zero limit",
			body:  "some text",
			limit: 0,
			want:  "",
		},
		{
			name:  "negative limit",
			body:  "some text",
			limit: -5,
			want:  "",
		},
		{
			name:  "body with whitespace",
			body:  "  trimmed  ",
			limit: 100,
			want:  "trimmed",
		},
		{
			name:  "body with newlines",
			body:  "line1\nline2",
			limit: 100,
			want:  "line1\nline2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snippet([]byte(tt.body), tt.limit)
			if got != tt.want {
				t.Errorf("snippet(%q, %d) = %q, want %q", tt.body, tt.limit, got, tt.want)
			}
		})
	}
}

func TestExtractProductIDMoreCases(t *testing.T) {
	// Test edge cases for extractProductID regex
	tests := []struct {
		name string
		url  string
		want int
	}{
		{
			name: "URL with query params",
			url:  "/product-d12345.htm?ref=search",
			want: 12345,
		},
		{
			name: "URL with anchor",
			url:  "/product-d12345.htm#section",
			want: 12345,
		},
		{
			name: "multiple d patterns - takes first",
			url:  "/product-d111.htm/related-d222.htm",
			want: 111,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractProductID(tt.url)
			if got != tt.want {
				t.Errorf("extractProductID(%q) = %d, want %d", tt.url, got, tt.want)
			}
		})
	}
}

func TestParsePriceEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want float64
	}{
		{
			name: "multiple dots (becomes invalid after comma->dot replacement)",
			raw:  "1.234.567",
			want: 0, // parsePrice replaces comma with dot, so "1.234.567" stays as is and fails ParseFloat
		},
		{
			name: "just comma",
			raw:  ",",
			want: 0,
		},
		{
			name: "just dot",
			raw:  ".",
			want: 0,
		},
		{
			name: "leading zeros",
			raw:  "0,99",
			want: 0.99,
		},
		{
			name: "trailing comma",
			raw:  "100,",
			want: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePrice(tt.raw)
			if got != tt.want {
				t.Errorf("parsePrice(%q) = %f, want %f", tt.raw, got, tt.want)
			}
		})
	}
}
