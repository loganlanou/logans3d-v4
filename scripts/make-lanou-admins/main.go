package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Get database path from environment or use default
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/database.db"
	}

	// Open database connection
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Update all @lanou.com users to be admins
	result, err := db.Exec(`
		UPDATE users
		SET is_admin = TRUE
		WHERE email LIKE '%@lanou.com'
	`)
	if err != nil {
		log.Fatalf("Failed to update users: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Fatalf("Failed to get rows affected: %v", err)
	}

	fmt.Printf("✓ Updated %d user(s) with @lanou.com email to admin status\n", rowsAffected)

	// Display all admin users
	rows, err := db.Query(`
		SELECT email, is_admin
		FROM users
		WHERE email LIKE '%@lanou.com'
		ORDER BY email
	`)
	if err != nil {
		log.Fatalf("Failed to query users: %v", err)
	}
	defer rows.Close()

	fmt.Println("\nAdmin users with @lanou.com domain:")
	for rows.Next() {
		var email string
		var isAdmin bool
		if err := rows.Scan(&email, &isAdmin); err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}
		adminStatus := "No"
		if isAdmin {
			adminStatus = "Yes"
		}
		fmt.Printf("  %s - Admin: %s\n", email, adminStatus)
	}

	if err := rows.Err(); err != nil {
		log.Fatalf("Error iterating rows: %v", err)
	}
}
