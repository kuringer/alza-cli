package client

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	http "github.com/bogdanfinn/fhttp"
)

type HTTPError struct {
	Status int
	URL    string
	Body   string
}

func (e *HTTPError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("HTTP %d", e.Status)
	}
	return fmt.Sprintf("HTTP %d: %s", e.Status, e.Body)
}

func (c *TLSClient) doRequest(method, endpoint string, body io.Reader, contentType, debugBody string) ([]byte, error) {
	urlStr, err := resolveURL(endpoint)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	if c.debug {
		fmt.Printf("[DEBUG] %s %s\n", method, urlStr)
		if debugBody != "" {
			fmt.Printf("[DEBUG] Body: %s\n", snippet([]byte(debugBody), 500))
		}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if c.debug {
		fmt.Printf("[DEBUG] Response: %d %s\n", resp.StatusCode, snippet(bodyBytes, 500))
	}

	if resp.StatusCode >= 400 {
		httpErr := &HTTPError{
			Status: resp.StatusCode,
			URL:    urlStr,
			Body:   snippet(bodyBytes, 500),
		}
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			return nil, errors.Join(ErrAuthRequired, httpErr)
		}
		return nil, httpErr
	}

	return bodyBytes, nil
}

func resolveURL(endpoint string) (string, error) {
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		_, err := url.Parse(endpoint)
		return endpoint, err
	}
	return BaseURL + endpoint, nil
}
