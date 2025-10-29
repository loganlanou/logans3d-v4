package storage

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/loganlanou/logans3d-v4/storage/db"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var testMigrations embed.FS

// NewTestDB creates an in-memory SQLite database for testing
func NewTestDB() (*sql.DB, *db.Queries, func(), error) {
	// Create in-memory database
	database, err := sql.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to open test database: %w", err)
	}

	// Run migrations
	goose.SetBaseFS(testMigrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		database.Close()
		return nil, nil, nil, fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := goose.Up(database, "migrations"); err != nil {
		database.Close()
		return nil, nil, nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	queries := db.New(database)

	// Cleanup function
	cleanup := func() {
		database.Close()
	}

	return database, queries, cleanup, nil
}

// WithTransaction executes a function within a transaction and rolls it back
// Useful for tests that need to ensure no side effects
func WithTransaction(database *sql.DB, fn func(*sql.Tx) error) error {
	tx, err := database.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer tx.Rollback() // Always rollback in tests

	if err := fn(tx); err != nil {
		return err
	}

	return nil
}
