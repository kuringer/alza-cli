package client

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// PriceHistoryEnv overrides the price history file path.
const PriceHistoryEnv = "ALZA_PRICE_HISTORY"

// historyDateLayout is the per-day key of a price record (local date).
const historyDateLayout = "2006-01-02"

// PriceRecord is one Alza price observation of a product on one day.
type PriceRecord struct {
	Date         string  `json:"date"`
	ProductID    int     `json:"productId"`
	Name         string  `json:"name,omitempty"`
	Price        float64 `json:"price"`
	ComparePrice float64 `json:"comparePrice,omitempty"`
}

// PriceStats summarizes a product's own price history over a window.
type PriceStats struct {
	Days   int     `json:"days"`
	Count  int     `json:"count"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Median float64 `json:"median"`
}

// PriceHistoryPath returns the JSONL history path (env override, else ~/.config/alza/price-history.jsonl).
func PriceHistoryPath() (string, error) {
	if p := strings.TrimSpace(os.Getenv(PriceHistoryEnv)); p != "" {
		return p, nil
	}
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "price-history.jsonl"), nil
}

// HistoryDate returns the day key for t.
func HistoryDate(t time.Time) string {
	return t.Format(historyDateLayout)
}

// ReadPriceHistory reads all records. A missing file is an empty history.
func ReadPriceHistory(path string) ([]PriceRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var out []PriceRecord
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	line := 0
	for sc.Scan() {
		line++
		text := strings.TrimSpace(sc.Text())
		if text == "" {
			continue
		}
		var rec PriceRecord
		if err := json.Unmarshal([]byte(text), &rec); err != nil {
			return nil, fmt.Errorf("%s:%d: %w", path, line, err)
		}
		out = append(out, rec)
	}
	return out, sc.Err()
}

// AppendPriceRecords appends records that are not yet stored for the same
// product and day. It returns how many records were written.
func AppendPriceRecords(path string, records []PriceRecord) (int, error) {
	existing, err := ReadPriceHistory(path)
	if err != nil {
		return 0, err
	}
	seen := make(map[string]bool, len(existing))
	for _, r := range existing {
		seen[historyKey(r)] = true
	}

	var buf strings.Builder
	written := 0
	for _, r := range records {
		if r.ProductID <= 0 || r.Price <= 0 || r.Date == "" {
			continue
		}
		key := historyKey(r)
		if seen[key] {
			continue
		}
		seen[key] = true
		data, err := json.Marshal(r)
		if err != nil {
			return 0, err
		}
		buf.Write(data)
		buf.WriteByte('\n')
		written++
	}
	if written == 0 {
		return 0, nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return 0, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return 0, err
	}
	if _, err := f.WriteString(buf.String()); err != nil {
		f.Close()
		return 0, err
	}
	return written, f.Close()
}

func historyKey(r PriceRecord) string {
	return fmt.Sprintf("%s|%d", r.Date, r.ProductID)
}

// HistoryStatsFor returns price stats of productID over the last `days` days
// (today included). ok is false when there is no record in the window.
func HistoryStatsFor(records []PriceRecord, productID, days int, now time.Time) (PriceStats, bool) {
	from := HistoryDate(now.AddDate(0, 0, -(days - 1)))
	to := HistoryDate(now)
	var prices []float64
	for _, r := range records {
		if r.ProductID != productID || r.Price <= 0 {
			continue
		}
		// Day keys are ISO dates, so string order is date order.
		if r.Date < from || r.Date > to {
			continue
		}
		prices = append(prices, r.Price)
	}
	if len(prices) == 0 {
		return PriceStats{Days: days}, false
	}
	sort.Float64s(prices)
	return PriceStats{
		Days:   days,
		Count:  len(prices),
		Min:    prices[0],
		Max:    prices[len(prices)-1],
		Median: median(prices),
	}, true
}

// median expects sorted input.
func median(sorted []float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}
