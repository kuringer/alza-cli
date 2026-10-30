package client

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// Fixtures mirror the page-script output shape with made-up products and shops.
const fixtureSearchByPart = `[
 {"type":"product","name":"Acme LP2000D","slug":"acme-lp2000d","categorySlug":"tlaciarne","minPrice":139.61,"offerCount":9},
 {"type":"product","name":"Acme LP-2000D","slug":"acme-lp-2000d","categorySlug":"multifunkcne-zariadenia","minPrice":133,"offerCount":37},
 {"type":"product","name":"Acme TN-200 - originálny","slug":"acme-tn-200-originalny","categorySlug":"naplne-a-tonery","minPrice":30.39,"offerCount":59}
]`

const fixtureProduct = `[{"name":"Acme LP-2000D","minPrice":133,"maxPrice":177.5,"offerCount":36,"offers":[
 {"name":"Acme LP-2000D","shop":"Shop B","price":135.71,"ship":0},
 {"name":"Acme LP-2000D LP2000DXX","shop":"Shop A","price":126,"ship":3.9},
 {"name":"Acme LP-2000D","shop":"Alza.sk","price":91.9,"ship":0},
 {"name":"Acme LP-2000D","shop":"Shop C","price":134,"ship":null},
 {"name":"broken","shop":"Shop D","price":null,"ship":1}
]}]`

func TestParseHeurekaSearch(t *testing.T) {
	got, err := ParseHeurekaSearch(json.RawMessage(fixtureSearchByPart))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("candidates = %d, want 3", len(got))
	}
	if got[1].URL != "https://multifunkcne-zariadenia.heureka.sk/acme-lp-2000d/" {
		t.Errorf("url = %q", got[1].URL)
	}
	if got[1].MinPrice != 133 || got[1].OfferCount != 37 {
		t.Errorf("candidate = %+v", got[1])
	}

	redirect := `[{"type":"product","name":"Acme X1","url":"https://x.heureka.sk/acme-x1/","minPrice":10,"offerCount":2},{"type":"offer","name":"skip"}]`
	got, err = ParseHeurekaSearch(json.RawMessage(redirect))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].URL != "https://x.heureka.sk/acme-x1/" {
		t.Errorf("redirect candidates = %+v", got)
	}

	if _, err := ParseHeurekaSearch(json.RawMessage(`{"bad":1}`)); err == nil {
		t.Error("want error on non-array")
	}
}

func TestParseHeurekaProduct(t *testing.T) {
	p, err := ParseHeurekaProduct(json.RawMessage(fixtureProduct))
	if err != nil {
		t.Fatal(err)
	}
	if p.MinPrice != 133 || p.MaxPrice != 177.5 || p.OfferCount != 36 {
		t.Errorf("product = %+v", p)
	}
	// Alza and the offer without price are dropped; sorted by total.
	if len(p.Offers) != 3 {
		t.Fatalf("offers = %d, want 3: %+v", len(p.Offers), p.Offers)
	}
	for _, o := range p.Offers {
		if isAlzaShop(o.Shop) {
			t.Errorf("Alza offer kept: %+v", o)
		}
	}
	first := p.Offers[0]
	if first.Shop != "Shop A" || first.Total != 129.9 || !first.ShippingKnown {
		t.Errorf("cheapest = %+v", first)
	}
	if p.Offers[1].Shop != "Shop C" || p.Offers[1].ShippingKnown {
		t.Errorf("second = %+v (unknown shipping expected)", p.Offers[1])
	}
	if p.CheapestTotal() != 129.9 {
		t.Errorf("CheapestTotal = %v", p.CheapestTotal())
	}
	// Listed offer (126) is below Heureka's own minPrice (133).
	if p.MarketMin() != 126 {
		t.Errorf("MarketMin = %v", p.MarketMin())
	}

	if _, err := ParseHeurekaProduct(json.RawMessage(`[{"error":"no __NEXT_DATA__ on page"}]`)); err == nil {
		t.Error("want error from script error row")
	}
	if _, err := ParseHeurekaProduct(json.RawMessage(`[]`)); err == nil {
		t.Error("want error on empty")
	}
}

func TestScoreHeurekaName(t *testing.T) {
	tests := []struct {
		name, alza, part, heureka string
		min, max                  float64
		reason                    string
	}{
		{"same name with dash", "Acme LP-2000D", "LP2000DXX", "Acme LP2000D", 1, 1, "same name"},
		{"diacritics", "Čistič Žltý 5", "", "cistic zlty 5", 1, 1, "same name"},
		{"part number in name", "Acme PG-50 + CL-51 multipack", "5225B013", "Acme 5225B013 - originálny", 0.99, 1, "part number"},
		{"bundle with extra part", "Acme PG-50 + CL-51 multipack", "5225B013", "Acme 5225B006 + 5225B013 - originálny", 0.9, 0.95, "part number"},
		{"subset name", "Acme Sonic Premium All-in-One HX90/88 2 ks", "", "Acme Sonic HX90/88", 0.99, 1, "name tokens"},
		{"compatible clone", "Acme PG-50 + CL-51 multipack", "5225B013", "Clone Acme PG-50 + CL-51 Multipack - kompatibilný", 0.6, 0.8, "name tokens"},
		{"unrelated", "Acme LP-2000D", "", "Other Thing 9", 0, 0.01, "name tokens"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, reason := ScoreHeurekaName(tt.alza, tt.part, tt.heureka)
			if got < tt.min || got > tt.max {
				t.Errorf("score = %.3f, want %.2f..%.2f", got, tt.min, tt.max)
			}
			if reason != tt.reason {
				t.Errorf("reason = %q, want %q", reason, tt.reason)
			}
		})
	}
}

func TestMatchHeureka(t *testing.T) {
	cands, err := ParseHeurekaSearch(json.RawMessage(fixtureSearchByPart))
	if err != nil {
		t.Fatal(err)
	}
	m := MatchHeureka("Acme LP-2000D", "LP2000DXX", cands)
	if m == nil {
		t.Fatal("no match")
	}
	// Both listings have the same name; more offers wins, still certain.
	if m.Candidate.OfferCount != 37 || !m.Certain {
		t.Errorf("match = %+v", m)
	}
	if len(m.Alternatives) == 0 {
		t.Error("want alternatives")
	}

	// Two different products with close scores: uncertain.
	amb := []HeurekaCandidate{
		{Name: "Acme Widget 100 blue", URL: "u1"},
		{Name: "Acme Widget 100 red", URL: "u2"},
	}
	m = MatchHeureka("Acme Widget 100", "", amb)
	if m == nil || m.Certain {
		t.Errorf("want uncertain match, got %+v", m)
	}

	if MatchHeureka("Acme", "", nil) != nil {
		t.Error("want nil for no candidates")
	}
	if MatchHeureka("Acme LP-2000D", "", []HeurekaCandidate{{Name: "Other Thing", URL: "u"}}) != nil {
		t.Error("want nil when nothing scores")
	}
}

type fakeSurf struct {
	pages map[string]string // URL substring -> data JSON
	calls []string
}

func (f *fakeSurf) Extract(_ context.Context, url, _ string) (json.RawMessage, error) {
	f.calls = append(f.calls, url)
	for k, v := range f.pages {
		if strings.Contains(url, k) {
			return json.RawMessage(v), nil
		}
	}
	return nil, errors.New("not found: " + url)
}

func TestHeurekaLookupByPartNumber(t *testing.T) {
	fs := &fakeSurf{pages: map[string]string{
		"fraze%5D=LP2000DXX": fixtureSearchByPart,
		"/acme-lp-2000d/":    fixtureProduct,
	}}
	h := &Heureka{Surf: fs}
	m, err := h.Lookup(context.Background(), "Acme LP-2000D", "LP2000DXX")
	if err != nil {
		t.Fatal(err)
	}
	if !m.Certain || m.Product == nil || m.Product.CheapestTotal() != 129.9 {
		t.Errorf("match = %+v", m)
	}
	// Certain match after the part number search: no name search.
	if len(fs.calls) != 2 {
		t.Errorf("surf calls = %v", fs.calls)
	}
}

func TestHeurekaLookupFallsBackToName(t *testing.T) {
	fs := &fakeSurf{pages: map[string]string{
		"fraze%5D=NOPE1":       `[]`,
		"fraze%5D=Acme+LP-200": fixtureSearchByPart,
		"/acme-lp-2000d/":      fixtureProduct,
	}}
	h := &Heureka{Surf: fs}
	m, err := h.Lookup(context.Background(), "Acme LP-2000D", "NOPE1")
	if err != nil {
		t.Fatal(err)
	}
	if m.Product == nil || m.Query != "NOPE1 | Acme LP-2000D" {
		t.Errorf("match = %+v", m)
	}

	empty := &Heureka{Surf: &fakeSurf{pages: map[string]string{"fraze": `[]`}}}
	if _, err := empty.Lookup(context.Background(), "Nothing", ""); err == nil {
		t.Error("want error when nothing found")
	}
}

func TestParseSurfOutput(t *testing.T) {
	data, err := parseSurfOutput([]byte(`{"data":[{"a":1}],"rows":[{"a":1}],"rowCount":1}`))
	if err != nil || string(data) != `[{"a":1}]` {
		t.Errorf("data = %s, err = %v", data, err)
	}
	if _, err := parseSurfOutput([]byte(`{"error":{"code":"invalid_output","message":"too big"}}`)); err == nil || !strings.Contains(err.Error(), "too big") {
		t.Errorf("want surf error, got %v", err)
	}
	if _, err := parseSurfOutput(nil); err == nil {
		t.Error("want error on empty output")
	}
}

func TestNewExecSurfMissing(t *testing.T) {
	t.Setenv(SurfBinEnv, "/nonexistent/surf-binary")
	_, err := NewExecSurf()
	if !errors.Is(err, ErrSurfMissing) {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(err.Error(), "--no-heureka") {
		t.Errorf("error should mention --no-heureka: %v", err)
	}
}
