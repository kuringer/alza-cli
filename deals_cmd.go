package main

import (
	"fmt"
	"os"
	"time"

	"github.com/kuringer/alza-cli/client"
)

// === DEALS ===

type DealsCmd struct {
	Snapshot DealsSnapshotCmd `cmd:"" help:"Store today's Alza price of favorites (+ --id) in the local price history"`
	Check    DealsCheckCmd    `cmd:"" help:"Is the Alza discount real? Compare with Heureka and own price history"`
	Scan     DealsScanCmd     `cmd:"" help:"Run check for every favorite, sorted by real saving vs market"`
}

type DealsSnapshotCmd struct {
	ID          []int `help:"Extra product ID (repeatable)" name:"id"`
	NoFavorites bool  `help:"Skip the favorites list" name:"no-favorites"`
}

type snapshotResult struct {
	Path     string               `json:"path"`
	Written  int                  `json:"written"`
	Records  []client.PriceRecord `json:"records"`
	Warnings []string             `json:"warnings,omitempty"`
}

func (c *DealsSnapshotCmd) Run(g *Globals) error {
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}
	path, err := client.PriceHistoryPath()
	if err != nil {
		return err
	}

	today := client.HistoryDate(time.Now())
	res := snapshotResult{Path: path}
	seen := map[int]bool{}

	if !c.NoFavorites {
		items, err := favoriteItems(cl)
		if err != nil {
			res.Warnings = append(res.Warnings, fmt.Sprintf("favorites: %v", err))
		}
		for _, it := range items {
			if it.ID <= 0 || seen[it.ID] {
				continue
			}
			seen[it.ID] = true
			res.Records = append(res.Records, client.PriceRecord{
				Date:         today,
				ProductID:    it.ID,
				Name:         it.Name,
				Price:        it.PriceNoCurrency,
				ComparePrice: client.ParsePrice(it.ComparePrice),
			})
		}
	}

	for _, id := range c.ID {
		if seen[id] {
			continue
		}
		seen[id] = true
		p, err := cl.GetProductSummary(id)
		if err != nil {
			res.Warnings = append(res.Warnings, fmt.Sprintf("product %d: %v", id, err))
			continue
		}
		res.Records = append(res.Records, recordFromProduct(today, p))
	}

	if len(res.Records) == 0 {
		return fmt.Errorf("no products to snapshot (favorites empty and no --id)")
	}

	res.Written, err = client.AppendPriceRecords(path, res.Records)
	if err != nil {
		return err
	}

	if g.Format == "json" {
		outputJSON(res)
		return nil
	}
	for _, w := range res.Warnings {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", w)
	}
	fmt.Printf("Snapshot %s: %d products, %d new records -> %s\n", today, len(res.Records), res.Written, path)
	for _, r := range res.Records {
		fmt.Printf("  %d  %s  %s\n", r.ProductID, formatEUR(r.Price), r.Name)
	}
	return nil
}

func favoriteItems(cl *client.TLSClient) ([]client.ListItem, error) {
	list, err := resolveFavoritesList(cl)
	if err != nil {
		return nil, err
	}
	return cl.GetListItems(list.ID)
}

func recordFromProduct(date string, p *client.ProductDetail) client.PriceRecord {
	return client.PriceRecord{
		Date:         date,
		ProductID:    p.ID,
		Name:         p.Name,
		Price:        p.PriceNoCurrency,
		ComparePrice: p.ComparePriceNoCurrency,
	}
}

func formatEUR(v float64) string {
	if v <= 0 {
		return "-"
	}
	return fmt.Sprintf("%.2f EUR", v)
}
