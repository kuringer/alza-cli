package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	xhtml "golang.org/x/net/html"
)

// DefaultListingURL is the listing `alza deals browse` reads by default.
const DefaultListingURL = BaseURL + "/black-friday"

// ListingPageSize is how many product boxes one listing page returns.
const ListingPageSize = 24

// listingAttempts: Cloudflare sometimes answers the first request with a
// challenge (403 or an HTML page); a retry with the cookies it set passes.
const listingAttempts = 3

// ListingSorts maps CLI sort names to Alza's sort ids (data-sort on the listing page).
var ListingSorts = map[string]int{
	"recommended": 0,
	"cheapest":    1,
	"expensive":   2,
	"rating":      6,
	"sales":       7,
	"discount":    11,
	"newest":      19,
}

// Listing identifies a product listing (category, Black Friday, sale page).
// The values come from the page's _pageData and feed the Filter service.
type Listing struct {
	URL             string `json:"url"`
	Heading         string `json:"heading,omitempty"`
	CategoryID      int    `json:"categoryId"`
	CategoryType    int    `json:"categoryType"`
	IDPrefix        int    `json:"idPrefix"`
	PrefixType      int    `json:"prefixType"`
	ConfigurationID int    `json:"configurationId"`
}

// ListingItem is one product box of a listing.
type ListingItem struct {
	ID           int     `json:"productId"`
	Code         string  `json:"code,omitempty"`
	Name         string  `json:"name"`
	URL          string  `json:"url,omitempty"`
	Price        float64 `json:"price"`
	ComparePrice float64 `json:"comparePrice,omitempty"`
	// ClaimedDiscount is the % Alza shows (label), else computed from ComparePrice.
	ClaimedDiscount int     `json:"claimedDiscountPercent,omitempty"`
	Label           string  `json:"label,omitempty"`
	Rating          float64 `json:"rating,omitempty"`
	ReviewCount     int     `json:"reviewCount,omitempty"`
	// Used: a used/returned item (Alza "bazar"); its % is against the new price.
	Used bool `json:"used,omitempty"`
}

// ClaimedSaving is ComparePrice - Price in EUR (0 without a crossed-out price).
func (it ListingItem) ClaimedSaving() float64 {
	if it.ComparePrice <= it.Price {
		return 0
	}
	return round2(it.ComparePrice - it.Price)
}

// ListingClient reads Alza listings anonymously (no auth token needed).
type ListingClient struct {
	http    tls_client.HttpClient
	BaseURL string
	Debug   bool
	// Retry is the pause before a retry after a Cloudflare challenge.
	Retry time.Duration
}

// NewListingClient returns an anonymous client with a Chrome TLS fingerprint.
func NewListingClient(debug bool) (*ListingClient, error) {
	c, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(),
		tls_client.WithTimeoutSeconds(30),
		tls_client.WithClientProfile(profiles.Chrome_120),
		tls_client.WithCookieJar(tls_client.NewCookieJar()),
		tls_client.WithRandomTLSExtensionOrder(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS client: %w", err)
	}
	return &ListingClient{http: c, BaseURL: BaseURL, Debug: debug, Retry: 1500 * time.Millisecond}, nil
}

func (c *ListingClient) do(method, rawURL, body string, html bool) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < listingAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(c.Retry * time.Duration(attempt))
		}
		var rd io.Reader
		if body != "" {
			rd = strings.NewReader(body)
		}
		req, err := http.NewRequest(method, rawURL, rd)
		if err != nil {
			return nil, err
		}
		req.Header = baseHeaders()
		if html {
			req.Header.Set("Accept", "text/html,application/xhtml+xml")
			req.Header.Set("Sec-Fetch-Mode", "navigate")
			req.Header.Set("Sec-Fetch-Dest", "document")
			req.Header.Del("Origin")
		} else {
			req.Header.Set("Content-Type", "application/json; charset=UTF-8")
			req.Header.Set("X-Requested-With", "XMLHttpRequest")
		}
		if c.Debug {
			fmt.Printf("[DEBUG] %s %s\n", method, rawURL)
		}
		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}
		b, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			continue
		}
		if resp.StatusCode >= 400 {
			lastErr = &HTTPError{Status: resp.StatusCode, URL: rawURL, Body: snippet(b, 200)}
			if resp.StatusCode == 404 {
				return nil, lastErr
			}
			continue
		}
		// A challenge page instead of JSON also means "retry".
		if !html && !json.Valid(b) {
			lastErr = fmt.Errorf("%s: not JSON (bot check?): %s", rawURL, snippet(b, 120))
			continue
		}
		return b, nil
	}
	return nil, lastErr
}

// OpenListing loads a listing page and reads its identifiers.
func (c *ListingClient) OpenListing(pageURL string) (*Listing, error) {
	u, err := url.Parse(pageURL)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("invalid listing URL %q", pageURL)
	}
	if !strings.HasSuffix(u.Hostname(), "alza.sk") {
		return nil, fmt.Errorf("listing URL must be on alza.sk: %q", pageURL)
	}
	u.Fragment = ""
	body, err := c.do("GET", u.String(), "", true)
	if err != nil {
		return nil, fmt.Errorf("listing page: %w", err)
	}
	l, err := ParseListingPageData(string(body))
	if err != nil {
		return nil, err
	}
	l.URL = u.String()
	return l, nil
}

type pageDataJSON struct {
	CategoryID      int `json:"categoryId"`
	ConfigurationID int `json:"configurationId"`
	Data            struct {
		CategoryID      int    `json:"categoryId"`
		CategoryTypeID  int    `json:"categoryTypeId"`
		IDPrefix        int    `json:"idPrefix"`
		PrefixType      int    `json:"prefixType"`
		CategoryHeading string `json:"categoryHeading"`
	} `json:"data"`
}

// ParseListingPageData reads `var _pageData = {...}` of a listing page.
func ParseListingPageData(page string) (*Listing, error) {
	const marker = "_pageData ="
	i := strings.Index(page, marker)
	if i < 0 {
		return nil, errors.New("not a product listing page (no _pageData)")
	}
	var pd pageDataJSON
	if err := json.NewDecoder(strings.NewReader(page[i+len(marker):])).Decode(&pd); err != nil {
		return nil, fmt.Errorf("parse _pageData: %w", err)
	}
	cat := pd.Data.CategoryID
	if cat == 0 {
		cat = pd.CategoryID
	}
	if cat == 0 || pd.Data.CategoryTypeID == 0 {
		return nil, errors.New("not a product listing page (no category in _pageData)")
	}
	return &Listing{
		Heading:         pd.Data.CategoryHeading,
		CategoryID:      cat,
		CategoryType:    pd.Data.CategoryTypeID,
		IDPrefix:        pd.Data.IDPrefix,
		PrefixType:      pd.Data.PrefixType,
		ConfigurationID: pd.ConfigurationID,
	}, nil
}

// filterRequest is the body of the Filter service (as sent by the "24 more" button).
type filterRequest struct {
	IDCategory                    int      `json:"idCategory"`
	Producers                     string   `json:"producers"`
	Parameters                    []string `json:"parameters"`
	IDPrefix                      int      `json:"idPrefix"`
	PrefixType                    int      `json:"prefixType"`
	Page                          int      `json:"page"`
	PageTo                        int      `json:"pageTo"`
	AvailabilityType              int      `json:"availabilityType"`
	NewsOnly                      bool     `json:"newsOnly"`
	CommodityWears                []string `json:"commodityWears"`
	UpperDescriptionStatus        int      `json:"upperDescriptionStatus"`
	BranchID                      int      `json:"branchId"`
	Sort                          int      `json:"sort"`
	CategoryType                  int      `json:"categoryType"`
	SearchTerm                    string   `json:"searchTerm"`
	Author                        string   `json:"author"`
	Append                        bool     `json:"append"`
	YearFrom                      *int     `json:"yearFrom"`
	YearTo                        *int     `json:"yearTo"`
	ArtistID                      *int     `json:"artistId"`
	MinPrice                      int      `json:"minPrice"`
	MaxPrice                      int      `json:"maxPrice"`
	ShowOnlyActionCommodities     bool     `json:"showOnlyActionCommodities"`
	UseRatingThreshold            bool     `json:"useRatingThreshold"`
	ShowOnlyAlzaPlusCommodities   bool     `json:"showOnlyAlzaPlusCommodities"`
	CashbackCodeStyleVariant      int      `json:"cashbackCodeStyleVariant"`
	CallFromParametrizationDialog bool     `json:"callFromParametrizationDialog"`
	ConfigurationID               int      `json:"configurationId"`
	Scroll                        int      `json:"scroll"`
	Hash                          string   `json:"hash"`
	Counter                       int      `json:"counter"`
}

type filterResponse struct {
	D *struct {
		Boxes string `json:"Boxes"`
		Count int    `json:"Count"`
		Page  int    `json:"Page"`
	} `json:"d"`
}

// ListingPage returns the products of one listing page (1-based) and the
// total product count of the listing.
func (c *ListingClient) ListingPage(l *Listing, sort, page int) ([]ListingItem, int, error) {
	if page < 1 {
		page = 1
	}
	req := filterRequest{
		IDCategory: l.CategoryID, Parameters: []string{}, IDPrefix: l.IDPrefix, PrefixType: l.PrefixType,
		Page: page, PageTo: page, CommodityWears: []string{}, BranchID: -2, Sort: sort,
		CategoryType: l.CategoryType, MinPrice: -1, MaxPrice: -1, CashbackCodeStyleVariant: 2,
		ConfigurationID: l.ConfigurationID, Hash: fmt.Sprintf("#f&cud=0&pg=%d&prod=", page), Counter: 1,
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, 0, err
	}
	base := c.BaseURL
	if base == "" {
		base = BaseURL
	}
	raw, err := c.do("POST", base+EndpointCatalogFilter, string(body), false)
	if err != nil {
		return nil, 0, fmt.Errorf("listing page %d: %w", page, err)
	}
	var resp filterResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, 0, fmt.Errorf("listing page %d: %w", page, err)
	}
	if resp.D == nil {
		return nil, 0, fmt.Errorf("listing page %d: empty response", page)
	}
	items, err := ParseListingBoxes(resp.D.Boxes)
	return items, resp.D.Count, err
}

var labelPercent = regexp.MustCompile(`-\s*(\d{1,2})\s*%`)

// ParseListingBoxes parses `.browsingitem` product boxes (listing HTML).
func ParseListingBoxes(boxes string) ([]ListingItem, error) {
	doc, err := xhtml.Parse(strings.NewReader(boxes))
	if err != nil {
		return nil, fmt.Errorf("parse listing boxes: %w", err)
	}
	var out []ListingItem
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		if n.Type == xhtml.ElementNode && hasClass(n, "browsingitem") && attr(n, "data-id") != "" {
			if it, ok := parseListingBox(n); ok {
				out = append(out, it)
			}
			return
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			walk(ch)
		}
	}
	walk(doc)
	return out, nil
}

func parseListingBox(n *xhtml.Node) (ListingItem, bool) {
	id, err := strconv.Atoi(attr(n, "data-id"))
	if err != nil || id <= 0 {
		return ListingItem{}, false
	}
	it := ListingItem{ID: id, Code: attr(n, "data-code")}
	if a := findNode(n, func(x *xhtml.Node) bool { return x.Data == "a" && hasClass(x, "name") }); a != nil {
		it.Name = collapseSpaces(nodeText(a))
		if href := attr(a, "href"); href != "" {
			it.URL = BaseURL + href
			if strings.HasPrefix(href, "http") {
				it.URL = href
			}
		}
	}
	it.Price = parsePrice(classText(n, "js-price-box__primary-price__value"))
	it.Label = collapseSpaces(classText(n, "ads-pb__header"))
	if s := classText(n, "js-compare-price"); s != "" {
		it.ComparePrice = parsePrice(s)
	} else if s := classText(n, "ads-pb__original-price"); strings.Contains(strings.ToLower(s), "ušetríte") {
		// "Ušetríte 288,09 €": the crossed-out price is not shown, only the saving.
		if saving := parsePrice(s); saving > 0 && it.Price > 0 {
			it.ComparePrice = round2(it.Price + saving)
		}
	}
	if m := labelPercent.FindStringSubmatch(it.Label); m != nil {
		it.ClaimedDiscount, _ = strconv.Atoi(m[1])
	} else {
		it.ClaimedDiscount = ClaimedDiscount(it.Price, it.ComparePrice)
	}
	it.Used = strings.Contains(it.URL, "-bazar-") || strings.Contains(strings.ToLower(it.Label), "z nového")
	it.Rating = parsePrice(classText(n, "star-rating-block__value"))
	it.ReviewCount = int(parsePrice(classText(n, "star-rating-block__count")))
	if it.Name == "" || it.Price <= 0 {
		return it, false
	}
	return it, true
}

func classText(n *xhtml.Node, class string) string {
	if x := findNode(n, func(x *xhtml.Node) bool { return hasClass(x, class) }); x != nil {
		return strings.TrimSpace(nodeText(x))
	}
	return ""
}

func findNode(n *xhtml.Node, pred func(*xhtml.Node) bool) *xhtml.Node {
	for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
		if ch.Type == xhtml.ElementNode && pred(ch) {
			return ch
		}
		if x := findNode(ch, pred); x != nil {
			return x
		}
	}
	return nil
}

func attr(n *xhtml.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func hasClass(n *xhtml.Node, class string) bool {
	for _, c := range strings.Fields(attr(n, "class")) {
		if c == class {
			return true
		}
	}
	return false
}
