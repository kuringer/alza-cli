package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kuringer/alza-cli/client"
)

func TestFilterOrdersByQueryReturnsOnlyMatchingOrders(t *testing.T) {
	orders := []client.Order{
		{
			ID: "ORD-1",
			Items: []client.OrderItem{
				{CommodityName: "Potravinová fólia VIPOR", Count: 1, Status: "Vybavené"},
				{CommodityName: "Káva", Count: 1, Status: "Vybavené"},
			},
		},
		{
			ID: "ORD-2",
			Items: []client.OrderItem{
				{CommodityName: "Proteín", Count: 1, Status: "Vybavené"},
			},
		},
	}

	filtered := filterOrdersByQuery(orders, "FÓLIA")
	if len(filtered) != 1 {
		t.Fatalf("len(filtered) = %d, want 1", len(filtered))
	}
	if filtered[0].ID != "ORD-1" {
		t.Fatalf("filtered[0].ID = %q, want ORD-1", filtered[0].ID)
	}
	if len(filtered[0].Items) != 1 {
		t.Fatalf("len(filtered[0].Items) = %d, want 1", len(filtered[0].Items))
	}
	if filtered[0].Items[0].CommodityName != "Potravinová fólia VIPOR" {
		t.Fatalf("filtered item = %q", filtered[0].Items[0].CommodityName)
	}
}

func TestFormatOrdersTextCompactByDefault(t *testing.T) {
	orders := []client.Order{{
		ID:         "ORD-1",
		Date:       "2025-10-16",
		Status:     "Vybavená",
		TotalPrice: "32,28 €",
		Items: []client.OrderItem{{
			CommodityName: "Potravinová fólia VIPOR",
			Count:         1,
			Status:        "Vybavené",
		}},
	}}

	out := formatOrdersText(orders, 1, "", false)
	if !strings.Contains(out, "Orders (showing 1 of 1):") {
		t.Fatalf("missing compact header in %q", out)
	}
	if strings.Contains(out, "Potravinová fólia VIPOR") {
		t.Fatalf("compact output unexpectedly included item lines: %q", out)
	}
}

func TestFormatOrdersTextShowsItemsWhenRequested(t *testing.T) {
	orders := []client.Order{{
		ID:         "ORD-1",
		Date:       "2025-10-16",
		Status:     "Vybavená",
		TotalPrice: "32,28 €",
		Items: []client.OrderItem{{
			CommodityName: "Potravinová fólia VIPOR",
			Count:         1,
			Status:        "Vybavené",
		}},
	}}

	out := formatOrdersText(orders, 1, "", true)
	if !strings.Contains(out, "Potravinová fólia VIPOR") {
		t.Fatalf("expected item line in %q", out)
	}
	if !strings.Contains(out, "qty 1") {
		t.Fatalf("expected quantity in %q", out)
	}
}

func TestFormatOrdersTextShowsQueryHeader(t *testing.T) {
	orders := []client.Order{{
		ID:         "ORD-1",
		Date:       "2025-10-16",
		Status:     "Vybavená",
		TotalPrice: "32,28 €",
		Items: []client.OrderItem{{
			CommodityName: "Potravinová fólia VIPOR",
			Count:         1,
			Status:        "Vybavené",
		}},
	}}

	out := formatOrdersText(orders, 25, "fólia", false)
	if !strings.Contains(out, `Orders matching "fólia" (showing 1 of 25):`) {
		t.Fatalf("missing query header in %q", out)
	}
	if !strings.Contains(out, "Potravinová fólia VIPOR") {
		t.Fatalf("query output should show matching items: %q", out)
	}
}

func TestCollectArchiveOrdersFetchesAllPages(t *testing.T) {
	calls := []int{}
	fetchPage := func(offset, limit int) ([]client.Order, int, error) {
		calls = append(calls, offset)
		switch offset {
		case 0:
			return []client.Order{{ID: "ORD-1"}, {ID: "ORD-2"}}, 3, nil
		case 2:
			return []client.Order{{ID: "ORD-3"}}, 3, nil
		default:
			t.Fatalf("unexpected offset %d", offset)
			return nil, 0, nil
		}
	}

	orders, total, err := collectArchiveOrders(fetchPage, 2)
	if err != nil {
		t.Fatalf("collectArchiveOrders returned error: %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d, want 3", total)
	}
	if len(orders) != 3 {
		t.Fatalf("len(orders) = %d, want 3", len(orders))
	}
	if strings.Join([]string{orders[0].ID, orders[1].ID, orders[2].ID}, ",") != "ORD-1,ORD-2,ORD-3" {
		t.Fatalf("unexpected order IDs: %+v", orders)
	}
	if strings.Join([]string{fmt.Sprint(calls[0]), fmt.Sprint(calls[1])}, ",") != "0,2" {
		t.Fatalf("unexpected offsets: %v", calls)
	}
}

func TestCollectArchiveOrdersHandlesUnderreportedTotal(t *testing.T) {
	calls := []int{}
	fetchPage := func(offset, limit int) ([]client.Order, int, error) {
		calls = append(calls, offset)
		switch offset {
		case 0:
			return []client.Order{{ID: "ORD-1"}, {ID: "ORD-2"}}, 1, nil
		case 2:
			return []client.Order{{ID: "ORD-3"}}, 1, nil
		default:
			t.Fatalf("unexpected offset %d", offset)
			return nil, 0, nil
		}
	}

	orders, total, err := collectArchiveOrders(fetchPage, 2)
	if err != nil {
		t.Fatalf("collectArchiveOrders returned error: %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d, want 3", total)
	}
	if len(orders) != 3 {
		t.Fatalf("len(orders) = %d, want 3", len(orders))
	}
	if strings.Join([]string{fmt.Sprint(calls[0]), fmt.Sprint(calls[1])}, ",") != "0,2" {
		t.Fatalf("unexpected offsets: %v", calls)
	}
}

func TestApplyOrdersQueryBeforeLimit(t *testing.T) {
	orders := []client.Order{
		{ID: "ORD-1", Items: []client.OrderItem{{CommodityName: "Proteín"}}},
		{ID: "ORD-2", Items: []client.OrderItem{{CommodityName: "Potravinová fólia VIPOR"}}},
	}

	prepared := limitOrders(filterOrdersByQuery(orders, "fólia"), 1)
	if len(prepared) != 1 {
		t.Fatalf("len(prepared) = %d, want 1", len(prepared))
	}
	if prepared[0].ID != "ORD-2" {
		t.Fatalf("prepared[0].ID = %q, want ORD-2", prepared[0].ID)
	}
}

func TestLimitOrdersDefaultsToTen(t *testing.T) {
	orders := make([]client.Order, 12)
	for i := range orders {
		orders[i] = client.Order{
			ID: fmt.Sprintf("ORD-%d", i+1),
			Items: []client.OrderItem{{
				CommodityName: "Potravinová fólia VIPOR",
			}},
		}
	}

	prepared := limitOrders(orders, 0)
	if len(prepared) != 10 {
		t.Fatalf("len(prepared) = %d, want 10", len(prepared))
	}
}

func TestFormatOrdersTextKeepsFractionalQuantity(t *testing.T) {
	orders := []client.Order{{
		ID:         "ORD-1",
		Date:       "2025-10-16",
		Status:     "Vybavená",
		TotalPrice: "32,28 €",
		Items: []client.OrderItem{{
			CommodityName: "Káva",
			Count:         1.5,
			Status:        "Vybavené",
		}},
	}}

	out := formatOrdersText(orders, 1, "", true)
	if !strings.Contains(out, "qty 1.5") {
		t.Fatalf("expected non-lossy quantity in %q", out)
	}
}

func TestOrdersForJSONDropsItemsUnlessRequested(t *testing.T) {
	orders := []client.Order{{
		ID: "ORD-1",
		Items: []client.OrderItem{{
			CommodityName: "Potravinová fólia VIPOR",
			Count:         1,
			Status:        "Vybavené",
		}},
	}}

	withoutItems := ordersForJSON(orders, false)
	if len(withoutItems[0].Items) != 0 {
		t.Fatalf("expected items removed, got %+v", withoutItems[0].Items)
	}

	withItems := ordersForJSON(orders, true)
	if len(withItems[0].Items) != 1 {
		t.Fatalf("expected items preserved, got %+v", withItems[0].Items)
	}
}
