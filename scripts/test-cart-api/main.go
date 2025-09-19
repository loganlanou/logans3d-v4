package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/loganlanou/logans3d-v4/storage/db"
	_ "modernc.org/sqlite"
)

func main() {
	// Open database connection
	database, err := sql.Open("sqlite", "./data/database.db")
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	// Create queries instance
	queries := db.New(database)

	// Get a cart session that has Bone Dragons
	ctx := context.Background()

	// First, find a session with Bone Dragons
	rows, err := database.Query(`
		SELECT DISTINCT ci.session_id
		FROM cart_items ci
		JOIN products p ON ci.product_id = p.id
		WHERE p.name = 'Bone Dragons (no wings)'
		LIMIT 1
	`)
	if err != nil {
		log.Fatal("Failed to find session:", err)
	}
	defer rows.Close()

	var sessionID string
	if rows.Next() {
		var sessionIDNull sql.NullString
		err := rows.Scan(&sessionIDNull)
		if err != nil {
			log.Fatal(err)
		}
		sessionID = sessionIDNull.String
	} else {
		log.Fatal("No cart with Bone Dragons found")
	}

	fmt.Printf("Testing session: %s\n", sessionID)
	fmt.Println("===============================")

	// Now get the cart data like the API does
	items, err := queries.GetCartBySession(ctx, sql.NullString{String: sessionID, Valid: true})
	if err != nil {
		log.Fatal("Failed to get cart items:", err)
	}

	// Simulate the API response format
	response := map[string]interface{}{
		"items":       items,
		"totalCents":  0,
		"totalDollar": 0,
	}

	// Print the response as JSON
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		log.Fatal("Failed to marshal JSON:", err)
	}

	fmt.Println("API Response:")
	fmt.Println(string(jsonData))
}