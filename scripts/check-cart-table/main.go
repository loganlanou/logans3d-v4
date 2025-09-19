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

	// Query cart_items table
	rows, err := db.Query(`
		SELECT
			ci.id,
			ci.session_id,
			ci.quantity,
			p.name,
			COALESCE(pi.image_url, '') as image_url
		FROM cart_items ci
		JOIN products p ON ci.product_id = p.id
		LEFT JOIN product_images pi ON p.id = pi.product_id AND pi.is_primary = TRUE
		ORDER BY ci.created_at DESC
	`)
	if err != nil {
		log.Fatal("Failed to query cart_items:", err)
	}
	defer rows.Close()

	fmt.Println("Cart Items Table Data:")
	fmt.Println("======================")

	for rows.Next() {
		var id, sessionId sql.NullString
		var quantity int64
		var name, imageUrl string
		err := rows.Scan(&id, &sessionId, &quantity, &name, &imageUrl)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Cart Item ID: %s\n", id.String)
		fmt.Printf("Session ID: %s\n", sessionId.String)
		fmt.Printf("Product: %s\n", name)
		fmt.Printf("Quantity: %d\n", quantity)
		fmt.Printf("Image URL: '%s'\n", imageUrl)
		fmt.Println("---")
	}
}