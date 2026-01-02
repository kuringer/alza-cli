package client

import (
	"errors"
	"fmt"
	"os"
	"strings"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

const BaseURL = "https://www.alza.sk"

// TLSClient uses tls-client library to bypass Cloudflare
// It stores auth/session context for all API calls.
type TLSClient struct {
	client    tls_client.HttpClient
	authToken string
	userID    string
	basketID  string
	debug     bool
}

// NewTLSClient creates a client with Chrome TLS fingerprint
func NewTLSClient(debug bool) (*TLSClient, error) {
	authTokenPath, err := TokenPath()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve token path: %w", err)
	}
	tokenData, err := os.ReadFile(authTokenPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read auth token from %s: %w\nRun token refresh or `alza token pull --from <ssh-host>` first", authTokenPath, err)
	}
	authToken := strings.TrimSpace(string(tokenData))

	// Create TLS client with Chrome 120 profile
	jar := tls_client.NewCookieJar()
	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(30),
		tls_client.WithClientProfile(profiles.Chrome_120),
		tls_client.WithCookieJar(jar),
		tls_client.WithRandomTLSExtensionOrder(),
	}

	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS client: %w", err)
	}

	c := &TLSClient{
		client:    client,
		authToken: authToken,
		debug:     debug,
	}

	// Validate token by checking user status
	if err := c.validateToken(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *TLSClient) validateToken() error {
	status, err := c.GetUserStatus()
	if err != nil {
		return fmt.Errorf("failed to validate token: %w", err)
	}

	if status.UserID <= 0 {
		return fmt.Errorf("%w\n\n"+
			"╔═══════════════════════════════════════════════════════════╗\n"+
			"║  🔑 TOKEN EXPIROVAL!                                      ║\n"+
			"╠═══════════════════════════════════════════════════════════╣\n"+
			"║  Lokálny refresh (Chrome cookies):                        ║\n"+
			"║  $ alza token refresh                                     ║\n"+
			"║                                                           ║\n"+
			"║  Alebo pull z remote hosta:                               ║\n"+
			"║  $ alza token pull --from <ssh-host>                      ║\n"+
			"║  (Na hoste spusti: alza token refresh)                    ║\n"+
			"╚═══════════════════════════════════════════════════════════╝",
			errors.Join(ErrAuthRequired, ErrTokenExpired))
	}

	return nil
}

func (c *TLSClient) setHeaders(req *http.Request) {
	req.Header = baseHeaders()
	req.Header.Set("Authorization", c.authToken)
}

// Get performs a GET request
func (c *TLSClient) Get(endpoint string) ([]byte, error) {
	return c.doRequest("GET", endpoint, nil, "", "")
}

// Post performs a POST request
func (c *TLSClient) Post(endpoint string, bodyStr string) ([]byte, error) {
	return c.doRequest("POST", endpoint, strings.NewReader(bodyStr), "application/json", bodyStr)
}

// Delete performs a DELETE request
func (c *TLSClient) Delete(endpoint string) ([]byte, error) {
	return c.doRequest("DELETE", endpoint, nil, "", "")
}

// SetUserID sets the user ID
func (c *TLSClient) SetUserID(id string) {
	c.userID = id
}

// SetBasketID sets the basket ID
func (c *TLSClient) SetBasketID(id string) {
	c.basketID = id
}

// GetUserID returns the user ID
func (c *TLSClient) GetUserID() string {
	return c.userID
}

// GetBasketID returns the basket ID
func (c *TLSClient) GetBasketID() string {
	return c.basketID
}
