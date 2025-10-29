package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"

	"github.com/loganlanou/logans3d-v4/storage/db"
)

func main() {
	// Open database connection
	database, err := sql.Open("sqlite", "./data/database.db")
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	// Create queries object
	queries := db.New(database)
	ctx := context.Background()

	// Create a category first
	categoryID := uuid.New().String()
	category, err := queries.UpsertCategory(ctx, db.UpsertCategoryParams{
		ID:   categoryID,
		Name: "Test Products",
		Slug: "test-products",
	})
	if err != nil {
		log.Fatal("Error creating category:", err)
	}
	fmt.Printf("Created category: %s\n", category.Name)

	// Sample products with various prices to test rounding
	products := []struct {
		name       string
		priceCents int64
		slug       string
	}{
		{"Test Dragon Model", 2999, "test-dragon-model"}, // $29.99 -> $30.00
		{"Tiny Toothless", 499, "tiny-toothless"},        // $4.99 -> $5.00
		{"Crystal Dragon", 12500, "crystal-dragon"},      // $125.00 -> $125.00 (no change)
		{"Sample Fidget", 751, "sample-fidget"},          // $7.51 -> $8.00
		{"Demo Product", 1234, "demo-product"},           // $12.34 -> $12.00
	}

	for _, p := range products {
		productID := uuid.New().String()

		product, err := queries.UpsertProduct(ctx, db.UpsertProductParams{
			ID:            productID,
			Name:          p.name,
			Slug:          p.slug,
			Description:   sql.NullString{String: "Sample product for testing price rounding", Valid: true},
			PriceCents:    p.priceCents,
			CategoryID:    sql.NullString{String: categoryID, Valid: true},
			StockQuantity: sql.NullInt64{Int64: 10, Valid: true},
			IsFeatured:    sql.NullBool{Bool: false, Valid: true},
		})
		if err != nil {
			log.Printf("Error creating product %s: %v", p.name, err)
			continue
		}

		price := float64(p.priceCents) / 100.0
		fmt.Printf("Created product: %s - $%.2f\n", product.Name, price)
	}

	fmt.Println("\nSample products created successfully!")
}
