package client

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestPriceHistoryPathEnvOverride(t *testing.T) {
	t.Setenv(PriceHistoryEnv, "/tmp/example/history.jsonl")
	got, err := PriceHistoryPath()
	if err != nil {
		t.Fatal(err)
	}
	if got != "/tmp/example/history.jsonl" {
		t.Errorf("path = %q", got)
	}

	t.Setenv(PriceHistoryEnv, "")
	got, err = PriceHistoryPath()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "price-history.jsonl" || filepath.Base(filepath.Dir(got)) != "alza" {
		t.Errorf("default path = %q", got)
	}
}

func TestReadPriceHistoryMissingFile(t *testing.T) {
	recs, err := ReadPriceHistory(filepath.Join(t.TempDir(), "none.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 0 {
		t.Errorf("want empty, got %d", len(recs))
	}
}

func TestAppendPriceRecordsDedupe(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "history.jsonl")
	day1 := []PriceRecord{
		{Date: "2026-01-01", ProductID: 1, Name: "Product A", Price: 10, ComparePrice: 15},
		{Date: "2026-01-01", ProductID: 2, Name: "Product B", Price: 20},
		{Date: "2026-01-01", ProductID: 1, Name: "Product A", Price: 11}, // same day dup in batch
		{Date: "2026-01-01", ProductID: 3, Name: "No price", Price: 0},   // invalid
	}
	n, err := AppendPriceRecords(path, day1)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("written = %d, want 2", n)
	}

	// Same day again: nothing new. Next day: new record.
	n, err = AppendPriceRecords(path, []PriceRecord{
		{Date: "2026-01-01", ProductID: 1, Price: 9},
		{Date: "2026-01-02", ProductID: 1, Price: 9},
	})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("written = %d, want 1", n)
	}

	recs, err := ReadPriceHistory(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 3 {
		t.Fatalf("records = %d, want 3", len(recs))
	}
	if recs[0].Price != 10 || recs[0].ComparePrice != 15 || recs[0].Name != "Product A" {
		t.Errorf("first record = %+v", recs[0])
	}
	if recs[2].Date != "2026-01-02" || recs[2].Price != 9 {
		t.Errorf("last record = %+v", recs[2])
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("perm = %v, want 0600", info.Mode().Perm())
	}
}

func TestReadPriceHistoryBadLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "h.jsonl")
	if err := os.WriteFile(path, []byte("{\"date\":\"2026-01-01\",\"productId\":1,\"price\":5}\nnot json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadPriceHistory(path); err == nil {
		t.Fatal("want error for bad line")
	}
}

func TestHistoryStatsFor(t *testing.T) {
	now := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)
	recs := []PriceRecord{
		{Date: "2026-03-31", ProductID: 1, Price: 100},
		{Date: "2026-03-20", ProductID: 1, Price: 80},
		{Date: "2026-03-02", ProductID: 1, Price: 120}, // day 30 of the 30-day window
		{Date: "2026-03-01", ProductID: 1, Price: 60},  // outside 30 days, inside 90
		{Date: "2026-03-25", ProductID: 2, Price: 10},  // other product
		{Date: "2025-12-01", ProductID: 1, Price: 50},  // outside 90 days
	}

	s30, ok := HistoryStatsFor(recs, 1, 30, now)
	if !ok {
		t.Fatal("want stats for 30 days")
	}
	if s30.Count != 3 || s30.Min != 80 || s30.Max != 120 || s30.Median != 100 {
		t.Errorf("30d = %+v", s30)
	}

	s90, ok := HistoryStatsFor(recs, 1, 90, now)
	if !ok {
		t.Fatal("want stats for 90 days")
	}
	if s90.Count != 4 || s90.Min != 60 || s90.Median != 90 {
		t.Errorf("90d = %+v", s90)
	}

	if _, ok := HistoryStatsFor(recs, 99, 90, now); ok {
		t.Error("unknown product must have no stats")
	}
}

// Regression: parallel writers must not store the same product twice a day.
func TestAppendPriceRecordsConcurrent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "h.jsonl")
	rec := []PriceRecord{{Date: "2026-03-31", ProductID: 7, Price: 10}}
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := AppendPriceRecords(path, rec); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	got, err := ReadPriceHistory(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Errorf("records = %d, want 1", len(got))
	}
}

// Regression: a duplicate day (old file, unlocked writer) counts once.
func TestHistoryStatsDedupeDay(t *testing.T) {
	now := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)
	recs := []PriceRecord{
		{Date: "2026-03-31", ProductID: 7, Price: 10},
		{Date: "2026-03-31", ProductID: 7, Price: 10},
		{Date: "2026-03-30", ProductID: 7, Price: 20},
	}
	st, ok := HistoryStatsFor(recs, 7, 90, now)
	if !ok || st.Count != 2 || st.Median != 15 {
		t.Errorf("stats = %+v", st)
	}
}
