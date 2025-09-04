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
	rows, err := db.Query("SELECT id, image_url FROM product_images LIMIT 5")
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

	// Count URLs that need fixing
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM product_images WHERE image_url LIKE '/public/images/products/%'").Scan(&count)
	if err != nil {
		log.Fatal("Failed to count rows with problematic paths:", err)
	}

	fmt.Printf("\nFound %d images with /public prefix that need fixing\n", count)

	if count > 0 {
		fmt.Println("Fixing image paths by removing /public prefix...")
		// Remove /public prefix from URLs since templates already add /public/images/products/
		_, err = db.Exec(`
			UPDATE product_images 
			SET image_url = REPLACE(image_url, '/public/images/products/', '') 
			WHERE image_url LIKE '/public/images/products/%'
		`)
		if err != nil {
			log.Fatal("Failed to update product_images:", err)
		}

		fmt.Printf("Successfully updated %d image paths\n", count)
	}

	// Verify the fix
	rows2, err := db.Query("SELECT id, image_url FROM product_images LIMIT 5")
	if err != nil {
		log.Fatal("Failed to query product_images:", err)
	}
	defer rows2.Close()

	fmt.Println("\nUpdated image URLs in database:")
	for rows2.Next() {
		var id string
		var imageUrl string
		err := rows2.Scan(&id, &imageUrl)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("ID: %s, URL: %s\n", id, imageUrl)
	}
}