package client

import (
	"encoding/json"
	"fmt"
	"time"
)

type ordersArchiveResponse struct {
	Paging struct {
		Size int `json:"size"`
	} `json:"paging"`
	Value []struct {
		OrderID    string `json:"orderId"`
		Created    string `json:"created"`
		State      string `json:"state"`
		TotalPrice string `json:"totalPrice"`
	} `json:"value"`
}

type ordersActiveResponse struct {
	Groups []struct {
		Orders []struct {
			OrderID string `json:"orderId"`
			Created string `json:"created"`
			Parts   []struct {
				Status     string `json:"status"`
				TotalPrice string `json:"totalPrice"`
			} `json:"parts"`
		} `json:"orders"`
	} `json:"groups"`
}

func (c *TLSClient) GetOrders(limit int) ([]Order, int, error) {
	if c.userID == "" {
		if _, err := c.GetUserStatus(); err != nil {
			return nil, 0, err
		}
	}

	if limit <= 0 {
		limit = 10
	}

	activeOrders, err := c.getActiveOrders()
	if err != nil {
		return nil, 0, err
	}

	remaining := limit - len(activeOrders)
	if remaining < 0 {
		remaining = 0
	}

	archiveLimit := remaining
	if archiveLimit <= 0 {
		archiveLimit = 1
	}

	archiveOrders, archiveTotal, err := c.getArchiveOrders(archiveLimit)
	if err != nil {
		return nil, 0, err
	}

	orders := make([]Order, 0, len(activeOrders)+len(archiveOrders))
	orders = append(orders, activeOrders...)
	if remaining > 0 {
		orders = append(orders, archiveOrders...)
	}
	if len(orders) > limit {
		orders = orders[:limit]
	}

	total := archiveTotal + len(activeOrders)
	return orders, total, nil
}

func (c *TLSClient) getArchiveOrders(limit int) ([]Order, int, error) {
	endpoint := fmt.Sprintf(EndpointOrdersArchive, c.userID, limit)
	data, err := c.Get(endpoint)
	if err != nil {
		return nil, 0, err
	}

	var resp ordersArchiveResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, 0, fmt.Errorf("failed to parse orders: %w", err)
	}

	orders := make([]Order, 0, len(resp.Value))
	for _, o := range resp.Value {
		orders = append(orders, Order{
			ID:         o.OrderID,
			Date:       formatOrderDate(o.Created),
			Status:     o.State,
			TotalPrice: o.TotalPrice,
		})
	}

	return orders, resp.Paging.Size, nil
}

func (c *TLSClient) getActiveOrders() ([]Order, error) {
	endpoint := fmt.Sprintf(EndpointOrdersActive, c.userID)
	data, err := c.Get(endpoint)
	if err != nil {
		return nil, err
	}

	var resp ordersActiveResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse active orders: %w", err)
	}

	orders := []Order{}
	for _, group := range resp.Groups {
		for _, o := range group.Orders {
			status := ""
			total := ""
			if len(o.Parts) > 0 {
				status = o.Parts[0].Status
				total = o.Parts[0].TotalPrice
			}

			orders = append(orders, Order{
				ID:         o.OrderID,
				Date:       formatOrderDate(o.Created),
				Status:     status,
				TotalPrice: total,
			})
		}
	}

	return orders, nil
}

func formatOrderDate(value string) string {
	if value == "" {
		return value
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	return parsed.Format("2006-01-02")
}
