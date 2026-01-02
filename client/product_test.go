package client

import (
	"encoding/json"
	"testing"
)

func TestPickPrice(t *testing.T) {
	tests := []struct {
		name string
		data productDetailData
		want string
	}{
		{
			name: "from PriceInfoV3 MainPriceTag",
			data: productDetailData{
				Price: "old",
				PriceInfoV3: &productPriceInfoV3{
					MainPriceTag: struct {
						PrimaryPrice           string  `json:"primaryPrice"`
						PrimaryPriceNoCurrency float64 `json:"primaryPriceNoCurrency"`
						SecondaryPrice         string  `json:"secondaryPrice"`
					}{
						PrimaryPrice: "1299,99 €",
					},
				},
			},
			want: "1299,99 €",
		},
		{
			name: "from PriceInfoV2 when V3 missing",
			data: productDetailData{
				Price: "old",
				PriceInfoV2: &productPriceInfoV2{
					PriceWithVat: "999,99 €",
				},
			},
			want: "999,99 €",
		},
		{
			name: "fallback to Price field",
			data: productDetailData{
				Price: "599,99 €",
			},
			want: "599,99 €",
		},
		{
			name: "V3 has empty PrimaryPrice",
			data: productDetailData{
				Price: "fallback",
				PriceInfoV3: &productPriceInfoV3{
					MainPriceTag: struct {
						PrimaryPrice           string  `json:"primaryPrice"`
						PrimaryPriceNoCurrency float64 `json:"primaryPriceNoCurrency"`
						SecondaryPrice         string  `json:"secondaryPrice"`
					}{
						PrimaryPrice: "",
					},
				},
				PriceInfoV2: &productPriceInfoV2{
					PriceWithVat: "v2price",
				},
			},
			want: "v2price",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pickPrice(tt.data)
			if got != tt.want {
				t.Errorf("pickPrice() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPickPriceWithoutVat(t *testing.T) {
	tests := []struct {
		name string
		data productDetailData
		want string
	}{
		{
			name: "from PriceInfoV3",
			data: productDetailData{
				PriceInfoV3: &productPriceInfoV3{
					MainPriceTag: struct {
						PrimaryPrice           string  `json:"primaryPrice"`
						PrimaryPriceNoCurrency float64 `json:"primaryPriceNoCurrency"`
						SecondaryPrice         string  `json:"secondaryPrice"`
					}{
						SecondaryPrice: "1073,55 €",
					},
				},
			},
			want: "1073,55 €",
		},
		{
			name: "from PriceInfoV2 when V3 has no secondary",
			data: productDetailData{
				PriceInfoV3: &productPriceInfoV3{},
				PriceInfoV2: &productPriceInfoV2{
					PriceWithoutVat: "826,44 €",
				},
			},
			want: "826,44 €",
		},
		{
			name: "empty when nothing available",
			data: productDetailData{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pickPriceWithoutVat(tt.data)
			if got != tt.want {
				t.Errorf("pickPriceWithoutVat() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPickPriceNoCurrency(t *testing.T) {
	tests := []struct {
		name string
		data productDetailData
		want float64
	}{
		{
			name: "from PriceInfoV3",
			data: productDetailData{
				GaPrice: 100,
				PriceInfoV3: &productPriceInfoV3{
					MainPriceTag: struct {
						PrimaryPrice           string  `json:"primaryPrice"`
						PrimaryPriceNoCurrency float64 `json:"primaryPriceNoCurrency"`
						SecondaryPrice         string  `json:"secondaryPrice"`
					}{
						PrimaryPriceNoCurrency: 1299.99,
					},
				},
			},
			want: 1299.99,
		},
		{
			name: "from PriceInfoV2",
			data: productDetailData{
				GaPrice: 100,
				PriceInfoV2: &productPriceInfoV2{
					PriceNoCurrency: 999.99,
				},
			},
			want: 999.99,
		},
		{
			name: "fallback to GaPrice",
			data: productDetailData{
				GaPrice: 599.99,
			},
			want: 599.99,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pickPriceNoCurrency(tt.data)
			if got != tt.want {
				t.Errorf("pickPriceNoCurrency() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestMapProductParameters(t *testing.T) {
	groups := []productParameterGroup{
		{
			Name: "Display",
			Params: []productParameterEntry{
				{
					Name:   "Size",
					Values: []productParameterValue{{Desc: "6.1 inch"}},
				},
				{
					Name:   "Type",
					Values: []productParameterValue{{Desc: "OLED"}, {Desc: "Super Retina XDR"}},
				},
			},
		},
		{
			Name: "Battery",
			Params: []productParameterEntry{
				{
					Name:   "Capacity",
					Values: []productParameterValue{{Desc: "3095 mAh"}},
				},
			},
		},
		{
			Name: "", // Should be skipped
			Params: []productParameterEntry{
				{Name: "Skipped", Values: []productParameterValue{{Desc: "value"}}},
			},
		},
	}

	result := mapProductParameters(groups)

	if len(result) != 2 {
		t.Fatalf("mapProductParameters() returned %d groups, want 2", len(result))
	}

	if result[0].Name != "Display" {
		t.Errorf("first group name = %q, want Display", result[0].Name)
	}

	if len(result[0].Parameters) != 2 {
		t.Errorf("Display group has %d params, want 2", len(result[0].Parameters))
	}

	// Check multi-value parameter
	typeParam := result[0].Parameters[1]
	if len(typeParam.Values) != 2 {
		t.Errorf("Type param has %d values, want 2", len(typeParam.Values))
	}
}

func TestMapProductParametersEmptyValues(t *testing.T) {
	groups := []productParameterGroup{
		{
			Name: "Test",
			Params: []productParameterEntry{
				{
					Name:   "Empty",
					Values: []productParameterValue{}, // No values
				},
				{
					Name:   "HasValue",
					Values: []productParameterValue{{Desc: "value"}},
				},
			},
		},
	}

	result := mapProductParameters(groups)

	if len(result) != 1 {
		t.Fatalf("mapProductParameters() returned %d groups, want 1", len(result))
	}

	// Empty param should be skipped
	if len(result[0].Parameters) != 1 {
		t.Errorf("group has %d params, want 1 (empty should be skipped)", len(result[0].Parameters))
	}
}

func TestMapProductVariants(t *testing.T) {
	info := &productVariantsInfo{
		Type: 1,
		ProductVariants: []productVariantRaw{
			{ID: 1, Name: "Black", ImageURL: "https://example.com/black.jpg", IsSelected: true},
			{ID: 2, Name: "White", ImageURL: "https://example.com/white.jpg", IsSelected: false},
		},
	}

	result := mapProductVariants(info)

	if len(result) != 2 {
		t.Fatalf("mapProductVariants() returned %d variants, want 2", len(result))
	}

	if result[0].ID != 1 || result[0].Name != "Black" || !result[0].IsSelected {
		t.Errorf("first variant = %+v, expected Black selected", result[0])
	}

	if result[1].ID != 2 || result[1].Name != "White" || result[1].IsSelected {
		t.Errorf("second variant = %+v, expected White not selected", result[1])
	}
}

func TestMapProductVariantsNil(t *testing.T) {
	result := mapProductVariants(nil)
	if result != nil {
		t.Errorf("mapProductVariants(nil) = %v, want nil", result)
	}
}

func TestMapProductVariantsEmpty(t *testing.T) {
	info := &productVariantsInfo{ProductVariants: nil}
	result := mapProductVariants(info)
	if result != nil {
		t.Errorf("mapProductVariants(empty) = %v, want nil", result)
	}
}

func TestPickPromoPrices(t *testing.T) {
	data := productDetailData{
		PriceInfoV3: &productPriceInfoV3{
			PromoPrices: []productPromoPrice{
				{
					Name:               "Member Discount",
					PrimaryPrice:       "899,99 €",
					UnformattedPrice:   899.99,
					DiscountCouponCode: "MEMBER10",
				},
			},
		},
	}

	result := pickPromoPrices(data)

	if len(result) != 1 {
		t.Fatalf("pickPromoPrices() returned %d promos, want 1", len(result))
	}

	if result[0].Name != "Member Discount" {
		t.Errorf("promo name = %q, want Member Discount", result[0].Name)
	}
	if result[0].Price != "899,99 €" {
		t.Errorf("promo price = %q, want 899,99 €", result[0].Price)
	}
	if result[0].Code != "MEMBER10" {
		t.Errorf("promo code = %q, want MEMBER10", result[0].Code)
	}
}

func TestPickPromoPricesFromV2(t *testing.T) {
	data := productDetailData{
		PriceInfoV2: &productPriceInfoV2{
			PromoPrices: []productPromoPrice{
				{
					Name:           "V2 Promo",
					FormattedPrice: "799,99 €",
				},
			},
		},
	}

	result := pickPromoPrices(data)

	if len(result) != 1 {
		t.Fatalf("pickPromoPrices() returned %d promos, want 1", len(result))
	}

	// Should use FormattedPrice when PrimaryPrice is empty
	if result[0].Price != "799,99 €" {
		t.Errorf("promo price = %q, want 799,99 €", result[0].Price)
	}
}

func TestProductVariantsInfoUnmarshalUppercase(t *testing.T) {
	jsonData := `{
		"Type": 1,
		"ProductVariants": [
			{"Id": 123, "Name": "Test", "ImageUrl": "", "IsSelected": true}
		]
	}`

	var info productVariantsInfo
	if err := json.Unmarshal([]byte(jsonData), &info); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if info.Type != 1 {
		t.Errorf("Type = %d, want 1", info.Type)
	}
	if len(info.ProductVariants) != 1 {
		t.Errorf("ProductVariants len = %d, want 1", len(info.ProductVariants))
	}
}

func TestProductVariantsInfoUnmarshalLowercase(t *testing.T) {
	jsonData := `{
		"type": 2,
		"productVariants": [
			{"Id": 456, "Name": "Test2", "ImageUrl": "", "IsSelected": false}
		]
	}`

	var info productVariantsInfo
	if err := json.Unmarshal([]byte(jsonData), &info); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if info.Type != 2 {
		t.Errorf("Type = %d, want 2", info.Type)
	}
}

func TestProductVariantsInfoUnmarshalNull(t *testing.T) {
	var info productVariantsInfo
	if err := json.Unmarshal([]byte("null"), &info); err != nil {
		t.Fatalf("failed to unmarshal null: %v", err)
	}
}

func TestNormalizeExternalURL(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{
			name:  "protocol-relative URL",
			value: "//cdn.example.com/image.jpg",
			want:  "https://cdn.example.com/image.jpg",
		},
		{
			name:  "already https",
			value: "https://example.com/image.jpg",
			want:  "https://example.com/image.jpg",
		},
		{
			name:  "with whitespace",
			value: "  //cdn.example.com/image.jpg  ",
			want:  "https://cdn.example.com/image.jpg",
		},
		{
			name:  "relative path",
			value: "/images/photo.jpg",
			want:  "/images/photo.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeExternalURL(tt.value)
			if got != tt.want {
				t.Errorf("normalizeExternalURL(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestCollapseSpaces(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello   world", "hello world"},
		{"  leading", "leading"},
		{"trailing  ", "trailing"},
		{"  both  ", "both"},
		{"no extra spaces", "no extra spaces"},
		{"tabs\t\tand\nnewlines", "tabs and newlines"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := collapseSpaces(tt.input)
			if got != tt.want {
				t.Errorf("collapseSpaces(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestLooksLikeCookieNotice(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{"We use cookies", true},
		{"Cookie settings", true},
		{"COOKIES", true},
		{"Product description", false},
		{"", false},
		{"This is a coookie typo", false},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			got := looksLikeCookieNotice(tt.text)
			if got != tt.want {
				t.Errorf("looksLikeCookieNotice(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestExtractDescriptionFromHTML(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "basic paragraph",
			html: "<html><body><p>Product description</p></body></html>",
			want: "Product description",
		},
		{
			name: "multiple elements",
			html: "<html><body><h1>Title</h1><p>Description</p><li>Feature</li></body></html>",
			want: "Title\nDescription\nFeature",
		},
		{
			name: "skip scripts",
			html: "<html><body><script>alert('x')</script><p>Content</p></body></html>",
			want: "Content",
		},
		{
			name: "skip cookie notice",
			html: "<html><body><p>We use cookies</p><p>Real content</p></body></html>",
			want: "Real content",
		},
		{
			name: "empty string",
			html: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractDescriptionFromHTML(tt.html)
			if got != tt.want {
				t.Errorf("extractDescriptionFromHTML() = %q, want %q", got, tt.want)
			}
		})
	}
}
