package client

import (
	"encoding/json"
	"fmt"
)

type CommodityListsResponse struct {
	DataCnt  int             `json:"data_cnt"`
	Data     []CommodityList `json:"data"`
	UserID   int             `json:"user_id"`
	UserName string          `json:"user_name"`
}

type ListItemsResponse struct {
	Data []struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		ItemCount int    `json:"itemCount"`
		Items     []struct {
			NavigationURL string `json:"navigationUrl"`
			Count         int    `json:"count"`
			PriceInfoV2   struct {
				PriceWithVat string `json:"priceWithVat"`
			} `json:"priceInfoV2"`
		} `json:"items"`
	} `json:"data"`
}

func (c *TLSClient) GetLists() ([]CommodityList, error) {
	data, err := c.Get(EndpointCommodityLists)
	if err != nil {
		return nil, err
	}

	var resp CommodityListsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse lists: %w", err)
	}

	if resp.UserID > 0 {
		c.SetUserID(fmt.Sprintf("%d", resp.UserID))
	}

	return resp.Data, nil
}

func (c *TLSClient) GetListItems(listID int) ([]ListItem, error) {
	endpoint := fmt.Sprintf(EndpointCommodityListItems, listID)
	data, err := c.Get(endpoint)
	if err != nil {
		return nil, err
	}

	var resp ListItemsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("list not found")
	}

	var items []ListItem
	for _, item := range resp.Data[0].Items {
		items = append(items, ListItem{
			NavigationURL: item.NavigationURL,
			Count:         item.Count,
			Price:         item.PriceInfoV2.PriceWithVat,
		})
	}
	return items, nil
}

// CreateList creates a new commodity list
func (c *TLSClient) CreateList(name string) (*CommodityList, error) {
	body := fmt.Sprintf(`{"name":"%s","type":0}`, name)
	data, err := c.Post(EndpointCommodityListCreate, body)
	if err != nil {
		return nil, err
	}

	var resp CommodityListsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no list returned in response")
	}

	return &resp.Data[0], nil
}

// AddToList adds a product to a specific list
func (c *TLSClient) AddToList(listID int, productID int) error {
	body := fmt.Sprintf(`{"listID":%d,"cId":%d,"path":"","pageType":0}`, listID, productID)
	_, err := c.Post(EndpointCommodityListAddItem, body)
	return err
}

func (c *TLSClient) AddToFavorites(productID int) error {
	if c.userID == "" {
		_, err := c.GetUserStatus()
		if err != nil {
			return err
		}
	}

	endpoint := fmt.Sprintf(EndpointUserCommodityListItems, c.userID)
	body := fmt.Sprintf(`{"items":{"%d":1},"listType":1,"country":"SK"}`, productID)

	data, err := c.Post(endpoint, body)
	if err != nil {
		return err
	}

	var resp AddToListResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !resp.IsSuccess {
		return fmt.Errorf("failed to add: %s", resp.ErrorMessage)
	}

	return nil
}

func (c *TLSClient) RemoveFromList(listID int, productID int) error {
	body := fmt.Sprintf(`{"id":%d,"productId":%d}`, listID, productID)
	_, err := c.Post(EndpointCommodityListDelete, body)
	return err
}
