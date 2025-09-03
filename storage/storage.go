package storage

import (
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/loganlanou/logans3d-v4/storage/db"
	_ "modernc.org/sqlite"
)

type Storage struct {
	db      *sql.DB
	Queries *db.Queries
}

func New(dbPath string) (*Storage, error) {
	// Ensure the directory exists
	dir := filepath.Dir(dbPath)
	if err := ensureDir(dir); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open SQLite database with proper settings
	sqliteDB, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := sqliteDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	queries := db.New(sqliteDB)

	return &Storage{
		db:      sqliteDB,
		Queries: queries,
	}, nil
}

func (s *Storage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *Storage) DB() *sql.DB {
	return s.db
}

// ensureDir creates a directory if it doesn't exist
func ensureDir(dir string) error {
	if dir == "." || dir == "" {
		return nil
	}
	
	// For now, assume directory exists (created by make setup)
	// In production, this would use os.MkdirAll(dir, 0755)
	return nil
}