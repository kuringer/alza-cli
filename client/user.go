package client

import (
	"encoding/json"
	"fmt"
)

func (c *TLSClient) GetUserStatus() (*UserStatusResponse, error) {
	// Get basic user info from lists endpoint
	data, err := c.Get(EndpointCommodityLists)
	if err != nil {
		return nil, err
	}

	var listResp struct {
		UserID    int    `json:"user_id"`
		UserName  string `json:"user_name"`
		BasketCnt int    `json:"basket_cnt"`
	}
	if err := json.Unmarshal(data, &listResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// UserID -1 or 0 means not logged in / token expired
	if listResp.UserID <= 0 {
		return &UserStatusResponse{
			UserID:   listResp.UserID,
			UserName: listResp.UserName,
		}, nil
	}

	c.SetUserID(fmt.Sprintf("%d", listResp.UserID))

	// Get detailed status from statusSummary endpoint
	statusEndpoint := fmt.Sprintf(EndpointUserStatusSummary, c.userID)
	statusData, err := c.Get(statusEndpoint)
	if err != nil {
		// Fallback to basic info if statusSummary fails
		return &UserStatusResponse{
			UserID:    listResp.UserID,
			UserName:  listResp.UserName,
			BasketCnt: listResp.BasketCnt,
		}, nil
	}

	// Parse statusSummary response (different structure)
	var statusResp struct {
		NotificationsCount  int `json:"notificationsCount"`
		BasketProductsCount int `json:"basketProductsCount"`
		OrdersStatusInfo    struct {
			ActiveOrdersCount   int `json:"activeOrdersCount"`
			WaitingOrdersCount  int `json:"waitingOrdersCount"`
			OverdueOrdersCount  int `json:"overdueOrdersCount"`
			InactiveOrdersCount int `json:"inactiveOrdersCount"`
		} `json:"ordersStatusInfo"`
		BasketPreviewAction struct {
			Href string `json:"href"`
		} `json:"basketPreviewAction"`
		IsAlzaPlus bool `json:"isAlzaPlus"`
	}
	if err := json.Unmarshal(statusData, &statusResp); err != nil {
		// Fallback to basic info
		return &UserStatusResponse{
			UserID:    listResp.UserID,
			UserName:  listResp.UserName,
			BasketCnt: listResp.BasketCnt,
		}, nil
	}

	// Extract basket ID from preview URL (e.g., /api/basket/1538710316/preview)
	basketID := extractBasketID(statusResp.BasketPreviewAction.Href)

	return &UserStatusResponse{
		UserID:    listResp.UserID,
		UserName:  listResp.UserName,
		BasketID:  basketID,
		BasketCnt: statusResp.BasketProductsCount,
		OrdersCnt: statusResp.OrdersStatusInfo.ActiveOrdersCount + statusResp.OrdersStatusInfo.OverdueOrdersCount,
		IsPremium: statusResp.IsAlzaPlus,
	}, nil
}

func extractBasketID(href string) int {
	// Extract basket ID from URL like "/api/basket/1538710316/preview"
	if href == "" {
		return 0
	}
	var id int
	fmt.Sscanf(href, "https://www.alza.sk/api/basket/%d/preview", &id)
	if id == 0 {
		fmt.Sscanf(href, "/api/basket/%d/preview", &id)
	}
	return id
}
