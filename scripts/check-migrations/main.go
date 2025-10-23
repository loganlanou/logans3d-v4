package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "./data/database.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Check goose version table
	fmt.Println("=== Goose Migration Status ===")
	rows, err := db.Query("SELECT version_id, is_applied, tstamp FROM goose_db_version ORDER BY id")
	if err != nil {
		fmt.Printf("Error querying goose_db_version: %v\n", err)
		fmt.Println("Table might not exist yet")
	} else {
		defer rows.Close()
		for rows.Next() {
			var versionID int64
			var isApplied bool
			var tstamp string
			if err := rows.Scan(&versionID, &isApplied, &tstamp); err != nil {
				log.Fatal(err)
			}
			fmt.Printf("Version: %d, Applied: %v, Timestamp: %s\n", versionID, isApplied, tstamp)
		}
	}

	// Check if clerk_id column exists
	fmt.Println("\n=== Users Table Schema ===")
	rows, err = db.Query("PRAGMA table_info(users)")
	if err != nil {
		fmt.Printf("Error querying users table: %v\n", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var cid int
			var name string
			var typ string
			var notnull int
			var dfltValue sql.NullString
			var pk int
			if err := rows.Scan(&cid, &name, &typ, &notnull, &dfltValue, &pk); err != nil {
				log.Fatal(err)
			}
			fmt.Printf("Column: %s (%s)\n", name, typ)
		}
	}
}
