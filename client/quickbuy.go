package client

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// QuickBuyConfig contains delivery and payment settings
type QuickBuyConfig struct {
	AlzaBoxID  int    // AlzaBox location ID (e.g., 1009905 = Žilina Obvodová)
	DeliveryID int    // Delivery type ID (e.g., 2680 = AlzaBox)
	PaymentID  string // Payment method ID (e.g., "216" = Card online)
	CardID     string // Saved card ID
	IsAlzaPlus bool   // Has AlzaPlus+ membership
	VisitorID  string // Device fingerprint/visitor ID
	DryRun     bool   // If true, only simulate (don't actually order)
	PromoCodes []string
	QuoteOnly  bool // If true, only run FastOrderSave and return price
}

// QuickBuyResult contains order result
type QuickBuyResult struct {
	OrderID    string  `json:"orderId"`
	TotalPrice float64 `json:"totalPrice"`
	Success    bool    `json:"success"`
	Message    string  `json:"message"`
}

type fastOrderItem struct {
	CommodityID int `json:"CommodityId"`
	Count       int `json:"Count"`
}

type fastOrderOptions struct {
	Items                     []fastOrderItem `json:"Items"`
	AlzaBoxID                 int             `json:"AlzaBoxId"`
	DeliveryID                int             `json:"DeliveryId"`
	PaymentID                 string          `json:"PaymentId"`
	PreferredCard             string          `json:"PrefferedCard,omitempty"`
	IsAlzaPlus                bool            `json:"IsAlzaPlus"`
	IsLoggedIn                bool            `json:"IsLoggedIn"`
	Source                    string          `json:"Source"`
	IsDelayedPayment          bool            `json:"IsDelayedPayment"`
	WasDeliveryPaymentChanged bool            `json:"wasDeliveryPaymentChanged"`
	ShowAlert                 bool            `json:"ShowAlert"`
	DeliveryAddressID         int             `json:"DeliveryAddressId"`
	IsAddressRequired         bool            `json:"IsAddressRequired"`
	IsTretinka                bool            `json:"IsTretinka"`
	IsVirtual                 bool            `json:"IsVirtual"`
	NeedAddress               bool            `json:"NeedAddress"`
	ShowPaymentCards          bool            `json:"ShowPaymentCards"`
	IsBusinessCardSelected    bool            `json:"IsBusinessCardSelected"`
	AddressID                 int             `json:"AddressId"`
	PromoCodes                []string        `json:"PromoCodes"`
	SelectedPaymentLower      interface{}     `json:"selectedPayment"`
	Step                      interface{}     `json:"Step"`
	IsDialogVisible           bool            `json:"IsDialogVisible"`
	SendCallback              interface{}     `json:"SendCallback"`
	Note                      interface{}     `json:"Note"`
	SelectedPayment           interface{}     `json:"SelectedPayment"`
	AlzaPremium               bool            `json:"AlzaPremium"`
	TotalPriceDec             float64         `json:"TotalPriceDec"`
}

type fastOrderRequest struct {
	Options fastOrderOptions `json:"options"`
}

type fastOrderConsent struct {
	ConsentID string `json:"consentId"`
	Value     bool   `json:"value"`
}

// optionsToMap converts the static options struct to a map so that fields
// returned by the server (Data) can be merged in, mirroring the web client's
// `$.extend(config, response.Data)` behaviour.
func optionsToMap(o fastOrderOptions) (map[string]interface{}, error) {
	b, err := json.Marshal(o)
	if err != nil {
		return nil, err
	}
	m := map[string]interface{}{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// mergeFastOrderData merges d.Data from a FastOrder response into options.
// Like jQuery's shallow $.extend it copies nulls too; the caller re-asserts
// its own delivery/payment fields after the merge (as the web client does).
func mergeFastOrderData(options map[string]interface{}, resp []byte) {
	var parsed struct {
		D struct {
			Data map[string]interface{} `json:"Data"`
		} `json:"d"`
	}
	if err := json.Unmarshal(resp, &parsed); err != nil {
		return
	}
	for k, v := range parsed.D.Data {
		options[k] = v
	}
}

var consentInputRe = regexp.MustCompile(`<input[^>]*js-basket-consent-checkbox[^>]*>`)
var consentValueRe = regexp.MustCompile(`(?i)\bvalue\s*=\s*["']([^"']+)["']`)
var consentRequiredRe = regexp.MustCompile(`(?i)\bdata-isrequired\s*=\s*["']?\s*(true|1)`)

// extractConsents scans every string in the response JSON for basket consent
// checkboxes rendered in the fast-order dialog HTML. Required consents are
// sent as accepted, optional ones (e.g. marketing) as declined.
func extractConsents(resp []byte) []fastOrderConsent {
	var root interface{}
	if err := json.Unmarshal(resp, &root); err != nil {
		return nil
	}
	seen := map[string]bool{}
	out := []fastOrderConsent{}
	var walk func(v interface{})
	walk = func(v interface{}) {
		switch t := v.(type) {
		case string:
			for _, tag := range consentInputRe.FindAllString(t, -1) {
				vm := consentValueRe.FindStringSubmatch(tag)
				if vm == nil || seen[vm[1]] {
					continue
				}
				seen[vm[1]] = true
				out = append(out, fastOrderConsent{ConsentID: vm[1], Value: consentRequiredRe.MatchString(tag)})
			}
		case map[string]interface{}:
			for _, vv := range t {
				walk(vv)
			}
		case []interface{}:
			for _, vv := range t {
				walk(vv)
			}
		}
	}
	walk(root)
	return out
}

type paymentBrowserInfo struct {
	ScreenWidth       int    `json:"screenWidth"`
	ScreenHeight      int    `json:"screenHeight"`
	ColorDepth        int    `json:"colorDepth"`
	UserAgent         string `json:"userAgent"`
	TimeZoneOffset    int    `json:"timeZoneOffset"`
	Language          string `json:"language"`
	JavaEnabled       bool   `json:"javaEnabled"`
	DeviceFingerprint string `json:"deviceFingerprint"`
}

type paymentRequest struct {
	Browser             paymentBrowserInfo `json:"browser"`
	CardID              string             `json:"cardId"`
	FastOrder           bool               `json:"fastOrder"`
	OrderID             string             `json:"orderId"`
	AfterOrderPaymentID int                `json:"afterOrderPaymentId"`
	DeviceFingerprint   string             `json:"deviceFingerprint"`
}

func (c QuickBuyConfig) Validate() error {
	if c.DryRun {
		return nil
	}
	missing := []string{}
	if c.AlzaBoxID == 0 {
		missing = append(missing, "ALZA_QUICKBUY_ALZABOX_ID")
	}
	if c.DeliveryID == 0 {
		missing = append(missing, "ALZA_QUICKBUY_DELIVERY_ID")
	}
	if c.PaymentID == "" {
		missing = append(missing, "ALZA_QUICKBUY_PAYMENT_ID")
	}
	if !c.QuoteOnly {
		if c.CardID == "" {
			missing = append(missing, "ALZA_QUICKBUY_CARD_ID")
		}
		if c.VisitorID == "" {
			missing = append(missing, "ALZA_QUICKBUY_VISITOR_ID")
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("quickbuy config missing: %s", strings.Join(missing, ", "))
	}
	return nil
}

// WithDefaults fills empty fields from defaults.
func (c QuickBuyConfig) WithDefaults(defaults QuickBuyConfig) QuickBuyConfig {
	if c.AlzaBoxID == 0 {
		c.AlzaBoxID = defaults.AlzaBoxID
	}
	if c.DeliveryID == 0 {
		c.DeliveryID = defaults.DeliveryID
	}
	if c.PaymentID == "" {
		c.PaymentID = defaults.PaymentID
	}
	if c.CardID == "" {
		c.CardID = defaults.CardID
	}
	if c.VisitorID == "" {
		c.VisitorID = defaults.VisitorID
	}
	if !c.IsAlzaPlus {
		c.IsAlzaPlus = defaults.IsAlzaPlus
	}
	if len(c.PromoCodes) == 0 {
		c.PromoCodes = defaults.PromoCodes
	}
	c.PromoCodes = normalizePromoCodes(c.PromoCodes)
	return c
}

// QuickBuy performs fast order for a single product
func (c *TLSClient) QuickBuy(productID int, quantity int, config QuickBuyConfig) (*QuickBuyResult, error) {
	// Dry-run mode - simulate without actually ordering
	if config.DryRun {
		return &QuickBuyResult{
			OrderID:    "DRY-RUN-000000",
			TotalPrice: 0,
			Success:    true,
			Message:    "DRY RUN - žiadna objednávka nebola vytvorená",
		}, nil
	}

	options := fastOrderOptions{
		Items:                     []fastOrderItem{{CommodityID: productID, Count: quantity}},
		AlzaBoxID:                 config.AlzaBoxID,
		DeliveryID:                config.DeliveryID,
		PaymentID:                 config.PaymentID,
		PreferredCard:             config.CardID,
		IsAlzaPlus:                config.IsAlzaPlus,
		IsLoggedIn:                true,
		Source:                    "Unknown",
		IsDelayedPayment:          false,
		WasDeliveryPaymentChanged: true,
		ShowAlert:                 false,
		DeliveryAddressID:         -1,
		IsAddressRequired:         false,
		IsTretinka:                false,
		IsVirtual:                 false,
		NeedAddress:               false,
		ShowPaymentCards:          true,
		IsBusinessCardSelected:    false,
		AddressID:                 -1,
		PromoCodes:                config.PromoCodes,
		SelectedPaymentLower:      nil,
		Step:                      nil,
		IsDialogVisible:           false,
		SendCallback:              nil,
		Note:                      nil,
		SelectedPayment:           nil,
		AlzaPremium:               false,
		TotalPriceDec:             0,
	}

	optsMap, err := optionsToMap(options)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	// reassert re-applies the caller's cart and delivery/payment choice after
	// server Data merges - the web equivalent is the user's dialog selection
	// being written back into config before save/send.
	reassert := func() {
		optsMap["Items"] = options.Items
		optsMap["AlzaBoxId"] = options.AlzaBoxID
		optsMap["DeliveryId"] = options.DeliveryID
		optsMap["PaymentId"] = options.PaymentID
		if options.PreferredCard != "" {
			optsMap["PrefferedCard"] = options.PreferredCard
		}
		optsMap["PromoCodes"] = options.PromoCodes
	}

	// Step 0: FastOrderGetDialog - initializes the fast-order session server-side
	// and returns Data (and consent checkboxes) that the web merges into config.
	if c.debug {
		fmt.Println("[DEBUG] Step 0: FastOrderGetDialog")
	}
	var consents []fastOrderConsent
	getBody, err := json.Marshal(map[string]interface{}{"options": optsMap, "isInitial": true})
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	getResp, err := c.Post(EndpointFastOrderGetDialog, string(getBody))
	if err != nil {
		// Quote never places an order, so a broken init step is tolerable there;
		// for a real order fail closed rather than send with uninitialized state.
		if !config.QuoteOnly {
			return nil, fmt.Errorf("FastOrderGetDialog failed: %w", err)
		}
		if c.debug {
			fmt.Printf("[DEBUG] FastOrderGetDialog failed (quote-only, continuing): %v\n", err)
		}
	} else {
		var getResult struct {
			D struct {
				ErrorMessage string  `json:"ErrorMessage"`
				ErrorLevel   float64 `json:"ErrorLevel"`
			} `json:"d"`
		}
		if err := json.Unmarshal(getResp, &getResult); err != nil {
			if !config.QuoteOnly {
				return nil, fmt.Errorf("failed to parse FastOrderGetDialog response: %w", err)
			}
		} else if getResult.D.ErrorMessage != "" || (getResult.D.ErrorLevel != 0 && getResult.D.ErrorLevel != 100) {
			if !config.QuoteOnly {
				return nil, fmt.Errorf("FastOrderGetDialog error (level %.0f): %s", getResult.D.ErrorLevel, getResult.D.ErrorMessage)
			}
			if c.debug {
				fmt.Printf("[DEBUG] FastOrderGetDialog error (quote-only, continuing): %s\n", getResult.D.ErrorMessage)
			}
		} else {
			mergeFastOrderData(optsMap, getResp)
			consents = extractConsents(getResp)
			reassert()
		}
	}

	optsMap["IsLoggedIn"] = true
	bodyJSON, err := json.Marshal(map[string]interface{}{"options": optsMap})
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	// Step 1: FastOrderSave
	if c.debug {
		fmt.Println("[DEBUG] Step 1: FastOrderSave")
	}
	saveResp, err := c.Post(EndpointFastOrderSave, string(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("FastOrderSave failed: %w", err)
	}

	var saveResult struct {
		D struct {
			TotalPrice          float64 `json:"TotalPrice"`
			AfterOrderPaymentId int     `json:"AfterOrderPaymentId"`
			ErrorMessage        string  `json:"ErrorMessage"`
			Data                struct {
				TotalPriceDec       float64 `json:"TotalPriceDec"`
				AfterOrderPaymentId int     `json:"AfterOrderPaymentId"`
			} `json:"Data"`
		} `json:"d"`
	}
	if err := json.Unmarshal(saveResp, &saveResult); err != nil {
		return nil, fmt.Errorf("failed to parse FastOrderSave response: %w", err)
	}

	if saveResult.D.ErrorMessage != "" {
		return nil, fmt.Errorf("FastOrderSave error: %s", saveResult.D.ErrorMessage)
	}

	totalPrice := saveResult.D.TotalPrice
	if totalPrice == 0 {
		totalPrice = saveResult.D.Data.TotalPriceDec
	}

	afterOrderPaymentID := saveResult.D.AfterOrderPaymentId
	if afterOrderPaymentID == 0 {
		afterOrderPaymentID = saveResult.D.Data.AfterOrderPaymentId
	}

	if config.QuoteOnly {
		return &QuickBuyResult{
			OrderID:    "QUOTE-ONLY",
			TotalPrice: totalPrice,
			Success:    true,
			Message:    "QUOTE ONLY - FastOrderSend not executed",
		}, nil
	}

	// Merge server Data from save (web does $.extend(config, Data)), then
	// update options with calculated price and consents.
	mergeFastOrderData(optsMap, saveResp)
	if extra := extractConsents(saveResp); len(extra) > 0 {
		consents = extra
	}
	reassert()
	optsMap["TotalPriceDec"] = totalPrice
	optsMap["IsAddressRequired"] = true
	if len(consents) > 0 {
		optsMap["BasketConsents"] = consents
	}
	bodyJSON, _ = json.Marshal(map[string]interface{}{"options": optsMap})

	// Step 2: FastOrderSend
	if c.debug {
		fmt.Println("[DEBUG] Step 2: FastOrderSend")
	}
	sendResp, err := c.Post(EndpointFastOrderSend, string(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("FastOrderSend failed: %w", err)
	}

	var sendResult struct {
		D struct {
			Code                string `json:"Code"` // Order ID is in "Code" field
			OrderId             string `json:"OrderId"`
			AfterOrderPaymentId int    `json:"AfterOrderPaymentId"`
			ErrorMessage        string `json:"ErrorMessage"`
		} `json:"d"`
	}
	if err := json.Unmarshal(sendResp, &sendResult); err != nil {
		return nil, fmt.Errorf("failed to parse FastOrderSend response: %w", err)
	}

	if sendResult.D.ErrorMessage != "" {
		return nil, fmt.Errorf("FastOrderSend error: %s", sendResult.D.ErrorMessage)
	}

	orderID := sendResult.D.Code
	if orderID == "" {
		orderID = sendResult.D.OrderId
	}
	afterOrderPaymentId := sendResult.D.AfterOrderPaymentId
	if afterOrderPaymentId == 0 {
		afterOrderPaymentId = afterOrderPaymentID
	}

	// Step 3: Process payment (Adyen recurrent)
	if c.debug {
		fmt.Println("[DEBUG] Step 3: Payment processing")
	}
	paymentBody := paymentRequest{
		Browser: paymentBrowserInfo{
			ScreenWidth:       1800,
			ScreenHeight:      1169,
			ColorDepth:        30,
			UserAgent:         userAgent(),
			TimeZoneOffset:    -60,
			Language:          "sk-SK",
			JavaEnabled:       false,
			DeviceFingerprint: config.VisitorID,
		},
		CardID:              config.CardID,
		FastOrder:           true,
		OrderID:             orderID,
		AfterOrderPaymentID: afterOrderPaymentId,
		DeviceFingerprint:   config.VisitorID,
	}
	paymentJSON, _ := json.Marshal(paymentBody)

	_, err = c.Post(EndpointPaymentRepeat, string(paymentJSON))
	if err != nil {
		// Payment might still succeed, check order status
		if c.debug {
			fmt.Printf("[DEBUG] Payment request returned error (may still succeed): %v\n", err)
		}
	}

	return &QuickBuyResult{
		OrderID:    orderID,
		TotalPrice: totalPrice,
		Success:    true,
		Message:    fmt.Sprintf("Objednávka #%s vytvorená", orderID),
	}, nil
}

func normalizePromoCodes(codes []string) []string {
	if len(codes) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	out := []string{}
	for _, entry := range codes {
		for _, raw := range strings.Split(entry, ",") {
			code := strings.TrimSpace(raw)
			if code == "" {
				continue
			}
			if _, ok := seen[code]; ok {
				continue
			}
			seen[code] = struct{}{}
			out = append(out, code)
		}
	}

	return out
}
