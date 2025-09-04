package main

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

func main() {
	// Open database (using relative path since we'll run from project root)
	db, err := sql.Open("sqlite", "db/logans3d.db?_pragma=busy_timeout(5000)")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Get all product images
	rows, err := db.Query("SELECT id, image_url FROM product_images")
	if err != nil {
		log.Fatal("Failed to query product_images:", err)
	}
	defer rows.Close()

	updates := []struct {
		id       string
		filename string
	}{}

	for rows.Next() {
		var id, imageURL string
		if err := rows.Scan(&id, &imageURL); err != nil {
			log.Fatal("Failed to scan row:", err)
		}

		// Extract just the filename from paths like "/images/products/file.jpeg"
		filename := filepath.Base(imageURL)
		if strings.HasPrefix(imageURL, "/images/products/") && filename != imageURL {
			updates = append(updates, struct {
				id       string
				filename string
			}{id, filename})
		}
	}

	fmt.Printf("Found %d image URLs to update\n", len(updates))

	// Update each image URL to just the filename
	for _, update := range updates {
		_, err = db.Exec("UPDATE product_images SET image_url = ? WHERE id = ?", update.filename, update.id)
		if err != nil {
			log.Printf("Failed to update image %s: %v", update.id, err)
		} else {
			fmt.Printf("Updated %s to %s\n", update.id, update.filename)
		}
	}

	fmt.Println("Image URL update completed!")
}