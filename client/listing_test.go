package client

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
)

func readListingFixture(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("testdata/listing_boxes.html")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestParseListingBoxes(t *testing.T) {
	items, err := ParseListingBoxes(readListingFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("items = %d, want 3 (box without price skipped): %+v", len(items), items)
	}

	a := items[0]
	if a.ID != 1001 || a.Code != "ABC001" || a.Name != "Acme Perfume 85 ml" {
		t.Errorf("item 0 identity = %+v", a)
	}
	if a.URL != BaseURL+"/example/acme-perfume-85-ml-d1001.htm" {
		t.Errorf("url = %q", a.URL)
	}
	if a.Price != 18.99 || a.ComparePrice != 27.9 || a.ClaimedDiscount != 31 || a.Label != "BLACK FRIDAY -31 %" {
		t.Errorf("item 0 price = %+v", a)
	}
	if a.Rating != 4.8 || a.ReviewCount != 599 {
		t.Errorf("item 0 rating = %v / %d", a.Rating, a.ReviewCount)
	}
	if got := a.ClaimedSaving(); got != 8.91 {
		t.Errorf("saving = %v", got)
	}

	b := items[1]
	if b.Price != 1029.9 || b.ComparePrice != 1300 || b.ClaimedDiscount != 21 || b.ReviewCount != 1234 || b.Rating != 4.5 {
		t.Errorf("item 1 (saving variant) = %+v", b)
	}

	if a.Used || b.Used {
		t.Error("new items marked as used")
	}

	c := items[2]
	if c.Price != 9.51 || c.ComparePrice != 0 || c.ClaimedDiscount != 40 || !c.Used || c.Rating != 0 || c.ReviewCount != 0 || c.ClaimedSaving() != 0 {
		t.Errorf("item 2 (used, no compare price) = %+v", c)
	}
}

const testPageData = `<script>var _root='/';var _pageData ={"categoryId":1,"configurationId":9,"data":{"idPrefix":0,"categoryId":1,"categoryTypeId":29,"prefixType":3,"categoryHeading":"Black Friday","categoryProducers":[1,2]}};var x = 1;</script>`

func TestParseListingPageData(t *testing.T) {
	l, err := ParseListingPageData(testPageData)
	if err != nil {
		t.Fatal(err)
	}
	if l.CategoryID != 1 || l.CategoryType != 29 || l.PrefixType != 3 || l.ConfigurationID != 9 || l.Heading != "Black Friday" {
		t.Errorf("listing = %+v", l)
	}
	if _, err := ParseListingPageData("<html>product detail</html>"); err == nil {
		t.Error("want error for a page without _pageData")
	}
	if _, err := ParseListingPageData(`var _pageData ={"categoryId":0,"data":{}};`); err == nil {
		t.Error("want error for a page without category")
	}
}

func TestListingPageRetriesChallenge(t *testing.T) {
	boxes := readListingFixture(t)
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != EndpointCatalogFilter || r.Method != "POST" {
			http.NotFound(w, r)
			return
		}
		if calls.Add(1) == 1 {
			_, _ = io.WriteString(w, "<html>challenge</html>")
			return
		}
		var req filterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Page != 3 || req.PageTo != 3 || req.Sort != 11 || req.CategoryType != 29 {
			t.Errorf("request = %+v (%v)", req, err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"d": map[string]any{"Boxes": boxes, "Count": 38557, "Page": 3}})
	}))
	defer srv.Close()

	c, err := NewListingClient(false)
	if err != nil {
		t.Fatal(err)
	}
	c.BaseURL = srv.URL
	c.Retry = 0
	items, total, err := c.ListingPage(&Listing{CategoryID: 1, CategoryType: 29, PrefixType: 3, ConfigurationID: 9}, ListingSorts["discount"], 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 || total != 38557 || calls.Load() != 2 {
		t.Errorf("items=%d total=%d calls=%d", len(items), total, calls.Load())
	}
}

func TestOpenListingRejectsForeignHost(t *testing.T) {
	c := &ListingClient{}
	for _, u := range []string{"https://example.com/black-friday", "not a url"} {
		if _, err := c.OpenListing(u); err == nil || !strings.Contains(err.Error(), "listing URL") {
			t.Errorf("%q: err = %v", u, err)
		}
	}
}
