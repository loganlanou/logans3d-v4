package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

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

	// Get all products
	products, err := queries.ListProducts(ctx)
	if err != nil {
		log.Fatal(err)
	}

	if len(products) == 0 {
		fmt.Println("No products found in database")
		return
	}

	fmt.Printf("Found %d products:\n", len(products))

	for _, product := range products {
		priceInDollars := float64(product.PriceCents) / 100.0
		fmt.Printf("- %s: $%.2f (%d cents)\n", product.Name, priceInDollars, product.PriceCents)
	}
}