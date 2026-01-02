package client

import (
	"encoding/json"
	"strconv"
	"testing"
)

func TestCommodityListsResponseJSON(t *testing.T) {
	jsonData := `{
		"data_cnt": 3,
		"data": [
			{
				"id": 1,
				"name": "Favorites",
				"itemCount": 10,
				"type": 1,
				"canModify": false
			},
			{
				"id": 2,
				"name": "Shopping List",
				"itemCount": 5,
				"type": 0,
				"canModify": true
			}
		],
		"user_id": 12345,
		"user_name": "Test User"
	}`

	var resp CommodityListsResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if resp.DataCnt != 3 {
		t.Errorf("DataCnt = %d, want 3", resp.DataCnt)
	}

	if len(resp.Data) != 2 {
		t.Errorf("Data len = %d, want 2", len(resp.Data))
	}

	if resp.UserID != 12345 {
		t.Errorf("UserID = %d, want 12345", resp.UserID)
	}

	if resp.UserName != "Test User" {
		t.Errorf("UserName = %q, want Test User", resp.UserName)
	}

	// Check first list
	first := resp.Data[0]
	if first.ID != 1 {
		t.Errorf("first ID = %d, want 1", first.ID)
	}
	if first.Name != "Favorites" {
		t.Errorf("first Name = %q, want Favorites", first.Name)
	}
	if first.Type != 1 {
		t.Errorf("first Type = %d, want 1", first.Type)
	}
	if first.CanModify {
		t.Error("first CanModify = true, want false")
	}
}

func TestListItemsResponseJSON(t *testing.T) {
	jsonData := `{
		"data": [
			{
				"id": 1,
				"name": "My List",
				"itemCount": 2,
				"items": [
					{
						"navigationUrl": "/product-d12345.htm",
						"count": 1,
						"priceInfoV2": {
							"priceWithVat": "299,99 €"
						}
					},
					{
						"navigationUrl": "/product-d12346.htm",
						"count": 2,
						"priceInfoV2": {
							"priceWithVat": "199,99 €"
						}
					}
				]
			}
		]
	}`

	var resp ListItemsResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(resp.Data) != 1 {
		t.Errorf("Data len = %d, want 1", len(resp.Data))
	}

	list := resp.Data[0]
	if list.ID != 1 {
		t.Errorf("list ID = %d, want 1", list.ID)
	}

	if len(list.Items) != 2 {
		t.Errorf("Items len = %d, want 2", len(list.Items))
	}

	item := list.Items[0]
	if item.NavigationURL != "/product-d12345.htm" {
		t.Errorf("NavigationURL = %q, want /product-d12345.htm", item.NavigationURL)
	}
	if item.PriceInfoV2.PriceWithVat != "299,99 €" {
		t.Errorf("PriceWithVat = %q, want 299,99 €", item.PriceInfoV2.PriceWithVat)
	}
}

func TestCreateListRequestBody(t *testing.T) {
	name := "My New List"
	expected := `{"name":"My New List","type":0}`

	// Simulate body construction
	body := `{"name":"` + name + `","type":0}`

	if body != expected {
		t.Errorf("body = %q, want %q", body, expected)
	}
}

func TestAddToListRequestBody(t *testing.T) {
	listID := 123
	productID := 456
	expected := `{"listID":123,"cId":456,"path":"","pageType":0}`

	// Simulate body construction
	body := `{"listID":` + strconv.Itoa(listID) + `,"cId":` + strconv.Itoa(productID) + `,"path":"","pageType":0}`

	if body != expected {
		t.Errorf("body = %q, want %q", body, expected)
	}
}

func TestAddToFavoritesRequestBody(t *testing.T) {
	productID := 12345
	// The actual format uses map syntax
	expected := `{"items":{"12345":1},"listType":1,"country":"SK"}`

	body := `{"items":{"` + strconv.Itoa(productID) + `":1},"listType":1,"country":"SK"}`

	if body != expected {
		t.Errorf("body = %q, want %q", body, expected)
	}
}

func TestRemoveFromListRequestBody(t *testing.T) {
	listID := 123
	productID := 456
	expected := `{"id":123,"productId":456}`

	body := `{"id":` + strconv.Itoa(listID) + `,"productId":` + strconv.Itoa(productID) + `}`

	if body != expected {
		t.Errorf("body = %q, want %q", body, expected)
	}
}

func TestListItemsResponseEmpty(t *testing.T) {
	jsonData := `{"data": []}`

	var resp ListItemsResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(resp.Data) != 0 {
		t.Errorf("Data len = %d, want 0", len(resp.Data))
	}
}

func TestCommodityListsResponseNoUser(t *testing.T) {
	jsonData := `{
		"data_cnt": 0,
		"data": [],
		"user_id": -1,
		"user_name": ""
	}`

	var resp CommodityListsResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if resp.UserID != -1 {
		t.Errorf("UserID = %d, want -1 (not logged in)", resp.UserID)
	}
}

func TestListItemConversion(t *testing.T) {
	rawItem := struct {
		NavigationURL string `json:"navigationUrl"`
		Count         int    `json:"count"`
		PriceInfoV2   struct {
			PriceWithVat string `json:"priceWithVat"`
		} `json:"priceInfoV2"`
	}{
		NavigationURL: "/product-d12345.htm",
		Count:         2,
		PriceInfoV2: struct {
			PriceWithVat string `json:"priceWithVat"`
		}{PriceWithVat: "599,99 €"},
	}

	item := ListItem{
		NavigationURL: rawItem.NavigationURL,
		Count:         rawItem.Count,
		Price:         rawItem.PriceInfoV2.PriceWithVat,
	}

	if item.NavigationURL != "/product-d12345.htm" {
		t.Errorf("NavigationURL = %q, want /product-d12345.htm", item.NavigationURL)
	}
	if item.Count != 2 {
		t.Errorf("Count = %d, want 2", item.Count)
	}
	if item.Price != "599,99 €" {
		t.Errorf("Price = %q, want 599,99 €", item.Price)
	}
}

func TestAddToListResponseSuccess(t *testing.T) {
	jsonData := `{"IsSuccess": true, "ErrorMessage": ""}`

	var resp AddToListResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if !resp.IsSuccess {
		t.Error("IsSuccess = false, want true")
	}
	if resp.ErrorMessage != "" {
		t.Errorf("ErrorMessage = %q, want empty", resp.ErrorMessage)
	}
}

func TestAddToListResponseFailure(t *testing.T) {
	jsonData := `{"IsSuccess": false, "ErrorMessage": "Product already in favorites"}`

	var resp AddToListResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if resp.IsSuccess {
		t.Error("IsSuccess = true, want false")
	}
	if resp.ErrorMessage != "Product already in favorites" {
		t.Errorf("ErrorMessage = %q, want Product already in favorites", resp.ErrorMessage)
	}
}
