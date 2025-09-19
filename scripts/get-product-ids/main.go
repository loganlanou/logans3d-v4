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

	// Query product IDs
	rows, err := db.Query(`SELECT id, name FROM products LIMIT 5`)
	if err != nil {
		log.Fatal("Failed to query products:", err)
	}
	defer rows.Close()

	fmt.Println("Available Product IDs:")
	fmt.Println("=====================")

	for rows.Next() {
		var id, name string
		err := rows.Scan(&id, &name)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("ID: %s, Name: %s\n", id, name)
	}
}