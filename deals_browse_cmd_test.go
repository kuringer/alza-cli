package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kuringer/alza-cli/client"
)

var browseItems = []client.ListingItem{
	{ID: 1, Name: "Acme Žehlička 3000", Price: 50, ComparePrice: 100, ClaimedDiscount: 50, Rating: 4.5, ReviewCount: 40},
	{ID: 2, Name: "Acme Laptop 15", Price: 800, ComparePrice: 1000, ClaimedDiscount: 20, Rating: 4.8, ReviewCount: 120},
	{ID: 3, Name: "Acme Cable", Price: 5, ComparePrice: 20, ClaimedDiscount: 75, Rating: 4.9, ReviewCount: 3},
	{ID: 4, Name: "Acme Phone (kat. B)", Price: 300, ClaimedDiscount: 40, Rating: 4.0, ReviewCount: 50, Used: true},
	{ID: 5, Name: "Acme Mouse", Price: 30, ComparePrice: 35, ClaimedDiscount: 14, Rating: 4.2, ReviewCount: 500},
	{ID: 6, Name: "Acme TV 55", Price: 400, ComparePrice: 500, ClaimedDiscount: 20, Rating: 3.1, ReviewCount: 12},
}

func ids(items []client.ListingItem) []int {
	out := []int{}
	for _, it := range items {
		out = append(out, it.ID)
	}
	return out
}

func TestRankCandidates(t *testing.T) {
	cases := []struct {
		name string
		f    browseFilter
		want []int
	}{
		{"defaults rank by EUR saving", browseFilter{MinDiscount: 20, MinReviews: 10}, []int{2, 6, 1}},
		{"used included", browseFilter{MinDiscount: 20, MinReviews: 10, Used: true}, []int{2, 6, 1, 4}},
		{"rating", browseFilter{MinDiscount: 20, MinReviews: 10, MinRating: 4.5}, []int{2, 1}},
		{"price range", browseFilter{MinDiscount: 20, MinPrice: 10, MaxPrice: 500}, []int{6, 1}},
		{"query ignores case and diacritics", browseFilter{Query: "zehlicka"}, []int{1}},
		{"no filters (used still excluded)", browseFilter{}, []int{2, 6, 1, 3, 5}},
	}
	for _, tc := range cases {
		got := ids(rankCandidates(browseItems, tc.f))
		if len(got) != len(tc.want) {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
				break
			}
		}
	}
}

// countingMarket returns a fixed match and counts concurrent calls.
type countingMarket struct {
	active, peak atomic.Int32
}

func (m *countingMarket) Lookup(_ context.Context, name, _ string) (*client.HeurekaMatch, error) {
	n := m.active.Add(1)
	for {
		p := m.peak.Load()
		if n <= p || m.peak.CompareAndSwap(p, n) {
			break
		}
	}
	time.Sleep(10 * time.Millisecond)
	m.active.Add(-1)
	match := testMatch(!strings.Contains(name, "uncertain"))
	return match, nil
}

func TestCheckSubjectsKeepsOrderAndLimitsWorkers(t *testing.T) {
	subjects := []dealSubject{
		{ID: 1, Name: "cheap", Price: 100},
		{ID: 2, Name: "expensive", Price: 150},
		{ID: 3, Name: "uncertain cheap", Price: 90},
		{ID: 4, Name: "four", Price: 129},
		{ID: 5, Name: "five", Price: 140},
	}
	m := &countingMarket{}
	var progress atomic.Int32
	reports := checkSubjects(context.Background(), subjects, m, nil, time.Now(), 2, 0, func(int, string) { progress.Add(1) })
	if len(reports) != 5 || progress.Load() != 5 {
		t.Fatalf("reports=%d progress=%d", len(reports), progress.Load())
	}
	for i, r := range reports {
		if r.ProductID != subjects[i].ID {
			t.Errorf("report %d is product %d", i, r.ProductID)
		}
	}
	if p := m.peak.Load(); p > 2 {
		t.Errorf("peak concurrency %d > 2 workers", p)
	}

	rows := make([]browseRow, len(reports))
	for i, r := range reports {
		rows[i] = browseRow{dealReport: r, RealDiscount: realDiscount(r)}
	}
	sortRowsBySaving(rows)
	if rows[0].ProductID != 3 || rows[len(rows)-1].ProductID != 2 {
		t.Errorf("order = %v", []int{rows[0].ProductID, rows[1].ProductID, rows[2].ProductID, rows[3].ProductID, rows[4].ProductID})
	}
	// Market reference 129.9 (cheapest with shipping): 100 is 23 % below it.
	var cheap browseRow
	for _, r := range rows {
		if r.ProductID == 1 {
			cheap = r
		}
	}
	if cheap.RealDiscount == nil || *cheap.RealDiscount < 23 || *cheap.RealDiscount > 23.1 {
		t.Errorf("real discount = %v", cheap.RealDiscount)
	}

	var buf bytes.Buffer
	printBrowseTable(&buf, rows)
	out := buf.String()
	if !strings.Contains(out, "129.90?") || !strings.Contains(out, client.VerdictLabels[client.VerdictUncertain]) {
		t.Errorf("table misses uncertain mark:\n%s", out)
	}
	if !strings.Contains(out, "-15 %") {
		t.Errorf("table misses negative real discount for the expensive item:\n%s", out)
	}
}

func TestBrowseSubjectPrefersDetail(t *testing.T) {
	it := client.ListingItem{ID: 7, Name: "Acme Kettle", Price: 20, ComparePrice: 40, Label: "BLACK FRIDAY -50 %"}
	if s := browseSubject(it, nil); s.Price != 20 || s.CatalogNumber != "" || s.PriceHeader != it.Label {
		t.Errorf("listing subject = %+v", s)
	}
	d := fakeDetails{p: &client.ProductDetail{ID: 7, Name: "Acme Kettle 1.7 l", CatalogNumber: "AK17", PriceNoCurrency: 19.5, ComparePriceNoCurrency: 40}}
	if s := browseSubject(it, d); s.Price != 19.5 || s.CatalogNumber != "AK17" || s.PriceHeader != it.Label {
		t.Errorf("detail subject = %+v", s)
	}
	if s := browseSubject(it, fakeDetails{}); s.Price != 20 {
		t.Errorf("fallback subject = %+v", s)
	}
}

type fakeDetails struct{ p *client.ProductDetail }

func (f fakeDetails) GetProductSummary(int) (*client.ProductDetail, error) {
	if f.p == nil {
		return nil, context.DeadlineExceeded
	}
	return f.p, nil
}

type failingMarket struct{}

func (failingMarket) Lookup(context.Context, string, string) (*client.HeurekaMatch, error) {
	return nil, errors.New("Heureka product page: no __NEXT_DATA__ on page")
}

func TestBrowseRowsKeepListingLabelAndUsed(t *testing.T) {
	// Listing label says 31 %, prices give 32 % (1 - 18.99/27.90).
	top := []client.ListingItem{
		{ID: 1, Name: "Acme Toaster", URL: "https://www.alza.sk/x-d1.htm", Price: 18.99, ComparePrice: 27.9, ClaimedDiscount: 31, Label: "BLACK FRIDAY -31 %"},
		{ID: 2, Name: "Acme Phone", Price: 300, ComparePrice: 500, ClaimedDiscount: 40, Used: true},
	}
	reports := make([]dealReport, len(top))
	for i, it := range top {
		reports[i] = buildDealReport(context.Background(), browseSubject(it, nil), nil, nil, time.Now())
	}
	if reports[0].ClaimedDiscount != 32 {
		t.Fatalf("precondition: computed %% = %d, want 32", reports[0].ClaimedDiscount)
	}
	rows := buildBrowseRows(top, reports)
	var buf bytes.Buffer
	printBrowseTable(&buf, rows)
	out := buf.String()
	toaster := ""
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "Acme Toaster") {
			toaster = line
		}
	}
	if !strings.Contains(toaster, "31 %") || strings.Contains(toaster, "32 %") {
		t.Errorf("UVÁDZA must show the label 31 %%:\n%s", out)
	}
	if !strings.Contains(out, "[bazár] Acme Phone") || strings.Contains(out, "[bazár] Acme Toaster") {
		t.Errorf("used row not marked:\n%s", out)
	}
	for _, r := range rows {
		if r.ProductID == 2 && !r.Used {
			t.Error("used flag missing on row 2")
		}
		if r.ProductID == 1 && (r.ListingClaimedDiscount != 31 || r.ClaimedDiscount != 32) {
			t.Errorf("row 1: listing %d, computed %d", r.ListingClaimedDiscount, r.ClaimedDiscount)
		}
	}
	var cands bytes.Buffer
	printCandidates(&cands, top, 0)
	if !strings.Contains(cands.String(), "[bazár] Acme Phone") {
		t.Errorf("candidates table: used item not marked:\n%s", cands.String())
	}
}

func TestAllLookupsFailed(t *testing.T) {
	top := []client.ListingItem{{ID: 1, Name: "A", Price: 10}, {ID: 2, Name: "B", Price: 20}}
	subjects := []dealSubject{browseSubject(top[0], nil), browseSubject(top[1], nil)}
	failed := buildBrowseRows(top, checkSubjects(context.Background(), subjects, failingMarket{}, nil, time.Now(), 2, 0, func(int, string) {}))
	err := allLookupsFailed(failed)
	if err == nil || !strings.Contains(err.Error(), "všetkých 2") || !strings.Contains(err.Error(), "__NEXT_DATA__") {
		t.Errorf("all failed: err = %v", err)
	}
	// One lookup that worked (or found no other shop) is enough: no error.
	mixed := append([]browseRow{}, failed...)
	mixed[1].lookupFailed = false
	if err := allLookupsFailed(mixed); err != nil {
		t.Errorf("mixed: err = %v", err)
	}
	if err := allLookupsFailed(nil); err != nil {
		t.Errorf("no rows: err = %v", err)
	}
}
