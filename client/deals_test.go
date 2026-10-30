package client

import (
	"strings"
	"testing"
)

func TestDecideDeal(t *testing.T) {
	tests := []struct {
		name        string
		in          DealInput
		want        string
		wantMissing int
	}{
		{"cheaper than market incl. shipping", DealInput{AlzaPrice: 91.9, MarketMin: 126, CheapestTotal: 129.9, HistoryMedian: 100, HistoryRecords: 30}, VerdictRealDiscount, 0},
		// Regression: matching the market is not a discount (Alza 189 = market 189 was "real").
		{"equal to cheapest total", DealInput{AlzaPrice: 104, MarketMin: 100, CheapestTotal: 104, HistoryMedian: 104, HistoryRecords: 30}, VerdictNormalPrice, 0},
		{"at market min, no totals", DealInput{AlzaPrice: 100, MarketMin: 100, HistoryMedian: 100, HistoryRecords: 30}, VerdictNormalPrice, 0},
		{"4 % under cheapest total", DealInput{AlzaPrice: 191.79, MarketMin: 195, CheapestTotal: 200.48, HistoryMedian: 191.79, HistoryRecords: 30}, VerdictNormalPrice, 0},
		{"5 % under cheapest total", DealInput{AlzaPrice: 95, MarketMin: 97, CheapestTotal: 100, HistoryMedian: 95, HistoryRecords: 30}, VerdictRealDiscount, 0},
		{"only cheapest total, above", DealInput{AlzaPrice: 110, CheapestTotal: 100, HistoryMedian: 110, HistoryRecords: 30}, VerdictAboveMarket, 0},
		{"own history drop", DealInput{AlzaPrice: 95, MarketMin: 92, CheapestTotal: 94, HistoryMedian: 100, HistoryRecords: 30}, VerdictRealDiscount, 0},
		{"just above history ratio", DealInput{AlzaPrice: 95.5, MarketMin: 92, CheapestTotal: 94, HistoryMedian: 100, HistoryRecords: 30}, VerdictNormalPrice, 0},
		{"within 5 % above market", DealInput{AlzaPrice: 105, MarketMin: 100, CheapestTotal: 104.5, HistoryMedian: 105, HistoryRecords: 30}, VerdictNormalPrice, 0},
		{"over 5 % above market", DealInput{AlzaPrice: 105.5, MarketMin: 100, CheapestTotal: 104, HistoryMedian: 105.5, HistoryRecords: 30}, VerdictAboveMarket, 0},
		{"short history is ignored", DealInput{AlzaPrice: 80, MarketMin: 70, CheapestTotal: 75, HistoryMedian: 100, HistoryRecords: 3}, VerdictAboveMarket, 1},
		{"history only, drop", DealInput{AlzaPrice: 90, HistoryMedian: 100, HistoryRecords: 10}, VerdictRealDiscount, 1},
		{"history only, flat", DealInput{AlzaPrice: 99, HistoryMedian: 100, HistoryRecords: 10}, VerdictNormalPrice, 1},
		{"no data", DealInput{AlzaPrice: 99}, VerdictUnknown, 2},
		{"no Alza price", DealInput{MarketMin: 10}, VerdictUnknown, 1},
		// Regression: a weak Heureka match must not give a confident verdict.
		{"uncertain match, market says discount", DealInput{AlzaPrice: 50, MarketMin: 60, CheapestTotal: 62, MarketUncertain: true}, VerdictUncertain, 2},
		{"uncertain match, market says above", DealInput{AlzaPrice: 80, MarketMin: 60, CheapestTotal: 62, MarketUncertain: true, HistoryMedian: 80, HistoryRecords: 10}, VerdictUncertain, 1},
		{"unverified market min", DealInput{AlzaPrice: 80, MarketMin: 60, MarketMinUnverified: true}, VerdictUncertain, 2},
		{"uncertain match, own history drop still counts", DealInput{AlzaPrice: 90, MarketMin: 60, CheapestTotal: 62, MarketUncertain: true, HistoryMedian: 100, HistoryRecords: 10}, VerdictRealDiscount, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DecideDeal(tt.in)
			if got.Code != tt.want {
				t.Errorf("verdict = %s (%s), want %s", got.Code, got.Reason, tt.want)
			}
			if got.Label != VerdictLabels[tt.want] {
				t.Errorf("label = %q", got.Label)
			}
			if len(got.Missing) != tt.wantMissing {
				t.Errorf("missing = %v, want %d items", got.Missing, tt.wantMissing)
			}
			if got.Reason == "" {
				t.Error("empty reason")
			}
		})
	}
}

func TestDecideDealMissingNames(t *testing.T) {
	got := DecideDeal(DealInput{AlzaPrice: 10})
	joined := strings.Join(got.Missing, "; ")
	if !strings.Contains(joined, "Heureka") || !strings.Contains(joined, "história") {
		t.Errorf("missing = %q", joined)
	}
}

func TestClaimedDiscount(t *testing.T) {
	if got := ClaimedDiscount(91.9, 139.9); got != 34 {
		t.Errorf("got %d, want 34", got)
	}
	if got := ClaimedDiscount(100, 0); got != 0 {
		t.Errorf("no compare price: got %d", got)
	}
	if got := ClaimedDiscount(100, 90); got != 0 {
		t.Errorf("compare below price: got %d", got)
	}
}

func TestPercentDiff(t *testing.T) {
	if got := PercentDiff(110, 100); got < 9.99 || got > 10.01 {
		t.Errorf("got %v", got)
	}
	if PercentDiff(1, 0) != 0 {
		t.Error("zero base must give 0")
	}
}

func TestDecideDealUncertainKeepsMarketHint(t *testing.T) {
	got := DecideDeal(DealInput{AlzaPrice: 80, MarketMin: 60, MarketUncertain: true})
	if !strings.Contains(got.Reason, VerdictLabels[VerdictAboveMarket]) || got.Label != "neisté porovnanie" {
		t.Errorf("verdict = %+v", got)
	}
}
