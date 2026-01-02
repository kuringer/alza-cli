package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

const tokenEndpoint = BaseURL + EndpointAccessTokenPath

// RefreshTokenWithCookies fetches a new Bearer token using Chrome cookies.
func RefreshTokenWithCookies(ctx context.Context, cookieHeader string, debug bool) (string, error) {
	return refreshTokenWithCookies(ctx, cookieHeader, debug, tokenEndpoint)
}

func refreshTokenWithCookies(ctx context.Context, cookieHeader string, debug bool, endpoint string) (string, error) {
	cookieHeader = strings.TrimSpace(cookieHeader)
	if cookieHeader == "" {
		return "", errors.New("cookie header missing (are you logged in in Chrome?)")
	}

	jar := tls_client.NewCookieJar()
	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(30),
		tls_client.WithClientProfile(profiles.Chrome_120),
		tls_client.WithCookieJar(jar),
		tls_client.WithRandomTLSExtensionOrder(),
	}

	httpClient, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		return "", fmt.Errorf("failed to create TLS client: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return "", err
	}
	setTokenHeaders(req, cookieHeader)

	if debug {
		fmt.Printf("[DEBUG] GET %s\n", endpoint)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read token response: %w", err)
	}

	if debug {
		contentType := resp.Header.Get("Content-Type")
		fmt.Printf("[DEBUG] Token response: %d %s\n", resp.StatusCode, contentType)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("token endpoint failed: HTTP %d: %s", resp.StatusCode, snippet(body, 200))
	}

	contentType := resp.Header.Get("Content-Type")
	if looksLikeHTML(body, contentType) {
		return "", errors.Join(ErrAuthRequired, fmt.Errorf("token endpoint returned HTML (session missing or expired): %s", snippet(body, 200)))
	}

	payload := map[string]interface{}{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w (%s)", err, snippet(body, 200))
	}

	accessToken := firstString(payload, "accessToken", "AccessToken")
	logOut := firstBool(payload, "logOut", "LogOut")
	if accessToken == "" {
		return "", errors.Join(ErrAuthRequired, errors.New("token response missing accessToken (are you logged in in Chrome?)"))
	}
	if logOut {
		return "", errors.Join(ErrAuthRequired, errors.New("token response returned logOut=true (session expired)"))
	}

	return "Bearer " + accessToken, nil
}

func setTokenHeaders(req *http.Request, cookieHeader string) {
	req.Header = baseHeaders()
	req.Header.Set("Cookie", cookieHeader)
}

func firstString(m map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			if str, ok := value.(string); ok && str != "" {
				return str
			}
		}
	}
	return ""
}

func firstBool(m map[string]interface{}, keys ...string) bool {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			if b, ok := value.(bool); ok {
				return b
			}
		}
	}
	return false
}

func looksLikeHTML(body []byte, contentType string) bool {
	if strings.Contains(strings.ToLower(contentType), "text/html") {
		return true
	}
	trimmed := strings.TrimSpace(string(body))
	low := strings.ToLower(trimmed)
	return strings.HasPrefix(low, "<!doctype") || strings.HasPrefix(low, "<html") || strings.Contains(low, "<html")
}
