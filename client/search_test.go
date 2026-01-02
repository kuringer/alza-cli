package client

import (
	"encoding/json"
	"testing"
)

func TestSearchV5ResponseJSON(t *testing.T) {
	jsonData := `{
		"data2": [
			{
				"id": 12345,
				"name": "iPhone 15 Pro",
				"code": "IPH15PRO",
				"price": "1 299,99 €",
				"priceNoCurrency": 1299.99,
				"avail": "In Stock",
				"img": "https://example.com/iphone.jpg",
				"url": "/iphone-15-pro-d12345.htm"
			},
			{
				"id": 12346,
				"name": "iPhone 15",
				"code": "IPH15",
				"price": "999,99 €",
				"priceNoCurrency": 999.99,
				"avail": "Available",
				"img": "https://example.com/iphone15.jpg",
				"url": "/iphone-15-d12346.htm"
			}
		]
	}`

	var resp struct {
		Data2 []struct {
			ID              int     `json:"id"`
			Name            string  `json:"name"`
			Code            string  `json:"code"`
			Price           string  `json:"price"`
			PriceNoCurrency float64 `json:"priceNoCurrency"`
			Avail           string  `json:"avail"`
			Img             string  `json:"img"`
			URL             string  `json:"url"`
		} `json:"data2"`
	}

	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(resp.Data2) != 2 {
		t.Errorf("Data2 len = %d, want 2", len(resp.Data2))
	}

	first := resp.Data2[0]
	if first.ID != 12345 {
		t.Errorf("first ID = %d, want 12345", first.ID)
	}
	if first.Name != "iPhone 15 Pro" {
		t.Errorf("first Name = %q, want iPhone 15 Pro", first.Name)
	}
	if first.PriceNoCurrency != 1299.99 {
		t.Errorf("first PriceNoCurrency = %f, want 1299.99", first.PriceNoCurrency)
	}
}

func TestSearchWhisperResponseJSON(t *testing.T) {
	jsonData := `{
		"commodities": [
			{
				"imageUrl": "https://example.com/product.jpg",
				"clickAction": {
					"name": "MacBook Air M2",
					"webLink": "https://www.alza.sk/macbook-air-m2-d7654321.htm",
					"href": "/macbook-air-m2-d7654321.htm"
				}
			}
		]
	}`

	var resp struct {
		Commodities []struct {
			ImageURL    string `json:"imageUrl"`
			ClickAction struct {
				Name    string `json:"name"`
				WebLink string `json:"webLink"`
				Href    string `json:"href"`
			} `json:"clickAction"`
		} `json:"commodities"`
	}

	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(resp.Commodities) != 1 {
		t.Errorf("Commodities len = %d, want 1", len(resp.Commodities))
	}

	item := resp.Commodities[0]
	if item.ClickAction.Name != "MacBook Air M2" {
		t.Errorf("name = %q, want MacBook Air M2", item.ClickAction.Name)
	}
	if item.ImageURL != "https://example.com/product.jpg" {
		t.Errorf("imageUrl = %q, expected url", item.ImageURL)
	}
}

func TestSearchWhisperResponseEmptyLink(t *testing.T) {
	jsonData := `{
		"commodities": [
			{
				"imageUrl": "",
				"clickAction": {
					"name": "Product",
					"webLink": "",
					"href": "/product-d123.htm"
				}
			}
		]
	}`

	var resp struct {
		Commodities []struct {
			ImageURL    string `json:"imageUrl"`
			ClickAction struct {
				Name    string `json:"name"`
				WebLink string `json:"webLink"`
				Href    string `json:"href"`
			} `json:"clickAction"`
		} `json:"commodities"`
	}

	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	item := resp.Commodities[0]
	// When webLink is empty, should fallback to href
	link := item.ClickAction.WebLink
	if link == "" {
		link = item.ClickAction.Href
	}

	if link != "/product-d123.htm" {
		t.Errorf("link = %q, want /product-d123.htm", link)
	}
}

func TestSearchResultBuildingFromV5(t *testing.T) {
	item := struct {
		ID              int     `json:"id"`
		Name            string  `json:"name"`
		Code            string  `json:"code"`
		Price           string  `json:"price"`
		PriceNoCurrency float64 `json:"priceNoCurrency"`
		Avail           string  `json:"avail"`
		Img             string  `json:"img"`
		URL             string  `json:"url"`
	}{
		ID:              12345,
		Name:            "Test Product",
		Code:            "TEST123",
		Price:           "99,99 €",
		PriceNoCurrency: 99.99,
		Avail:           "In Stock",
		Img:             "https://example.com/img.jpg",
		URL:             "/product-d12345.htm",
	}

	// Simulate building SearchResult from v5 response
	price := item.PriceNoCurrency
	if price == 0 {
		price = parsePrice(item.Price)
	}

	result := SearchResult{
		ID:           item.ID,
		Name:         item.Name,
		Code:         item.Code,
		Price:        price,
		PriceStr:     item.Price,
		Availability: item.Avail,
		ImageURL:     item.Img,
		URL:          item.URL,
	}

	if result.ID != 12345 {
		t.Errorf("ID = %d, want 12345", result.ID)
	}
	if result.Price != 99.99 {
		t.Errorf("Price = %f, want 99.99", result.Price)
	}
	if result.Code != "TEST123" {
		t.Errorf("Code = %q, want TEST123", result.Code)
	}
}

func TestSearchResultBuildingFromWhisper(t *testing.T) {
	link := "https://www.alza.sk/product-d7654321.htm"
	productID := extractProductID(link)

	result := SearchResult{
		ID:       productID,
		Name:     "Test Product",
		ImageURL: "https://example.com/img.jpg",
		URL:      link,
	}

	if result.ID != 7654321 {
		t.Errorf("ID = %d, want 7654321", result.ID)
	}
}

func TestSearchV5EmptyResponse(t *testing.T) {
	jsonData := `{"data2": []}`

	var resp struct {
		Data2 []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"data2"`
	}

	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(resp.Data2) != 0 {
		t.Errorf("Data2 len = %d, want 0", len(resp.Data2))
	}
}

func TestSearchWhisperEmptyResponse(t *testing.T) {
	jsonData := `{"commodities": []}`

	var resp struct {
		Commodities []struct {
			ImageURL    string `json:"imageUrl"`
			ClickAction struct {
				Name string `json:"name"`
			} `json:"clickAction"`
		} `json:"commodities"`
	}

	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(resp.Commodities) != 0 {
		t.Errorf("Commodities len = %d, want 0", len(resp.Commodities))
	}
}

func TestParsePriceInSearch(t *testing.T) {
	// When priceNoCurrency is 0, should use parsePrice
	priceStr := "1 299,99 €"
	priceNoCurrency := 0.0

	price := priceNoCurrency
	if price == 0 {
		price = parsePrice(priceStr)
	}

	// parsePrice removes spaces and converts comma to dot
	if price != 1299.99 {
		t.Errorf("price = %f, want 1299.99", price)
	}
}
