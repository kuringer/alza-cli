package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"

	http "github.com/bogdanfinn/fhttp"
)

func (c *TLSClient) Search(query string, limit int) ([]SearchResult, error) {
	results, err := c.searchV5(query, limit)
	if err == nil && len(results) > 0 {
		return results, nil
	}

	if c.debug {
		if err != nil {
			fmt.Printf("[DEBUG] Search v5 failed, falling back to whisperer: %v\n", err)
		} else {
			fmt.Printf("[DEBUG] Search v5 returned no items, falling back to whisperer\n")
		}
	}

	fallback, fallbackErr := c.searchWhisper(query, limit)
	if fallbackErr != nil {
		if err != nil {
			return nil, fmt.Errorf("search v5 failed: %w; whisperer failed: %v", err, fallbackErr)
		}
		return nil, fallbackErr
	}
	if len(fallback) > 0 {
		return fallback, nil
	}
	if err != nil {
		return nil, err
	}
	return fallback, nil
}

func (c *TLSClient) searchV5(query string, limit int) ([]SearchResult, error) {
	payload := map[string]string{
		"searchTerm": query,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to build search payload: %w", err)
	}

	data, err := c.Post(EndpointSearchService, string(body))
	if err != nil {
		return nil, err
	}

	var searchResp struct {
		Data2 []struct {
			ID              int     `json:"id"`
			Name            string  `json:"name"`
			Code            string  `json:"code"`
			Price           string  `json:"price"`
			PriceNoCurrency float64 `json:"priceNoCurrency"`
			Avail           string  `json:"avail"`
			Img             string  `json:"img"`
			URL             string  `json:"url"`
		} `json:"data2"`
	}

	if err := json.Unmarshal(data, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse search: %w", err)
	}

	results := make([]SearchResult, 0, min(len(searchResp.Data2), limit))
	for _, item := range searchResp.Data2 {
		price := item.PriceNoCurrency
		if price == 0 {
			price = parsePrice(item.Price)
		}
		results = append(results, SearchResult{
			ID:           item.ID,
			Name:         item.Name,
			Code:         item.Code,
			Price:        price,
			PriceStr:     item.Price,
			Availability: item.Avail,
			ImageURL:     item.Img,
			URL:          item.URL,
		})
		if len(results) >= limit {
			break
		}
	}
	return results, nil
}

func (c *TLSClient) searchWhisper(query string, limit int) ([]SearchResult, error) {
	endpoint := EndpointWhisperAnon
	if c.userID != "" {
		endpoint = fmt.Sprintf(EndpointWhisperUser, c.userID)
	}

	params := url.Values{}
	params.Set("country", "SK")
	params.Set("eshopUrl", "https://www.alza.sk/")
	params.Set("searchTerm", query)
	params.Set("visitor", "00000000-0000-0000-0000-000000000000")
	searchURL := fmt.Sprintf("%s?%s", endpoint, params.Encode())

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	if c.debug {
		fmt.Printf("[DEBUG] GET %s\n", searchURL)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if c.debug {
		fmt.Printf("[DEBUG] Search whisper response: %d %s\n", resp.StatusCode, string(data)[:min(300, len(data))])
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(data)[:min(500, len(data))])
	}

	var searchResp struct {
		Commodities []struct {
			ImageURL    string `json:"imageUrl"`
			ClickAction struct {
				Name    string `json:"name"`
				WebLink string `json:"webLink"`
				Href    string `json:"href"`
			} `json:"clickAction"`
		} `json:"commodities"`
	}

	if err := json.Unmarshal(data, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse whisper search: %w", err)
	}

	results := make([]SearchResult, 0, min(len(searchResp.Commodities), limit))
	for _, item := range searchResp.Commodities {
		link := item.ClickAction.WebLink
		if link == "" {
			link = item.ClickAction.Href
		}
		id := extractProductID(link)
		results = append(results, SearchResult{
			ID:       id,
			Name:     item.ClickAction.Name,
			ImageURL: item.ImageURL,
			URL:      link,
		})
		if len(results) >= limit {
			break
		}
	}
	return results, nil
}
