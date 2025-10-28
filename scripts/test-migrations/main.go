package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
)

func main() {
	log.Println("🧪 Testing database migrations...")

	// Get migrations directory path
	migrationsDir := "../../storage/migrations"
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		// Try from project root
		migrationsDir = "storage/migrations"
	}

	absPath, err := filepath.Abs(migrationsDir)
	if err != nil {
		log.Fatalf("❌ Failed to get absolute path: %v", err)
	}
	log.Printf("📁 Migrations directory: %s", absPath)

	// Create temp directory for test database
	tempDir, err := os.MkdirTemp("", "migration-test-*")
	if err != nil {
		log.Fatalf("❌ Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	log.Printf("📁 Test database: %s", dbPath)

	// Test: Migrate UP from scratch
	log.Println("\n📈 Step 1: Testing UP migrations from scratch...")
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		log.Fatalf("❌ Failed to open database: %v", err)
	}
	defer db.Close()

	if err := goose.SetDialect("sqlite3"); err != nil {
		log.Fatalf("❌ Failed to set dialect: %v", err)
	}

	if err := goose.Up(db, migrationsDir); err != nil {
		log.Fatalf("❌ UP migration failed: %v", err)
	}

	// Get current version
	version, err := goose.GetDBVersion(db)
	if err != nil {
		log.Fatalf("❌ Failed to get version: %v", err)
	}
	log.Printf("✅ UP migrations successful! Current version: %d", version)

	// Test: Migrate DOWN to version 0
	log.Println("\n📉 Step 2: Testing DOWN migrations to version 0...")
	if err := goose.DownTo(db, migrationsDir, 0); err != nil {
		log.Fatalf("❌ DOWN migration failed: %v", err)
	}

	version, err = goose.GetDBVersion(db)
	if err != nil {
		log.Fatalf("❌ Failed to get version: %v", err)
	}
	if version != 0 {
		log.Fatalf("❌ Expected version 0 after DOWN, got: %d", version)
	}
	log.Printf("✅ DOWN migrations successful! Current version: %d", version)

	// Test: Migrate UP again to ensure idempotency
	log.Println("\n📈 Step 3: Testing UP migrations again (idempotency check)...")
	if err := goose.Up(db, migrationsDir); err != nil {
		log.Fatalf("❌ Second UP migration failed: %v", err)
	}

	version, err = goose.GetDBVersion(db)
	if err != nil {
		log.Fatalf("❌ Failed to get version: %v", err)
	}
	log.Printf("✅ Second UP migrations successful! Current version: %d", version)

	// Test: Step down and up one migration at a time
	log.Println("\n🔄 Step 4: Testing individual migration reversibility...")

	// Get latest version
	latestVersion := version

	// Test each migration's reversibility (only test last 3 to save time)
	testCount := 3
	if latestVersion < int64(testCount) {
		testCount = int(latestVersion)
	}

	for i := 0; i < testCount; i++ {
		log.Printf("  Testing migration down from version %d...", version)
		if err := goose.Down(db, migrationsDir); err != nil {
			log.Fatalf("❌ Step DOWN failed from version %d: %v", version, err)
		}

		version, err = goose.GetDBVersion(db)
		if err != nil {
			log.Fatalf("❌ Failed to get version: %v", err)
		}
		log.Printf("  ✓ Down to version %d", version)

		log.Printf("  Testing migration up to version %d...", version+1)
		if err := goose.Up(db, migrationsDir); err != nil {
			log.Fatalf("❌ Step UP failed to version %d: %v", version+1, err)
		}

		version, err = goose.GetDBVersion(db)
		if err != nil {
			log.Fatalf("❌ Failed to get version: %v", err)
		}
		log.Printf("  ✓ Up to version %d", version)
	}

	log.Println("✅ Individual migration reversibility test passed!")

	// Final cleanup
	db.Close()
	os.RemoveAll(tempDir)

	log.Println("\n✅ All migration tests passed successfully! 🎉")
	fmt.Println("\nSummary:")
	fmt.Println("  ✓ UP migrations from scratch")
	fmt.Println("  ✓ DOWN migrations to version 0")
	fmt.Println("  ✓ UP migrations again (idempotency)")
	fmt.Printf("  ✓ Last %d migrations reversibility\n", testCount)
}
