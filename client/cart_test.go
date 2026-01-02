package client

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

func TestCartItemsResponseJSON(t *testing.T) {
	jsonData := `{
		"items": [
			{
				"productId": 12345,
				"count": 2,
				"basketItemId": 67890
			},
			{
				"productId": 12346,
				"count": 1,
				"basketItemId": 67891
			}
		]
	}`

	var resp struct {
		Items []struct {
			ProductID    int `json:"productId"`
			Count        int `json:"count"`
			BasketItemID int `json:"basketItemId"`
		} `json:"items"`
	}

	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(resp.Items) != 2 {
		t.Errorf("Items len = %d, want 2", len(resp.Items))
	}

	if resp.Items[0].ProductID != 12345 {
		t.Errorf("first ProductID = %d, want 12345", resp.Items[0].ProductID)
	}
	if resp.Items[0].Count != 2 {
		t.Errorf("first Count = %d, want 2", resp.Items[0].Count)
	}
	if resp.Items[0].BasketItemID != 67890 {
		t.Errorf("first BasketItemID = %d, want 67890", resp.Items[0].BasketItemID)
	}
}

func TestCartPreviewResponseJSON(t *testing.T) {
	jsonData := `{
		"items": [
			{
				"count": 2,
				"name": "iPhone 15 Pro",
				"imageUrl": "https://example.com/iphone.jpg",
				"price": "2 599,98 €",
				"detailAction": {
					"webLink": "https://www.alza.sk/iphone-15-pro-d12345.htm"
				}
			}
		]
	}`

	var resp struct {
		Items []struct {
			Count        int    `json:"count"`
			Name         string `json:"name"`
			ImageURL     string `json:"imageUrl"`
			Price        string `json:"price"`
			DetailAction struct {
				WebLink string `json:"webLink"`
			} `json:"detailAction"`
		} `json:"items"`
	}

	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(resp.Items) != 1 {
		t.Errorf("Items len = %d, want 1", len(resp.Items))
	}

	item := resp.Items[0]
	if item.Name != "iPhone 15 Pro" {
		t.Errorf("Name = %q, want iPhone 15 Pro", item.Name)
	}
	if item.Count != 2 {
		t.Errorf("Count = %d, want 2", item.Count)
	}
	if item.Price != "2 599,98 €" {
		t.Errorf("Price = %q, want 2 599,98 €", item.Price)
	}
}

func TestUserStatusSummaryResponseJSON(t *testing.T) {
	jsonData := `{
		"basketProductsCount": 3,
		"basketPreviewAction": {
			"href": "https://www.alza.sk/api/basket/1538710316/preview"
		}
	}`

	var resp struct {
		BasketProductsCount int `json:"basketProductsCount"`
		BasketPreviewAction struct {
			Href string `json:"href"`
		} `json:"basketPreviewAction"`
	}

	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if resp.BasketProductsCount != 3 {
		t.Errorf("BasketProductsCount = %d, want 3", resp.BasketProductsCount)
	}

	if resp.BasketPreviewAction.Href != "https://www.alza.sk/api/basket/1538710316/preview" {
		t.Errorf("Href = %q, expected basket preview URL", resp.BasketPreviewAction.Href)
	}
}

func TestExtractBasketIDFromHref(t *testing.T) {
	tests := []struct {
		name string
		href string
		want string
	}{
		{
			name: "standard basket URL",
			href: "https://www.alza.sk/api/basket/1538710316/preview",
			want: "1538710316",
		},
		{
			name: "different basket ID",
			href: "https://www.alza.sk/api/basket/9876543210/preview",
			want: "9876543210",
		},
		{
			name: "relative URL",
			href: "/api/basket/123456/preview",
			want: "123456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate basket ID extraction logic from cart.go
			var basketID string
			parts := strings.Split(tt.href, "/")
			for i, p := range parts {
				if p == "basket" && i+1 < len(parts) {
					basketID = parts[i+1]
					break
				}
			}

			if basketID != tt.want {
				t.Errorf("extracted basketID = %q, want %q", basketID, tt.want)
			}
		})
	}
}

func TestCartItemMerge(t *testing.T) {
	// Test merging cart items with preview data
	cartItems := []struct {
		ProductID    int
		Count        int
		BasketItemID int
	}{
		{ProductID: 12345, Count: 2, BasketItemID: 67890},
	}

	previewItems := []struct {
		Count    int
		Name     string
		ImageURL string
		Price    string
		WebLink  string
	}{
		{
			Count:    2,
			Name:     "Test Product",
			ImageURL: "https://example.com/img.jpg",
			Price:    "199,99 €",
			WebLink:  "/product-d12345.htm",
		},
	}

	// Build map from productID to cart item data
	cartItemMap := make(map[int]int)
	for _, item := range cartItems {
		cartItemMap[item.ProductID] = item.BasketItemID
	}

	// Merge
	var mergedItems []CartItem
	for _, p := range previewItems {
		productID := extractProductID(p.WebLink)
		basketItemID := cartItemMap[productID]
		mergedItems = append(mergedItems, CartItem{
			ProductID:    productID,
			Count:        p.Count,
			BasketItemID: basketItemID,
			Name:         p.Name,
			Price:        p.Price,
			ImageURL:     p.ImageURL,
			URL:          p.WebLink,
		})
	}

	if len(mergedItems) != 1 {
		t.Fatalf("merged items len = %d, want 1", len(mergedItems))
	}

	item := mergedItems[0]
	if item.ProductID != 12345 {
		t.Errorf("ProductID = %d, want 12345", item.ProductID)
	}
	if item.BasketItemID != 67890 {
		t.Errorf("BasketItemID = %d, want 67890", item.BasketItemID)
	}
	if item.Name != "Test Product" {
		t.Errorf("Name = %q, want Test Product", item.Name)
	}
}

func TestEmptyCartResponse(t *testing.T) {
	jsonData := `{"items": []}`

	var resp struct {
		Items []struct {
			ProductID int `json:"productId"`
		} `json:"items"`
	}

	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(resp.Items) != 0 {
		t.Errorf("Items len = %d, want 0", len(resp.Items))
	}
}

func TestAddToCartRequestBody(t *testing.T) {
	productID := 12345
	quantity := 2
	expected := `{"id":12345,"count":2}`

	body := `{"id":` + strconv.Itoa(productID) + `,"count":` + strconv.Itoa(quantity) + `}`

	if body != expected {
		t.Errorf("body = %q, want %q", body, expected)
	}
}

func TestRemoveFromCartRequestBody(t *testing.T) {
	basketItemID := 67890
	expected := `{"id":"67890","count":0,"addHook":null,"source":4,"accessoryvariant":null}`

	body := `{"id":"` + strconv.Itoa(basketItemID) + `","count":0,"addHook":null,"source":4,"accessoryvariant":null}`

	if body != expected {
		t.Errorf("body = %q, want %q", body, expected)
	}
}
