package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kuringer/alza-cli/client"
)

// === QUICKBUY ===

type QuickbuyCmd struct {
	ProductIDs []int    `arg:"" name:"product-id" help:"Product ID to order (only 1 supported)"`
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

func buildQuickbuyConfig(cmd *QuickbuyCmd, envCfg client.QuickBuyConfig) client.QuickBuyConfig {
	config := client.QuickBuyConfig{
		AlzaBoxID:  cmd.AlzaBoxID,
		DeliveryID: cmd.DeliveryID,
		PaymentID:  cmd.PaymentID,
		CardID:     cmd.CardID,
		IsAlzaPlus: cmd.AlzaPlus,
		VisitorID:  cmd.VisitorID,
		DryRun:     cmd.DryRun,
		QuoteOnly:  cmd.QuoteOnly,
		PromoCodes: cmd.Coupons,
	}
	config = config.WithDefaults(envCfg)
	if cmd.NoCoupon {
		config.PromoCodes = []string{}
	}
	return config
}

func (c *QuickbuyCmd) Run(g *Globals) error {
	// Validate single product - quickbuy only supports 1 product at a time
	if len(c.ProductIDs) == 0 {
		return fmt.Errorf("product ID is required")
	}
	if len(c.ProductIDs) > 1 {
		return fmt.Errorf("quickbuy supports only 1 product at a time\nFor multiple products, run quickbuy separately for each or use cart")
	}
	productID := c.ProductIDs[0]

	// Load env config first (before auth) to validate coupon requirement
	envCfg, err := client.QuickbuyConfigFromEnvFile("")
	if err != nil {
		return err
	}

	config := buildQuickbuyConfig(c, envCfg)

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
	fmt.Printf("║  Produkt ID: %-45d ║\n", productID)
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

	result, err := cl.QuickBuy(productID, c.Quantity, config)
	if err != nil {
		return fmt.Errorf("quickbuy failed: %w", err)
	}

	var cartErr error
	if !c.QuoteOnly && !c.DryRun {
		result.RemovedFromCart, cartErr = removeOrderedFromCart(cl, productID)
	}

	if g.Format == "json" {
		outputJSON(result)
		return nil
	}

	if c.QuoteOnly || c.DryRun {
		fmt.Println()
		if c.DryRun {
			fmt.Println("🧪 DRY RUN - nič sa neobjednalo")
		} else {
			fmt.Println("✅ CENOVÁ PONUKA VYTVORENÁ")
		}
		fmt.Printf("   ID ponuky:    %s\n", result.OrderID)
		fmt.Printf("   Celková suma: %.2f €\n", result.TotalPrice)
		return nil
	}

	fmt.Println()
	fmt.Println(orderHeadline(result.PaymentStatus))
	fmt.Printf("   Číslo objednávky: %s\n", result.OrderID)
	fmt.Printf("   Celková suma:     %.2f €\n", result.TotalPrice)
	fmt.Printf("   Doručenie:        AlzaBox ID %d\n", config.AlzaBoxID)
	fmt.Printf("   Platba:           %s\n", paymentLine(result))
	switch {
	case cartErr != nil:
		fmt.Printf("   Košík:            nepodarilo sa upratať (%v)\n", cartErr)
	case result.RemovedFromCart:
		fmt.Println("   Košík:            produkt odstránený z košíka")
	}

	return nil
}

func orderHeadline(paymentStatus string) string {
	switch paymentStatus {
	case client.PaymentStatusPaid:
		return "✅ OBJEDNÁVKA VYTVORENÁ A ZAPLATENÁ"
	case client.PaymentStatusFailed:
		return "❌ OBJEDNÁVKA VYTVORENÁ, ALE ZRUŠENÁ / PLATBA ZLYHALA"
	default:
		return "⚠️  OBJEDNÁVKA VYTVORENÁ, PLATBA NEOVERENÁ"
	}
}

func paymentLine(r *client.QuickBuyResult) string {
	var line string
	switch r.PaymentStatus {
	case client.PaymentStatusPaid:
		line = "zaplatené kartou"
	case client.PaymentStatusPending:
		line = "čaká na platbu - over v appke Alza alebo `alza orders`"
	case client.PaymentStatusFailed:
		line = "zlyhala"
	default:
		line = "neoverená - skontroluj `alza orders`"
	}
	if r.OrderStatus != "" {
		line += fmt.Sprintf(" (stav: %s)", r.OrderStatus)
	}
	if r.PaymentError != "" {
		line += fmt.Sprintf(" [chyba platby: %s]", r.PaymentError)
	}
	return line
}

type cartAPI interface {
	GetCart() ([]client.CartItem, error)
	RemoveFromCart(productID int) error
}

// removeOrderedFromCart drops the just-ordered product from the web cart so it
// is not bought a second time from there. Returns true if it removed anything.
func removeOrderedFromCart(cl cartAPI, productID int) (bool, error) {
	items, err := cl.GetCart()
	if err != nil {
		return false, err
	}
	for _, item := range items {
		if item.ProductID == productID {
			if err := cl.RemoveFromCart(productID); err != nil {
				return false, err
			}
			return true, nil
		}
	}
	return false, nil
}
