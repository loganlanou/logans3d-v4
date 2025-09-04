package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	// Open database
	db, err := sql.Open("sqlite", "../db/logans3d.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Update product image paths from /images/products/ to /public/images/products/
	_, err = db.Exec(`
		UPDATE product_images 
		SET image_url = '/public' || image_url 
		WHERE image_url LIKE '/images/products/%'
	`)
	if err != nil {
		log.Fatal("Failed to update product_images:", err)
	}

	// Check results
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM product_images WHERE image_url LIKE '/public/images/products/%'").Scan(&count)
	if err != nil {
		log.Fatal("Failed to count updated rows:", err)
	}

	fmt.Printf("Successfully updated %d image paths\n", count)
}