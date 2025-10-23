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

	// Delete migration 005 and 006 from goose version table so they can run again
	fmt.Println("Resetting migrations 005 and 006...")

	_, err = db.Exec("DELETE FROM goose_db_version WHERE version_id IN (5, 6)")
	if err != nil {
		log.Fatalf("Failed to reset migrations: %v", err)
	}

	fmt.Println("✅ Migrations 005 and 006 have been reset and will run on next server start")

	// Verify
	rows, err := db.Query("SELECT version_id FROM goose_db_version ORDER BY id")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("\nCurrent migration status:")
	for rows.Next() {
		var versionID int64
		if err := rows.Scan(&versionID); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("  ✓ Migration %d applied\n", versionID)
	}
}
