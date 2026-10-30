package main

import (
	"fmt"
	"strings"

	"github.com/kuringer/alza-cli/client"
)

const (
	archiveOrdersPageSize = 100
	defaultOrdersLimit    = 10
)

type archiveOrdersPageFetcher func(offset, limit int) ([]client.Order, int, error)

func filterOrdersByQuery(orders []client.Order, query string) []client.Order {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return orders
	}

	filtered := make([]client.Order, 0, len(orders))
	for _, order := range orders {
		matchedItems := make([]client.OrderItem, 0, len(order.Items))
		for _, item := range order.Items {
			if strings.Contains(strings.ToLower(item.CommodityName), query) {
				matchedItems = append(matchedItems, item)
			}
		}
		if len(matchedItems) == 0 {
			continue
		}
		order.Items = matchedItems
		filtered = append(filtered, order)
	}

	return filtered
}

func normalizeOrdersLimit(limit int) int {
	if limit <= 0 {
		return defaultOrdersLimit
	}
	return limit
}

func limitOrders(orders []client.Order, limit int) []client.Order {
	limit = normalizeOrdersLimit(limit)
	if len(orders) <= limit {
		return orders
	}
	return orders[:limit]
}

func ordersForJSON(orders []client.Order, includeItems bool) []client.Order {
	if includeItems {
		return orders
	}

	clean := make([]client.Order, len(orders))
	for i, order := range orders {
		order.Items = nil
		clean[i] = order
	}
	return clean
}

func collectArchiveOrders(fetchPage archiveOrdersPageFetcher, pageSize int) ([]client.Order, int, error) {
	if pageSize <= 0 {
		pageSize = archiveOrdersPageSize
	}

	all := []client.Order{}
	offset := 0
	total := 0
	for {
		page, pageTotal, err := fetchPage(offset, pageSize)
		if err != nil {
			return nil, 0, err
		}
		if pageTotal > total {
			total = pageTotal
		}
		all = append(all, page...)
		offset += len(page)
		if len(all) > total {
			total = len(all)
		}
		if len(page) == 0 || (len(page) < pageSize && offset >= total) {
			break
		}
	}

	return all, total, nil
}

func formatOrdersText(orders []client.Order, total int, query string, withItems bool) string {
	var b strings.Builder

	showItems := withItems || strings.TrimSpace(query) != ""
	if strings.TrimSpace(query) != "" {
		fmt.Fprintf(&b, "Orders matching %q (showing %d of %d):\n\n", query, len(orders), total)
	} else {
		fmt.Fprintf(&b, "Orders (showing %d of %d):\n\n", len(orders), total)
	}

	for _, order := range orders {
		fmt.Fprintf(&b, "  #%s | %s | %s | %s\n", order.ID, order.Date, order.Status, order.TotalPrice)
		if !showItems {
			continue
		}
		for _, item := range order.Items {
			fmt.Fprintf(&b, "    - %s | qty %g | %s\n", item.CommodityName, item.Count, item.Status)
		}
	}

	return b.String()
}

// === ORDERS ===

type OrdersCmd struct {
	Limit     int    `help:"Max orders to show" default:"10" short:"n"`
	WithItems bool   `help:"Show item lines under each order" name:"with-items"`
	Query     string `help:"Filter past orders by item name"`
}

func (c *OrdersCmd) Run(g *Globals) error {
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}

	query := strings.TrimSpace(c.Query)
	if query != "" {
		archiveOrders, historyTotal, err := collectArchiveOrders(cl.GetArchiveOrdersPage, archiveOrdersPageSize)
		if err != nil {
			return err
		}

		filtered := filterOrdersByQuery(archiveOrders, query)
		orders := limitOrders(filtered, c.Limit)

		if g.Format == "json" {
			outputJSON(map[string]interface{}{
				"orders":              ordersForJSON(orders, true),
				"totalCount":          len(filtered),
				"historyCount":        historyTotal,
				"query":               query,
				"searchesArchiveOnly": true,
			})
			return nil
		}

		fmt.Print(formatOrdersText(orders, len(filtered), query, true))
		return nil
	}

	orders, total, err := cl.GetOrders(c.Limit)
	if err != nil {
		return err
	}

	if g.Format == "json" {
		outputJSON(map[string]interface{}{
			"orders":     ordersForJSON(orders, c.WithItems),
			"totalCount": total,
		})
		return nil
	}

	fmt.Print(formatOrdersText(orders, total, "", c.WithItems))
	return nil
}
