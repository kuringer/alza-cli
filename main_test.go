package main

import "testing"

func TestRenderStars(t *testing.T) {
	tests := []struct {
		name   string
		rating float64
		want   string
	}{
		{"zero", 0, "☆☆☆☆☆"},
		{"one", 1, "★☆☆☆☆"},
		{"two", 2, "★★☆☆☆"},
		{"three", 3, "★★★☆☆"},
		{"four", 4, "★★★★☆"},
		{"five", 5, "★★★★★"},
		{"half star", 2.5, "★★½☆☆"},
		{"4.5 stars", 4.5, "★★★★½"},
		{"4.8 stars", 4.8, "★★★★½"},
		{"4.3 stars", 4.3, "★★★★☆"},
		{"negative clamped", -1, "☆☆☆☆☆"},
		{"over 5 clamped", 6, "★★★★★"},
		{"fractional low", 3.2, "★★★☆☆"},
		{"fractional high", 3.7, "★★★½☆"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderStars(tt.rating)
			if got != tt.want {
				t.Errorf("renderStars(%f) = %q, want %q", tt.rating, got, tt.want)
			}
		})
	}
}
