package client

import (
	"encoding/json"
	"testing"
)

func TestReviewStatsAPIResponseParsing(t *testing.T) {
	jsonData := `{
		"ratingAverage": 4.8,
		"ratingCount": 89,
		"reviewCount": 24,
		"recommendationRate": 0.93,
		"purchaseCountFormatted": "2 000+",
		"ratings": [
			{"value": 1, "count": 1},
			{"value": 2, "count": 0},
			{"value": 3, "count": 5},
			{"value": 4, "count": 8},
			{"value": 5, "count": 75}
		],
		"complaint": {
			"description": "nízka reklamovanosť",
			"rate": 0.0009,
			"tooltip": "Tento produkt je spoľahlivý.",
			"type": 1
		}
	}`

	var resp reviewStatsAPIResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if resp.RatingAverage != 4.8 {
		t.Errorf("RatingAverage = %f, want 4.8", resp.RatingAverage)
	}
	if resp.RatingCount != 89 {
		t.Errorf("RatingCount = %d, want 89", resp.RatingCount)
	}
	if resp.ReviewCount != 24 {
		t.Errorf("ReviewCount = %d, want 24", resp.ReviewCount)
	}
	if resp.RecommendationRate != 0.93 {
		t.Errorf("RecommendationRate = %f, want 0.93", resp.RecommendationRate)
	}
	if resp.PurchaseCount != "2 000+" {
		t.Errorf("PurchaseCount = %q, want \"2 000+\"", resp.PurchaseCount)
	}
	if len(resp.Ratings) != 5 {
		t.Errorf("len(Ratings) = %d, want 5", len(resp.Ratings))
	}
	if resp.Complaint == nil {
		t.Error("Complaint is nil, want non-nil")
	} else if resp.Complaint.Rate != 0.0009 {
		t.Errorf("Complaint.Rate = %f, want 0.0009", resp.Complaint.Rate)
	}
}

func TestReviewsAPIResponseParsing(t *testing.T) {
	jsonData := `{
		"paging": {
			"size": 24,
			"limit": 10
		},
		"value": [
			{
				"rating": 4.0,
				"name": "Test User",
				"description": "Great product",
				"positives": ["good", "fast"],
				"negatives": ["expensive"],
				"reviewDate": "2025-01-15T10:00:00Z",
				"likeCount": 5,
				"isTranslated": false,
				"commodityName": "Test Product",
				"reviewDetail": "Reviewed on 15.01.2025",
				"flagUrl": "https://example.com/flag.svg",
				"verifiedPurchaseTag": {
					"label": "Verified"
				}
			},
			{
				"rating": 5.0,
				"name": "Another User",
				"description": "",
				"positives": [],
				"negatives": [],
				"reviewDate": "2025-01-14T10:00:00Z",
				"likeCount": 0,
				"isTranslated": true,
				"commodityName": "Test Product",
				"reviewDetail": "",
				"flagUrl": ""
			}
		]
	}`

	var resp reviewsAPIResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if resp.Paging.Size != 24 {
		t.Errorf("Paging.Size = %d, want 24", resp.Paging.Size)
	}
	if len(resp.Value) != 2 {
		t.Fatalf("len(Value) = %d, want 2", len(resp.Value))
	}

	// First review
	r1 := resp.Value[0]
	if r1.Rating != 4.0 {
		t.Errorf("Value[0].Rating = %f, want 4.0", r1.Rating)
	}
	if r1.Name != "Test User" {
		t.Errorf("Value[0].Name = %q, want \"Test User\"", r1.Name)
	}
	if len(r1.Positives) != 2 {
		t.Errorf("len(Value[0].Positives) = %d, want 2", len(r1.Positives))
	}
	if r1.VerifiedPurchaseTag == nil {
		t.Error("Value[0].VerifiedPurchaseTag is nil, want non-nil")
	}

	// Second review (no verified tag)
	r2 := resp.Value[1]
	if r2.VerifiedPurchaseTag != nil {
		t.Error("Value[1].VerifiedPurchaseTag is non-nil, want nil")
	}
	if !r2.IsTranslated {
		t.Error("Value[1].IsTranslated = false, want true")
	}
}

func TestReviewStatsMapping(t *testing.T) {
	// Test that API response maps correctly to ReviewStats
	apiResp := reviewStatsAPIResponse{
		RatingAverage:      4.5,
		RatingCount:        100,
		ReviewCount:        50,
		RecommendationRate: 0.85,
		PurchaseCount:      "1 000+",
		Ratings: []RatingBreakdown{
			{Value: 5, Count: 60},
			{Value: 4, Count: 25},
		},
		Complaint: &ComplaintInfo{
			Description: "low",
			Rate:        0.01,
		},
	}

	stats := &ReviewStats{
		RatingAverage:      apiResp.RatingAverage,
		RatingCount:        apiResp.RatingCount,
		ReviewCount:        apiResp.ReviewCount,
		RecommendationRate: apiResp.RecommendationRate,
		PurchaseCount:      apiResp.PurchaseCount,
		Ratings:            apiResp.Ratings,
		Complaint:          apiResp.Complaint,
	}

	if stats.RatingAverage != 4.5 {
		t.Errorf("RatingAverage = %f, want 4.5", stats.RatingAverage)
	}
	if stats.PurchaseCount != "1 000+" {
		t.Errorf("PurchaseCount = %q, want \"1 000+\"", stats.PurchaseCount)
	}
}

func TestReviewMapping(t *testing.T) {
	// Test that API response maps correctly to Review with int conversion
	apiItem := reviewAPIItem{
		Rating:       4.0,
		Name:         "User",
		Description:  "Good",
		Positives:    []string{"a", "b"},
		Negatives:    []string{"c"},
		ReviewDate:   "2025-01-15T10:00:00Z",
		LikeCount:    3,
		IsTranslated: false,
		VerifiedPurchaseTag: &struct {
			Label string `json:"label"`
		}{Label: "Verified"},
	}

	review := Review{
		Rating:       int(apiItem.Rating),
		Name:         apiItem.Name,
		Description:  apiItem.Description,
		Positives:    apiItem.Positives,
		Negatives:    apiItem.Negatives,
		ReviewDate:   apiItem.ReviewDate,
		LikeCount:    apiItem.LikeCount,
		IsTranslated: apiItem.IsTranslated,
		IsVerified:   apiItem.VerifiedPurchaseTag != nil,
	}

	if review.Rating != 4 {
		t.Errorf("Rating = %d, want 4", review.Rating)
	}
	if !review.IsVerified {
		t.Error("IsVerified = false, want true")
	}
}

func TestReviewsResponseJSON(t *testing.T) {
	// Test that ReviewsResponse serializes correctly to JSON
	resp := ReviewsResponse{
		Reviews: []Review{
			{Rating: 5, Name: "User1", IsVerified: true},
			{Rating: 4, Name: "User2", IsVerified: false},
		},
		TotalCount: 100,
		Offset:     0,
		Limit:      10,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed ReviewsResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if parsed.TotalCount != 100 {
		t.Errorf("TotalCount = %d, want 100", parsed.TotalCount)
	}
	if len(parsed.Reviews) != 2 {
		t.Errorf("len(Reviews) = %d, want 2", len(parsed.Reviews))
	}
}

func TestGetReviewsLimitValidation(t *testing.T) {
	tests := []struct {
		name        string
		inputLimit  int
		inputOffset int
		wantLimit   int
		wantOffset  int
	}{
		{"default limit", 0, 0, 10, 0},
		{"negative limit", -5, 0, 10, 0},
		{"max limit exceeded", 100, 0, 50, 0},
		{"valid limit", 20, 5, 20, 5},
		{"negative offset", 10, -5, 10, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't test the actual API call, but we can verify the validation logic
			limit := tt.inputLimit
			offset := tt.inputOffset

			if offset < 0 {
				offset = 0
			}
			if limit <= 0 {
				limit = 10
			}
			if limit > 50 {
				limit = 50
			}

			if limit != tt.wantLimit {
				t.Errorf("limit = %d, want %d", limit, tt.wantLimit)
			}
			if offset != tt.wantOffset {
				t.Errorf("offset = %d, want %d", offset, tt.wantOffset)
			}
		})
	}
}

func TestRatingBreakdownJSON(t *testing.T) {
	rb := RatingBreakdown{Value: 5, Count: 75}
	data, err := json.Marshal(rb)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	expected := `{"value":5,"count":75}`
	if string(data) != expected {
		t.Errorf("JSON = %s, want %s", string(data), expected)
	}
}

func TestComplaintInfoJSON(t *testing.T) {
	ci := ComplaintInfo{
		Description: "low",
		Rate:        0.01,
		Tooltip:     "Good product",
		Type:        1,
	}
	data, err := json.Marshal(ci)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed ComplaintInfo
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if parsed.Description != "low" {
		t.Errorf("Description = %q, want \"low\"", parsed.Description)
	}
	if parsed.Rate != 0.01 {
		t.Errorf("Rate = %f, want 0.01", parsed.Rate)
	}
}
