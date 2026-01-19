package client

import (
	"encoding/json"
	"fmt"
)

// ReviewStats contains aggregate review statistics for a product
type ReviewStats struct {
	RatingAverage      float64           `json:"ratingAverage"`
	RatingCount        int               `json:"ratingCount"`
	ReviewCount        int               `json:"reviewCount"`
	RecommendationRate float64           `json:"recommendationRate"`
	PurchaseCount      string            `json:"purchaseCountFormatted,omitempty"`
	Ratings            []RatingBreakdown `json:"ratings,omitempty"`
	Complaint          *ComplaintInfo    `json:"complaint,omitempty"`
}

// RatingBreakdown shows count per star rating
type RatingBreakdown struct {
	Value int `json:"value"`
	Count int `json:"count"`
}

// ComplaintInfo contains product reliability/complaint data
type ComplaintInfo struct {
	Description string  `json:"description"`
	Rate        float64 `json:"rate"`
	Tooltip     string  `json:"tooltip"`
	Type        int     `json:"type"`
}

// Review represents a single product review
type Review struct {
	Rating        int      `json:"rating"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Positives     []string `json:"positives"`
	Negatives     []string `json:"negatives"`
	ReviewDate    string   `json:"reviewDate"`
	LikeCount     int      `json:"likeCount"`
	IsTranslated  bool     `json:"isTranslated"`
	IsVerified    bool     `json:"isVerified"`
	CommodityName string   `json:"commodityName,omitempty"`
	ReviewDetail  string   `json:"reviewDetail,omitempty"`
	FlagURL       string   `json:"flagUrl,omitempty"`
}

// ReviewsResponse contains paginated reviews
type ReviewsResponse struct {
	Reviews    []Review `json:"reviews"`
	TotalCount int      `json:"totalCount"`
	Offset     int      `json:"offset"`
	Limit      int      `json:"limit"`
}

// Internal API response types
type reviewStatsAPIResponse struct {
	RatingAverage      float64           `json:"ratingAverage"`
	RatingCount        int               `json:"ratingCount"`
	ReviewCount        int               `json:"reviewCount"`
	RecommendationRate float64           `json:"recommendationRate"`
	PurchaseCount      string            `json:"purchaseCountFormatted"`
	Ratings            []RatingBreakdown `json:"ratings"`
	Complaint          *ComplaintInfo    `json:"complaint"`
}

type reviewsAPIResponse struct {
	Paging struct {
		Size int `json:"size"`
	} `json:"paging"`
	Value []reviewAPIItem `json:"value"`
}

type reviewAPIItem struct {
	Rating              float64  `json:"rating"`
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	Positives           []string `json:"positives"`
	Negatives           []string `json:"negatives"`
	ReviewDate          string   `json:"reviewDate"`
	LikeCount           int      `json:"likeCount"`
	IsTranslated        bool     `json:"isTranslated"`
	CommodityName       string   `json:"commodityName"`
	ReviewDetail        string   `json:"reviewDetail"`
	FlagURL             string   `json:"flagUrl"`
	VerifiedPurchaseTag *struct {
		Label string `json:"label"`
	} `json:"verifiedPurchaseTag"`
}

// GetReviewStats fetches aggregate review statistics for a product
func (c *TLSClient) GetReviewStats(productID int) (*ReviewStats, error) {
	endpoint := fmt.Sprintf(EndpointReviewStats, productID)
	data, err := c.Get(endpoint)
	if err != nil {
		return nil, err
	}

	var resp reviewStatsAPIResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse review stats: %w", err)
	}

	return &ReviewStats{
		RatingAverage:      resp.RatingAverage,
		RatingCount:        resp.RatingCount,
		ReviewCount:        resp.ReviewCount,
		RecommendationRate: resp.RecommendationRate,
		PurchaseCount:      resp.PurchaseCount,
		Ratings:            resp.Ratings,
		Complaint:          resp.Complaint,
	}, nil
}

// GetReviews fetches individual reviews for a product with pagination
func (c *TLSClient) GetReviews(productID, offset, limit int) (*ReviewsResponse, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	endpoint := fmt.Sprintf(EndpointReviews, productID, offset, limit)
	data, err := c.Get(endpoint)
	if err != nil {
		return nil, err
	}

	var resp reviewsAPIResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse reviews: %w", err)
	}

	reviews := make([]Review, 0, len(resp.Value))
	for _, item := range resp.Value {
		review := Review{
			Rating:        int(item.Rating),
			Name:          item.Name,
			Description:   item.Description,
			Positives:     item.Positives,
			Negatives:     item.Negatives,
			ReviewDate:    item.ReviewDate,
			LikeCount:     item.LikeCount,
			IsTranslated:  item.IsTranslated,
			CommodityName: item.CommodityName,
			ReviewDetail:  item.ReviewDetail,
			FlagURL:       item.FlagURL,
			IsVerified:    item.VerifiedPurchaseTag != nil,
		}
		reviews = append(reviews, review)
	}

	return &ReviewsResponse{
		Reviews:    reviews,
		TotalCount: resp.Paging.Size,
		Offset:     offset,
		Limit:      limit,
	}, nil
}
