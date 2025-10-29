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

	// Query cart_items with Bone Dragons specifically
	rows, err := db.Query(`
		SELECT
			ci.id,
			ci.session_id,
			ci.quantity,
			p.name,
			pi.image_url,
			length(pi.image_url) as url_length
		FROM cart_items ci
		JOIN products p ON ci.product_id = p.id
		LEFT JOIN product_images pi ON p.id = pi.product_id AND pi.is_primary = TRUE
		WHERE p.name = 'Bone Dragons (no wings)'
		ORDER BY ci.created_at DESC
		LIMIT 3
	`)
	if err != nil {
		log.Fatal("Failed to query cart_items:", err)
	}
	defer rows.Close()

	fmt.Println("Bone Dragons Cart Items Debug:")
	fmt.Println("==============================")

	for rows.Next() {
		var id, sessionId sql.NullString
		var quantity int64
		var name, imageUrl string
		var urlLength int
		err := rows.Scan(&id, &sessionId, &quantity, &name, &imageUrl, &urlLength)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Cart Item ID: %s\n", id.String)
		fmt.Printf("Session ID: %s\n", sessionId.String)
		fmt.Printf("Product: %s\n", name)
		fmt.Printf("Quantity: %d\n", quantity)
		fmt.Printf("Image URL: '%s'\n", imageUrl)
		fmt.Printf("URL Length: %d\n", urlLength)
		fmt.Printf("Bytes: %v\n", []byte(imageUrl))
		fmt.Println("---")
	}
}
