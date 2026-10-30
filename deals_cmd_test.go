package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kuringer/alza-cli/client"
)

type fakeMarket struct {
	match *client.HeurekaMatch
	err   error
}

func (f fakeMarket) Lookup(context.Context, string, string) (*client.HeurekaMatch, error) {
	return f.match, f.err
}

func testMatch(certain bool) *client.HeurekaMatch {
	return &client.HeurekaMatch{
		Query:     "LP2000DXX",
		Candidate: client.HeurekaCandidate{Name: "Acme LP-2000D", Score: 1, Reason: "same name"},
		Certain:   certain,
		Product: &client.HeurekaProduct{
			Name: "Acme LP-2000D", URL: "https://example.heureka.sk/acme-lp-2000d/",
			MinPrice: 133, OfferCount: 36,
			Offers: []client.HeurekaOffer{
				{Shop: "Shop A", Price: 126, Shipping: 3.9, ShippingKnown: true, Total: 129.9},
				{Shop: "Shop B", Price: 127.55, Shipping: 3.9, ShippingKnown: true, Total: 131.45},
				{Shop: "Shop C", Price: 134, Total: 134},
				{Shop: "Shop D", Price: 140, Total: 140},
			},
		},
	}
}

var testSubject = dealSubject{ID: 1001, Name: "Acme LP-2000D", CatalogNumber: "LP2000DXX", Price: 91.9, ComparePrice: 139.9, PriceHeader: "SALE -34 %"}

func TestBuildDealReportWithMarket(t *testing.T) {
	now := time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC)
	history := []client.PriceRecord{
		{Date: "2026-03-31", ProductID: 1001, Price: 91.9},
		{Date: "2026-03-01", ProductID: 1001, Price: 120},
	}
	r := buildDealReport(context.Background(), testSubject, fakeMarket{match: testMatch(true)}, history, now)

	if r.ClaimedDiscount != 34 {
		t.Errorf("claimed = %d", r.ClaimedDiscount)
	}
	if r.Market == nil || r.Market.MarketMin != 126 || r.Market.CheapestTotal != 129.9 || len(r.Market.Cheapest) != dealsCheapestShown {
		t.Fatalf("market = %+v", r.Market)
	}
	if r.SavingVsMarket == nil || *r.SavingVsMarket < 37.99 || *r.SavingVsMarket > 38.01 {
		t.Errorf("saving = %v", r.SavingVsMarket)
	}
	if r.History30 == nil || r.History30.Count != 1 || r.History90 == nil || r.History90.Count != 2 {
		t.Errorf("history = %+v / %+v", r.History30, r.History90)
	}
	if r.Verdict.Code != client.VerdictRealDiscount {
		t.Errorf("verdict = %+v", r.Verdict)
	}

	var buf bytes.Buffer
	printDealReport(&buf, r)
	out := buf.String()
	for _, want := range []string{"Acme LP-2000D", "kód LP2000DXX", "-34 %", "Shop A", "129.90", "SKUTOČNÁ ZĽAVA", "ušetríš 38.00", "Heureka uvádza od 133.00"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestBuildDealReportUncertainAndErrors(t *testing.T) {
	now := time.Now()
	r := buildDealReport(context.Background(), testSubject, fakeMarket{match: testMatch(false)}, nil, now)
	var buf bytes.Buffer
	printDealReport(&buf, r)
	if !strings.Contains(buf.String(), "uncertain match") {
		t.Errorf("want uncertain match note:\n%s", buf.String())
	}

	r = buildDealReport(context.Background(), testSubject, fakeMarket{err: errors.New("surf down")}, nil, now)
	if r.Market != nil || r.MarketError != "surf down" || r.Verdict.Code != client.VerdictUnknown {
		t.Errorf("report = %+v", r)
	}

	r = buildDealReport(context.Background(), testSubject, nil, nil, now)
	if !strings.Contains(r.MarketError, "--no-heureka") {
		t.Errorf("marketError = %q", r.MarketError)
	}
	buf.Reset()
	printDealReport(&buf, r)
	if !strings.Contains(buf.String(), "Chýba:") {
		t.Errorf("want missing data lines:\n%s", buf.String())
	}
}

func TestSortBySaving(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	reports := []dealReport{
		{ProductID: 1},
		{ProductID: 2, SavingVsMarket: f(-5)},
		{ProductID: 3, SavingVsMarket: f(20)},
		{ProductID: 4, SavingVsMarket: f(3)},
	}
	sortBySaving(reports)
	got := []int{reports[0].ProductID, reports[1].ProductID, reports[2].ProductID, reports[3].ProductID}
	want := []int{3, 4, 2, 1}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
	var buf bytes.Buffer
	printScanTable(&buf, reports)
	if !strings.Contains(buf.String(), "+20.00") {
		t.Errorf("table:\n%s", buf.String())
	}
}

func TestTruncate(t *testing.T) {
	if truncate("abc", 5) != "abc" || truncate("ábcdef", 4) != "ábc…" {
		t.Error("truncate")
	}
}
