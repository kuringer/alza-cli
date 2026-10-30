package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/kuringer/alza-cli/client"
	"github.com/kuringer/alza-cli/internal/chromecookies"
)

// Globals contains shared configuration
type Globals struct {
	Format string `help:"Output format (text|json)" enum:"text,json" default:"text"`
	Debug  bool   `help:"Enable debug mode" short:"d"`
}

// CLI is the main command structure
var CLI struct {
	Globals

	Whoami    WhoamiCmd    `cmd:"" help:"Show logged in user info"`
	Search    SearchCmd    `cmd:"" help:"Search for products"`
	Product   ProductCmd   `cmd:"" help:"Show product detail"`
	Reviews   ReviewsCmd   `cmd:"" help:"Show product reviews"`
	Cart      CartCmd      `cmd:"" help:"Manage shopping cart"`
	Favorites FavoritesCmd `cmd:"" help:"Manage favorites list"`
	Lists     ListsCmd     `cmd:"" help:"Manage commodity lists"`
	Orders    OrdersCmd    `cmd:"" help:"View order history"`
	Quickbuy  QuickbuyCmd  `cmd:"" help:"Quick order to AlzaBox (WILL CHARGE YOUR CARD!)"`
	Token     TokenCmd     `cmd:"" help:"Manage auth token"`
	Version   VersionCmd   `cmd:"" help:"Show version info"`
}

// VersionCmd shows version information
type VersionCmd struct{}

func (c *VersionCmd) Run(g *Globals) error {
	fmt.Printf("alza %s\n", client.Version)
	return nil
}

func newClient(g *Globals) (*client.TLSClient, error) {
	return client.NewTLSClient(g.Debug)
}

// newClientWithAutoRefresh creates a client, auto-refreshing token if expired
func newClientWithAutoRefresh(g *Globals) (*client.TLSClient, error) {
	cl, err := client.NewTLSClient(g.Debug)
	if err == nil {
		return cl, nil
	}

	// Check if it's a token expiration error
	if !isTokenExpiredError(err) {
		return nil, err
	}

	fmt.Println("🔄 Token expiroval, skúšam automatický refresh...")

	// Try to refresh token
	if refreshErr := doAutoRefresh(g); refreshErr != nil {
		// Return original error with refresh failure info
		return nil, fmt.Errorf("%w\n\nAuto-refresh zlyhal: %v", err, refreshErr)
	}

	fmt.Println("✓ Token refreshnutý, pokračujem...")

	// Retry with new token
	return client.NewTLSClient(g.Debug)
}

func isTokenExpiredError(err error) bool {
	// Use sentinel errors from client package
	if errors.Is(err, client.ErrTokenExpired) || errors.Is(err, client.ErrAuthRequired) {
		return true
	}
	// Fallback: check error message for TOKEN EXPIROVAL (from validateToken)
	return strings.Contains(err.Error(), "TOKEN EXPIROVAL")
}

func doAutoRefresh(g *Globals) error {
	cacheDir, err := expandHomePath("~/.cache/alza/chromecookies")
	if err != nil {
		return err
	}

	profile, err := defaultChromeProfile()
	if err != nil {
		return fmt.Errorf("nepodarilo sa zistiť Chrome profil: %w", err)
	}

	opts := chromecookies.Options{
		TargetURL:     client.BaseURL,
		ChromeProfile: profile,
		CacheDir:      cacheDir,
		Timeout:       15 * time.Second,
	}
	if g.Debug {
		opts.LogWriter = os.Stderr
	}

	res, err := chromecookies.LoadCookieHeader(context.Background(), opts)
	if err != nil {
		return fmt.Errorf("nepodarilo sa načítať cookies: %w", err)
	}
	if strings.TrimSpace(res.CookieHeader) == "" {
		return fmt.Errorf("žiadne cookies (si prihlásený v Chrome?)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	token, err := client.RefreshTokenWithCookies(ctx, res.CookieHeader, g.Debug)
	if err != nil {
		return err
	}

	return client.SaveToken(token)
}

func outputJSON(v interface{}) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}

func main() {
	ctx := kong.Parse(&CLI,
		kong.Name("alza"),
		kong.Description("CLI for Alza.sk - search products, manage cart and favorites"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}),
		kong.Vars{"version": client.Version},
	)

	err := ctx.Run(&CLI.Globals)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Check for updates (async-cached, won't slow down)
	if msg := client.CheckForUpdate(); msg != "" {
		fmt.Fprint(os.Stderr, msg)
	}
}
