package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRefreshTokenWithCookiesSuccess(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Cookie"); got == "" {
			t.Errorf("expected cookie header")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"accessToken":"abc","logOut":false}`)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	token, err := refreshTokenWithCookies(ctx, "cf=1", false, server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "Bearer abc" {
		t.Fatalf("unexpected token: %s", token)
	}
}

func TestRefreshTokenWithCookiesHTMLResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, "<html>login</html>")
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := refreshTokenWithCookies(ctx, "cf=1", false, server.URL)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "html") {
		t.Fatalf("expected HTML error, got: %v", err)
	}
}

func TestRefreshTokenWithCookiesLogout(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"accessToken":"abc","logOut":true}`)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := refreshTokenWithCookies(ctx, "cf=1", false, server.URL)
	if err == nil || !strings.Contains(err.Error(), "logOut") {
		t.Fatalf("expected logOut error, got: %v", err)
	}
}

func TestRefreshTokenWithCookiesMissingHeader(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := refreshTokenWithCookies(ctx, "", false, "http://example.test")
	if err == nil {
		t.Fatal("expected error for missing cookie header")
	}
}

func TestRefreshTokenWithCookiesWhitespaceHeader(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := refreshTokenWithCookies(ctx, "   ", false, "http://example.test")
	if err == nil {
		t.Fatal("expected error for whitespace-only cookie header")
	}
}

func TestRefreshTokenWithCookiesMissingAccessToken(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"someOther":"field"}`)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := refreshTokenWithCookies(ctx, "cf=1", false, server.URL)
	if err == nil || !strings.Contains(err.Error(), "accessToken") {
		t.Fatalf("expected missing accessToken error, got: %v", err)
	}
}

func TestRefreshTokenWithCookiesInvalidJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{invalid json}`)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := refreshTokenWithCookies(ctx, "cf=1", false, server.URL)
	if err == nil || !strings.Contains(err.Error(), "decode") {
		t.Fatalf("expected decode error, got: %v", err)
	}
}

func TestRefreshTokenWithCookiesHTTP400(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		_, _ = io.WriteString(w, `Bad Request`)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := refreshTokenWithCookies(ctx, "cf=1", false, server.URL)
	if err == nil || !strings.Contains(err.Error(), "400") {
		t.Fatalf("expected HTTP 400 error, got: %v", err)
	}
}

func TestRefreshTokenWithCookiesUppercaseAccessToken(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"AccessToken":"uppercase","LogOut":false}`)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	token, err := refreshTokenWithCookies(ctx, "cf=1", false, server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "Bearer uppercase" {
		t.Fatalf("unexpected token: %s", token)
	}
}

func TestFirstString(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]interface{}
		keys []string
		want string
	}{
		{
			name: "first key present",
			m:    map[string]interface{}{"a": "value_a", "b": "value_b"},
			keys: []string{"a", "b"},
			want: "value_a",
		},
		{
			name: "second key present",
			m:    map[string]interface{}{"b": "value_b"},
			keys: []string{"a", "b"},
			want: "value_b",
		},
		{
			name: "no keys present",
			m:    map[string]interface{}{"c": "value_c"},
			keys: []string{"a", "b"},
			want: "",
		},
		{
			name: "empty string value",
			m:    map[string]interface{}{"a": "", "b": "value_b"},
			keys: []string{"a", "b"},
			want: "value_b",
		},
		{
			name: "non-string value",
			m:    map[string]interface{}{"a": 123, "b": "value_b"},
			keys: []string{"a", "b"},
			want: "value_b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := firstString(tt.m, tt.keys...)
			if got != tt.want {
				t.Errorf("firstString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFirstBool(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]interface{}
		keys []string
		want bool
	}{
		{
			name: "first key true",
			m:    map[string]interface{}{"a": true, "b": false},
			keys: []string{"a", "b"},
			want: true,
		},
		{
			name: "first key false",
			m:    map[string]interface{}{"a": false, "b": true},
			keys: []string{"a", "b"},
			want: false,
		},
		{
			name: "second key present",
			m:    map[string]interface{}{"b": true},
			keys: []string{"a", "b"},
			want: true,
		},
		{
			name: "no keys present",
			m:    map[string]interface{}{"c": true},
			keys: []string{"a", "b"},
			want: false,
		},
		{
			name: "non-bool value",
			m:    map[string]interface{}{"a": "true", "b": true},
			keys: []string{"a", "b"},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := firstBool(tt.m, tt.keys...)
			if got != tt.want {
				t.Errorf("firstBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLooksLikeHTML(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		contentType string
		want        bool
	}{
		{
			name:        "text/html content type",
			body:        "anything",
			contentType: "text/html; charset=utf-8",
			want:        true,
		},
		{
			name:        "doctype html",
			body:        "<!DOCTYPE html><html>",
			contentType: "application/json",
			want:        true,
		},
		{
			name:        "html tag at start",
			body:        "<html><body>",
			contentType: "",
			want:        true,
		},
		{
			name:        "html tag in body",
			body:        "Some text <html> more",
			contentType: "",
			want:        true,
		},
		{
			name:        "json content",
			body:        `{"key": "value"}`,
			contentType: "application/json",
			want:        false,
		},
		{
			name:        "plain text",
			body:        "Just some text",
			contentType: "text/plain",
			want:        false,
		},
		{
			name:        "uppercase DOCTYPE",
			body:        "<!DOCTYPE HTML>",
			contentType: "",
			want:        true,
		},
		{
			name:        "whitespace before doctype",
			body:        "  <!doctype html>",
			contentType: "",
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := looksLikeHTML([]byte(tt.body), tt.contentType)
			if got != tt.want {
				t.Errorf("looksLikeHTML(%q, %q) = %v, want %v", tt.body, tt.contentType, got, tt.want)
			}
		})
	}
}
