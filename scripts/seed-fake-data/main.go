package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

const (
	// Configuration
	numUsers            = 25
	numOrders           = 45
	numActiveCarts      = 8
	numAbandonedCarts   = 15
	numContactRequests  = 20
	numFavoritesPerUser = 3
	numCollections      = 10
)

var (
	db         *sql.DB
	productIDs []string
	userIDs    []string
	sessionIDs []string
)

type User struct {
	ID              string
	Email           string
	FullName        string
	FirstName       string
	LastName        string
	Username        string
	ProfileImageURL string
	ClerkID         string
	CreatedAt       time.Time
	LastSyncedAt    time.Time
}

type Order struct {
	ID                   string
	UserID               string
	CustomerName         string
	CustomerEmail        string
	CustomerPhone        string
	ShippingAddressLine1 string
	ShippingAddressLine2 string
	ShippingCity         string
	ShippingState        string
	ShippingPostalCode   string
	ShippingCountry      string
	SubtotalCents        int64
	TaxCents             int64
	ShippingCents        int64
	TotalCents           int64
	Status               string
	CreatedAt            time.Time
}

func main() {
	rand.Seed(time.Now().UnixNano())
	gofakeit.Seed(time.Now().UnixNano())

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/database.db"
	}

	var err error
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	fmt.Println("🌱 Starting database seeding...")
	fmt.Println()

	// Load existing product IDs
	loadProductIDs()

	if len(productIDs) == 0 {
		log.Fatal("❌ No products found in database. Please seed products first.")
	}

	fmt.Printf("✓ Found %d products in database\n", len(productIDs))
	fmt.Println()

	// Clear existing fake data
	clearFakeData()

	// Seed data
	seedUsers()
	seedOrders()
	seedActiveCarts()
	seedAbandonedCarts()
	seedContactRequests()
	seedUserFavorites()
	seedUserCollections()

	fmt.Println()
	fmt.Println("✅ Database seeding completed!")
	fmt.Println()
	printSummary()
}

func loadProductIDs() {
	rows, err := db.Query("SELECT id FROM products WHERE is_active = 1 LIMIT 50")
	if err != nil {
		log.Fatalf("Failed to load products: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			log.Printf("Error scanning product: %v", err)
			continue
		}
		productIDs = append(productIDs, id)
	}
}

func clearFakeData() {
	fmt.Println("🧹 Clearing existing fake data...")

	// Clear in reverse dependency order
	tables := []string{
		"collection_items",
		"user_collections",
		"user_favorites",
		"contact_requests",
		"cart_recovery_attempts",
		"cart_snapshots",
		"abandoned_carts",
		"cart_items",
		"order_items",
		"orders",
		// Only delete non-admin users (preserve the real admin)
	}

	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			log.Printf("Warning: failed to clear %s: %v", table, err)
		}
	}

	// Delete fake users (preserve admin users)
	_, err := db.Exec("DELETE FROM users WHERE is_admin = FALSE")
	if err != nil {
		log.Printf("Warning: failed to clear users: %v", err)
	}

	fmt.Println("✓ Cleared existing fake data")
	fmt.Println()
}

func seedUsers() {
	fmt.Println("👥 Creating users...")

	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO users (id, email, full_name, first_name, last_name, username, profile_image_url, clerk_id, created_at, last_synced_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		log.Fatalf("Failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	now := time.Now()

	for i := 0; i < numUsers; i++ {
		user := generateUser(i, now)
		userIDs = append(userIDs, user.ID)

		_, err = stmt.Exec(
			user.ID,
			user.Email,
			user.FullName,
			user.FirstName,
			user.LastName,
			user.Username,
			user.ProfileImageURL,
			user.ClerkID,
			formatSQLiteTime(user.CreatedAt),
			formatSQLiteTime(user.LastSyncedAt),
		)
		if err != nil {
			log.Printf("Failed to insert user %s: %v", user.Email, err)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("Failed to commit users: %v", err)
	}

	fmt.Printf("✓ Created %d users\n", numUsers)
}

func generateUser(index int, now time.Time) User {
	firstName := gofakeit.FirstName()
	lastName := gofakeit.LastName()
	username := gofakeit.Username()

	var createdAt, lastSyncedAt time.Time

	// Create different user segments
	switch {
	case index < 3:
		// Inactive users (no activity in 90+ days)
		createdAt = now.AddDate(0, -6, -rand.Intn(60))
		lastSyncedAt = now.AddDate(0, 0, -90-rand.Intn(30))
	case index < 8:
		// New users (registered within 30 days)
		createdAt = now.AddDate(0, 0, -rand.Intn(30))
		lastSyncedAt = now.AddDate(0, 0, -rand.Intn(2))
	default:
		// Regular users
		createdAt = now.AddDate(0, -rand.Intn(12), -rand.Intn(30))
		lastSyncedAt = now.AddDate(0, 0, -rand.Intn(7))
	}

	// Some users have profile images
	profileImageURL := ""
	if rand.Float32() < 0.4 {
		profileImageURL = fmt.Sprintf("https://i.pravatar.cc/150?u=%s", username)
	}

	// Generate fake Clerk ID
	clerkID := fmt.Sprintf("user_fake_%s", uuid.New().String()[:16])

	return User{
		ID:              uuid.New().String(),
		Email:           gofakeit.Email(),
		FullName:        fmt.Sprintf("%s %s", firstName, lastName),
		FirstName:       firstName,
		LastName:        lastName,
		Username:        username,
		ProfileImageURL: profileImageURL,
		ClerkID:         clerkID,
		CreatedAt:       createdAt,
		LastSyncedAt:    lastSyncedAt,
	}
}

func seedOrders() {
	fmt.Println("📦 Creating orders...")

	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	orderStmt, err := tx.Prepare(`
		INSERT INTO orders (
			id, user_id, customer_name, customer_email, customer_phone,
			shipping_address_line1, shipping_address_line2, shipping_city, shipping_state, shipping_postal_code, shipping_country,
			subtotal_cents, tax_cents, shipping_cents, total_cents, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		log.Fatalf("Failed to prepare order statement: %v", err)
	}
	defer orderStmt.Close()

	itemStmt, err := tx.Prepare(`
		INSERT INTO order_items (id, order_id, product_id, quantity, unit_price_cents, total_price_cents, product_name, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		log.Fatalf("Failed to prepare order item statement: %v", err)
	}
	defer itemStmt.Close()

	now := time.Now()
	statuses := []string{"received", "in_production", "shipped", "delivered", "cancelled"}

	// Distribute orders to create VIP users (5+ orders or $500+ spend)
	ordersPerUser := make(map[string]int)

	for i := 0; i < numOrders; i++ {
		// Pick user - bias toward first few users to create VIPs
		var userID string
		if i < 20 && rand.Float32() < 0.6 {
			userID = userIDs[rand.Intn(min(5, len(userIDs)))]
		} else {
			userID = userIDs[rand.Intn(len(userIDs))]
		}
		ordersPerUser[userID]++

		order := generateOrder(userID, now)

		_, err = orderStmt.Exec(
			order.ID, order.UserID, order.CustomerName, order.CustomerEmail, order.CustomerPhone,
			order.ShippingAddressLine1, order.ShippingAddressLine2, order.ShippingCity,
			order.ShippingState, order.ShippingPostalCode, order.ShippingCountry,
			order.SubtotalCents, order.TaxCents, order.ShippingCents, order.TotalCents,
			statuses[rand.Intn(len(statuses))], formatSQLiteTime(order.CreatedAt), formatSQLiteTime(order.CreatedAt),
		)
		if err != nil {
			log.Printf("Failed to insert order: %v", err)
			continue
		}

		// Add 1-4 items per order
		numItems := 1 + rand.Intn(4)
		for j := 0; j < numItems; j++ {
			productID := productIDs[rand.Intn(len(productIDs))]
			quantity := 1 + rand.Intn(3)
			unitPrice := 499 + rand.Intn(2000) // $4.99 - $24.99
			totalPrice := unitPrice * quantity

			// Get product name
			var productName string
			err := db.QueryRow("SELECT name FROM products WHERE id = ?", productID).Scan(&productName)
			if err != nil {
				productName = "Unknown Product"
			}

			_, err = itemStmt.Exec(
				uuid.New().String(),
				order.ID,
				productID,
				quantity,
				unitPrice,
				totalPrice,
				productName,
				formatSQLiteTime(order.CreatedAt),
			)
			if err != nil {
				log.Printf("Failed to insert order item: %v", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("Failed to commit orders: %v", err)
	}

	fmt.Printf("✓ Created %d orders\n", numOrders)
}

func generateOrder(userID string, now time.Time) Order {
	// Get user details
	var customerName, customerEmail string
	err := db.QueryRow("SELECT full_name, email FROM users WHERE id = ?", userID).Scan(&customerName, &customerEmail)
	if err != nil {
		customerName = gofakeit.Name()
		customerEmail = gofakeit.Email()
	}

	// Generate random time in past 6 months
	daysAgo := rand.Intn(180)
	createdAt := now.AddDate(0, 0, -daysAgo)

	address := gofakeit.Address()
	subtotal := int64(1000 + rand.Intn(15000)) // $10 - $160
	tax := subtotal * 8 / 100                  // 8% tax
	shipping := int64(500 + rand.Intn(1500))   // $5 - $20

	return Order{
		ID:                   uuid.New().String(),
		UserID:               userID,
		CustomerName:         customerName,
		CustomerEmail:        customerEmail,
		CustomerPhone:        gofakeit.Phone(),
		ShippingAddressLine1: address.Street,
		ShippingAddressLine2: "",
		ShippingCity:         address.City,
		ShippingState:        address.State,
		ShippingPostalCode:   address.Zip,
		ShippingCountry:      "US",
		SubtotalCents:        subtotal,
		TaxCents:             tax,
		ShippingCents:        shipping,
		TotalCents:           subtotal + tax + shipping,
		CreatedAt:            createdAt,
	}
}

func seedActiveCarts() {
	fmt.Println("🛒 Creating active carts...")

	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO cart_items (id, session_id, user_id, product_id, quantity, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		log.Fatalf("Failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	now := time.Now()

	for i := 0; i < numActiveCarts; i++ {
		// 50% guest carts, 50% user carts
		var sessionID, userID sql.NullString
		if rand.Float32() < 0.5 {
			// Guest cart
			sessionID = sql.NullString{String: uuid.New().String(), Valid: true}
			sessionIDs = append(sessionIDs, sessionID.String)
		} else {
			// User cart
			userID = sql.NullString{String: userIDs[rand.Intn(len(userIDs))], Valid: true}
		}

		// Add 1-5 items to cart
		numItems := 1 + rand.Intn(5)
		updatedAt := now.Add(-time.Duration(rand.Intn(7*24)) * time.Hour) // Within last 7 days

		for j := 0; j < numItems; j++ {
			productID := productIDs[rand.Intn(len(productIDs))]
			quantity := 1 + rand.Intn(3)

			_, err = stmt.Exec(
				uuid.New().String(),
				sessionID,
				userID,
				productID,
				quantity,
				formatSQLiteTime(updatedAt),
				formatSQLiteTime(updatedAt),
			)
			if err != nil {
				log.Printf("Failed to insert cart item: %v", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("Failed to commit active carts: %v", err)
	}

	fmt.Printf("✓ Created %d active carts\n", numActiveCarts)
}

func seedAbandonedCarts() {
	fmt.Println("🛒 Creating abandoned carts...")

	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	cartStmt, err := tx.Prepare(`
		INSERT INTO cart_items (id, session_id, user_id, product_id, quantity, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		log.Fatalf("Failed to prepare cart statement: %v", err)
	}
	defer cartStmt.Close()

	now := time.Now()

	for i := 0; i < numAbandonedCarts; i++ {
		// 60% guest carts, 40% user carts
		var sessionID, userID sql.NullString

		if rand.Float32() < 0.6 {
			// Guest cart
			sessionID = sql.NullString{String: uuid.New().String(), Valid: true}
		} else {
			// User cart
			uid := userIDs[rand.Intn(len(userIDs))]
			userID = sql.NullString{String: uid, Valid: true}
		}

		// Abandoned 8-60 days ago
		daysAgo := 8 + rand.Intn(52)
		abandonedAt := now.AddDate(0, 0, -daysAgo).Add(-time.Duration(rand.Intn(24)) * time.Hour)

		// Add 1-4 items to cart
		numItems := 1 + rand.Intn(4)
		totalValue := int64(0)

		for j := 0; j < numItems; j++ {
			productID := productIDs[rand.Intn(len(productIDs))]
			quantity := 1 + rand.Intn(2)

			// Get product price
			var price int64
			err := db.QueryRow("SELECT price_cents FROM products WHERE id = ?", productID).Scan(&price)
			if err != nil {
				price = 999
			}
			totalValue += price * int64(quantity)

			_, err = cartStmt.Exec(
				uuid.New().String(),
				sessionID,
				userID,
				productID,
				quantity,
				formatSQLiteTime(abandonedAt),
				formatSQLiteTime(abandonedAt),
			)
			if err != nil {
				log.Printf("Failed to insert abandoned cart item: %v", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("Failed to commit abandoned carts: %v", err)
	}

	fmt.Printf("✓ Created %d abandoned carts\n", numAbandonedCarts)
}

func seedContactRequests() {
	fmt.Println("📧 Creating contact requests...")

	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO contact_requests (
			id, first_name, last_name, email, phone, subject, message,
			newsletter_subscribe, status, priority, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		log.Fatalf("Failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	now := time.Now()
	statuses := []string{"new", "in_progress", "responded", "resolved", "spam"}
	priorities := []string{"low", "normal", "high", "urgent"}
	subjects := []string{
		"Question about custom orders",
		"Shipping inquiry",
		"Product availability",
		"Bulk order pricing",
		"Event collaboration",
		"Product defect report",
		"Custom design request",
		"Partnership opportunity",
		"General inquiry",
	}

	for i := 0; i < numContactRequests; i++ {
		daysAgo := rand.Intn(90)
		createdAt := now.AddDate(0, 0, -daysAgo)

		_, err = stmt.Exec(
			uuid.New().String(),
			gofakeit.FirstName(),
			gofakeit.LastName(),
			gofakeit.Email(),
			gofakeit.Phone(),
			subjects[rand.Intn(len(subjects))],
			gofakeit.Paragraph(2, 4, 10, " "),
			rand.Float32() < 0.3,
			statuses[rand.Intn(len(statuses))],
			priorities[rand.Intn(len(priorities))],
			formatSQLiteTime(createdAt),
			formatSQLiteTime(createdAt),
		)
		if err != nil {
			log.Printf("Failed to insert contact request: %v", err)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("Failed to commit contact requests: %v", err)
	}

	fmt.Printf("✓ Created %d contact requests\n", numContactRequests)
}

func seedUserFavorites() {
	fmt.Println("⭐ Creating user favorites...")

	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO user_favorites (id, user_id, product_id, created_at)
		VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		log.Fatalf("Failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	now := time.Now()
	totalFavorites := 0

	for _, userID := range userIDs {
		// Some users have favorites
		if rand.Float32() < 0.6 {
			numFavs := 1 + rand.Intn(numFavoritesPerUser)
			usedProducts := make(map[string]bool)

			for i := 0; i < numFavs; i++ {
				productID := productIDs[rand.Intn(len(productIDs))]

				// Avoid duplicates
				if usedProducts[productID] {
					continue
				}
				usedProducts[productID] = true

				daysAgo := rand.Intn(60)
				createdAt := now.AddDate(0, 0, -daysAgo)

				_, err = stmt.Exec(
					uuid.New().String(),
					userID,
					productID,
					formatSQLiteTime(createdAt),
				)
				if err != nil {
					log.Printf("Failed to insert favorite: %v", err)
					continue
				}
				totalFavorites++
			}
		}
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("Failed to commit favorites: %v", err)
	}

	fmt.Printf("✓ Created %d favorites\n", totalFavorites)
}

func seedUserCollections() {
	fmt.Println("📚 Creating user collections...")

	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	collStmt, err := tx.Prepare(`
		INSERT INTO user_collections (id, user_id, name, description, is_quote_requested, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		log.Fatalf("Failed to prepare collection statement: %v", err)
	}
	defer collStmt.Close()

	itemStmt, err := tx.Prepare(`
		INSERT INTO collection_items (id, collection_id, product_id, notes, created_at)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		log.Fatalf("Failed to prepare collection item statement: %v", err)
	}
	defer itemStmt.Close()

	now := time.Now()
	collectionNames := []string{
		"Dinosaur Collection",
		"Gift Ideas",
		"Office Decorations",
		"Kids' Room",
		"Custom Project Ideas",
		"Birthday Wishlist",
		"Educational Models",
		"Event Display",
	}

	totalCollections := 0

	for i := 0; i < numCollections; i++ {
		userID := userIDs[rand.Intn(len(userIDs))]
		daysAgo := rand.Intn(90)
		createdAt := now.AddDate(0, 0, -daysAgo)

		collectionID := uuid.New().String()
		isQuoteRequested := rand.Float32() < 0.3

		_, err = collStmt.Exec(
			collectionID,
			userID,
			collectionNames[rand.Intn(len(collectionNames))],
			gofakeit.Sentence(8),
			isQuoteRequested,
			formatSQLiteTime(createdAt),
			formatSQLiteTime(createdAt),
		)
		if err != nil {
			log.Printf("Failed to insert collection: %v", err)
			continue
		}
		totalCollections++

		// Add 2-6 items to collection
		numItems := 2 + rand.Intn(5)
		usedProducts := make(map[string]bool)

		for j := 0; j < numItems; j++ {
			productID := productIDs[rand.Intn(len(productIDs))]

			if usedProducts[productID] {
				continue
			}
			usedProducts[productID] = true

			notes := ""
			if rand.Float32() < 0.3 {
				notes = gofakeit.Sentence(5)
			}

			_, err = itemStmt.Exec(
				uuid.New().String(),
				collectionID,
				productID,
				notes,
				formatSQLiteTime(createdAt),
			)
			if err != nil {
				log.Printf("Failed to insert collection item: %v", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("Failed to commit collections: %v", err)
	}

	fmt.Printf("✓ Created %d collections\n", totalCollections)
}

func printSummary() {
	fmt.Println("📊 Summary:")
	fmt.Println()

	// Count records
	counts := make(map[string]int)
	tables := []string{
		"users WHERE is_admin = FALSE",
		"orders",
		"order_items",
		"cart_items",
		"contact_requests",
		"user_favorites",
		"user_collections",
	}

	for _, table := range tables {
		var count int
		tableName := table
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)

		if err := db.QueryRow(query).Scan(&count); err != nil {
			log.Printf("Failed to count %s: %v", table, err)
			continue
		}

		// Clean up table name for display
		for idx := 0; idx < len(tableName); idx++ {
			if tableName[idx] == ' ' {
				tableName = tableName[:idx]
				break
			}
		}

		counts[tableName] = count
	}

	// VIP users
	var vipUsers int
	db.QueryRow(`
		SELECT COUNT(DISTINCT u.id)
		FROM users u
		LEFT JOIN orders o ON o.user_id = u.id
		WHERE u.is_admin = FALSE
		GROUP BY u.id
		HAVING COUNT(o.id) >= 5 OR COALESCE(SUM(o.total_cents), 0) >= 50000
	`).Scan(&vipUsers)

	// New users
	var newUsers int
	db.QueryRow(`
		SELECT COUNT(*)
		FROM users
		WHERE is_admin = FALSE
		AND created_at > datetime('now', '-30 days')
	`).Scan(&newUsers)

	// Abandoned carts
	var abandonedCarts int
	db.QueryRow("SELECT COUNT(*) FROM abandoned_carts").Scan(&abandonedCarts)

	fmt.Printf("  Users:              %d (%d VIP, %d New)\n", counts["users"], vipUsers, newUsers)
	fmt.Printf("  Orders:             %d (%d items)\n", counts["orders"], counts["order_items"])
	fmt.Printf("  Active Carts:       %d\n", counts["cart_items"])
	fmt.Printf("  Abandoned Carts:    %d\n", abandonedCarts)
	fmt.Printf("  Contact Requests:   %d\n", counts["contact_requests"])
	fmt.Printf("  Favorites:          %d\n", counts["user_favorites"])
	fmt.Printf("  Collections:        %d\n", counts["user_collections"])
	fmt.Println()
	fmt.Println("🎉 You can now view the admin dashboard with realistic data!")
	fmt.Println("   Navigate to: http://localhost:8007/admin")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// formatSQLiteTime formats a time.Time as SQLite-compatible datetime string without timezone
func formatSQLiteTime(t time.Time) string {
	return t.UTC().Format("2006-01-02 15:04:05")
}
