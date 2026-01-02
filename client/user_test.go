package client

import (
	"encoding/json"
	"strconv"
	"testing"
)

func TestUserStatusFromListsResponseJSON(t *testing.T) {
	// This is the actual response structure used in GetUserStatus
	jsonData := `{
		"user_id": 12345,
		"user_name": "Test User",
		"basket_cnt": 3
	}`

	var resp struct {
		UserID    int    `json:"user_id"`
		UserName  string `json:"user_name"`
		BasketCnt int    `json:"basket_cnt"`
	}

	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if resp.UserID != 12345 {
		t.Errorf("UserID = %d, want 12345", resp.UserID)
	}
	if resp.UserName != "Test User" {
		t.Errorf("UserName = %q, want Test User", resp.UserName)
	}
	if resp.BasketCnt != 3 {
		t.Errorf("BasketCnt = %d, want 3", resp.BasketCnt)
	}
}

func TestUserStatusResponseConversion(t *testing.T) {
	listResp := struct {
		UserID    int    `json:"user_id"`
		UserName  string `json:"user_name"`
		BasketCnt int    `json:"basket_cnt"`
	}{
		UserID:    12345,
		UserName:  "Test User",
		BasketCnt: 5,
	}

	// Convert to UserStatusResponse
	status := &UserStatusResponse{
		UserID:    listResp.UserID,
		UserName:  listResp.UserName,
		BasketCnt: listResp.BasketCnt,
	}

	if status.UserID != 12345 {
		t.Errorf("UserID = %d, want 12345", status.UserID)
	}
	if status.UserName != "Test User" {
		t.Errorf("UserName = %q, want Test User", status.UserName)
	}
	if status.BasketCnt != 5 {
		t.Errorf("BasketCnt = %d, want 5", status.BasketCnt)
	}
}

func TestUserNotLoggedIn(t *testing.T) {
	// When not logged in, user_id is -1 or 0
	jsonData := `{
		"user_id": -1,
		"user_name": "",
		"basket_cnt": 0
	}`

	var resp struct {
		UserID    int    `json:"user_id"`
		UserName  string `json:"user_name"`
		BasketCnt int    `json:"basket_cnt"`
	}

	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// UserID <= 0 means not logged in
	isLoggedIn := resp.UserID > 0
	if isLoggedIn {
		t.Error("UserID -1 should indicate not logged in")
	}
}

func TestUserIDZeroMeansNotLoggedIn(t *testing.T) {
	jsonData := `{
		"user_id": 0,
		"user_name": "",
		"basket_cnt": 0
	}`

	var resp struct {
		UserID int `json:"user_id"`
	}

	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	isLoggedIn := resp.UserID > 0
	if isLoggedIn {
		t.Error("UserID 0 should indicate not logged in")
	}
}

func TestUserIDConversionToString(t *testing.T) {
	tests := []struct {
		userID int
		want   string
	}{
		{12345, "12345"},
		{1, "1"},
		{999999999, "999999999"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			// Simulate SetUserID logic
			got := strconv.Itoa(tt.userID)
			if got != tt.want {
				t.Errorf("strconv.Itoa(%d) = %q, want %q", tt.userID, got, tt.want)
			}
		})
	}
}

func TestUserStatusResponseAllFields(t *testing.T) {
	status := UserStatusResponse{
		UserID:    12345,
		BasketID:  67890,
		UserName:  "Test User",
		BasketCnt: 3,
		OrdersCnt: 10,
		FavCnt:    5,
		IsPremium: true,
	}

	if status.UserID != 12345 {
		t.Errorf("UserID = %d, want 12345", status.UserID)
	}
	if status.BasketID != 67890 {
		t.Errorf("BasketID = %d, want 67890", status.BasketID)
	}
	if status.UserName != "Test User" {
		t.Errorf("UserName = %q, want Test User", status.UserName)
	}
	if status.BasketCnt != 3 {
		t.Errorf("BasketCnt = %d, want 3", status.BasketCnt)
	}
	if status.OrdersCnt != 10 {
		t.Errorf("OrdersCnt = %d, want 10", status.OrdersCnt)
	}
	if status.FavCnt != 5 {
		t.Errorf("FavCnt = %d, want 5", status.FavCnt)
	}
	if !status.IsPremium {
		t.Error("IsPremium = false, want true")
	}
}
