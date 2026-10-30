package main

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/kuringer/alza-cli/client"
)

// browseWorkers is how many products the Heureka check runs at once (pages in one browser).
const browseWorkers = 3

// browseLookupDelay is each worker's pause between two products (polite request rate).
const browseLookupDelay = time.Second

// browsePageDelay is the pause between two listing pages.
const browsePageDelay = 300 * time.Millisecond

type DealsBrowseCmd struct {
	URL         string  `help:"Alza listing URL (Black Friday, category, sale page)" default:"https://www.alza.sk/black-friday" name:"url"`
	Sort        string  `help:"Listing order on Alza" enum:"discount,recommended,sales,rating,cheapest,expensive,newest" default:"discount"`
	Pages       int     `help:"Listing pages to read (24 products each)" default:"10"`
	MinDiscount int     `help:"Minimum discount Alza claims, in %" default:"20" name:"min-discount"`
	MinRating   float64 `help:"Minimum rating (0-5)" name:"min-rating"`
	MinReviews  int     `help:"Minimum review count" default:"10" name:"min-reviews"`
	MinPrice    float64 `help:"Minimum Alza price in EUR" name:"min-price"`
	MaxPrice    float64 `help:"Maximum Alza price in EUR (0 = no limit)" name:"max-price"`
	Query       string  `help:"Name contains this text (case and diacritics ignored)"`
	Used        bool    `help:"Include used items (Alza bazar, % against the new price)"`
	Check       int     `help:"Check the top N candidates on Heureka (0 = no browser)" default:"20"`
	Limit       int     `help:"Max candidate rows in text output" default:"30"`
}

// browseFilter is the cheap pre-filter (no Heureka).
type browseFilter struct {
	MinDiscount int
	MinRating   float64
	MinReviews  int
	MinPrice    float64
	MaxPrice    float64
	Query       string
	Used        bool
}

func (c *DealsBrowseCmd) filter() browseFilter {
	return browseFilter{MinDiscount: c.MinDiscount, MinRating: c.MinRating, MinReviews: c.MinReviews,
		MinPrice: c.MinPrice, MaxPrice: c.MaxPrice, Query: c.Query, Used: c.Used}
}

func (f browseFilter) keep(it client.ListingItem) bool {
	switch {
	case it.Used && !f.Used:
		return false
	case it.ClaimedDiscount < f.MinDiscount:
		return false
	case it.Rating < f.MinRating:
		return false
	case it.ReviewCount < f.MinReviews:
		return false
	case it.Price < f.MinPrice:
		return false
	case f.MaxPrice > 0 && it.Price > f.MaxPrice:
		return false
	case f.Query != "" && !client.NameContains(it.Name, f.Query):
		return false
	}
	return true
}

// rankCandidates filters items and sorts them by the saving Alza claims, in EUR.
func rankCandidates(items []client.ListingItem, f browseFilter) []client.ListingItem {
	out := []client.ListingItem{}
	for _, it := range items {
		if f.keep(it) {
			out = append(out, it)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if sa, sb := a.ClaimedSaving(), b.ClaimedSaving(); sa != sb {
			return sa > sb
		}
		return a.ClaimedDiscount > b.ClaimedDiscount
	})
	return out
}

// browseRow is a checked candidate: deal report + listing facts.
type browseRow struct {
	dealReport
	URL string `json:"url,omitempty"`
	// ListingClaimedDiscount is the % from the listing box (Alza label), kept
	// apart from claimedDiscountPercent, which is computed from the detail prices.
	ListingClaimedDiscount int `json:"listingClaimedDiscountPercent,omitempty"`
	// Used marks Alza bazar items (% against the new price).
	Used bool `json:"used,omitempty"`
	// RealDiscount: how much cheaper Alza is than the market reference, in % (negative = more expensive).
	RealDiscount *float64 `json:"realDiscountPercent,omitempty"`
}

type browseResult struct {
	Listing      *client.Listing      `json:"listing"`
	Sort         string               `json:"sort"`
	ListingTotal int                  `json:"listingTotal"`
	PagesRead    int                  `json:"pagesRead"`
	Scanned      int                  `json:"scanned"`
	Candidates   []client.ListingItem `json:"candidates"`
	Checked      []browseRow          `json:"checked,omitempty"`
	Warnings     []string             `json:"warnings,omitempty"`
	Error        string               `json:"error,omitempty"`
	Seconds      float64              `json:"seconds"`
}

// productSummarizer gives the Alza detail (part number for Heureka, current price).
type productSummarizer interface {
	GetProductSummary(id int) (*client.ProductDetail, error)
}

func (c *DealsBrowseCmd) Run(g *Globals) error {
	start := time.Now()
	logf := func(format string, a ...any) {
		if g.Format != "json" {
			fmt.Fprintf(os.Stderr, format+"\n", a...)
		}
	}
	if c.Pages < 1 {
		return fmt.Errorf("--pages must be at least 1")
	}

	lc, err := client.NewListingClient(g.Debug)
	if err != nil {
		return err
	}
	listing, err := lc.OpenListing(c.URL)
	if err != nil {
		return err
	}
	res := browseResult{Listing: listing, Sort: c.Sort}

	items, total, pages, err := fetchListing(lc, listing, client.ListingSorts[c.Sort], c.Pages, func(page, total int) {
		logf("%s: strana %d/%d (%d produktov v zozname)", listing.Heading, page, c.Pages, total)
	})
	res.ListingTotal, res.PagesRead, res.Scanned = total, pages, len(items)
	if err != nil {
		if len(items) == 0 {
			return err
		}
		res.Warnings = append(res.Warnings, err.Error())
	}
	res.Candidates = rankCandidates(items, c.filter())
	logf("%d produktov, %d kandidátov po filtri", len(items), len(res.Candidates))

	if c.Check > 0 && len(res.Candidates) > 0 {
		top := res.Candidates
		if len(top) > c.Check {
			top = top[:c.Check]
		}
		rows, warnings, err := c.checkTop(g, top, logf)
		if err != nil {
			return err
		}
		res.Checked = rows
		res.Warnings = append(res.Warnings, warnings...)
	}
	res.Seconds = time.Since(start).Round(100 * time.Millisecond).Seconds()
	failErr := allLookupsFailed(res.Checked)
	if failErr != nil {
		res.Error = failErr.Error()
	}

	if g.Format == "json" {
		outputJSON(res)
		return failErr
	}
	for _, w := range res.Warnings {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", w)
	}
	if res.Checked != nil {
		printBrowseTable(os.Stdout, res.Checked)
		if rest := len(res.Candidates) - len(res.Checked); rest > 0 {
			fmt.Printf("\nĎalších %d kandidátov bez kontroly (--check N alebo --format json).\n", rest)
		}
	} else {
		printCandidates(os.Stdout, res.Candidates, c.Limit)
	}
	fmt.Fprintf(os.Stderr, "hotovo za %s\n", time.Since(start).Round(time.Second))
	return failErr
}

// allLookupsFailed: an error when every Heureka lookup failed (blocked or
// broken browser), so the table has no market data at all.
func allLookupsFailed(rows []browseRow) error {
	if len(rows) == 0 {
		return nil
	}
	for _, r := range rows {
		if !r.lookupFailed {
			return nil
		}
	}
	return fmt.Errorf("Heureka: všetkých %d vyhľadaní zlyhalo, porovnanie s trhom chýba (prvá chyba: %s)", len(rows), rows[0].MarketError)
}

// fetchListing reads listing pages 1..pages, deduplicated by product id. It
// stops early on an empty page or at the end of the listing.
func fetchListing(lc *client.ListingClient, l *client.Listing, sortID, pages int, progress func(page, total int)) ([]client.ListingItem, int, int, error) {
	seen := map[int]bool{}
	var out []client.ListingItem
	total, read := 0, 0
	for p := 1; p <= pages; p++ {
		if p > 1 {
			time.Sleep(browsePageDelay)
		}
		items, t, err := lc.ListingPage(l, sortID, p)
		if err != nil {
			return out, total, read, err
		}
		read, total = p, t
		progress(p, t)
		for _, it := range items {
			if !seen[it.ID] {
				seen[it.ID] = true
				out = append(out, it)
			}
		}
		if len(items) == 0 || p*client.ListingPageSize >= t {
			break
		}
	}
	return out, total, read, nil
}

func (c *DealsBrowseCmd) checkTop(g *Globals, top []client.ListingItem, logf func(string, ...any)) ([]browseRow, []string, error) {
	var warnings []string
	market, closeMarket, err := newMarket(false)
	if err != nil {
		return nil, nil, err
	}
	defer closeMarket()

	// Part numbers make Heureka matches certain; they need the (logged-in) product API.
	var details productSummarizer
	if cl, err := client.NewTLSClient(g.Debug); err == nil {
		details = cl
	} else {
		warnings = append(warnings, "bez Alza tokenu: Heureka hľadá len podľa názvu (alza token refresh)")
	}

	var history []client.PriceRecord
	if path, err := client.PriceHistoryPath(); err == nil {
		if history, err = client.ReadPriceHistory(path); err != nil {
			warnings = append(warnings, fmt.Sprintf("price history: %v", err))
		}
	}

	subjects := make([]dealSubject, len(top))
	for i, it := range top {
		subjects[i] = browseSubject(it, details)
	}
	logf("Heureka: kontrolujem %d produktov (%d naraz)", len(top), browseWorkers)
	reports := checkSubjects(context.Background(), subjects, market, history, time.Now(), browseWorkers, browseLookupDelay, func(done int, name string) {
		logf("[%d/%d] %s", done, len(subjects), name)
	})

	return buildBrowseRows(top, reports), warnings, nil
}

// buildBrowseRows joins listing items and their reports (same order), sorted by saving vs market.
func buildBrowseRows(top []client.ListingItem, reports []dealReport) []browseRow {
	rows := make([]browseRow, len(reports))
	for i, r := range reports {
		rows[i] = browseRow{dealReport: r, URL: top[i].URL, ListingClaimedDiscount: top[i].ClaimedDiscount,
			Used: top[i].Used, RealDiscount: realDiscount(r)}
	}
	sortRowsBySaving(rows)
	return rows
}

// usedPrefix marks Alza bazar items in text tables.
func usedPrefix(used bool) string {
	if used {
		return "[bazár] "
	}
	return ""
}

// browseSubject uses the product detail when available (part number, current
// price), else the listing box.
func browseSubject(it client.ListingItem, details productSummarizer) dealSubject {
	s := dealSubject{ID: it.ID, Name: it.Name, Price: it.Price, ComparePrice: it.ComparePrice, PriceHeader: it.Label}
	if details == nil {
		return s
	}
	p, err := details.GetProductSummary(it.ID)
	if err != nil || p == nil || p.PriceNoCurrency <= 0 {
		return s
	}
	d := subjectFromProduct(p)
	if d.Name == "" {
		d.Name = s.Name
	}
	if d.PriceHeader == "" {
		d.PriceHeader = s.PriceHeader
	}
	return d
}

// checkSubjects runs buildDealReport with a few workers; results keep input order.
func checkSubjects(ctx context.Context, subjects []dealSubject, market marketLooker, history []client.PriceRecord, now time.Time, workers int, delay time.Duration, progress func(done int, name string)) []dealReport {
	out := make([]dealReport, len(subjects))
	jobs := make(chan int)
	var mu sync.Mutex
	done := 0
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			first := true
			for i := range jobs {
				if !first {
					time.Sleep(delay)
				}
				first = false
				out[i] = buildDealReport(ctx, subjects[i], market, history, now)
				mu.Lock()
				done++
				progress(done, subjects[i].Name)
				mu.Unlock()
			}
		}()
	}
	for i := range subjects {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	return out
}

// marketRef is the market price a saving is measured against (cheapest offer with shipping, else min price).
func marketRef(r dealReport) float64 {
	if r.Market == nil {
		return 0
	}
	if r.Market.CheapestTotal > 0 {
		return r.Market.CheapestTotal
	}
	return r.Market.MarketMin
}

func realDiscount(r dealReport) *float64 {
	ref := marketRef(r)
	if r.SavingVsMarket == nil || ref <= 0 {
		return nil
	}
	v := math.Round(*r.SavingVsMarket/ref*1000) / 10
	return &v
}

// sortRowsBySaving: biggest saving vs market first, unknown last.
func sortRowsBySaving(rows []browseRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i].SavingVsMarket, rows[j].SavingVsMarket
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

func printBrowseTable(w io.Writer, rows []browseRow) {
	fmt.Fprintf(w, "%-18s %9s %10s %9s %7s %7s  %s\n", "VERDIKT", "ALZA", "TRH*", "ÚSPORA", "UVÁDZA", "REÁLNE", "PRODUKT")
	for _, r := range rows {
		market, saving, real := "-", "-", "-"
		ref := marketRef(r.dealReport)
		if ref > 0 {
			market = fmt.Sprintf("%.2f", ref)
			if !r.Market.Certain || r.Market.MinUnverified {
				market += "?"
			}
		}
		if r.SavingVsMarket != nil {
			saving = fmt.Sprintf("%+.2f", *r.SavingVsMarket)
			if ref > 0 {
				// From the raw ratio: the 1-decimal JSON value would round twice.
				real = fmt.Sprintf("%.0f %%", *r.SavingVsMarket/ref*100)
			}
		}
		// The Alza label from the listing wins over the % computed from prices.
		claimed := "-"
		if r.ListingClaimedDiscount > 0 {
			claimed = fmt.Sprintf("%d %%", r.ListingClaimedDiscount)
		} else if r.ClaimedDiscount > 0 {
			claimed = fmt.Sprintf("%d %%", r.ClaimedDiscount)
		}
		fmt.Fprintf(w, "%-18s %9.2f %10s %9s %7s %7s  %s%s (%d)\n", r.Verdict.Label, r.AlzaPrice, market, saving, claimed, real, usedPrefix(r.Used), truncate(r.Name, 50), r.ProductID)
	}
	fmt.Fprintln(w, "* TRH = najlacnejšia ponuka na Heureke s približnou dopravou; ? = neistá zhoda produktu alebo neoverená cena")
	fmt.Fprintln(w, "  UVÁDZA = zľava podľa Alzy; REÁLNE = o koľko je Alza lacnejšia ako trh (záporné = drahšia)")
}

func printCandidates(w io.Writer, items []client.ListingItem, limit int) {
	fmt.Fprintf(w, "%9s %6s %9s %9s %5s %7s  %s\n", "ÚSPORA*", "ZĽAVA", "ALZA", "PÔVODNE", "HOD.", "RECENZ.", "PRODUKT")
	for i, it := range items {
		if limit > 0 && i >= limit {
			fmt.Fprintf(w, "... a ďalších %d (--limit, --format json)\n", len(items)-limit)
			break
		}
		cmp := "-"
		if it.ComparePrice > 0 {
			cmp = fmt.Sprintf("%.2f", it.ComparePrice)
		}
		fmt.Fprintf(w, "%9.2f %5d%% %9.2f %9s %5.1f %7d  %s%s (%d)\n", it.ClaimedSaving(), it.ClaimedDiscount, it.Price, cmp, it.Rating, it.ReviewCount, usedPrefix(it.Used), truncate(it.Name, 50), it.ID)
	}
	fmt.Fprintln(w, "* úspora a zľava podľa Alzy (preškrtnutá cena), bez porovnania s trhom")
}
