package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"time"

	_ "modernc.org/sqlite"
)

func main() {
	// Parse command line flags
	dbPath := flag.String("db", "./data/database.db", "Path to the database file")
	flag.Parse()

	// Open database connection with same settings as main app
	db, err := sql.Open("sqlite", *dbPath+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Ensure connection is actually established
	log.Println("Testing database connection...")
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("Database connection successful")

	// Update all @lanou.com users to be admins with retry logic
	var result sql.Result
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		var err error
		result, err = db.Exec(`
			UPDATE users
			SET is_admin = TRUE
			WHERE email LIKE '%@lanou.com'
		`)
		if err == nil {
			break
		}
		if i == maxRetries-1 {
			log.Fatalf("Failed to update users after %d retries: %v", maxRetries, err)
		}
		waitTime := time.Duration(i+1) * 200 * time.Millisecond
		log.Printf("Database busy, retrying in %v... (attempt %d/%d)", waitTime, i+1, maxRetries)
		time.Sleep(waitTime)
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
