package client

import (
	"encoding/json"
	"fmt"
	"strings"
)

// fetchBasketID gets basket ID from user status summary
func (c *TLSClient) fetchBasketID() error {
	if c.userID == "" {
		_, err := c.GetUserStatus()
		if err != nil {
			return err
		}
	}

	endpoint := fmt.Sprintf(EndpointUserStatusSummary, c.userID)
	data, err := c.Get(endpoint)
	if err != nil {
		return err
	}

	var resp struct {
		BasketProductsCount int `json:"basketProductsCount"`
		BasketPreviewAction struct {
			Href string `json:"href"`
		} `json:"basketPreviewAction"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("failed to parse status: %w", err)
	}

	// If cart is empty, basketPreviewAction might not have href
	href := resp.BasketPreviewAction.Href
	if href == "" {
		if resp.BasketProductsCount == 0 {
			// Cart is empty, no basket ID needed
			c.basketID = ""
			return nil
		}
		return fmt.Errorf("no basket preview action in response")
	}

	// Parse: https://www.alza.sk/api/basket/1538710316/preview
	parts := strings.Split(href, "/")
	for i, p := range parts {
		if p == "basket" && i+1 < len(parts) {
			c.basketID = parts[i+1]
			break
		}
	}

	if c.basketID == "" {
		return fmt.Errorf("could not extract basket ID from: %s", href)
	}

	if c.debug {
		fmt.Printf("[DEBUG] Basket ID: %s\n", c.basketID)
	}

	return nil
}

func (c *TLSClient) GetCart() ([]CartItem, error) {
	if c.basketID == "" {
		if err := c.fetchBasketID(); err != nil {
			return nil, err
		}
		// If still empty after fetch, cart is empty
		if c.basketID == "" {
			return []CartItem{}, nil
		}
	}

	// Get cart items
	endpoint := fmt.Sprintf(EndpointCartItems, c.basketID)
	data, err := c.Get(endpoint)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Items []struct {
			ProductID    int `json:"productId"`
			Count        int `json:"count"`
			BasketItemID int `json:"basketItemId"`
		} `json:"items"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse cart items: %w", err)
	}

	// Get basket preview for more details (names, prices)
	previewEndpoint := fmt.Sprintf(EndpointCartPreview, c.basketID)
	previewData, err := c.Get(previewEndpoint)
	if err != nil {
		// If preview fails, return basic items
		var items []CartItem
		for _, item := range resp.Items {
			items = append(items, CartItem{
				ProductID:    item.ProductID,
				Count:        item.Count,
				BasketItemID: item.BasketItemID,
			})
		}
		return items, nil
	}

	var preview struct {
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
	if err := json.Unmarshal(previewData, &preview); err != nil {
		// Fallback to basic items
		var items []CartItem
		for _, item := range resp.Items {
			items = append(items, CartItem{
				ProductID:    item.ProductID,
				Count:        item.Count,
				BasketItemID: item.BasketItemID,
			})
		}
		return items, nil
	}

	// Build map from productID to cart item data
	cartItemMap := make(map[int]struct {
		BasketItemID int
	})
	for _, item := range resp.Items {
		cartItemMap[item.ProductID] = struct{ BasketItemID int }{item.BasketItemID}
	}

	// Merge data - match by extracted product ID
	var items []CartItem
	for _, p := range preview.Items {
		productID := extractProductID(p.DetailAction.WebLink)
		basketItemID := 0
		if cartData, ok := cartItemMap[productID]; ok {
			basketItemID = cartData.BasketItemID
		}
		items = append(items, CartItem{
			ProductID:    productID,
			Count:        p.Count,
			BasketItemID: basketItemID,
			Name:         p.Name,
			Price:        p.Price,
			ImageURL:     p.ImageURL,
			URL:          p.DetailAction.WebLink,
		})
	}

	return items, nil
}

func (c *TLSClient) AddToCart(productID, quantity int) error {
	body := fmt.Sprintf(`{"id":%d,"count":%d}`, productID, quantity)
	_, err := c.Post(EndpointOrderCommodity, body)
	return err
}

func (c *TLSClient) ClearCart() error {
	if c.basketID == "" {
		if err := c.fetchBasketID(); err != nil {
			return err
		}
		if c.basketID == "" {
			// Cart already empty
			return nil
		}
	}

	endpoint := fmt.Sprintf(EndpointCartItems, c.basketID)
	_, err := c.Delete(endpoint)
	return err
}

func (c *TLSClient) RemoveFromCart(productID int) error {
	// Get cart items to find basketItemId
	items, err := c.GetCart()
	if err != nil {
		return err
	}

	var basketItemID int
	for _, item := range items {
		if item.ProductID == productID {
			basketItemID = item.BasketItemID
			break
		}
	}

	if basketItemID == 0 {
		return fmt.Errorf("product %d not found in cart", productID)
	}

	// Use OrderUpdate endpoint with count=0 to remove item
	body := fmt.Sprintf(`{"id":"%d","count":0,"addHook":null,"source":4,"accessoryvariant":null}`, basketItemID)
	_, err = c.Post(EndpointOrderUpdate, body)
	return err
}
