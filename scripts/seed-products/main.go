package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

func main() {
	// Open database connection
	db, err := sql.Open("sqlite", "./data/database.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Read CSV file
	file, err := os.Open("./data/complete_products.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	// Skip header row
	if len(records) == 0 {
		log.Fatal("CSV file is empty")
	}
	records = records[1:]

	// Create categories first
	categories := make(map[string]string) // category name -> category id
	for _, record := range records {
		categoryName := record[5] // category column
		if _, exists := categories[categoryName]; !exists {
			categoryID := uuid.New().String()
			categories[categoryName] = categoryID

			slug := strings.ToLower(strings.ReplaceAll(categoryName, " ", "-"))
			
			_, err := db.ExecContext(ctx, `
				INSERT INTO categories (id, name, slug, created_at, updated_at) 
				VALUES (?, ?, ?, ?, ?)`,
				categoryID, categoryName, slug, time.Now(), time.Now())
			if err != nil {
				log.Printf("Error inserting category %s: %v", categoryName, err)
				continue
			}
			log.Printf("Created category: %s", categoryName)
		}
	}

	// Insert products
	for _, record := range records {
		if len(record) < 12 {
			log.Printf("Skipping malformed record: %v", record)
			continue
		}

		// Parse CSV fields
		csvID := record[0]
		name := record[1]
		description := record[2]
		_ = record[3] // priceMinStr - not used for now
		priceMaxStr := record[4]
		categoryName := record[5]
		imagePath := record[6]
		stockQuantityStr := record[7]
		featuredStr := record[8]

		// Convert price to cents (use max price for now)
		priceMax, err := strconv.ParseFloat(priceMaxStr, 64)
		if err != nil {
			log.Printf("Error parsing price for product %s: %v", name, err)
			continue
		}
		priceCents := int(priceMax * 100)

		// Parse stock quantity
		stockQuantity, err := strconv.Atoi(stockQuantityStr)
		if err != nil {
			log.Printf("Error parsing stock quantity for product %s: %v", name, err)
			stockQuantity = 0
		}

		// Parse featured flag
		featured := strings.ToLower(featuredStr) == "true"

		// Get category ID
		categoryID := categories[categoryName]

		// Generate product ID and slug
		productID := uuid.New().String()
		slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
		slug = strings.ReplaceAll(slug, "(", "")
		slug = strings.ReplaceAll(slug, ")", "")
		slug = strings.ReplaceAll(slug, "'", "")
		
		// Insert product
		_, err = db.ExecContext(ctx, `
			INSERT INTO products (
				id, name, slug, description, price_cents, category_id, 
				stock_quantity, is_active, is_featured, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			productID, name, slug, description, priceCents, categoryID,
			stockQuantity, true, featured, time.Now(), time.Now())
		if err != nil {
			log.Printf("Error inserting product %s: %v", name, err)
			continue
		}

		// Insert product image if provided
		if imagePath != "" {
			imageID := uuid.New().String()
			_, err = db.ExecContext(ctx, `
				INSERT INTO product_images (
					id, product_id, image_url, alt_text, display_order, is_primary, created_at
				) VALUES (?, ?, ?, ?, ?, ?, ?)`,
				imageID, productID, imagePath, name, 0, true, time.Now())
			if err != nil {
				log.Printf("Error inserting image for product %s: %v", name, err)
			}
		}

		log.Printf("Created product: %s (ID: %s)", name, csvID)
	}

	fmt.Println("Database seeding completed successfully!")
}