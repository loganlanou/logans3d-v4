package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	// Open database
	db, err := sql.Open("sqlite", "./data/database.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Query product_images table
	rows, err := db.Query(`
		SELECT p.name, pi.image_url, pi.is_primary
		FROM products p
		LEFT JOIN product_images pi ON p.id = pi.product_id
		ORDER BY p.name
		LIMIT 10
	`)
	if err != nil {
		log.Fatal("Failed to query product_images:", err)
	}
	defer rows.Close()

	fmt.Println("Product Images Table Data:")
	fmt.Println("==========================")

	for rows.Next() {
		var name string
		var imageUrl sql.NullString
		var isPrimary sql.NullBool
		err := rows.Scan(&name, &imageUrl, &isPrimary)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Product: %s\n", name)
		if imageUrl.Valid {
			fmt.Printf("  Image URL: '%s'\n", imageUrl.String)
			fmt.Printf("  Is Primary: %v\n", isPrimary.Bool)
		} else {
			fmt.Printf("  Image URL: NULL\n")
		}
		fmt.Println()
	}
}