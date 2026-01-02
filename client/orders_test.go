package client

import (
	"encoding/json"
	"testing"
)

func TestFormatOrderDate(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{
			name:  "RFC3339 format",
			value: "2024-01-15T10:30:00Z",
			want:  "2024-01-15",
		},
		{
			name:  "RFC3339 with timezone",
			value: "2024-03-20T15:45:00+01:00",
			want:  "2024-03-20",
		},
		{
			name:  "empty string",
			value: "",
			want:  "",
		},
		{
			name:  "invalid format (passthrough)",
			value: "2024/01/15",
			want:  "2024/01/15",
		},
		{
			name:  "already formatted",
			value: "2024-01-15",
			want:  "2024-01-15",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatOrderDate(tt.value)
			if got != tt.want {
				t.Errorf("formatOrderDate(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestOrdersArchiveResponseJSON(t *testing.T) {
	jsonData := `{
		"paging": {"size": 100},
		"value": [
			{
				"orderId": "ORD123",
				"created": "2024-01-15T10:00:00Z",
				"state": "Delivered",
				"totalPrice": "299,99 €"
			},
			{
				"orderId": "ORD124",
				"created": "2024-01-10T08:30:00Z",
				"state": "Completed",
				"totalPrice": "149,99 €"
			}
		]
	}`

	var resp ordersArchiveResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if resp.Paging.Size != 100 {
		t.Errorf("Paging.Size = %d, want 100", resp.Paging.Size)
	}

	if len(resp.Value) != 2 {
		t.Errorf("Value len = %d, want 2", len(resp.Value))
	}

	if resp.Value[0].OrderID != "ORD123" {
		t.Errorf("first order ID = %q, want ORD123", resp.Value[0].OrderID)
	}

	if resp.Value[0].State != "Delivered" {
		t.Errorf("first order state = %q, want Delivered", resp.Value[0].State)
	}
}

func TestOrdersActiveResponseJSON(t *testing.T) {
	jsonData := `{
		"groups": [
			{
				"orders": [
					{
						"orderId": "ORD200",
						"created": "2024-02-01T12:00:00Z",
						"parts": [
							{
								"status": "In Transit",
								"totalPrice": "599,99 €"
							}
						]
					}
				]
			}
		]
	}`

	var resp ordersActiveResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(resp.Groups) != 1 {
		t.Errorf("Groups len = %d, want 1", len(resp.Groups))
	}

	if len(resp.Groups[0].Orders) != 1 {
		t.Errorf("Orders len = %d, want 1", len(resp.Groups[0].Orders))
	}

	order := resp.Groups[0].Orders[0]
	if order.OrderID != "ORD200" {
		t.Errorf("order ID = %q, want ORD200", order.OrderID)
	}

	if len(order.Parts) != 1 {
		t.Errorf("Parts len = %d, want 1", len(order.Parts))
	}

	if order.Parts[0].Status != "In Transit" {
		t.Errorf("part status = %q, want In Transit", order.Parts[0].Status)
	}
}

func TestOrdersActiveResponseEmptyParts(t *testing.T) {
	jsonData := `{
		"groups": [
			{
				"orders": [
					{
						"orderId": "ORD300",
						"created": "2024-02-05T10:00:00Z",
						"parts": []
					}
				]
			}
		]
	}`

	var resp ordersActiveResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	order := resp.Groups[0].Orders[0]
	if len(order.Parts) != 0 {
		t.Errorf("Parts len = %d, want 0", len(order.Parts))
	}
}

func TestOrdersArchiveResponseEmpty(t *testing.T) {
	jsonData := `{
		"paging": {"size": 0},
		"value": []
	}`

	var resp ordersArchiveResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if resp.Paging.Size != 0 {
		t.Errorf("Paging.Size = %d, want 0", resp.Paging.Size)
	}

	if len(resp.Value) != 0 {
		t.Errorf("Value len = %d, want 0", len(resp.Value))
	}
}

func TestOrdersActiveResponseEmptyGroups(t *testing.T) {
	jsonData := `{"groups": []}`

	var resp ordersActiveResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(resp.Groups) != 0 {
		t.Errorf("Groups len = %d, want 0", len(resp.Groups))
	}
}
