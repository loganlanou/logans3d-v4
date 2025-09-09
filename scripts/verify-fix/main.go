package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
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

	fmt.Println("Products and their image URLs (fixed):")
	fmt.Println("=====================================")
	
	successCount := 0
	totalCount := 0
	
	for rows.Next() {
		var name string
		var imageUrl sql.NullString
		err := rows.Scan(&name, &imageUrl)
		if err != nil {
			log.Fatal(err)
		}
		
		totalCount++
		
		if imageUrl.Valid {
			fmt.Printf("Product: %s\n", name)
			fmt.Printf("  Image URL (DB): %s\n", imageUrl.String)
			
			// Check if file exists on filesystem
			fullImagePath := filepath.Join("public", "images", "products", imageUrl.String)
			if _, err := os.Stat(fullImagePath); os.IsNotExist(err) {
				fmt.Printf("  ❌ FILE MISSING: %s\n", fullImagePath)
			} else {
				fmt.Printf("  ✅ FILE EXISTS: %s\n", fullImagePath)
				
				// Check if file is accessible via HTTP
				resp, err := http.Get(fmt.Sprintf("http://localhost:8000/public/images/products/%s", imageUrl.String))
				if err != nil {
					fmt.Printf("  ❌ HTTP ERROR: %v\n", err)
				} else if resp.StatusCode == 200 {
					fmt.Printf("  ✅ HTTP ACCESSIBLE: 200 OK\n")
					successCount++
				} else {
					fmt.Printf("  ❌ HTTP STATUS: %d\n", resp.StatusCode)
				}
				resp.Body.Close()
			}
		} else {
			fmt.Printf("Product: %s - NO IMAGE\n", name)
		}
		fmt.Println()
	}
	
	// Check event images
	fmt.Println("Event Images:")
	fmt.Println("=============")
	
	eventImages := []string{
		"event-maker-fair.jpg",
		"event-library-workshop.jpg", 
		"event-craft-show.jpg",
	}
	
	for _, img := range eventImages {
		fullPath := filepath.Join("public", "images", img)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			fmt.Printf("❌ FILE MISSING: %s\n", fullPath)
		} else {
			fmt.Printf("✅ FILE EXISTS: %s\n", fullPath)
			
			// Check HTTP accessibility
			resp, err := http.Get(fmt.Sprintf("http://localhost:8000/public/images/%s", img))
			if err != nil {
				fmt.Printf("❌ HTTP ERROR: %v\n", err)
			} else if resp.StatusCode == 200 {
				fmt.Printf("✅ HTTP ACCESSIBLE: 200 OK\n")
			} else {
				fmt.Printf("❌ HTTP STATUS: %d\n", resp.StatusCode)
			}
			resp.Body.Close()
		}
		fmt.Println()
	}
	
	fmt.Printf("SUMMARY: %d/%d product images working correctly\n", successCount, totalCount)
}