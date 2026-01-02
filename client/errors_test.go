package client

import (
	"errors"
	"testing"
)

func TestErrorConstants(t *testing.T) {
	if ErrAuthRequired == nil {
		t.Error("ErrAuthRequired should not be nil")
	}
	if ErrTokenExpired == nil {
		t.Error("ErrTokenExpired should not be nil")
	}
}

func TestErrAuthRequiredMessage(t *testing.T) {
	if ErrAuthRequired.Error() != "auth required" {
		t.Errorf("ErrAuthRequired.Error() = %q, want %q", ErrAuthRequired.Error(), "auth required")
	}
}

func TestErrTokenExpiredMessage(t *testing.T) {
	if ErrTokenExpired.Error() != "auth token expired or invalid" {
		t.Errorf("ErrTokenExpired.Error() = %q, want %q", ErrTokenExpired.Error(), "auth token expired or invalid")
	}
}

func TestErrorsCanBeWrapped(t *testing.T) {
	wrapped := errors.Join(ErrAuthRequired, ErrTokenExpired)

	if !errors.Is(wrapped, ErrAuthRequired) {
		t.Error("wrapped error should contain ErrAuthRequired")
	}
	if !errors.Is(wrapped, ErrTokenExpired) {
		t.Error("wrapped error should contain ErrTokenExpired")
	}
}

func TestHTTPErrorWithAuth(t *testing.T) {
	httpErr := &HTTPError{Status: 401, URL: "https://example.com", Body: "Unauthorized"}
	combined := errors.Join(ErrAuthRequired, httpErr)

	if !errors.Is(combined, ErrAuthRequired) {
		t.Error("combined error should contain ErrAuthRequired")
	}

	var gotHTTPErr *HTTPError
	if !errors.As(combined, &gotHTTPErr) {
		t.Error("combined error should contain HTTPError")
	}
	if gotHTTPErr.Status != 401 {
		t.Errorf("HTTPError.Status = %d, want 401", gotHTTPErr.Status)
	}
}
