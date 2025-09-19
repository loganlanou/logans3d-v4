package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"math"

	_ "modernc.org/sqlite"

	"github.com/loganlanou/logans3d-v4/storage/db"
)

func main() {
	// Parse command line flags
	var dbPath string
	flag.StringVar(&dbPath, "db", "./data/database.db", "Path to the database file")
	flag.Parse()

	// Open database connection
	database, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	// Enable foreign keys
	if _, err := database.Exec("PRAGMA foreign_keys = ON"); err != nil {
		log.Fatal(err)
	}

	// Create queries object using SQLC generated code
	queries := db.New(database)
	ctx := context.Background()

	// Get all products
	products, err := queries.ListProducts(ctx)
	if err != nil {
		log.Fatal(err)
	}

	var updatedCount int

	// Update each product's price to nearest dollar
	for _, product := range products {
		// Convert cents to dollars, round, then convert back to cents
		priceInDollars := float64(product.PriceCents) / 100.0
		roundedDollars := math.Round(priceInDollars)
		newPriceCents := int64(roundedDollars * 100)

		// Only update if the price changed
		if newPriceCents != product.PriceCents {
			oldPrice := float64(product.PriceCents) / 100.0
			newPrice := float64(newPriceCents) / 100.0

			// Update the product price
			err := queries.UpdateProductPrice(ctx, db.UpdateProductPriceParams{
				PriceCents: newPriceCents,
				ID:         product.ID,
			})
			if err != nil {
				log.Printf("Error updating price for product %s: %v", product.Name, err)
				continue
			}

			fmt.Printf("Updated %s: $%.2f -> $%.2f\n", product.Name, oldPrice, newPrice)
			updatedCount++
		}
	}

	fmt.Printf("\nPrice rounding completed successfully!\n")
	fmt.Printf("Updated %d products\n", updatedCount)
}