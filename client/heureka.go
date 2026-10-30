package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

// HeurekaBaseURL is the Heureka.sk search entry point.
const HeurekaBaseURL = "https://www.heureka.sk/"

// heurekaMaxOffers caps offers returned by the page script (surf output is limited to ~50 kB).
const heurekaMaxOffers = 40

// heurekaSearchScript reads search results from the server-rendered
// category-search-state. When Heureka redirects the search straight to a
// product page, it returns that product.
const heurekaSearchScript = `
const next = document.getElementById('__NEXT_DATA__');
const st = document.getElementById('category-search-state');
if (!st && next) {
  const d = JSON.parse(next.textContent).props.pageProps.initialData.productDetail;
  if (d && d.product) return [{type: 'product', name: d.product.name, url: location.origin + location.pathname, minPrice: d.product.minPrice, offerCount: d.offers && d.offers.regularCount}];
}
if (!st) return [];
const s = JSON.parse(st.textContent);
const q = Object.values((s.oneRequestGetAllApi || {}).queries || {});
const out = [];
for (const e of q) {
  const items = (e.data && e.data.items && e.data.items.result) || [];
  for (const x of items) {
    if (x.type !== 'product') continue;
    out.push({type: x.type, name: x.name, slug: x.slug, categorySlug: x.categorySlug, minPrice: x.minPrice, offerCount: x.offerCount});
  }
}
return out.slice(0, 15);
`

// heurekaProductScript reads price and offers from a Heureka product page (Next.js).
var heurekaProductScript = fmt.Sprintf(`
const el = document.getElementById('__NEXT_DATA__');
if (!el) return [{error: 'no __NEXT_DATA__ on page'}];
const d = JSON.parse(el.textContent).props.pageProps.initialData.productDetail;
if (!d || !d.product) return [{error: 'no productDetail on page'}];
const raw = [...(d.offers.bidding || []), ...(d.offers.regular || []), ...(d.offers.topOffer ? [d.offers.topOffer] : [])];
const seen = new Set();
const offers = [];
for (const x of raw) {
  if (!x || seen.has(x.id)) continue;
  seen.add(x.id);
  offers.push({id: x.id, name: x.name, shop: x.shop && x.shop.name, price: x.price, ship: x.minDeliveryPrice});
}
offers.sort((a, b) => (a.price + (a.ship || 0)) - (b.price + (b.ship || 0)));
return [{name: d.product.name, minPrice: d.product.minPrice, maxPrice: d.product.maxPrice, offerCount: d.offers.regularCount, offers: offers.slice(0, %d)}];
`, heurekaMaxOffers)

// HeurekaCandidate is one product from Heureka search results.
type HeurekaCandidate struct {
	Name       string  `json:"name"`
	URL        string  `json:"url"`
	MinPrice   float64 `json:"minPrice,omitempty"`
	OfferCount int     `json:"offerCount,omitempty"`
	Score      float64 `json:"score"`
	Reason     string  `json:"reason,omitempty"`
}

// HeurekaOffer is one shop offer. Shipping is Heureka's "from" delivery price
// and can be wrong (free-shipping thresholds), so treat it as approximate.
type HeurekaOffer struct {
	Shop          string  `json:"shop"`
	Name          string  `json:"name,omitempty"`
	Price         float64 `json:"price"`
	Shipping      float64 `json:"shippingApprox"`
	ShippingKnown bool    `json:"shippingKnown"`
	Total         float64 `json:"totalApprox"`
}

// HeurekaProduct is the market view of one Heureka product (Alza offers excluded).
type HeurekaProduct struct {
	Name       string         `json:"name"`
	URL        string         `json:"url"`
	MinPrice   float64        `json:"minPrice"`
	MaxPrice   float64        `json:"maxPrice,omitempty"`
	OfferCount int            `json:"offerCount"`
	Offers     []HeurekaOffer `json:"offers"`
}

// HeurekaMatch is the result of mapping an Alza product to Heureka.
type HeurekaMatch struct {
	Query        string             `json:"query"`
	Candidate    HeurekaCandidate   `json:"candidate"`
	Certain      bool               `json:"certain"`
	Alternatives []HeurekaCandidate `json:"alternatives,omitempty"`
	Product      *HeurekaProduct    `json:"product,omitempty"`
}

// MarketMin is the lowest known market item price (no shipping).
func (p *HeurekaProduct) MarketMin() float64 {
	best := p.MinPrice
	for _, o := range p.Offers {
		if o.Price > 0 && (best <= 0 || o.Price < best) {
			best = o.Price
		}
	}
	return best
}

// CheapestTotal is the lowest price + approx. shipping among loaded offers.
func (p *HeurekaProduct) CheapestTotal() float64 {
	if len(p.Offers) == 0 {
		return 0
	}
	return p.Offers[0].Total
}

// Heureka maps Alza products to Heureka.sk via a browser (surf).
type Heureka struct {
	Surf    SurfRunner
	BaseURL string
	Timeout time.Duration
}

// NewHeureka returns a Heureka client using the surf CLI.
func NewHeureka() (*Heureka, error) {
	s, err := NewExecSurf()
	if err != nil {
		return nil, err
	}
	return &Heureka{Surf: s}, nil
}

// HeurekaSearchURL returns the search page URL for a query.
func (h *Heureka) HeurekaSearchURL(query string) string {
	base := h.BaseURL
	if base == "" {
		base = HeurekaBaseURL
	}
	return base + "?h%5Bfraze%5D=" + url.QueryEscape(query)
}

func (h *Heureka) extract(ctx context.Context, pageURL, script string) (json.RawMessage, error) {
	timeout := h.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return h.Surf.Extract(ctx, pageURL, script)
}

// Search returns product candidates for a query.
func (h *Heureka) Search(ctx context.Context, query string) ([]HeurekaCandidate, error) {
	data, err := h.extract(ctx, h.HeurekaSearchURL(query), heurekaSearchScript)
	if err != nil {
		return nil, err
	}
	return ParseHeurekaSearch(data)
}

// Product loads offers of a Heureka product page.
func (h *Heureka) Product(ctx context.Context, productURL string) (*HeurekaProduct, error) {
	data, err := h.extract(ctx, productURL, heurekaProductScript)
	if err != nil {
		return nil, err
	}
	p, err := ParseHeurekaProduct(data)
	if err != nil {
		return nil, err
	}
	p.URL = productURL
	return p, nil
}

// Lookup finds the Heureka product for an Alza product: search by part number
// first, then by name, pick the best match and load its offers.
func (h *Heureka) Lookup(ctx context.Context, name, partNumber string) (*HeurekaMatch, error) {
	var candidates []HeurekaCandidate
	var queries []string
	var searchErr error

	search := func(q string) {
		q = strings.TrimSpace(q)
		if q == "" {
			return
		}
		queries = append(queries, q)
		found, err := h.Search(ctx, q)
		if err != nil {
			searchErr = err
			return
		}
		candidates = mergeCandidates(candidates, found)
	}

	search(partNumber)
	match := MatchHeureka(name, partNumber, candidates)
	if match == nil || !match.Certain {
		search(name)
		match = MatchHeureka(name, partNumber, candidates)
	}
	if match == nil {
		if searchErr != nil {
			return nil, searchErr
		}
		return nil, fmt.Errorf("no Heureka product found for %q", strings.Join(queries, `" / "`))
	}
	match.Query = strings.Join(queries, " | ")

	product, err := h.Product(ctx, match.Candidate.URL)
	if err != nil {
		return match, err
	}
	match.Product = product
	return match, nil
}

func mergeCandidates(existing, found []HeurekaCandidate) []HeurekaCandidate {
	seen := map[string]bool{}
	for _, c := range existing {
		seen[c.URL] = true
	}
	for _, c := range found {
		if !seen[c.URL] {
			seen[c.URL] = true
			existing = append(existing, c)
		}
	}
	return existing
}

type heurekaSearchRow struct {
	Type         string   `json:"type"`
	Name         string   `json:"name"`
	URL          string   `json:"url"`
	Slug         string   `json:"slug"`
	CategorySlug string   `json:"categorySlug"`
	MinPrice     *float64 `json:"minPrice"`
	OfferCount   *int     `json:"offerCount"`
}

// ParseHeurekaSearch parses the search script output.
func ParseHeurekaSearch(data json.RawMessage) ([]HeurekaCandidate, error) {
	var rows []heurekaSearchRow
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, fmt.Errorf("parse Heureka search: %w", err)
	}
	var out []HeurekaCandidate
	for _, r := range rows {
		if r.Type != "" && r.Type != "product" {
			continue
		}
		u := r.URL
		if u == "" && r.Slug != "" && r.CategorySlug != "" {
			u = fmt.Sprintf("https://%s.heureka.sk/%s/", r.CategorySlug, r.Slug)
		}
		if u == "" || strings.TrimSpace(r.Name) == "" {
			continue
		}
		c := HeurekaCandidate{Name: strings.TrimSpace(r.Name), URL: u}
		if r.MinPrice != nil {
			c.MinPrice = *r.MinPrice
		}
		if r.OfferCount != nil {
			c.OfferCount = *r.OfferCount
		}
		out = append(out, c)
	}
	return out, nil
}

type heurekaProductRow struct {
	Error      string   `json:"error"`
	Name       string   `json:"name"`
	MinPrice   *float64 `json:"minPrice"`
	MaxPrice   *float64 `json:"maxPrice"`
	OfferCount *int     `json:"offerCount"`
	Offers     []struct {
		Name  string   `json:"name"`
		Shop  string   `json:"shop"`
		Price *float64 `json:"price"`
		Ship  *float64 `json:"ship"`
	} `json:"offers"`
}

// ParseHeurekaProduct parses the product script output. Alza's own offers are
// dropped: the point is to compare Alza against the rest of the market.
func ParseHeurekaProduct(data json.RawMessage) (*HeurekaProduct, error) {
	var rows []heurekaProductRow
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, fmt.Errorf("parse Heureka product: %w", err)
	}
	if len(rows) == 0 {
		return nil, errors.New("Heureka product page returned no data")
	}
	r := rows[0]
	if r.Error != "" {
		return nil, fmt.Errorf("Heureka product page: %s", r.Error)
	}
	p := &HeurekaProduct{Name: r.Name}
	if r.MinPrice != nil {
		p.MinPrice = *r.MinPrice
	}
	if r.MaxPrice != nil {
		p.MaxPrice = *r.MaxPrice
	}
	if r.OfferCount != nil {
		p.OfferCount = *r.OfferCount
	}
	for _, o := range r.Offers {
		if o.Price == nil || *o.Price <= 0 || isAlzaShop(o.Shop) {
			continue
		}
		offer := HeurekaOffer{Shop: o.Shop, Name: o.Name, Price: *o.Price, Total: *o.Price}
		if o.Ship != nil && *o.Ship >= 0 {
			offer.Shipping = *o.Ship
			offer.ShippingKnown = true
			offer.Total = round2(*o.Price + *o.Ship)
		}
		p.Offers = append(p.Offers, offer)
	}
	sort.SliceStable(p.Offers, func(i, j int) bool { return p.Offers[i].Total < p.Offers[j].Total })
	return p, nil
}

func isAlzaShop(name string) bool {
	return strings.Contains(strings.ToLower(name), "alza")
}

func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}
