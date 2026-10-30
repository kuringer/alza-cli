package client

import (
	"regexp"
	"strconv"
	"strings"
)

var productIDRe = regexp.MustCompile(`(?i)(?:-d|/d)(\d+)\.htm`)

func extractProductID(rawURL string) int {
	if rawURL == "" {
		return 0
	}
	matches := productIDRe.FindStringSubmatch(rawURL)
	if len(matches) < 2 {
		return 0
	}
	id, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0
	}
	return id
}

func parsePrice(raw string) float64 {
	if raw == "" {
		return 0
	}
	var b strings.Builder
	for _, r := range raw {
		if (r >= '0' && r <= '9') || r == ',' || r == '.' {
			b.WriteRune(r)
		}
	}
	cleaned := strings.ReplaceAll(b.String(), ",", ".")
	if cleaned == "" {
		return 0
	}
	value, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0
	}
	return value
}

func snippet(body []byte, limit int) string {
	if limit <= 0 {
		return ""
	}
	trimmed := strings.TrimSpace(string(body))
	if len(trimmed) <= limit {
		return trimmed
	}
	return trimmed[:limit] + "..."
}
