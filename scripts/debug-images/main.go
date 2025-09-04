package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
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

	// Query products with their images
	rows, err := db.Query(`
		SELECT p.name, pi.image_url 
		FROM products p 
		LEFT JOIN product_images pi ON p.id = pi.product_id 
		ORDER BY p.name 
		LIMIT 10
	`)
	if err != nil {
		log.Fatal("Failed to query products:", err)
	}
	defer rows.Close()

	fmt.Println("Products and their image URLs:")
	fmt.Println("=====================================")
	
	for rows.Next() {
		var name string
		var imageUrl sql.NullString
		err := rows.Scan(&name, &imageUrl)
		if err != nil {
			log.Fatal(err)
		}
		
		if imageUrl.Valid {
			fmt.Printf("Product: %s\n", name)
			fmt.Printf("  Image URL: %s\n", imageUrl.String)
			
			// Check if file exists
			imagePath := "." + imageUrl.String // Remove leading / and add .
			if _, err := os.Stat(imagePath); os.IsNotExist(err) {
				fmt.Printf("  ❌ FILE MISSING: %s\n", imagePath)
			} else {
				fmt.Printf("  ✅ FILE EXISTS: %s\n", imagePath)
			}
		} else {
			fmt.Printf("Product: %s - NO IMAGE\n", name)
		}
		fmt.Println()
	}
	
	// Check what image files actually exist
	fmt.Println("\nActual image files in public/images/products:")
	fmt.Println("=============================================")
	
	err = filepath.Walk("./public/images/products", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fmt.Printf("Found: %s\n", path)
		}
		return nil
	})
	
	if err != nil {
		log.Printf("Error walking directory: %v", err)
	}
}