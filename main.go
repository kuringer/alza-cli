package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	Cart      CartCmd      `cmd:"" help:"Manage shopping cart"`
	Favorites FavoritesCmd `cmd:"" help:"Manage favorites list"`
	Lists     ListsCmd     `cmd:"" help:"Manage commodity lists"`
	Orders    OrdersCmd    `cmd:"" help:"View order history"`
	Quickbuy  QuickbuyCmd  `cmd:"" help:"Quick order to AlzaBox (WILL CHARGE YOUR CARD!)"`
	Token     TokenCmd     `cmd:"" help:"Manage auth token"`
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

func resolveFavoritesList(cl *client.TLSClient) (*client.CommodityList, error) {
	lists, err := cl.GetLists()
	if err != nil {
		return nil, err
	}

	names := []string{}
	if envName := strings.TrimSpace(os.Getenv("ALZA_FAVORITES_LIST")); envName != "" {
		names = append(names, envName)
	}
	names = append(names, "AGENT", "AGENTS")

	for _, name := range names {
		for _, l := range lists {
			if strings.EqualFold(l.Name, name) {
				return &l, nil
			}
		}
	}

	return nil, fmt.Errorf("favorites list not found (set ALZA_FAVORITES_LIST or create list named AGENT)")
}

func extractBearerToken(output string) (string, error) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Bearer ") && len(line) > 40 {
			return line, nil
		}
	}
	return "", fmt.Errorf("no Bearer token found in SSH output")
}

func expandHomePath(input string) (string, error) {
	if input == "" {
		return "", nil
	}
	if strings.HasPrefix(input, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, input[2:]), nil
	}
	return input, nil
}

func defaultChromeProfile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	customProfile := filepath.Join(home, ".config", "alza", "pw-profile")
	if info, err := os.Stat(customProfile); err == nil && info.IsDir() {
		defaultDir := filepath.Join(customProfile, "Default")
		if info, err := os.Stat(defaultDir); err == nil && info.IsDir() {
			return defaultDir, nil
		}
		return customProfile, nil
	}
	return "Default", nil
}

// === TOKEN ===

type TokenCmd struct {
	Refresh TokenRefreshCmd `cmd:"" help:"Refresh auth token from Chrome cookies"`
	Pull    TokenPullCmd    `cmd:"" help:"Pull auth token from a remote host via SSH"`
}

type TokenRefreshCmd struct {
	ChromeProfile string        `help:"Chrome profile name or path (auto-detect if empty)"`
	CookiePath    string        `help:"Explicit path to Chrome Cookies DB" type:"path"`
	CacheDir      string        `help:"Cache dir for chrome-cookies-secure" default:"~/.cache/alza/chromecookies" type:"path"`
	Timeout       time.Duration `help:"Timeout for cookie read" default:"15s"`
	URL           string        `help:"Target URL to match cookies" default:"https://www.alza.sk/"`
}

func (c *TokenRefreshCmd) Run(g *Globals) error {
	cacheDir, err := expandHomePath(c.CacheDir)
	if err != nil {
		return err
	}
	if cacheDir == "" {
		return fmt.Errorf("cache dir is empty")
	}

	profile := strings.TrimSpace(c.ChromeProfile)
	if profile == "" {
		profile, err = defaultChromeProfile()
		if err != nil {
			return err
		}
	}

	targetURL := strings.TrimSpace(c.URL)
	if targetURL == "" {
		targetURL = client.BaseURL
	}

	opts := chromecookies.Options{
		TargetURL:          targetURL,
		ChromeProfile:      profile,
		ExplicitCookiePath: c.CookiePath,
		CacheDir:           cacheDir,
		Timeout:            c.Timeout,
	}
	if g.Debug {
		opts.LogWriter = os.Stderr
	}

	res, err := chromecookies.LoadCookieHeader(context.Background(), opts)
	if err != nil {
		return formatTokenRefreshError(err, profile, c.CookiePath)
	}
	if strings.TrimSpace(res.CookieHeader) == "" {
		return formatTokenRefreshError(fmt.Errorf("no cookies found for %s (are you logged in in Chrome?)", targetURL), profile, c.CookiePath)
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()

	token, err := client.RefreshTokenWithCookies(ctx, res.CookieHeader, g.Debug)
	if err != nil {
		return formatTokenRefreshError(err, profile, c.CookiePath)
	}
	if err := client.SaveToken(token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	fmt.Printf("✓ Token refreshed from Chrome cookies (%d cookies)\n", res.CookieCount)
	return nil
}

func formatTokenRefreshError(err error, profile, cookiePath string) error {
	msg := err.Error()
	if !needsLoginGuidance(msg) {
		return err
	}

	lines := []string{
		"Login required to refresh token.",
		"",
		"Local (Mac/Linux desktop):",
		"1) Open Chrome/Chromium and sign in to https://www.alza.sk/",
		"2) Run: alza token refresh",
		"",
		"Headless server:",
		"1) Run: ./scripts/remote-login.sh",
		"2) Connect via VNC and sign in",
		"3) Run: alza token refresh --chrome-profile ~/.config/alza/pw-profile",
	}

	if profile != "" {
		lines = append(lines, "", "Detected profile:", profile)
	}
	if cookiePath != "" {
		lines = append(lines, "", "Explicit cookie path:", cookiePath)
	}

	lines = append(lines, "", "Original error:", msg)
	return fmt.Errorf("%s", strings.Join(lines, "\n"))
}

func needsLoginGuidance(msg string) bool {
	needles := []string{
		"No Cookies DB found",
		"no cookies found",
		"cookie header missing",
		"token endpoint returned HTML",
		"missing accessToken",
		"session expired",
	}
	for _, needle := range needles {
		if strings.Contains(strings.ToLower(msg), strings.ToLower(needle)) {
			return true
		}
	}
	return false
}

type TokenPullCmd struct {
	From       string        `help:"SSH host (from ~/.ssh/config or user@host)"`
	RemotePath string        `help:"Remote auth_token.txt path" default:"~/.config/alza/auth_token.txt"`
	Timeout    time.Duration `help:"SSH timeout" default:"15s"`
}

func (c *TokenPullCmd) Run(g *Globals) error {
	if c.From == "" {
		return fmt.Errorf("missing --from (SSH host)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()

	var stdout bytes.Buffer
	cmd := exec.CommandContext(ctx, "ssh", c.From, "cat", "--", c.RemotePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("ssh timed out after %s", c.Timeout)
		}
		return fmt.Errorf("ssh failed: %w", err)
	}

	token, err := extractBearerToken(stdout.String())
	if err != nil {
		return err
	}

	if err := client.SaveToken(token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	fmt.Println("✓ Token pulled and saved to ~/.config/alza/auth_token.txt")
	return nil
}

// === WHOAMI ===

type WhoamiCmd struct{}

func (c *WhoamiCmd) Run(g *Globals) error {
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}

	status, err := cl.GetUserStatus()
	if err != nil {
		return err
	}

	if g.Format == "json" {
		outputJSON(status)
		return nil
	}

	fmt.Printf("User: %s (ID: %d)\n", status.UserName, status.UserID)
	fmt.Printf("Basket ID: %d\n", status.BasketID)
	fmt.Printf("Cart items: %d\n", status.BasketCnt)
	fmt.Printf("Orders: %d\n", status.OrdersCnt)
	if status.IsPremium {
		fmt.Println("Status: AlzaPlus+ member ✓")
	}

	return nil
}

// === SEARCH ===

type SearchCmd struct {
	Query string `arg:"" help:"Search query"`
	Limit int    `help:"Max results to show" default:"10" short:"n"`
}

func (c *SearchCmd) Run(g *Globals) error {
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}

	results, err := cl.Search(c.Query, c.Limit)
	if err != nil {
		return err
	}

	if g.Format == "json" {
		outputJSON(results)
		return nil
	}

	if len(results) == 0 {
		fmt.Println("No results found")
		return nil
	}

	for i, r := range results {
		fmt.Printf("%d. [%d] %s\n", i+1, r.ID, r.Name)
		fmt.Printf("   Price: %s | %s\n", r.PriceStr, r.Availability)
		fmt.Printf("   %s\n\n", r.URL)
	}

	return nil
}

// === PRODUCT ===

type ProductCmd struct {
	ProductID int `arg:"" help:"Product ID"`
}

func (c *ProductCmd) Run(g *Globals) error {
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}

	product, err := cl.GetProduct(c.ProductID)
	if err != nil {
		return err
	}

	if g.Format == "json" {
		outputJSON(product)
		return nil
	}

	fmt.Printf("[%d] %s\n", product.ID, product.Name)

	if product.Price != "" {
		fmt.Printf("Price: %s", product.Price)
		if product.PriceWithoutVat != "" {
			fmt.Printf(" (bez DPH %s)", product.PriceWithoutVat)
		}
		if product.DiscountPercent != nil {
			fmt.Printf(" | Discount: %d%%", *product.DiscountPercent)
		}
		fmt.Println()
	}

	if len(product.PromoPrices) > 0 {
		for _, promo := range product.PromoPrices {
			line := fmt.Sprintf("Promo: %s", promo.Name)
			if promo.Price != "" {
				line += fmt.Sprintf(" (%s)", promo.Price)
			}
			if promo.Code != "" {
				line += fmt.Sprintf(" [code: %s]", promo.Code)
			}
			fmt.Println(line)
		}
	}

	if product.CashBackPriceLabel != "" || product.CashBackPrice != "" {
		line := "Promo"
		if product.CashBackPriceLabel != "" {
			line += fmt.Sprintf(": %s", product.CashBackPriceLabel)
		}
		if product.CashBackPrice != "" {
			line += fmt.Sprintf(" (%s)", product.CashBackPrice)
		}
		fmt.Println(line)
	}

	if product.Availability != "" {
		fmt.Printf("Availability: %s\n", product.Availability)
	}
	if product.AvailabilityDetail != "" {
		fmt.Printf("  %s\n", product.AvailabilityDetail)
	}
	if product.ExpectedStockDate != "" {
		fmt.Printf("  %s\n", product.ExpectedStockDate)
	}

	if product.Description != "" {
		fmt.Println("\nDescription:")
		for _, line := range strings.Split(product.Description, "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			fmt.Printf("  %s\n", line)
		}
	}

	if len(product.Parameters) > 0 {
		fmt.Println("\nParameters:")
		for _, group := range product.Parameters {
			fmt.Printf("  %s:\n", group.Name)
			for _, param := range group.Parameters {
				fmt.Printf("    - %s: %s\n", param.Name, strings.Join(param.Values, ", "))
			}
		}
	}

	if len(product.Variants) > 0 {
		fmt.Println("\nVariants:")
		for _, variant := range product.Variants {
			marker := " "
			if variant.IsSelected {
				marker = "*"
			}
			fmt.Printf("  %s [%d] %s\n", marker, variant.ID, variant.Name)
		}
	}

	return nil
}

// === CART ===

type CartCmd struct {
	Show   CartShowCmd   `cmd:"" default:"1" help:"Show cart contents"`
	Add    CartAddCmd    `cmd:"" help:"Add product to cart"`
	Remove CartRemoveCmd `cmd:"" help:"Remove product from cart"`
	Clear  CartClearCmd  `cmd:"" help:"Clear entire cart"`
}

type CartShowCmd struct{}

func (c *CartShowCmd) Run(g *Globals) error {
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}

	items, err := cl.GetCart()
	if err != nil {
		return err
	}

	if g.Format == "json" {
		outputJSON(items)
		return nil
	}

	if len(items) == 0 {
		fmt.Println("Cart is empty")
		return nil
	}

	fmt.Printf("Cart (%d items):\n\n", len(items))
	for i, item := range items {
		if item.Name != "" {
			fmt.Printf("%d. [%d] %s\n", i+1, item.ProductID, item.Name)
			fmt.Printf("   Price: %s | Qty: %d\n", item.Price, item.Count)
			if item.URL != "" {
				fmt.Printf("   %s\n", item.URL)
			}
			fmt.Println()
		} else {
			fmt.Printf("%d. Product ID: %d (qty: %d)\n", i+1, item.ProductID, item.Count)
		}
	}

	return nil
}

type CartAddCmd struct {
	ProductID int `arg:"" help:"Product ID to add"`
	Quantity  int `help:"Quantity" default:"1" short:"q"`
}

func (c *CartAddCmd) Run(g *Globals) error {
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}

	if err := cl.AddToCart(c.ProductID, c.Quantity); err != nil {
		return err
	}

	fmt.Printf("✓ Added product %d to cart (qty: %d)\n", c.ProductID, c.Quantity)
	return nil
}

type CartRemoveCmd struct {
	ProductID int `arg:"" help:"Product ID to remove"`
}

func (c *CartRemoveCmd) Run(g *Globals) error {
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}

	if err := cl.RemoveFromCart(c.ProductID); err != nil {
		return err
	}

	fmt.Printf("✓ Removed product %d from cart\n", c.ProductID)
	return nil
}

type CartClearCmd struct{}

func (c *CartClearCmd) Run(g *Globals) error {
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}

	if err := cl.ClearCart(); err != nil {
		return err
	}

	fmt.Println("✓ Cart cleared")
	return nil
}

// === FAVORITES ===

type FavoritesCmd struct {
	Show   FavoritesShowCmd   `cmd:"" default:"1" help:"Show favorites"`
	Add    FavoritesAddCmd    `cmd:"" help:"Add product to favorites"`
	Remove FavoritesRemoveCmd `cmd:"" help:"Remove product from favorites"`
}

type FavoritesShowCmd struct{}

func (c *FavoritesShowCmd) Run(g *Globals) error {
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}

	list, err := resolveFavoritesList(cl)
	if err != nil {
		return err
	}

	items, err := cl.GetListItems(list.ID)
	if err != nil {
		return err
	}

	if g.Format == "json" {
		outputJSON(items)
		return nil
	}

	if len(items) == 0 {
		fmt.Printf("List '%s' is empty\n", list.Name)
		return nil
	}

	fmt.Printf("List '%s' (%d items):\n\n", list.Name, len(items))
	for i, item := range items {
		fmt.Printf("%d. %s\n", i+1, item.NavigationURL)
		if item.Price != "" {
			fmt.Printf("   Price: %s\n", item.Price)
		}
	}

	return nil
}

type FavoritesAddCmd struct {
	ProductID int `arg:"" help:"Product ID to add"`
}

func (c *FavoritesAddCmd) Run(g *Globals) error {
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}

	list, err := resolveFavoritesList(cl)
	if err != nil {
		return err
	}

	if err := cl.AddToList(list.ID, c.ProductID); err != nil {
		return err
	}

	fmt.Printf("✓ Added product %d to list '%s'\n", c.ProductID, list.Name)
	return nil
}

type FavoritesRemoveCmd struct {
	ProductID int `arg:"" help:"Product ID to remove"`
}

func (c *FavoritesRemoveCmd) Run(g *Globals) error {
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}

	list, err := resolveFavoritesList(cl)
	if err != nil {
		return err
	}

	if err := cl.RemoveFromList(list.ID, c.ProductID); err != nil {
		return err
	}

	fmt.Printf("✓ Removed product %d from list '%s'\n", c.ProductID, list.Name)
	return nil
}

// === LISTS ===

type ListsCmd struct {
	Show   ListsShowCmd   `cmd:"" default:"1" help:"Show all commodity lists"`
	Items  ListsItemsCmd  `cmd:"" help:"Show items in a specific list"`
	Create ListsCreateCmd `cmd:"" help:"Create a new list"`
	Add    ListsAddCmd    `cmd:"" help:"Add product to a list"`
}

type ListsShowCmd struct{}

func (c *ListsShowCmd) Run(g *Globals) error {
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}

	lists, err := cl.GetLists()
	if err != nil {
		return err
	}

	if g.Format == "json" {
		outputJSON(lists)
		return nil
	}

	listTypes := map[int]string{
		0:  "custom",
		1:  "favorites",
		9:  "frequent",
		14: "buy-later",
		15: "pc-config",
	}

	fmt.Printf("Commodity Lists (%d):\n\n", len(lists))
	for _, l := range lists {
		typeStr := listTypes[l.Type]
		if typeStr == "" {
			typeStr = fmt.Sprintf("type-%d", l.Type)
		}
		fmt.Printf("  [%d] %s (%d items) - %s\n", l.ID, l.Name, l.ItemCount, typeStr)
	}

	return nil
}

type ListsItemsCmd struct {
	ListID int `arg:"" help:"List ID"`
}

func (c *ListsItemsCmd) Run(g *Globals) error {
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}

	items, err := cl.GetListItems(c.ListID)
	if err != nil {
		return err
	}

	if g.Format == "json" {
		outputJSON(items)
		return nil
	}

	if len(items) == 0 {
		fmt.Println("List is empty")
		return nil
	}

	fmt.Printf("List items (%d):\n\n", len(items))
	for i, item := range items {
		fmt.Printf("%d. %s\n", i+1, item.NavigationURL)
		if item.Price != "" {
			fmt.Printf("   Price: %s\n", item.Price)
		}
	}

	return nil
}

type ListsCreateCmd struct {
	Name string `arg:"" help:"Name for the new list"`
}

func (c *ListsCreateCmd) Run(g *Globals) error {
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}

	list, err := cl.CreateList(c.Name)
	if err != nil {
		return err
	}

	if g.Format == "json" {
		outputJSON(list)
		return nil
	}

	fmt.Printf("✓ List '%s' vytvorený (ID: %d)\n", list.Name, list.ID)
	return nil
}

type ListsAddCmd struct {
	ListID    int `arg:"" help:"List ID"`
	ProductID int `arg:"" help:"Product ID to add"`
}

func (c *ListsAddCmd) Run(g *Globals) error {
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}

	if err := cl.AddToList(c.ListID, c.ProductID); err != nil {
		return err
	}

	fmt.Printf("✓ Produkt %d pridaný do listu %d\n", c.ProductID, c.ListID)
	return nil
}

// === ORDERS ===

type OrdersCmd struct {
	Limit int `help:"Max orders to show" default:"10" short:"n"`
}

func (c *OrdersCmd) Run(g *Globals) error {
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}

	orders, total, err := cl.GetOrders(c.Limit)
	if err != nil {
		return err
	}

	if g.Format == "json" {
		outputJSON(map[string]interface{}{
			"orders":     orders,
			"totalCount": total,
		})
		return nil
	}

	fmt.Printf("Orders (showing %d of %d):\n\n", len(orders), total)
	for _, o := range orders {
		fmt.Printf("  #%s | %s | %s | %s\n", o.ID, o.Date, o.Status, o.TotalPrice)
	}

	return nil
}

// === QUICKBUY ===

type QuickbuyCmd struct {
	ProductID  int      `arg:"" help:"Product ID to order"`
	Quantity   int      `help:"Quantity" default:"1" short:"q"`
	Yes        bool     `help:"Skip countdown (DANGEROUS!)" short:"y"`
	DryRun     bool     `help:"Simulate only, don't actually order" name:"dry-run"`
	QuoteOnly  bool     `help:"Get price quote only (no order)" name:"quote"`
	Timeout    int      `help:"Countdown seconds before ordering" default:"10" short:"t"`
	AlzaBoxID  int      `help:"AlzaBox location ID (required unless --dry-run)" env:"ALZA_QUICKBUY_ALZABOX_ID"`
	DeliveryID int      `help:"Delivery type ID (required unless --dry-run)" env:"ALZA_QUICKBUY_DELIVERY_ID"`
	PaymentID  string   `help:"Payment method ID (required unless --dry-run)" env:"ALZA_QUICKBUY_PAYMENT_ID"`
	CardID     string   `help:"Saved card ID (required unless --dry-run)" env:"ALZA_QUICKBUY_CARD_ID"`
	VisitorID  string   `help:"Device fingerprint/visitor ID (required unless --dry-run)" env:"ALZA_QUICKBUY_VISITOR_ID"`
	AlzaPlus   bool     `help:"Use AlzaPlus+ pricing" env:"ALZA_QUICKBUY_ALZAPLUS"`
	Coupons    []string `help:"Promo code(s), comma-separated or repeated" name:"coupon" sep:"," env:"ALZA_QUICKBUY_COUPON"`
	NoCoupon   bool     `help:"Explicitly proceed without coupon" name:"no-coupon"`
}

func (c *QuickbuyCmd) Run(g *Globals) error {
	// Load env config first (before auth) to validate coupon requirement
	envCfg, err := client.QuickbuyConfigFromEnvFile("")
	if err != nil {
		return err
	}

	config := client.QuickBuyConfig{
		AlzaBoxID:  c.AlzaBoxID,
		DeliveryID: c.DeliveryID,
		PaymentID:  c.PaymentID,
		CardID:     c.CardID,
		IsAlzaPlus: c.AlzaPlus,
		VisitorID:  c.VisitorID,
		DryRun:     c.DryRun,
		QuoteOnly:  c.QuoteOnly,
		PromoCodes: c.Coupons,
	}
	config = config.WithDefaults(envCfg)

	// Coupon is required unless --no-coupon is explicitly set (not for dry-run)
	if len(config.PromoCodes) == 0 && !c.NoCoupon && !c.DryRun {
		return fmt.Errorf("coupon is required\nUse --coupon <CODE> or --no-coupon to proceed without discount")
	}

	if err := config.Validate(); err != nil {
		return fmt.Errorf("%w\nSet required flags or env vars: ALZA_QUICKBUY_ALZABOX_ID, ALZA_QUICKBUY_DELIVERY_ID, ALZA_QUICKBUY_PAYMENT_ID, ALZA_QUICKBUY_CARD_ID, ALZA_QUICKBUY_VISITOR_ID", err)
	}

	// Create client (requires auth)
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}

	// Show order info
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	if c.DryRun {
		fmt.Println("║  🧪 DRY RUN - SIMULÁCIA                                   ║")
	} else if c.QuoteOnly {
		fmt.Println("║  🧾 QUOTE ONLY - CENOVÁ PONUKA                            ║")
	} else {
		fmt.Println("║  🛒 QUICKBUY - RÝCHLA OBJEDNÁVKA                          ║")
	}
	fmt.Println("╠═══════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Produkt ID: %-45d ║\n", c.ProductID)
	fmt.Printf("║  Množstvo:   %-45d ║\n", c.Quantity)
	fmt.Printf("║  AlzaBox ID: %-43d ║\n", config.AlzaBoxID)
	fmt.Printf("║  Delivery ID: %-42d ║\n", config.DeliveryID)
	fmt.Printf("║  Payment ID: %-42s ║\n", config.PaymentID)
	if len(config.PromoCodes) > 0 {
		fmt.Printf("║  Coupon:     %-42s ║\n", strings.Join(config.PromoCodes, ", "))
	} else {
		fmt.Printf("║  Coupon:     %-42s ║\n", "(žiadny - --no-coupon)")
	}
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()

	if !c.DryRun && !c.Yes && !c.QuoteOnly {
		// Countdown with cancel option
		fmt.Println("╔═══════════════════════════════════════════════════════════╗")
		fmt.Println("║  💳 KARTA BUDE ZAŤAŽENÁ!                                  ║")
		fmt.Println("║                                                           ║")
		fmt.Println("║  Stlač Ctrl+C pre ZRUŠENIE                                ║")
		fmt.Println("╚═══════════════════════════════════════════════════════════╝")
		fmt.Println()

		// Countdown
		cancelled := make(chan bool, 1)

		// Listen for Enter key to cancel
		go func() {
			reader := bufio.NewReader(os.Stdin)
			reader.ReadString('\n')
			cancelled <- true
		}()

		for i := c.Timeout; i > 0; i-- {
			select {
			case <-cancelled:
				fmt.Println("\n\n❌ ZRUŠENÉ používateľom")
				return nil
			default:
				// Build progress bar
				progress := c.Timeout - i
				total := c.Timeout
				barWidth := 40
				filled := (progress * barWidth) / total
				bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

				// Color based on urgency
				var emoji string
				if i <= 3 {
					emoji = "🔴"
				} else if i <= 6 {
					emoji = "🟡"
				} else {
					emoji = "🟢"
				}

				fmt.Printf("\r  %s Objednávka za %2d sekúnd [%s] (Enter = zrušiť)", emoji, i, bar)

				time.Sleep(1 * time.Second)
			}
		}
		fmt.Println()
		fmt.Println()
	}

	if c.QuoteOnly {
		fmt.Println("⏳ Vytváram cenovú ponuku...")
	} else {
		fmt.Println("⏳ Vytváram objednávku...")
	}

	result, err := cl.QuickBuy(c.ProductID, c.Quantity, config)
	if err != nil {
		return fmt.Errorf("quickbuy failed: %w", err)
	}

	if g.Format == "json" {
		outputJSON(result)
		return nil
	}

	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	if c.QuoteOnly {
		fmt.Println("║  ✅ CENOVÁ PONUKA VYTVORENÁ                              ║")
	} else {
		fmt.Println("║  ✅ OBJEDNÁVKA ÚSPEŠNE VYTVORENÁ!                        ║")
	}
	fmt.Println("╠═══════════════════════════════════════════════════════════╣")
	if c.QuoteOnly {
		fmt.Printf("║  ID ponuky:        %-40s ║\n", result.OrderID)
	} else {
		fmt.Printf("║  Číslo objednávky: %-40s ║\n", result.OrderID)
	}
	fmt.Printf("║  Celková suma:     %.2f €%s ║\n", result.TotalPrice, strings.Repeat(" ", 34-len(fmt.Sprintf("%.2f", result.TotalPrice))))
	if !c.QuoteOnly {
		fmt.Println("║                                                           ║")
		fmt.Println("║  Doručenie: AlzaBox Žilina - Obvodová (Tesco)             ║")
		fmt.Println("║  Platba:    Kartou online                                 ║")
	}
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")

	return nil
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
	)

	err := ctx.Run(&CLI.Globals)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
