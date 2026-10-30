package client

import (
	"fmt"
	"math"
)

// Deal verdict thresholds.
const (
	// DealMarketTolerance: Alza within +/- this share of the market minimum is a normal price.
	DealMarketTolerance = 0.05
	// DealHistoryRatio: Alza at or below this share of its own median price is a real discount.
	DealHistoryRatio = 0.95
	// DealHistoryDays is the window of the own-price median.
	DealHistoryDays = 90
	// DealMinHistoryRecords: fewer daily records than this make the median unreliable.
	DealMinHistoryRecords = 7
)

// Verdict codes (stable, for JSON).
const (
	VerdictRealDiscount = "real_discount"
	VerdictNormalPrice  = "normal_price"
	VerdictAboveMarket  = "above_market"
	VerdictUncertain    = "uncertain_comparison"
	VerdictUnknown      = "unknown"
)

// VerdictLabels are the human labels of verdict codes.
var VerdictLabels = map[string]string{
	VerdictRealDiscount: "skutočná zľava",
	VerdictNormalPrice:  "bežná cena",
	VerdictAboveMarket:  "drahšie ako trh",
	VerdictUncertain:    "neisté porovnanie",
	VerdictUnknown:      "nedá sa určiť",
}

// DealInput holds the prices a verdict is based on. Zero means "not known".
type DealInput struct {
	AlzaPrice       float64
	MarketMin       float64 // lowest market item price (Heureka, Alza excluded)
	CheapestTotal   float64 // lowest market price + approx. shipping
	HistoryMedian   float64 // own Alza median over DealHistoryDays
	HistoryRecords  int
	MarketUncertain bool // Heureka product match is not certain
	// MarketMinUnverified: MarketMin is Heureka's minPrice without a known shop.
	MarketMinUnverified bool
}

// DealVerdict is the result of DecideDeal.
type DealVerdict struct {
	Code    string   `json:"verdict"`
	Label   string   `json:"label"`
	Reason  string   `json:"reason"`
	Missing []string `json:"missing,omitempty"`
}

// DecideDeal says whether the Alza price is a real discount:
//   - real discount: Alza <= cheapest market offer (with shipping) or <= market
//     minimum, or <= DealHistoryRatio of its own 90-day median
//   - normal price: within DealMarketTolerance of the market minimum
//   - above market: more than DealMarketTolerance over the market minimum
func DecideDeal(in DealInput) DealVerdict {
	var missing []string
	hasMarket := in.MarketMin > 0 || in.CheapestTotal > 0
	if !hasMarket {
		missing = append(missing, "trhová cena (Heureka)")
	} else {
		if in.MarketUncertain {
			missing = append(missing, "istá zhoda na Heureke (porovnanie môže byť s iným produktom)")
		}
		if in.MarketMinUnverified {
			missing = append(missing, "overená ponuka iného obchodu (cena trhu je súhrnná cena Heureky, môže to byť Alza)")
		}
	}
	hasHistory := in.HistoryMedian > 0 && in.HistoryRecords >= DealMinHistoryRecords
	if !hasHistory {
		missing = append(missing, fmt.Sprintf("história cien (dni so záznamom za %d dní: %d, treba aspoň %d)", DealHistoryDays, in.HistoryRecords, DealMinHistoryRecords))
	}

	v := func(code, reason string) DealVerdict {
		return DealVerdict{Code: code, Label: VerdictLabels[code], Reason: reason, Missing: missing}
	}

	if in.AlzaPrice <= 0 {
		return v(VerdictUnknown, "chýba cena na Alze")
	}
	marketCode, marketReason := decideByMarket(in)
	uncertain := marketCode != "" && (in.MarketUncertain || in.MarketMinUnverified)
	histDiscount := hasHistory && in.AlzaPrice <= DealHistoryRatio*in.HistoryMedian
	switch {
	case marketCode == VerdictRealDiscount && !uncertain:
		return v(marketCode, marketReason)
	case histDiscount:
		return v(VerdictRealDiscount, fmt.Sprintf("Alza %.2f <= %.0f %% vlastného %d-dňového mediánu %.2f EUR", in.AlzaPrice, DealHistoryRatio*100, DealHistoryDays, in.HistoryMedian))
	case uncertain:
		// A weak match must not give a confident verdict: show it only as a hint.
		return v(VerdictUncertain, fmt.Sprintf("trh nie je istý; podľa Heureky by to bola %s (%s)", VerdictLabels[marketCode], marketReason))
	case marketCode != "":
		return v(marketCode, marketReason)
	}
	if hasHistory {
		return v(VerdictNormalPrice, fmt.Sprintf("bez trhu; Alza nad %.0f %% vlastného mediánu %.2f EUR", DealHistoryRatio*100, in.HistoryMedian))
	}
	return v(VerdictUnknown, "chýba trhová cena aj história")
}

// decideByMarket compares the Alza price with the market only ("" = no market).
func decideByMarket(in DealInput) (string, string) {
	if in.CheapestTotal > 0 && in.AlzaPrice <= in.CheapestTotal {
		return VerdictRealDiscount, fmt.Sprintf("Alza %.2f <= najlacnejšia ponuka trhu s dopravou %.2f EUR", in.AlzaPrice, in.CheapestTotal)
	}
	if in.MarketMin <= 0 {
		return "", ""
	}
	if in.AlzaPrice <= in.MarketMin {
		return VerdictRealDiscount, fmt.Sprintf("Alza %.2f <= najnižšia cena trhu %.2f EUR", in.AlzaPrice, in.MarketMin)
	}
	diff := PercentDiff(in.AlzaPrice, in.MarketMin)
	if in.AlzaPrice > in.MarketMin*(1+DealMarketTolerance) {
		return VerdictAboveMarket, fmt.Sprintf("Alza o %.0f %% drahšia ako najnižšia cena trhu %.2f EUR", diff, in.MarketMin)
	}
	return VerdictNormalPrice, fmt.Sprintf("Alza do %.0f %% od najnižšej ceny trhu %.2f EUR (%+.0f %%)", DealMarketTolerance*100, in.MarketMin, diff)
}

// PercentDiff returns (a-b)/b in percent.
func PercentDiff(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return (a - b) / b * 100
}

// ClaimedDiscount returns the discount Alza claims against its crossed-out price, in whole percent.
func ClaimedDiscount(price, compare float64) int {
	if price <= 0 || compare <= price {
		return 0
	}
	return int(math.Round((1 - price/compare) * 100))
}
