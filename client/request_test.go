package client

import (
	"testing"
)

func TestResolveURL(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		want     string
		wantErr  bool
	}{
		{
			name:     "relative endpoint",
			endpoint: "/api/v1/users",
			want:     BaseURL + "/api/v1/users",
			wantErr:  false,
		},
		{
			name:     "absolute https URL",
			endpoint: "https://webapi.alza.cz/api/search",
			want:     "https://webapi.alza.cz/api/search",
			wantErr:  false,
		},
		{
			name:     "absolute http URL",
			endpoint: "http://example.com/api",
			want:     "http://example.com/api",
			wantErr:  false,
		},
		{
			name:     "empty endpoint",
			endpoint: "",
			want:     BaseURL,
			wantErr:  false,
		},
		{
			name:     "endpoint with query params",
			endpoint: "/api/search?q=test&limit=10",
			want:     BaseURL + "/api/search?q=test&limit=10",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveURL(tt.endpoint)
			if (err != nil) != tt.wantErr {
				t.Errorf("resolveURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("resolveURL(%q) = %q, want %q", tt.endpoint, got, tt.want)
			}
		})
	}
}

func TestHTTPError(t *testing.T) {
	tests := []struct {
		name       string
		err        HTTPError
		wantString string
	}{
		{
			name: "error with body",
			err: HTTPError{
				Status: 404,
				URL:    "https://example.com/api",
				Body:   "Not Found",
			},
			wantString: "HTTP 404: Not Found",
		},
		{
			name: "error without body",
			err: HTTPError{
				Status: 500,
				URL:    "https://example.com/api",
				Body:   "",
			},
			wantString: "HTTP 500",
		},
		{
			name: "auth error",
			err: HTTPError{
				Status: 401,
				URL:    "https://example.com/api",
				Body:   "Unauthorized",
			},
			wantString: "HTTP 401: Unauthorized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.wantString {
				t.Errorf("HTTPError.Error() = %q, want %q", got, tt.wantString)
			}
		})
	}
}

func TestBaseURLConstant(t *testing.T) {
	if BaseURL != "https://www.alza.sk" {
		t.Errorf("BaseURL = %q, want https://www.alza.sk", BaseURL)
	}
}
