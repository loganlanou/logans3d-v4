package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
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

	// Track stats
	var categoriesCreated, categoriesUpdated int
	var productsCreated, productsUpdated int

	// Process categories first
	categories := make(map[string]string) // category name -> category id
	processedCategories := make(map[string]bool)

	for _, record := range records {
		if len(record) < 9 {
			continue
		}
		categoryName := strings.TrimSpace(record[5]) // category column

		if processedCategories[categoryName] {
			continue
		}
		processedCategories[categoryName] = true

		slug := strings.ToLower(strings.ReplaceAll(categoryName, " ", "-"))

		// Check if category exists
		existingCategory, err := queries.GetCategoryByName(ctx, categoryName)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("Error checking category %s: %v", categoryName, err)
			continue
		}

		if err == sql.ErrNoRows {
			// Category doesn't exist, create it
			categoryID := uuid.New().String()
			category, err := queries.UpsertCategory(ctx, db.UpsertCategoryParams{
				ID:   categoryID,
				Name: categoryName,
				Slug: slug,
			})
			if err != nil {
				log.Printf("Error creating category %s: %v", categoryName, err)
				continue
			}
			categories[categoryName] = category.ID
			categoriesCreated++
			log.Printf("Created category: %s", categoryName)
		} else {
			// Category exists
			categories[categoryName] = existingCategory.ID
			// Update it to refresh updated_at timestamp
			_, err = queries.UpsertCategory(ctx, db.UpsertCategoryParams{
				ID:   existingCategory.ID,
				Name: categoryName,
				Slug: slug,
			})
			if err != nil {
				log.Printf("Error updating category %s: %v", categoryName, err)
			}
			categoriesUpdated++
		}
	}

	// Process products
	for _, record := range records {
		if len(record) < 9 {
			log.Printf("Skipping malformed record: %v", record)
			continue
		}

		// Parse CSV fields
		csvID := record[0]
		name := strings.TrimSpace(record[1])
		description := strings.TrimSpace(record[2])
		// record[3] is price_min, not used for now
		priceMaxStr := strings.TrimSpace(record[4])
		categoryName := strings.TrimSpace(record[5])
		imagePath := strings.TrimSpace(record[6])
		stockQuantityStr := strings.TrimSpace(record[7])
		featuredStr := strings.TrimSpace(record[8])

		// Convert price to cents
		priceMax, err := strconv.ParseFloat(priceMaxStr, 64)
		if err != nil {
			log.Printf("Error parsing price for product %s: %v", name, err)
			continue
		}
		priceCents := int64(priceMax * 100)

		// Parse stock quantity
		stockQuantity := int64(0)
		if stockQuantityStr != "" {
			qty, err := strconv.ParseInt(stockQuantityStr, 10, 64)
			if err != nil {
				log.Printf("Error parsing stock quantity for product %s: %v", name, err)
			} else {
				stockQuantity = qty
			}
		}

		// Parse featured flag
		featured := strings.ToLower(featuredStr) == "true"

		// Get category ID
		categoryID, ok := categories[categoryName]
		if !ok {
			log.Printf("Category not found for product %s: %s", name, categoryName)
			continue
		}

		// Generate slug
		slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
		slug = strings.ReplaceAll(slug, "(", "")
		slug = strings.ReplaceAll(slug, ")", "")
		slug = strings.ReplaceAll(slug, "'", "")
		slug = strings.ReplaceAll(slug, ",", "")

		// Check if product exists
		existingProduct, err := queries.GetProductByName(ctx, name)
		isNew := err == sql.ErrNoRows

		var productID string
		if isNew {
			productID = uuid.New().String()
			productsCreated++
			log.Printf("Created product: %s (CSV ID: %s)", name, csvID)
		} else {
			productID = existingProduct.ID
			productsUpdated++
			log.Printf("Updated product: %s (CSV ID: %s)", name, csvID)
		}

		// Upsert the product
		product, err := queries.UpsertProduct(ctx, db.UpsertProductParams{
			ID:            productID,
			Name:          name,
			Slug:          slug,
			Description:   sql.NullString{String: description, Valid: description != ""},
			PriceCents:    priceCents,
			CategoryID:    sql.NullString{String: categoryID, Valid: true},
			StockQuantity: sql.NullInt64{Int64: stockQuantity, Valid: true},
			IsFeatured:    sql.NullBool{Bool: featured, Valid: true},
		})
		if err != nil {
			log.Printf("Error upserting product %s: %v", name, err)
			continue
		}

		// Handle product image
		if imagePath != "" {
			// Check if primary image exists
			existingImage, err := queries.GetPrimaryProductImage(ctx, product.ID)

			switch err {
			case sql.ErrNoRows:
				// No primary image exists, create one
				imageID := uuid.New().String()
				_, err = queries.CreateProductImage(ctx, db.CreateProductImageParams{
					ID:           imageID,
					ProductID:    product.ID,
					ImageUrl:     imagePath,
					AltText:      sql.NullString{String: name, Valid: true},
					DisplayOrder: sql.NullInt64{Int64: 0, Valid: true},
					IsPrimary:    sql.NullBool{Bool: true, Valid: true},
				})
				if err != nil {
					log.Printf("Error creating image for product %s: %v", name, err)
				}
			case nil:
				// Primary image exists, update it if different
				if existingImage.ImageUrl != imagePath {
					err = queries.UpdateProductImage(ctx, db.UpdateProductImageParams{
						ImageUrl: imagePath,
						AltText:  sql.NullString{String: name, Valid: true},
						ID:       existingImage.ID,
					})
					if err != nil {
						log.Printf("Error updating image for product %s: %v", name, err)
					}
				}
			default:
				log.Printf("Error checking image for product %s: %v", name, err)
			}
		}
	}

	fmt.Println("\nDatabase seeding completed successfully!")
	fmt.Printf("Categories: %d created, %d updated\n", categoriesCreated, categoriesUpdated)
	fmt.Printf("Products: %d created, %d updated\n", productsCreated, productsUpdated)
}
