package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/loganlanou/logans3d-v4/storage/db"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/database.db"
	}

	database, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer database.Close()

	queries := db.New(database)
	ctx := context.Background()

	// Get all products
	products, err := queries.ListProducts(ctx)
	if err != nil {
		log.Fatal("Failed to list products:", err)
	}

	fmt.Printf("Found %d products to categorize\n", len(products))

	// Categorization logic based on price (in cents)
	// Small: $0-$15 (0-1500 cents)
	// Medium: $15-$30 (1500-3000 cents)
	// Large: $30-$50 (3000-5000 cents)
	// XLarge: $50+ (5000+ cents)

	for _, product := range products {
		category := determineShippingCategory(product)

		// Update the product's shipping category
		_, err := queries.UpdateProductShippingCategory(ctx, db.UpdateProductShippingCategoryParams{
			ID:               product.ID,
			ShippingCategory: sql.NullString{String: category, Valid: true},
		})
		if err != nil {
			log.Printf("Failed to update product %s (%s): %v", product.Name, product.ID, err)
			continue
		}

		fmt.Printf("Updated %s (ID: %s, Price: $%.2f) -> %s\n",
			product.Name,
			product.ID,
			float64(product.PriceCents)/100.0,
			category,
		)
	}

	fmt.Println("\nProduct categorization complete!")

	// Display summary
	counts, err := queries.CountCartItemsByShippingCategory(ctx, db.CountCartItemsByShippingCategoryParams{
		SessionID: sql.NullString{Valid: false},
		UserID:    sql.NullString{Valid: false},
	})
	if err != nil {
		log.Printf("Warning: Could not get cart summary: %v", err)
	} else {
		fmt.Printf("\nCurrent cart breakdown:\n")
		fmt.Printf("  Small: %.0f items\n", counts.SmallItems.Float64)
		fmt.Printf("  Medium: %.0f items\n", counts.MediumItems.Float64)
		fmt.Printf("  Large: %.0f items\n", counts.LargeItems.Float64)
		fmt.Printf("  XLarge: %.0f items\n", counts.XlargeItems.Float64)
	}
}

func determineShippingCategory(product db.Product) string {
	price := product.PriceCents

	// Use weight if available as a secondary signal
	// For 3D prints, weight is a good indicator of size
	if product.WeightGrams.Valid {
		weight := product.WeightGrams.Int64

		// Weight-based categorization (in grams)
		// Small: 0-100g
		// Medium: 100-250g
		// Large: 250-600g
		// XLarge: 600g+
		if weight < 100 {
			return "small"
		} else if weight < 250 {
			return "medium"
		} else if weight < 600 {
			return "large"
		} else {
			return "xlarge"
		}
	}

	// Fallback to price-based categorization
	if price < 1500 {
		return "small"
	} else if price < 3000 {
		return "medium"
	} else if price < 5000 {
		return "large"
	} else {
		return "xlarge"
	}
}