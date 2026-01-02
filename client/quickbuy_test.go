package client

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestNormalizePromoCodes(t *testing.T) {
	input := []string{" CODE1, CODE2", "CODE1", "", "CODE3", "CODE2"}
	want := []string{"CODE1", "CODE2", "CODE3"}

	got := normalizePromoCodes(input)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizePromoCodes mismatch: got %v want %v", got, want)
	}
}

func TestNormalizePromoCodesEmpty(t *testing.T) {
	got := normalizePromoCodes([]string{})
	if got != nil {
		t.Fatalf("normalizePromoCodes([]) = %v, want nil", got)
	}
}

func TestNormalizePromoCodesNil(t *testing.T) {
	got := normalizePromoCodes(nil)
	if got != nil {
		t.Fatalf("normalizePromoCodes(nil) = %v, want nil", got)
	}
}

func TestNormalizePromoCodesOnlyWhitespace(t *testing.T) {
	got := normalizePromoCodes([]string{"  ", " , "})
	if len(got) != 0 {
		t.Fatalf("normalizePromoCodes(whitespace) = %v, want empty", got)
	}
}

func TestQuickBuyValidateQuoteOnly(t *testing.T) {
	cfg := QuickBuyConfig{
		AlzaBoxID:  1,
		DeliveryID: 2,
		PaymentID:  "216",
		QuoteOnly:  true,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected quote-only config to be valid, got %v", err)
	}
}

func TestQuickBuyValidateRequiresPaymentDetails(t *testing.T) {
	cfg := QuickBuyConfig{
		AlzaBoxID:  1,
		DeliveryID: 2,
		PaymentID:  "216",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatalf("expected missing payment details error, got nil")
	}
	if !strings.Contains(err.Error(), "ALZA_QUICKBUY_CARD_ID") {
		t.Fatalf("expected missing card id error, got %v", err)
	}
	if !strings.Contains(err.Error(), "ALZA_QUICKBUY_VISITOR_ID") {
		t.Fatalf("expected missing visitor id error, got %v", err)
	}
}

func TestQuickBuyValidateDryRunSkipsAll(t *testing.T) {
	cfg := QuickBuyConfig{
		DryRun: true,
		// All other fields empty
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected dry-run config to be valid, got %v", err)
	}
}

func TestQuickBuyValidateMissingAlzaBoxID(t *testing.T) {
	cfg := QuickBuyConfig{
		DeliveryID: 2,
		PaymentID:  "216",
		CardID:     "card123",
		VisitorID:  "visitor456",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing AlzaBoxID")
	}
	if !strings.Contains(err.Error(), "ALZA_QUICKBUY_ALZABOX_ID") {
		t.Fatalf("expected ALZABOX_ID in error, got %v", err)
	}
}

func TestQuickBuyValidateMissingDeliveryID(t *testing.T) {
	cfg := QuickBuyConfig{
		AlzaBoxID: 1,
		PaymentID: "216",
		CardID:    "card123",
		VisitorID: "visitor456",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing DeliveryID")
	}
	if !strings.Contains(err.Error(), "ALZA_QUICKBUY_DELIVERY_ID") {
		t.Fatalf("expected DELIVERY_ID in error, got %v", err)
	}
}

func TestQuickBuyValidateMissingPaymentID(t *testing.T) {
	cfg := QuickBuyConfig{
		AlzaBoxID:  1,
		DeliveryID: 2,
		CardID:     "card123",
		VisitorID:  "visitor456",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing PaymentID")
	}
	if !strings.Contains(err.Error(), "ALZA_QUICKBUY_PAYMENT_ID") {
		t.Fatalf("expected PAYMENT_ID in error, got %v", err)
	}
}

func TestQuickBuyValidateCompleteConfig(t *testing.T) {
	cfg := QuickBuyConfig{
		AlzaBoxID:  12345,
		DeliveryID: 2680,
		PaymentID:  "216",
		CardID:     "card123",
		VisitorID:  "visitor456",
		IsAlzaPlus: true,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}
}

func TestQuickBuyConfigWithDefaults(t *testing.T) {
	defaults := QuickBuyConfig{
		AlzaBoxID:  12345,
		DeliveryID: 2680,
		PaymentID:  "216",
		CardID:     "defaultCard",
		VisitorID:  "defaultVisitor",
		IsAlzaPlus: true,
		PromoCodes: []string{"DEFAULT"},
	}

	cfg := QuickBuyConfig{
		AlzaBoxID: 99999, // Override
	}

	result := cfg.WithDefaults(defaults)

	if result.AlzaBoxID != 99999 {
		t.Errorf("AlzaBoxID = %d, want 99999 (should keep override)", result.AlzaBoxID)
	}
	if result.DeliveryID != 2680 {
		t.Errorf("DeliveryID = %d, want 2680 (from defaults)", result.DeliveryID)
	}
	if result.PaymentID != "216" {
		t.Errorf("PaymentID = %q, want 216 (from defaults)", result.PaymentID)
	}
	if result.CardID != "defaultCard" {
		t.Errorf("CardID = %q, want defaultCard (from defaults)", result.CardID)
	}
	if result.VisitorID != "defaultVisitor" {
		t.Errorf("VisitorID = %q, want defaultVisitor (from defaults)", result.VisitorID)
	}
	if !result.IsAlzaPlus {
		t.Error("IsAlzaPlus = false, want true (from defaults)")
	}
}

func TestQuickBuyConfigWithDefaultsPromoCodes(t *testing.T) {
	defaults := QuickBuyConfig{
		PromoCodes: []string{"DEFAULT1", "DEFAULT2"},
	}

	// Config without promo codes should use defaults
	cfg := QuickBuyConfig{}
	result := cfg.WithDefaults(defaults)

	if len(result.PromoCodes) != 2 {
		t.Errorf("PromoCodes len = %d, want 2", len(result.PromoCodes))
	}
}

func TestQuickBuyConfigWithDefaultsOwnPromoCodes(t *testing.T) {
	defaults := QuickBuyConfig{
		PromoCodes: []string{"DEFAULT"},
	}

	// Config with own promo codes should keep them
	cfg := QuickBuyConfig{
		PromoCodes: []string{"CUSTOM"},
	}
	result := cfg.WithDefaults(defaults)

	if len(result.PromoCodes) != 1 || result.PromoCodes[0] != "CUSTOM" {
		t.Errorf("PromoCodes = %v, want [CUSTOM]", result.PromoCodes)
	}
}

func TestQuickBuyResultJSON(t *testing.T) {
	result := QuickBuyResult{
		OrderID:    "ORD123456",
		TotalPrice: 299.99,
		Success:    true,
		Message:    "Order created",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed QuickBuyResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.OrderID != "ORD123456" {
		t.Errorf("OrderID = %q, want ORD123456", parsed.OrderID)
	}
	if parsed.TotalPrice != 299.99 {
		t.Errorf("TotalPrice = %f, want 299.99", parsed.TotalPrice)
	}
	if !parsed.Success {
		t.Error("Success = false, want true")
	}
}

func TestFastOrderRequestJSON(t *testing.T) {
	req := fastOrderRequest{
		Options: fastOrderOptions{
			Items:      []fastOrderItem{{CommodityID: 12345, Count: 1}},
			AlzaBoxID:  1009905,
			DeliveryID: 2680,
			PaymentID:  "216",
			IsLoggedIn: true,
			Source:     "Unknown",
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	if !strings.Contains(string(data), `"CommodityId":12345`) {
		t.Errorf("JSON missing CommodityId: %s", data)
	}
	if !strings.Contains(string(data), `"AlzaBoxId":1009905`) {
		t.Errorf("JSON missing AlzaBoxId: %s", data)
	}
}

func TestPaymentRequestJSON(t *testing.T) {
	req := paymentRequest{
		Browser: paymentBrowserInfo{
			ScreenWidth:       1920,
			ScreenHeight:      1080,
			ColorDepth:        24,
			UserAgent:         "test-agent",
			TimeZoneOffset:    -60,
			Language:          "sk-SK",
			JavaEnabled:       false,
			DeviceFingerprint: "fp123",
		},
		CardID:              "card456",
		FastOrder:           true,
		OrderID:             "ORD789",
		AfterOrderPaymentID: 12345,
		DeviceFingerprint:   "fp123",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	if !strings.Contains(string(data), `"cardId":"card456"`) {
		t.Errorf("JSON missing cardId: %s", data)
	}
	if !strings.Contains(string(data), `"orderId":"ORD789"`) {
		t.Errorf("JSON missing orderId: %s", data)
	}
}

func TestFastOrderItemJSON(t *testing.T) {
	item := fastOrderItem{CommodityID: 12345, Count: 2}

	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	expected := `{"CommodityId":12345,"Count":2}`
	if string(data) != expected {
		t.Errorf("JSON = %s, want %s", data, expected)
	}
}

func TestFastOrderSaveResponseParsing(t *testing.T) {
	jsonData := `{
		"d": {
			"TotalPrice": 299.99,
			"AfterOrderPaymentId": 12345,
			"ErrorMessage": "",
			"Data": {
				"TotalPriceDec": 299.99,
				"AfterOrderPaymentId": 12345
			}
		}
	}`

	var result struct {
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

	if err := json.Unmarshal([]byte(jsonData), &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if result.D.TotalPrice != 299.99 {
		t.Errorf("TotalPrice = %f, want 299.99", result.D.TotalPrice)
	}
	if result.D.AfterOrderPaymentId != 12345 {
		t.Errorf("AfterOrderPaymentId = %d, want 12345", result.D.AfterOrderPaymentId)
	}
}

func TestFastOrderSaveResponseWithError(t *testing.T) {
	jsonData := `{
		"d": {
			"TotalPrice": 0,
			"AfterOrderPaymentId": 0,
			"ErrorMessage": "Product not available"
		}
	}`

	var result struct {
		D struct {
			ErrorMessage string `json:"ErrorMessage"`
		} `json:"d"`
	}

	if err := json.Unmarshal([]byte(jsonData), &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if result.D.ErrorMessage != "Product not available" {
		t.Errorf("ErrorMessage = %q, want Product not available", result.D.ErrorMessage)
	}
}

func TestFastOrderSendResponseParsing(t *testing.T) {
	jsonData := `{
		"d": {
			"Code": "ORD123456",
			"OrderId": "",
			"AfterOrderPaymentId": 67890,
			"ErrorMessage": ""
		}
	}`

	var result struct {
		D struct {
			Code                string `json:"Code"`
			OrderId             string `json:"OrderId"`
			AfterOrderPaymentId int    `json:"AfterOrderPaymentId"`
			ErrorMessage        string `json:"ErrorMessage"`
		} `json:"d"`
	}

	if err := json.Unmarshal([]byte(jsonData), &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Code is the primary order ID field
	orderID := result.D.Code
	if orderID == "" {
		orderID = result.D.OrderId
	}

	if orderID != "ORD123456" {
		t.Errorf("OrderID = %q, want ORD123456", orderID)
	}
}

func TestQuickBuyDryRun(t *testing.T) {
	// Test the dry-run mode which doesn't make HTTP calls
	c := &TLSClient{}

	config := QuickBuyConfig{
		DryRun: true,
	}

	result, err := c.QuickBuy(12345, 1, config)
	if err != nil {
		t.Fatalf("QuickBuy dry-run error: %v", err)
	}

	if result == nil {
		t.Fatal("QuickBuy dry-run returned nil result")
	}

	if result.OrderID != "DRY-RUN-000000" {
		t.Errorf("OrderID = %q, want DRY-RUN-000000", result.OrderID)
	}

	if !result.Success {
		t.Error("Success = false, want true")
	}

	if result.TotalPrice != 0 {
		t.Errorf("TotalPrice = %f, want 0", result.TotalPrice)
	}

	if !strings.Contains(result.Message, "DRY RUN") {
		t.Errorf("Message = %q, expected to contain 'DRY RUN'", result.Message)
	}
}
