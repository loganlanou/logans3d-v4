package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/database.db"
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	email := "logan@lanou.com"

	// Update the user to be an admin
	result, err := db.Exec("UPDATE users SET is_admin = TRUE WHERE email = ?", email)
	if err != nil {
		log.Fatalf("Failed to update user: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Fatalf("Failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		fmt.Printf("No user found with email: %s\n", email)
		fmt.Println("User may need to log in first to be synced from Clerk.")
	} else {
		fmt.Printf("Successfully made %s an admin!\n", email)
	}
}
