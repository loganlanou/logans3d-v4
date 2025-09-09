package main

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"

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

	// Count URLs that need fixing (those that contain paths instead of just filenames)
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM product_images WHERE image_url LIKE '%/%'").Scan(&count)
	if err != nil {
		log.Fatal("Failed to count rows with path URLs:", err)
	}

	fmt.Printf("\nFound %d images with path URLs that need to be converted to filenames\n", count)

	if count > 0 {
		fmt.Println("Converting image URLs to just filenames...")
		
		// Get all rows that need updating
		updateRows, err := db.Query("SELECT id, image_url FROM product_images WHERE image_url LIKE '%/%'")
		if err != nil {
			log.Fatal("Failed to query for update:", err)
		}
		defer updateRows.Close()

		updates := 0
		for updateRows.Next() {
			var id string
			var imageUrl string
			err := updateRows.Scan(&id, &imageUrl)
			if err != nil {
				log.Fatal(err)
			}

			// Extract just the filename from the URL
			filename := filepath.Base(imageUrl)
			
			// Update the database
			_, err = db.Exec("UPDATE product_images SET image_url = ? WHERE id = ?", filename, id)
			if err != nil {
				log.Printf("Failed to update row %s: %v", id, err)
				continue
			}
			
			fmt.Printf("Updated: %s -> %s\n", imageUrl, filename)
			updates++
		}

		fmt.Printf("Successfully updated %d image paths\n", updates)
	}

	// Verify the fix
	rows2, err := db.Query("SELECT id, image_url FROM product_images LIMIT 10")
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