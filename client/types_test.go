package client

import (
	"encoding/json"
	"testing"
)

func TestUserStatusResponseJSON(t *testing.T) {
	jsonData := `{
		"userId": 12345,
		"basketId": 67890,
		"userName": "Test User",
		"basketItemsCount": 3,
		"ordersCount": 10,
		"watchDogCommoditiesCount": 5,
		"isAlzaPlus": true
	}`

	var resp UserStatusResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if resp.UserID != 12345 {
		t.Errorf("UserID = %d, want 12345", resp.UserID)
	}
	if resp.BasketID != 67890 {
		t.Errorf("BasketID = %d, want 67890", resp.BasketID)
	}
	if resp.UserName != "Test User" {
		t.Errorf("UserName = %q, want Test User", resp.UserName)
	}
	if resp.BasketCnt != 3 {
		t.Errorf("BasketCnt = %d, want 3", resp.BasketCnt)
	}
	if resp.OrdersCnt != 10 {
		t.Errorf("OrdersCnt = %d, want 10", resp.OrdersCnt)
	}
	if resp.FavCnt != 5 {
		t.Errorf("FavCnt = %d, want 5", resp.FavCnt)
	}
	if !resp.IsPremium {
		t.Error("IsPremium = false, want true")
	}
}

func TestSearchResultJSON(t *testing.T) {
	jsonData := `{
		"id": 12345,
		"name": "Test Product",
		"code": "ABC123",
		"price": 99.99,
		"priceStr": "99,99 €",
		"availability": "in stock",
		"imageUrl": "https://example.com/image.jpg",
		"url": "/product-d12345.htm"
	}`

	var result SearchResult
	if err := json.Unmarshal([]byte(jsonData), &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if result.ID != 12345 {
		t.Errorf("ID = %d, want 12345", result.ID)
	}
	if result.Name != "Test Product" {
		t.Errorf("Name = %q, want Test Product", result.Name)
	}
	if result.Code != "ABC123" {
		t.Errorf("Code = %q, want ABC123", result.Code)
	}
	if result.Price != 99.99 {
		t.Errorf("Price = %f, want 99.99", result.Price)
	}
}

func TestCommodityListJSON(t *testing.T) {
	jsonData := `{
		"id": 1,
		"name": "My List",
		"itemCount": 5,
		"type": 0,
		"canModify": true
	}`

	var list CommodityList
	if err := json.Unmarshal([]byte(jsonData), &list); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if list.ID != 1 {
		t.Errorf("ID = %d, want 1", list.ID)
	}
	if list.Name != "My List" {
		t.Errorf("Name = %q, want My List", list.Name)
	}
	if list.ItemCount != 5 {
		t.Errorf("ItemCount = %d, want 5", list.ItemCount)
	}
	if !list.CanModify {
		t.Error("CanModify = false, want true")
	}
}

func TestCartItemJSON(t *testing.T) {
	jsonData := `{
		"productId": 12345,
		"count": 2,
		"basketItemId": 67890,
		"name": "Test Product",
		"price": "199,99 €",
		"imageUrl": "https://example.com/image.jpg",
		"url": "/product-d12345.htm"
	}`

	var item CartItem
	if err := json.Unmarshal([]byte(jsonData), &item); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if item.ProductID != 12345 {
		t.Errorf("ProductID = %d, want 12345", item.ProductID)
	}
	if item.Count != 2 {
		t.Errorf("Count = %d, want 2", item.Count)
	}
	if item.BasketItemID != 67890 {
		t.Errorf("BasketItemID = %d, want 67890", item.BasketItemID)
	}
}

func TestOrderJSON(t *testing.T) {
	jsonData := `{
		"orderId": "ORD123",
		"orderDate": "2024-01-15",
		"status": "delivered",
		"totalPrice": "299,99 €"
	}`

	var order Order
	if err := json.Unmarshal([]byte(jsonData), &order); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if order.ID != "ORD123" {
		t.Errorf("ID = %q, want ORD123", order.ID)
	}
	if order.Status != "delivered" {
		t.Errorf("Status = %q, want delivered", order.Status)
	}
}

func TestProductDetailJSON(t *testing.T) {
	jsonData := `{
		"id": 12345,
		"name": "iPhone 15 Pro",
		"price": "1299,99 €",
		"priceWithoutVat": "1073,55 €",
		"priceNoCurrency": 1299.99,
		"discountPercent": 10,
		"availability": "In Stock",
		"description": "Amazing phone",
		"parameters": [
			{
				"name": "Display",
				"parameters": [
					{"name": "Size", "values": ["6.1\""]}
				]
			}
		],
		"variants": [
			{"id": 1, "name": "256GB", "isSelected": true}
		]
	}`

	var detail ProductDetail
	if err := json.Unmarshal([]byte(jsonData), &detail); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if detail.ID != 12345 {
		t.Errorf("ID = %d, want 12345", detail.ID)
	}
	if detail.Name != "iPhone 15 Pro" {
		t.Errorf("Name = %q, want iPhone 15 Pro", detail.Name)
	}
	if detail.PriceNoCurrency != 1299.99 {
		t.Errorf("PriceNoCurrency = %f, want 1299.99", detail.PriceNoCurrency)
	}
	if detail.DiscountPercent == nil || *detail.DiscountPercent != 10 {
		t.Errorf("DiscountPercent = %v, want 10", detail.DiscountPercent)
	}
}

func TestAddToListResponseJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		success bool
		errMsg  string
	}{
		{
			name:    "success",
			json:    `{"IsSuccess": true, "ErrorMessage": ""}`,
			success: true,
			errMsg:  "",
		},
		{
			name:    "failure",
			json:    `{"IsSuccess": false, "ErrorMessage": "Item already in list"}`,
			success: false,
			errMsg:  "Item already in list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp AddToListResponse
			if err := json.Unmarshal([]byte(tt.json), &resp); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			if resp.IsSuccess != tt.success {
				t.Errorf("IsSuccess = %v, want %v", resp.IsSuccess, tt.success)
			}
			if resp.ErrorMessage != tt.errMsg {
				t.Errorf("ErrorMessage = %q, want %q", resp.ErrorMessage, tt.errMsg)
			}
		})
	}
}

func TestProductVariantJSON(t *testing.T) {
	jsonData := `{
		"id": 123,
		"name": "Black 256GB",
		"imageUrl": "https://example.com/black.jpg",
		"isSelected": false
	}`

	var variant ProductVariant
	if err := json.Unmarshal([]byte(jsonData), &variant); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if variant.ID != 123 {
		t.Errorf("ID = %d, want 123", variant.ID)
	}
	if variant.Name != "Black 256GB" {
		t.Errorf("Name = %q, want Black 256GB", variant.Name)
	}
	if variant.IsSelected {
		t.Error("IsSelected = true, want false")
	}
}

func TestProductPromoPriceJSON(t *testing.T) {
	jsonData := `{
		"name": "Member Discount",
		"price": "999,99 €",
		"code": "MEMBER10",
		"unformattedPrice": 999.99
	}`

	var promo ProductPromoPrice
	if err := json.Unmarshal([]byte(jsonData), &promo); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if promo.Name != "Member Discount" {
		t.Errorf("Name = %q, want Member Discount", promo.Name)
	}
	if promo.Price != "999,99 €" {
		t.Errorf("Price = %q, want 999,99 €", promo.Price)
	}
	if promo.Code != "MEMBER10" {
		t.Errorf("Code = %q, want MEMBER10", promo.Code)
	}
	if promo.UnformattedPrice != 999.99 {
		t.Errorf("UnformattedPrice = %f, want 999.99", promo.UnformattedPrice)
	}
}
