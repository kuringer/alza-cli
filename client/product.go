package client

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"

	xhtml "golang.org/x/net/html"
)

type productDetailResponse struct {
	Data productDetailData `json:"data"`
}

type productDetailData struct {
	Name                      string                  `json:"name"`
	Price                     string                  `json:"price"`
	GaPrice                   float64                 `json:"gaPrice"`
	SalePercentage            *int                    `json:"salePercentage"`
	PriceInfoV2               *productPriceInfoV2     `json:"priceInfoV2"`
	PriceInfoV3               *productPriceInfoV3     `json:"priceInfoV3"`
	CashBackPriceLabel        string                  `json:"cashBackPriceLabel"`
	CashBackPrice             string                  `json:"cashBackPrice"`
	DescriptionBeforeDiscount string                  `json:"descriptionBeforeDiscount"`
	DescPageURL               string                  `json:"descPageUrl"`
	ParameterGroups           []productParameterGroup `json:"parameterGroups"`
	ProductVariantsInfo       *productVariantsInfo    `json:"productVariantsInfo"`
}

type productPriceInfoV2 struct {
	PriceWithVat     string              `json:"priceWithVat"`
	PriceWithoutVat  string              `json:"priceWithoutVat"`
	PriceNoCurrency  float64             `json:"priceNoCurrency"`
	UnitPriceWithVat string              `json:"unitPriceWithVat"`
	PromoPrices      []productPromoPrice `json:"promoPrices"`
}

type productPriceInfoV3 struct {
	PromoPrices  []productPromoPrice `json:"promoPrices"`
	MainPriceTag struct {
		PrimaryPrice           string  `json:"primaryPrice"`
		PrimaryPriceNoCurrency float64 `json:"primaryPriceNoCurrency"`
		SecondaryPrice         string  `json:"secondaryPrice"`
	} `json:"mainPriceTag"`
}

type productPromoPrice struct {
	Name               string  `json:"name"`
	FormattedPrice     string  `json:"formattedPrice"`
	PrimaryPrice       string  `json:"primaryPrice"`
	UnformattedPrice   float64 `json:"unformattedPrice"`
	DiscountCouponCode string  `json:"discountCouponCode"`
}

type productParameterGroup struct {
	Name   string                  `json:"name"`
	Params []productParameterEntry `json:"params"`
}

type productParameterEntry struct {
	Name   string                  `json:"name"`
	Values []productParameterValue `json:"values"`
}

type productParameterValue struct {
	Desc string `json:"desc"`
}

type productVariantsInfo struct {
	Type            int                 `json:"Type"`
	ProductVariants []productVariantRaw `json:"ProductVariants"`
}

func (p *productVariantsInfo) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}

	var probe map[string]json.RawMessage
	if err := json.Unmarshal(data, &probe); err != nil {
		return err
	}

	type upper struct {
		Type            int                 `json:"Type"`
		ProductVariants []productVariantRaw `json:"ProductVariants"`
	}
	type lower struct {
		Type            int                 `json:"type"`
		ProductVariants []productVariantRaw `json:"productVariants"`
	}

	if _, ok := probe["Type"]; ok {
		var u upper
		if err := json.Unmarshal(data, &u); err != nil {
			return err
		}
		p.Type = u.Type
		p.ProductVariants = u.ProductVariants
		return nil
	}
	if _, ok := probe["type"]; ok {
		var l lower
		if err := json.Unmarshal(data, &l); err != nil {
			return err
		}
		p.Type = l.Type
		p.ProductVariants = l.ProductVariants
		return nil
	}

	var fallback upper
	if err := json.Unmarshal(data, &fallback); err != nil {
		return err
	}
	p.Type = fallback.Type
	p.ProductVariants = fallback.ProductVariants
	return nil
}

type productVariantRaw struct {
	ID         int    `json:"Id"`
	Name       string `json:"Name"`
	ImageURL   string `json:"ImageUrl"`
	IsSelected bool   `json:"IsSelected"`
}

type productAvailabilityResponse struct {
	Title             string `json:"title"`
	Description       string `json:"description"`
	ExpectedStockDate string `json:"expectedStockDate"`
}

// GetProduct returns rich product info for a commodity ID.
func (c *TLSClient) GetProduct(productID int) (*ProductDetail, error) {
	endpoint := fmt.Sprintf(EndpointProductDetail, productID)
	data, err := c.Get(endpoint)
	if err != nil {
		return nil, err
	}

	var resp productDetailResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse product: %w", err)
	}

	detail := ProductDetail{
		ID:                  productID,
		Name:                resp.Data.Name,
		Price:               pickPrice(resp.Data),
		PriceWithoutVat:     pickPriceWithoutVat(resp.Data),
		PriceNoCurrency:     pickPriceNoCurrency(resp.Data),
		DiscountPercent:     resp.Data.SalePercentage,
		CashBackPriceLabel:  resp.Data.CashBackPriceLabel,
		CashBackPrice:       resp.Data.CashBackPrice,
		DiscountDescription: resp.Data.DescriptionBeforeDiscount,
		Parameters:          mapProductParameters(resp.Data.ParameterGroups),
		Variants:            mapProductVariants(resp.Data.ProductVariantsInfo),
		PromoPrices:         pickPromoPrices(resp.Data),
	}

	if resp.Data.DescPageURL != "" {
		descURL := normalizeExternalURL(resp.Data.DescPageURL)
		if descHTML, err := c.Get(descURL); err == nil {
			detail.Description = extractDescriptionFromHTML(string(descHTML))
		}
	}

	if availability, err := c.getProductAvailability(productID); err == nil {
		detail.Availability = availability.Title
		detail.AvailabilityDetail = availability.Description
		detail.ExpectedStockDate = availability.ExpectedStockDate
	}

	return &detail, nil
}

func (c *TLSClient) getProductAvailability(productID int) (*productAvailabilityResponse, error) {
	if c.userID == "" {
		_, _ = c.GetUserStatus()
	}

	var endpoint string
	if c.userID != "" {
		endpoint = fmt.Sprintf(EndpointProductAvailabilityUser, c.userID, productID)
	} else {
		endpoint = fmt.Sprintf(EndpointProductAvailabilityAnon, productID)
	}

	data, err := c.Get(endpoint)
	if err != nil {
		return nil, err
	}

	var resp productAvailabilityResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse availability: %w", err)
	}

	return &resp, nil
}

func pickPrice(data productDetailData) string {
	if data.PriceInfoV3 != nil && data.PriceInfoV3.MainPriceTag.PrimaryPrice != "" {
		return data.PriceInfoV3.MainPriceTag.PrimaryPrice
	}
	if data.PriceInfoV2 != nil && data.PriceInfoV2.PriceWithVat != "" {
		return data.PriceInfoV2.PriceWithVat
	}
	return data.Price
}

func pickPriceWithoutVat(data productDetailData) string {
	if data.PriceInfoV3 != nil && data.PriceInfoV3.MainPriceTag.SecondaryPrice != "" {
		return data.PriceInfoV3.MainPriceTag.SecondaryPrice
	}
	if data.PriceInfoV2 != nil {
		return data.PriceInfoV2.PriceWithoutVat
	}
	return ""
}

func pickPriceNoCurrency(data productDetailData) float64 {
	if data.PriceInfoV3 != nil && data.PriceInfoV3.MainPriceTag.PrimaryPriceNoCurrency > 0 {
		return data.PriceInfoV3.MainPriceTag.PrimaryPriceNoCurrency
	}
	if data.PriceInfoV2 != nil && data.PriceInfoV2.PriceNoCurrency > 0 {
		return data.PriceInfoV2.PriceNoCurrency
	}
	return data.GaPrice
}

func pickPromoPrices(data productDetailData) []ProductPromoPrice {
	var promos []ProductPromoPrice
	if data.PriceInfoV3 != nil && len(data.PriceInfoV3.PromoPrices) > 0 {
		for _, p := range data.PriceInfoV3.PromoPrices {
			promos = append(promos, ProductPromoPrice{
				Name:             p.Name,
				Price:            pickPromoPrice(p),
				Code:             p.DiscountCouponCode,
				UnformattedPrice: pickPromoPriceNoCurrency(p),
			})
		}
		return promos
	}
	if data.PriceInfoV2 != nil && len(data.PriceInfoV2.PromoPrices) > 0 {
		for _, p := range data.PriceInfoV2.PromoPrices {
			promos = append(promos, ProductPromoPrice{
				Name:             p.Name,
				Price:            pickPromoPrice(p),
				Code:             p.DiscountCouponCode,
				UnformattedPrice: pickPromoPriceNoCurrency(p),
			})
		}
	}
	return promos
}

func pickPromoPrice(p productPromoPrice) string {
	if p.PrimaryPrice != "" {
		return p.PrimaryPrice
	}
	return p.FormattedPrice
}

func pickPromoPriceNoCurrency(p productPromoPrice) float64 {
	if p.UnformattedPrice > 0 {
		return p.UnformattedPrice
	}
	return 0
}

func mapProductParameters(groups []productParameterGroup) []ProductParameterGroup {
	var out []ProductParameterGroup
	for _, g := range groups {
		if g.Name == "" {
			continue
		}
		group := ProductParameterGroup{Name: g.Name}
		for _, p := range g.Params {
			if p.Name == "" {
				continue
			}
			values := []string{}
			for _, v := range p.Values {
				if v.Desc != "" {
					values = append(values, v.Desc)
				}
			}
			if len(values) == 0 {
				continue
			}
			group.Parameters = append(group.Parameters, ProductParameter{
				Name:   p.Name,
				Values: values,
			})
		}
		if len(group.Parameters) > 0 {
			out = append(out, group)
		}
	}
	return out
}

func mapProductVariants(info *productVariantsInfo) []ProductVariant {
	if info == nil || len(info.ProductVariants) == 0 {
		return nil
	}
	var out []ProductVariant
	for _, v := range info.ProductVariants {
		out = append(out, ProductVariant{
			ID:         v.ID,
			Name:       v.Name,
			ImageURL:   v.ImageURL,
			IsSelected: v.IsSelected,
		})
	}
	return out
}

func extractDescriptionFromHTML(htmlStr string) string {
	if htmlStr == "" {
		return ""
	}

	doc, err := xhtml.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return ""
	}

	var parts []string
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		if n.Type == xhtml.ElementNode {
			switch n.Data {
			case "script", "style", "noscript":
				return
			case "h1", "h2", "h3", "p", "li":
				text := strings.TrimSpace(nodeText(n))
				text = collapseSpaces(html.UnescapeString(text))
				if text != "" && !looksLikeCookieNotice(text) {
					parts = append(parts, text)
				}
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return strings.Join(parts, "\n")
}

func normalizeExternalURL(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "//") {
		return "https:" + value
	}
	return value
}

func nodeText(n *xhtml.Node) string {
	if n.Type == xhtml.TextNode {
		return n.Data
	}
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		b.WriteString(nodeText(c))
	}
	return b.String()
}

func collapseSpaces(input string) string {
	return strings.Join(strings.Fields(input), " ")
}

func looksLikeCookieNotice(text string) bool {
	low := strings.ToLower(text)
	return strings.Contains(low, "cookie") || strings.Contains(low, "cookies")
}
