package client

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestProductVariantsInfoUnmarshalEmpty(t *testing.T) {
	var info productVariantsInfo
	if err := json.Unmarshal([]byte("{}"), &info); err != nil {
		t.Fatalf("failed to unmarshal empty object: %v", err)
	}

	if info.Type != 0 {
		t.Errorf("Type = %d, want 0", info.Type)
	}
	if info.ProductVariants != nil {
		t.Errorf("ProductVariants = %v, want nil", info.ProductVariants)
	}
}

func TestProductVariantsInfoUnmarshalMixedCase(t *testing.T) {
	// Test with mixed case - neither pure upper nor lower
	jsonData := `{
		"Type": 3,
		"productVariants": [
			{"Id": 789, "Name": "Mixed", "ImageUrl": "", "IsSelected": false}
		]
	}`

	var info productVariantsInfo
	if err := json.Unmarshal([]byte(jsonData), &info); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Type should be 3 (from uppercase)
	if info.Type != 3 {
		t.Errorf("Type = %d, want 3", info.Type)
	}
}

func TestProductPriceInfoV3JSON(t *testing.T) {
	jsonData := `{
		"promoPrices": [
			{
				"name": "Member",
				"formattedPrice": "899,99 €",
				"primaryPrice": "899,99 €",
				"unformattedPrice": 899.99,
				"discountCouponCode": "MEM10"
			}
		],
		"mainPriceTag": {
			"primaryPrice": "999,99 €",
			"primaryPriceNoCurrency": 999.99,
			"secondaryPrice": "826,44 €"
		}
	}`

	var info productPriceInfoV3
	if err := json.Unmarshal([]byte(jsonData), &info); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if info.MainPriceTag.PrimaryPrice != "999,99 €" {
		t.Errorf("PrimaryPrice = %q, want 999,99 €", info.MainPriceTag.PrimaryPrice)
	}
	if len(info.PromoPrices) != 1 {
		t.Errorf("PromoPrices len = %d, want 1", len(info.PromoPrices))
	}
}

func TestProductPriceInfoV2JSON(t *testing.T) {
	jsonData := `{
		"priceWithVat": "1299,99 €",
		"priceWithoutVat": "1073,55 €",
		"priceNoCurrency": 1299.99,
		"unitPriceWithVat": "1299,99 €/ks",
		"promoPrices": []
	}`

	var info productPriceInfoV2
	if err := json.Unmarshal([]byte(jsonData), &info); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if info.PriceWithVat != "1299,99 €" {
		t.Errorf("PriceWithVat = %q, want 1299,99 €", info.PriceWithVat)
	}
	if info.PriceNoCurrency != 1299.99 {
		t.Errorf("PriceNoCurrency = %f, want 1299.99", info.PriceNoCurrency)
	}
}

func TestProductParameterGroupJSON(t *testing.T) {
	jsonData := `{
		"name": "Display",
		"params": [
			{
				"name": "Size",
				"values": [{"desc": "6.1 inch"}]
			}
		]
	}`

	var group productParameterGroup
	if err := json.Unmarshal([]byte(jsonData), &group); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if group.Name != "Display" {
		t.Errorf("Name = %q, want Display", group.Name)
	}
	if len(group.Params) != 1 {
		t.Errorf("Params len = %d, want 1", len(group.Params))
	}
}

func TestProductDetailResponseJSON(t *testing.T) {
	jsonData := `{
		"data": {
			"name": "iPhone 15 Pro",
			"price": "1299,99 €",
			"gaPrice": 1299.99,
			"salePercentage": 10,
			"cashBackPriceLabel": "Cashback",
			"cashBackPrice": "50 €",
			"descriptionBeforeDiscount": "Was 1499 €",
			"descPageUrl": "//cdn.alza.sk/desc.html"
		}
	}`

	var resp productDetailResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if resp.Data.Name != "iPhone 15 Pro" {
		t.Errorf("Name = %q, want iPhone 15 Pro", resp.Data.Name)
	}
	if resp.Data.GaPrice != 1299.99 {
		t.Errorf("GaPrice = %f, want 1299.99", resp.Data.GaPrice)
	}
	if resp.Data.SalePercentage == nil || *resp.Data.SalePercentage != 10 {
		t.Errorf("SalePercentage = %v, want 10", resp.Data.SalePercentage)
	}
}

func TestProductAvailabilityResponseJSON(t *testing.T) {
	jsonData := `{
		"title": "In Stock",
		"description": "Available for immediate dispatch",
		"expectedStockDate": "2024-02-01"
	}`

	var resp productAvailabilityResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if resp.Title != "In Stock" {
		t.Errorf("Title = %q, want In Stock", resp.Title)
	}
	if resp.Description != "Available for immediate dispatch" {
		t.Errorf("Description = %q, expected dispatch message", resp.Description)
	}
}

func TestPickPromoPricesEmpty(t *testing.T) {
	data := productDetailData{}
	result := pickPromoPrices(data)
	if len(result) != 0 {
		t.Errorf("pickPromoPrices(empty) = %v, want empty", result)
	}
}

func TestMapProductParametersEmptyInput(t *testing.T) {
	result := mapProductParameters(nil)
	if result != nil {
		t.Errorf("mapProductParameters(nil) = %v, want nil", result)
	}
}

func TestMapProductParametersEmptyGroups(t *testing.T) {
	result := mapProductParameters([]productParameterGroup{})
	if len(result) != 0 {
		t.Errorf("mapProductParameters([]) = %v, want empty", result)
	}
}

func TestMapProductParametersSkipEmptyName(t *testing.T) {
	groups := []productParameterGroup{
		{
			Name: "",
			Params: []productParameterEntry{
				{Name: "Test", Values: []productParameterValue{{Desc: "value"}}},
			},
		},
		{
			Name: "Valid",
			Params: []productParameterEntry{
				{Name: "Param", Values: []productParameterValue{{Desc: "value"}}},
			},
		},
	}

	result := mapProductParameters(groups)
	if len(result) != 1 {
		t.Errorf("expected 1 group (skip empty name), got %d", len(result))
	}
	if result[0].Name != "Valid" {
		t.Errorf("expected Valid group, got %q", result[0].Name)
	}
}

func TestMapProductParametersSkipEmptyParamName(t *testing.T) {
	groups := []productParameterGroup{
		{
			Name: "Group",
			Params: []productParameterEntry{
				{Name: "", Values: []productParameterValue{{Desc: "value"}}},
				{Name: "Valid", Values: []productParameterValue{{Desc: "value"}}},
			},
		},
	}

	result := mapProductParameters(groups)
	if len(result) != 1 || len(result[0].Parameters) != 1 {
		t.Errorf("expected 1 param (skip empty name), got %+v", result)
	}
}

func TestMapProductParametersSkipEmptyDesc(t *testing.T) {
	groups := []productParameterGroup{
		{
			Name: "Group",
			Params: []productParameterEntry{
				{Name: "Param", Values: []productParameterValue{{Desc: ""}, {Desc: "valid"}}},
			},
		},
	}

	result := mapProductParameters(groups)
	if len(result[0].Parameters[0].Values) != 1 {
		t.Errorf("expected 1 value (skip empty desc), got %+v", result[0].Parameters[0].Values)
	}
}

func TestExtractDescriptionFromHTMLWithNestedElements(t *testing.T) {
	html := `<html><body>
		<h1>Title</h1>
		<div>
			<p>Paragraph <strong>with</strong> nested</p>
			<ul>
				<li>Item 1</li>
				<li>Item 2</li>
			</ul>
		</div>
	</body></html>`

	result := extractDescriptionFromHTML(html)
	if result == "" {
		t.Error("expected non-empty result")
	}
	if !strings.Contains(result, "Title") {
		t.Errorf("result should contain Title: %q", result)
	}
}

func TestExtractDescriptionFromHTMLSkipsStyleAndNoscript(t *testing.T) {
	html := `<html><body>
		<style>body { color: red; }</style>
		<noscript>Enable JS</noscript>
		<p>Actual content</p>
	</body></html>`

	result := extractDescriptionFromHTML(html)
	if strings.Contains(result, "color") || strings.Contains(result, "Enable") {
		t.Errorf("result should not contain style/noscript content: %q", result)
	}
	if !strings.Contains(result, "Actual") {
		t.Errorf("result should contain actual content: %q", result)
	}
}

func TestNodeTextRecursive(t *testing.T) {
	// This is a unit test for nodeText which is used by extractDescriptionFromHTML
	// We can test it indirectly through extractDescriptionFromHTML
	html := `<p>Level1 <span>Level2 <b>Level3</b></span></p>`
	result := extractDescriptionFromHTML(html)
	if !strings.Contains(result, "Level1") || !strings.Contains(result, "Level2") || !strings.Contains(result, "Level3") {
		t.Errorf("nodeText should extract all nested text: %q", result)
	}
}
