package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	// Open database with correct path
	db, err := sql.Open("sqlite", "./data/database.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Check current image URLs
	rows, err := db.Query("SELECT id, image_url FROM product_images LIMIT 10")
	if err != nil {
		log.Fatal("Failed to query product_images:", err)
	}
	defer rows.Close()

	fmt.Println("Current image URLs in database:")
	for rows.Next() {
		var id string
		var imageUrl string
		err := rows.Scan(&id, &imageUrl)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("ID: %s, URL: %s\n", id, imageUrl)
	}

	// Check for problematic paths
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM product_images WHERE image_url LIKE '/images/products/%'").Scan(&count)
	if err != nil {
		log.Fatal("Failed to count rows with old paths:", err)
	}

	fmt.Printf("\nFound %d images with old paths (/images/products/)\n", count)

	if count > 0 {
		fmt.Println("Fixing image paths...")
		// Update product image paths from /images/products/ to /public/images/products/
		_, err = db.Exec(`
			UPDATE product_images 
			SET image_url = '/public' || image_url 
			WHERE image_url LIKE '/images/products/%'
		`)
		if err != nil {
			log.Fatal("Failed to update product_images:", err)
		}

		fmt.Printf("Successfully updated %d image paths\n", count)
	}

	// Count final results
	err = db.QueryRow("SELECT COUNT(*) FROM product_images WHERE image_url LIKE '/public/images/products/%'").Scan(&count)
	if err != nil {
		log.Fatal("Failed to count updated rows:", err)
	}

	fmt.Printf("Total images with correct paths: %d\n", count)
}