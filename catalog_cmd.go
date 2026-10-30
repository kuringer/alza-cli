package main

import (
	"fmt"
	"strings"
)

// === SEARCH ===

type SearchCmd struct {
	Query string `arg:"" help:"Search query"`
	Limit int    `help:"Max results to show" default:"10" short:"n"`
}

func (c *SearchCmd) Run(g *Globals) error {
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}

	results, err := cl.Search(c.Query, c.Limit)
	if err != nil {
		return err
	}

	if g.Format == "json" {
		outputJSON(results)
		return nil
	}

	if len(results) == 0 {
		fmt.Println("No results found")
		return nil
	}

	for i, r := range results {
		fmt.Printf("%d. [%d] %s\n", i+1, r.ID, r.Name)
		fmt.Printf("   Price: %s | %s\n", r.PriceStr, r.Availability)
		fmt.Printf("   %s\n\n", r.URL)
	}

	return nil
}

// === PRODUCT ===

type ProductCmd struct {
	ProductID int `arg:"" help:"Product ID"`
}

func (c *ProductCmd) Run(g *Globals) error {
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}

	product, err := cl.GetProduct(c.ProductID)
	if err != nil {
		return err
	}

	if g.Format == "json" {
		outputJSON(product)
		return nil
	}

	fmt.Printf("[%d] %s\n", product.ID, product.Name)

	if product.Price != "" {
		fmt.Printf("Price: %s", product.Price)
		if product.PriceWithoutVat != "" {
			fmt.Printf(" (bez DPH %s)", product.PriceWithoutVat)
		}
		if product.DiscountPercent != nil {
			fmt.Printf(" | Discount: %d%%", *product.DiscountPercent)
		}
		fmt.Println()
	}

	if len(product.PromoPrices) > 0 {
		for _, promo := range product.PromoPrices {
			line := fmt.Sprintf("Promo: %s", promo.Name)
			if promo.Price != "" {
				line += fmt.Sprintf(" (%s)", promo.Price)
			}
			if promo.Code != "" {
				line += fmt.Sprintf(" [code: %s]", promo.Code)
			}
			fmt.Println(line)
		}
	}

	if product.CashBackPriceLabel != "" || product.CashBackPrice != "" {
		line := "Promo"
		if product.CashBackPriceLabel != "" {
			line += fmt.Sprintf(": %s", product.CashBackPriceLabel)
		}
		if product.CashBackPrice != "" {
			line += fmt.Sprintf(" (%s)", product.CashBackPrice)
		}
		fmt.Println(line)
	}

	// Rating summary
	if product.ReviewStats != nil && product.ReviewStats.RatingCount > 0 {
		stars := renderStars(product.ReviewStats.RatingAverage)
		fmt.Printf("Rating: %s %.1f/5 (%d hodnotení", stars, product.ReviewStats.RatingAverage, product.ReviewStats.RatingCount)
		if product.ReviewStats.ReviewCount > 0 {
			fmt.Printf(", %d recenzií", product.ReviewStats.ReviewCount)
		}
		fmt.Printf(")\n")
		if product.ReviewStats.RecommendationRate > 0 {
			fmt.Printf("  %.0f%% odporúča", product.ReviewStats.RecommendationRate*100)
			if product.ReviewStats.PurchaseCount != "" {
				fmt.Printf(" | %s predaných", product.ReviewStats.PurchaseCount)
			}
			fmt.Println()
		}
	}

	if product.Availability != "" {
		fmt.Printf("Availability: %s\n", product.Availability)
	}
	if product.AvailabilityDetail != "" {
		fmt.Printf("  %s\n", product.AvailabilityDetail)
	}
	if product.ExpectedStockDate != "" {
		fmt.Printf("  %s\n", product.ExpectedStockDate)
	}

	if product.Description != "" {
		fmt.Println("\nDescription:")
		for _, line := range strings.Split(product.Description, "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			fmt.Printf("  %s\n", line)
		}
	}

	if len(product.Parameters) > 0 {
		fmt.Println("\nParameters:")
		for _, group := range product.Parameters {
			fmt.Printf("  %s:\n", group.Name)
			for _, param := range group.Parameters {
				fmt.Printf("    - %s: %s\n", param.Name, strings.Join(param.Values, ", "))
			}
		}
	}

	if len(product.Variants) > 0 {
		fmt.Println("\nVariants:")
		for _, variant := range product.Variants {
			marker := " "
			if variant.IsSelected {
				marker = "*"
			}
			fmt.Printf("  %s [%d] %s\n", marker, variant.ID, variant.Name)
		}
	}

	return nil
}

// === REVIEWS ===

type ReviewsCmd struct {
	ProductID int  `arg:"" help:"Product ID"`
	Limit     int  `help:"Number of reviews to show" default:"10" short:"n"`
	Offset    int  `help:"Skip first N reviews" default:"0"`
	Stats     bool `help:"Show only stats, no individual reviews" short:"s"`
}

func (c *ReviewsCmd) Run(g *Globals) error {
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}

	// Always fetch stats
	stats, err := cl.GetReviewStats(c.ProductID)
	if err != nil {
		return fmt.Errorf("failed to fetch review stats: %w", err)
	}

	if g.Format == "json" {
		if c.Stats {
			outputJSON(stats)
			return nil
		}
		// Fetch reviews for JSON output
		reviews, err := cl.GetReviews(c.ProductID, c.Offset, c.Limit)
		if err != nil {
			return err
		}
		outputJSON(map[string]interface{}{
			"stats":   stats,
			"reviews": reviews,
		})
		return nil
	}

	// Text output - stats header
	fmt.Printf("Reviews for product %d\n", c.ProductID)
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println()

	// Rating summary
	stars := renderStars(stats.RatingAverage)
	fmt.Printf("Rating: %s %.1f/5\n", stars, stats.RatingAverage)
	fmt.Printf("Total ratings: %d | Text reviews: %d\n", stats.RatingCount, stats.ReviewCount)
	if stats.RecommendationRate > 0 {
		fmt.Printf("Recommendation: %.0f%%", stats.RecommendationRate*100)
		if stats.PurchaseCount != "" {
			fmt.Printf(" | Sold: %s", stats.PurchaseCount)
		}
		fmt.Println()
	}

	// Rating distribution
	if len(stats.Ratings) > 0 {
		fmt.Println("\nRating distribution:")
		// Build a map for quick lookup
		ratingMap := make(map[int]int)
		maxCount := 0
		for _, r := range stats.Ratings {
			ratingMap[r.Value] = r.Count
			if r.Count > maxCount {
				maxCount = r.Count
			}
		}
		// Show from 5 stars down to 1
		for star := 5; star >= 1; star-- {
			count := ratingMap[star]
			barLen := 0
			if maxCount > 0 {
				barLen = (count * 30) / maxCount
			}
			bar := strings.Repeat("█", barLen) + strings.Repeat("░", 30-barLen)
			fmt.Printf("  %d★ [%s] %d\n", star, bar, count)
		}
	}

	// Complaint/reliability info
	if stats.Complaint != nil {
		fmt.Printf("\nReliability: %s (%.2f%%)\n", stats.Complaint.Description, stats.Complaint.Rate*100)
	}

	if c.Stats {
		return nil
	}

	// Fetch and display individual reviews
	reviews, err := cl.GetReviews(c.ProductID, c.Offset, c.Limit)
	if err != nil {
		return err
	}

	if len(reviews.Reviews) == 0 {
		fmt.Println("\nNo text reviews available.")
		return nil
	}

	fmt.Printf("\nReviews (showing %d-%d of %d):\n", c.Offset+1, c.Offset+len(reviews.Reviews), reviews.TotalCount)
	fmt.Println(strings.Repeat("-", 50))

	for i, review := range reviews.Reviews {
		fmt.Println()
		// Header: rating, name, date
		stars := renderStars(float64(review.Rating))
		verified := ""
		if review.IsVerified {
			verified = " [Overený nákup]"
		}
		translated := ""
		if review.IsTranslated {
			translated = " [Preložené]"
		}
		fmt.Printf("%d. %s %d/5 - %s%s%s\n", c.Offset+i+1, stars, review.Rating, review.Name, verified, translated)

		// Date
		if review.ReviewDate != "" {
			date := review.ReviewDate
			if len(date) > 10 {
				date = date[:10] // Just YYYY-MM-DD
			}
			fmt.Printf("   %s", date)
			if review.LikeCount > 0 {
				fmt.Printf(" | %d likes", review.LikeCount)
			}
			fmt.Println()
		}

		// Positives/Negatives
		if len(review.Positives) > 0 {
			fmt.Printf("   + %s\n", strings.Join(review.Positives, ", "))
		}
		if len(review.Negatives) > 0 {
			fmt.Printf("   - %s\n", strings.Join(review.Negatives, ", "))
		}

		// Description text
		if review.Description != "" {
			// Wrap long descriptions
			desc := review.Description
			if len(desc) > 200 {
				desc = desc[:200] + "..."
			}
			fmt.Printf("   \"%s\"\n", desc)
		}
	}

	// Pagination hint
	if c.Offset+c.Limit < reviews.TotalCount {
		fmt.Printf("\n... ďalšie recenzie: alza reviews %d --offset %d\n", c.ProductID, c.Offset+c.Limit)
	}

	return nil
}

// renderStars converts a rating (0-5) to star characters
func renderStars(rating float64) string {
	// Clamp to valid range
	if rating < 0 {
		rating = 0
	}
	if rating > 5 {
		rating = 5
	}

	full := int(rating)
	half := rating-float64(full) >= 0.5
	empty := 5 - full
	if half {
		empty--
	}
	if empty < 0 {
		empty = 0
	}

	result := strings.Repeat("★", full)
	if half {
		result += "½"
	}
	result += strings.Repeat("☆", empty)
	return result
}
