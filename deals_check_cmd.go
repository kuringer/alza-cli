package main

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/kuringer/alza-cli/client"
)

// dealsScanDelay is the pause between Heureka lookups in scan (polite request rate).
const dealsScanDelay = 1500 * time.Millisecond

// dealsCheapestShown is how many cheapest market offers the report keeps.
const dealsCheapestShown = 3

type DealsCheckCmd struct {
	ID        int  `arg:"" help:"Product ID"`
	NoHeureka bool `help:"Skip the Heureka market check (no browser needed)" name:"no-heureka"`
	NoRecord  bool `help:"Do not store today's price in the price history" name:"no-record"`
}

type DealsScanCmd struct {
	Favorites bool `help:"Scan the favorites list (default and only source for now)" default:"true"`
	NoHeureka bool `help:"Skip the Heureka market check" name:"no-heureka"`
	NoRecord  bool `help:"Do not store today's prices in the price history" name:"no-record"`
}

// marketLooker maps an Alza product to market offers (client.Heureka in production).
type marketLooker interface {
	Lookup(ctx context.Context, name, partNumber string) (*client.HeurekaMatch, error)
}

// dealSubject is the Alza side of a deal check.
type dealSubject struct {
	ID            int
	Name          string
	CatalogNumber string
	Price         float64
	ComparePrice  float64
	PriceHeader   string
}

type dealMarket struct {
	Query         string                `json:"query"`
	Name          string                `json:"name"`
	URL           string                `json:"url"`
	Certain       bool                  `json:"certain"`
	MatchScore    float64               `json:"matchScore"`
	MatchReason   string                `json:"matchReason,omitempty"`
	HeurekaMin    float64               `json:"heurekaMinPrice"`
	MarketMin     float64               `json:"marketMin"`
	MinUnverified bool                  `json:"marketMinUnverified,omitempty"`
	CheapestTotal float64               `json:"cheapestTotalApprox"`
	OfferCount    int                   `json:"offerCount"`
	Cheapest      []client.HeurekaOffer `json:"cheapest"`
}

type dealReport struct {
	ProductID       int                `json:"productId"`
	Name            string             `json:"name"`
	CatalogNumber   string             `json:"catalogNumber,omitempty"`
	AlzaPrice       float64            `json:"alzaPrice"`
	ComparePrice    float64            `json:"comparePrice,omitempty"`
	ClaimedDiscount int                `json:"claimedDiscountPercent,omitempty"`
	PriceHeader     string             `json:"priceHeader,omitempty"`
	Market          *dealMarket        `json:"market,omitempty"`
	MarketError     string             `json:"marketError,omitempty"`
	History30       *client.PriceStats `json:"history30,omitempty"`
	History90       *client.PriceStats `json:"history90,omitempty"`
	SavingVsMarket  *float64           `json:"savingVsMarket,omitempty"`
	Verdict         client.DealVerdict `json:"verdict"`
	// lookupFailed: the Heureka lookup itself returned an error (not "no other shop").
	lookupFailed bool
}

func subjectFromProduct(p *client.ProductDetail) dealSubject {
	return dealSubject{ID: p.ID, Name: p.Name, CatalogNumber: p.CatalogNumber, Price: p.PriceNoCurrency, ComparePrice: p.ComparePriceNoCurrency, PriceHeader: p.PriceHeader}
}

func (s dealSubject) record(date string) client.PriceRecord {
	return client.PriceRecord{Date: date, ProductID: s.ID, Name: s.Name, Price: s.Price, ComparePrice: s.ComparePrice}
}

// buildDealReport combines Alza price, market (market may be nil) and own history into a verdict.
func buildDealReport(ctx context.Context, s dealSubject, market marketLooker, history []client.PriceRecord, now time.Time) dealReport {
	r := dealReport{
		ProductID:       s.ID,
		Name:            s.Name,
		CatalogNumber:   s.CatalogNumber,
		AlzaPrice:       s.Price,
		ComparePrice:    s.ComparePrice,
		ClaimedDiscount: client.ClaimedDiscount(s.Price, s.ComparePrice),
		PriceHeader:     s.PriceHeader,
	}
	in := client.DealInput{AlzaPrice: s.Price}

	if market != nil {
		m, err := market.Lookup(ctx, s.Name, s.CatalogNumber)
		if err != nil {
			r.MarketError = err.Error()
			r.lookupFailed = true
		}
		var marketMin float64
		var unverified bool
		if m != nil && m.Product != nil {
			marketMin, unverified = m.Product.MarketPrice(s.Price)
		}
		if m != nil && m.Product != nil && marketMin <= 0 {
			r.MarketError = fmt.Sprintf("na Heureke nie je iná ponuka ako Alza (%s)", m.Product.URL)
		} else if m != nil && m.Product != nil {
			p := m.Product
			r.Market = &dealMarket{
				Query: m.Query, Name: p.Name, URL: p.URL, Certain: m.Certain,
				MatchScore: m.Candidate.Score, MatchReason: m.Candidate.Reason,
				HeurekaMin: p.MinPrice, MarketMin: marketMin, MinUnverified: unverified,
				CheapestTotal: p.CheapestTotal(), OfferCount: p.OfferCount,
			}
			if len(p.Offers) > dealsCheapestShown {
				r.Market.Cheapest = p.Offers[:dealsCheapestShown]
			} else {
				r.Market.Cheapest = p.Offers
			}
			in.MarketMin = r.Market.MarketMin
			in.CheapestTotal = r.Market.CheapestTotal
			in.MarketUncertain = !m.Certain
			in.MarketMinUnverified = unverified
			ref := in.CheapestTotal
			if ref <= 0 {
				ref = in.MarketMin
			}
			if ref > 0 && s.Price > 0 {
				saving := math.Round((ref-s.Price)*100) / 100
				r.SavingVsMarket = &saving
			}
		}
	} else {
		r.MarketError = "vynechané (--no-heureka)"
	}

	if st, ok := client.HistoryStatsFor(history, s.ID, 30, now); ok {
		r.History30 = &st
	}
	if st, ok := client.HistoryStatsFor(history, s.ID, client.DealHistoryDays, now); ok {
		r.History90 = &st
		in.HistoryMedian = st.Median
		in.HistoryRecords = st.Count
	}
	r.Verdict = client.DecideDeal(in)
	return r
}

// recordAndReadHistory stores today's prices (unless skip) and returns the full history.
func recordAndReadHistory(subjects []dealSubject, now time.Time, skip bool) ([]client.PriceRecord, error) {
	path, err := client.PriceHistoryPath()
	if err != nil {
		return nil, err
	}
	if !skip {
		recs := make([]client.PriceRecord, 0, len(subjects))
		for _, s := range subjects {
			recs = append(recs, s.record(client.HistoryDate(now)))
		}
		if _, err := client.AppendPriceRecords(path, recs); err != nil {
			return nil, err
		}
	}
	return client.ReadPriceHistory(path)
}

// newMarket returns the Heureka looker (nil when skip) and its close func.
// All lookups share one headless browser.
func newMarket(skip bool) (marketLooker, func(), error) {
	if skip {
		return nil, func() {}, nil
	}
	h, closeFn, err := client.NewHeurekaShared()
	if err != nil {
		return nil, nil, err
	}
	return h, closeFn, nil
}

func (c *DealsCheckCmd) Run(g *Globals) error {
	market, closeMarket, err := newMarket(c.NoHeureka)
	if err != nil {
		return err
	}
	defer closeMarket()
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}
	p, err := cl.GetProductSummary(c.ID)
	if err != nil {
		return err
	}
	subject := subjectFromProduct(p)
	now := time.Now()
	history, err := recordAndReadHistory([]dealSubject{subject}, now, c.NoRecord)
	if err != nil {
		return err
	}

	r := buildDealReport(context.Background(), subject, market, history, now)
	if g.Format == "json" {
		outputJSON(r)
		return nil
	}
	printDealReport(os.Stdout, r)
	return nil
}

func (c *DealsScanCmd) Run(g *Globals) error {
	market, closeMarket, err := newMarket(c.NoHeureka)
	if err != nil {
		return err
	}
	defer closeMarket()
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}
	subjects, warnings := loadDealSubjects(cl, nil, true)
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", w)
	}
	if len(subjects) == 0 {
		return fmt.Errorf("no favorites to scan")
	}
	now := time.Now()
	history, err := recordAndReadHistory(subjects, now, c.NoRecord)
	if err != nil {
		return err
	}

	reports := make([]dealReport, 0, len(subjects))
	for i, s := range subjects {
		if i > 0 && market != nil {
			time.Sleep(dealsScanDelay)
		}
		if g.Format != "json" {
			fmt.Fprintf(os.Stderr, "[%d/%d] %s\n", i+1, len(subjects), s.Name)
		}
		reports = append(reports, buildDealReport(context.Background(), s, market, history, now))
	}
	sortBySaving(reports)

	if g.Format == "json" {
		outputJSON(reports)
		return nil
	}
	printScanTable(os.Stdout, reports)
	return nil
}

// sortBySaving puts the biggest saving vs market first; unknown saving last.
func sortBySaving(reports []dealReport) {
	sort.SliceStable(reports, func(i, j int) bool {
		a, b := reports[i].SavingVsMarket, reports[j].SavingVsMarket
		switch {
		case a == nil:
			return false
		case b == nil:
			return true
		default:
			return *a > *b
		}
	})
}

func printDealReport(w io.Writer, r dealReport) {
	fmt.Fprintf(w, "%s  (id %d", r.Name, r.ProductID)
	if r.CatalogNumber != "" {
		fmt.Fprintf(w, ", kód %s", r.CatalogNumber)
	}
	fmt.Fprintln(w, ")")

	fmt.Fprintf(w, "Alza:      %s", formatEUR(r.AlzaPrice))
	if r.ComparePrice > 0 {
		fmt.Fprintf(w, "  (pôvodne %s, Alza uvádza -%d %%)", formatEUR(r.ComparePrice), r.ClaimedDiscount)
	}
	if r.PriceHeader != "" {
		fmt.Fprintf(w, "  [%s]", r.PriceHeader)
	}
	fmt.Fprintln(w)

	if m := r.Market; m != nil {
		fmt.Fprintf(w, "Heureka:   min %s", formatEUR(m.MarketMin))
		if m.MinUnverified {
			fmt.Fprint(w, " (NEOVERENÉ: súhrnná cena Heureky bez obchodu, môže to byť Alza)")
		} else if m.HeurekaMin > 0 && m.HeurekaMin != m.MarketMin {
			fmt.Fprintf(w, " (Heureka uvádza od %s)", formatEUR(m.HeurekaMin))
		}
		fmt.Fprintf(w, ", %d ponúk\n", m.OfferCount)
		match := "zhoda"
		if !m.Certain {
			match = "NEISTÁ ZHODA (uncertain match)"
		}
		fmt.Fprintf(w, "           %s: %s (skóre %.2f, %s)\n           %s\n", match, m.Name, m.MatchScore, matchReasonLabels[m.MatchReason], m.URL)
		if len(m.Cheapest) > 0 {
			fmt.Fprintln(w, "           najlacnejšie (doprava približne):")
			for _, o := range m.Cheapest {
				ship := fmt.Sprintf("%.2f", o.Shipping)
				if !o.ShippingKnown {
					ship = "?"
				}
				fmt.Fprintf(w, "             %-24s %8.2f + %-5s = %8.2f EUR\n", truncate(o.Shop, 24), o.Price, ship, o.Total)
			}
		}
	} else if r.MarketError != "" {
		fmt.Fprintf(w, "Heureka:   - (%s)\n", r.MarketError)
	}

	fmt.Fprintf(w, "História:  %s\n", formatHistory(r))

	fmt.Fprintf(w, "Verdikt:   %s - %s\n", strings.ToUpper(r.Verdict.Label), r.Verdict.Reason)
	if r.SavingVsMarket != nil && r.Market != nil {
		ref := r.Market.CheapestTotal
		if ref <= 0 {
			ref = r.Market.MarketMin
		}
		if *r.SavingVsMarket >= 0 {
			fmt.Fprintf(w, "           oproti trhu ušetríš %.2f EUR (%.0f %%)\n", *r.SavingVsMarket, *r.SavingVsMarket/ref*100)
		} else {
			fmt.Fprintf(w, "           na trhu o %.2f EUR lacnejšie\n", -*r.SavingVsMarket)
		}
	}
	for _, m := range r.Verdict.Missing {
		fmt.Fprintf(w, "Chýba:     %s\n", m)
	}
}

var matchReasonLabels = map[string]string{
	"same name":   "rovnaký názov",
	"part number": "produktové číslo",
	"name tokens": "podobný názov",
}

func formatHistory(r dealReport) string {
	if r.History90 == nil {
		return "žiadne záznamy (alza deals snapshot)"
	}
	parts := []string{}
	if h := r.History30; h != nil {
		parts = append(parts, fmt.Sprintf("30 dní min %s (%d zázn.)", formatEUR(h.Min), h.Count))
	}
	h := r.History90
	parts = append(parts, fmt.Sprintf("90 dní min %s, medián %s (%d zázn.)", formatEUR(h.Min), formatEUR(h.Median), h.Count))
	return strings.Join(parts, " | ")
}

func printScanTable(w io.Writer, reports []dealReport) {
	fmt.Fprintf(w, "%-18s %10s %10s %10s  %s\n", "VERDIKT", "ALZA", "TRH*", "ÚSPORA", "PRODUKT")
	for _, r := range reports {
		market, saving := "-", "-"
		if r.Market != nil {
			ref := r.Market.CheapestTotal
			if ref <= 0 {
				ref = r.Market.MarketMin
			}
			market = fmt.Sprintf("%.2f", ref)
			if !r.Market.Certain || r.Market.MinUnverified {
				market += "?"
			}
		}
		if r.SavingVsMarket != nil {
			saving = fmt.Sprintf("%+.2f", *r.SavingVsMarket)
		}
		fmt.Fprintf(w, "%-18s %10.2f %10s %10s  %s (%d)\n", r.Verdict.Label, r.AlzaPrice, market, saving, truncate(r.Name, 50), r.ProductID)
	}
	fmt.Fprintln(w, "* najlacnejšia ponuka na Heureke s približnou dopravou; ? = neistá zhoda produktu alebo neoverená cena")
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
