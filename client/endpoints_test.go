package client

import (
	"fmt"
	"strings"
	"testing"
)

func TestEndpointConstants(t *testing.T) {
	endpoints := map[string]string{
		"EndpointAccessTokenPath":         EndpointAccessTokenPath,
		"EndpointCommodityLists":          EndpointCommodityLists,
		"EndpointCommodityListItems":      EndpointCommodityListItems,
		"EndpointCommodityListCreate":     EndpointCommodityListCreate,
		"EndpointCommodityListDelete":     EndpointCommodityListDelete,
		"EndpointCommodityListAddItem":    EndpointCommodityListAddItem,
		"EndpointUserCommodityListItems":  EndpointUserCommodityListItems,
		"EndpointUserStatusSummary":       EndpointUserStatusSummary,
		"EndpointCartItems":               EndpointCartItems,
		"EndpointCartPreview":             EndpointCartPreview,
		"EndpointOrderCommodity":          EndpointOrderCommodity,
		"EndpointOrderUpdate":             EndpointOrderUpdate,
		"EndpointSearchService":           EndpointSearchService,
		"EndpointWhisperAnon":             EndpointWhisperAnon,
		"EndpointWhisperUser":             EndpointWhisperUser,
		"EndpointOrdersArchive":           EndpointOrdersArchive,
		"EndpointOrdersActive":            EndpointOrdersActive,
		"EndpointProductDetail":           EndpointProductDetail,
		"EndpointProductAvailabilityUser": EndpointProductAvailabilityUser,
		"EndpointProductAvailabilityAnon": EndpointProductAvailabilityAnon,
		"EndpointFastOrderSave":           EndpointFastOrderSave,
		"EndpointFastOrderSend":           EndpointFastOrderSend,
		"EndpointPaymentRepeat":           EndpointPaymentRepeat,
	}

	for name, endpoint := range endpoints {
		if endpoint == "" {
			t.Errorf("%s is empty", name)
		}
	}
}

func TestEndpointFormatStrings(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		args     []interface{}
		wantOK   bool
	}{
		{
			name:     "CommodityListItems with ID",
			endpoint: EndpointCommodityListItems,
			args:     []interface{}{123},
			wantOK:   true,
		},
		{
			name:     "UserCommodityListItems with user ID",
			endpoint: EndpointUserCommodityListItems,
			args:     []interface{}{"user123"},
			wantOK:   true,
		},
		{
			name:     "UserStatusSummary with user ID",
			endpoint: EndpointUserStatusSummary,
			args:     []interface{}{"user123"},
			wantOK:   true,
		},
		{
			name:     "CartItems with basket ID",
			endpoint: EndpointCartItems,
			args:     []interface{}{"basket123"},
			wantOK:   true,
		},
		{
			name:     "CartPreview with basket ID",
			endpoint: EndpointCartPreview,
			args:     []interface{}{"basket123"},
			wantOK:   true,
		},
		{
			name:     "WhisperUser with user ID",
			endpoint: EndpointWhisperUser,
			args:     []interface{}{"user123"},
			wantOK:   true,
		},
		{
			name:     "OrdersArchive with user ID and limit",
			endpoint: EndpointOrdersArchive,
			args:     []interface{}{"user123", 10},
			wantOK:   true,
		},
		{
			name:     "OrdersActive with user ID",
			endpoint: EndpointOrdersActive,
			args:     []interface{}{"user123"},
			wantOK:   true,
		},
		{
			name:     "ProductDetail with product ID",
			endpoint: EndpointProductDetail,
			args:     []interface{}{12345},
			wantOK:   true,
		},
		{
			name:     "ProductAvailabilityUser with user and product ID",
			endpoint: EndpointProductAvailabilityUser,
			args:     []interface{}{"user123", 12345},
			wantOK:   true,
		},
		{
			name:     "ProductAvailabilityAnon with product ID",
			endpoint: EndpointProductAvailabilityAnon,
			args:     []interface{}{12345},
			wantOK:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fmt.Sprintf(tt.endpoint, tt.args...)
			// Should not contain unformatted %s or %d
			if strings.Contains(result, "%s") || strings.Contains(result, "%d") {
				t.Errorf("fmt.Sprintf(%q, %v) = %q, still contains format verbs", tt.endpoint, tt.args, result)
			}
		})
	}
}

func TestEndpointURLPatterns(t *testing.T) {
	// Test that endpoints start with expected patterns
	relativeEndpoints := []string{
		EndpointAccessTokenPath,
		EndpointCommodityLists,
		EndpointCommodityListItems,
		EndpointCommodityListCreate,
		EndpointCommodityListDelete,
		EndpointCommodityListAddItem,
		EndpointUserCommodityListItems,
		EndpointUserStatusSummary,
		EndpointCartItems,
		EndpointCartPreview,
		EndpointOrderCommodity,
		EndpointOrderUpdate,
		EndpointSearchService,
		EndpointOrdersArchive,
		EndpointOrdersActive,
		EndpointProductDetail,
		EndpointProductAvailabilityUser,
		EndpointProductAvailabilityAnon,
		EndpointFastOrderSave,
		EndpointFastOrderSend,
		EndpointPaymentRepeat,
	}

	for _, endpoint := range relativeEndpoints {
		if !strings.HasPrefix(endpoint, "/") {
			t.Errorf("endpoint %q should start with /", endpoint)
		}
	}

	absoluteEndpoints := []string{
		EndpointWhisperAnon,
		EndpointWhisperUser,
	}

	for _, endpoint := range absoluteEndpoints {
		if !strings.HasPrefix(endpoint, "https://") {
			t.Errorf("endpoint %q should start with https://", endpoint)
		}
	}
}
