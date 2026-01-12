package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/loganlanou/logans3d-v4/internal/sync"
	"github.com/loganlanou/logans3d-v4/storage"
	"github.com/loganlanou/logans3d-v4/storage/db"
)

func main() {
	listCmd := flag.NewFlagSet("list", flag.ExitOnError)
	listLimit := listCmd.Int("limit", 20, "Maximum number of products to list")

	syncCmd := flag.NewFlagSet("sync", flag.ExitOnError)
	syncProduct := syncCmd.String("product", "", "Product ID to sync")
	syncAll := syncCmd.Bool("all", false, "Sync all products")
	syncDryRun := syncCmd.Bool("dry-run", false, "Preview what would be synced")
	syncLimit := syncCmd.Int("limit", 100, "Maximum number of products to sync (with -all)")

	testCmd := flag.NewFlagSet("test", flag.ExitOnError)

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "list":
		listCmd.Parse(os.Args[2:])
		runList(*listLimit)
	case "sync":
		syncCmd.Parse(os.Args[2:])
		runSync(*syncProduct, *syncAll, *syncDryRun, *syncLimit)
	case "test":
		testCmd.Parse(os.Args[2:])
		runTest()
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Product Sync Tool - Sync products from local to production")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  sync list                    List products available for sync")
	fmt.Println("  sync sync -product <id>      Sync a single product by ID")
	fmt.Println("  sync sync -all               Sync all products")
	fmt.Println("  sync sync -all -dry-run      Preview what would be synced")
	fmt.Println("  sync test                    Test connection to production API")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  PRODUCTION_API_URL    Production API base URL (default: https://logans3dcreations.com)")
	fmt.Println("  PRODUCTION_API_KEY    API key for production (required)")
	fmt.Println("  DB_PATH               Local database path (default: ./data/database.db)")
}

func getStore() *storage.Storage {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/database.db"
	}

	store, err := storage.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	return store
}

func runList(limit int) {
	store := getStore()
	defer store.Close()

	ctx := context.Background()

	products, err := store.Queries.ListAllProducts(ctx)
	if err != nil {
		log.Fatalf("Failed to list products: %v", err)
	}

	if len(products) == 0 {
		fmt.Println("No products found in local database.")
		return
	}

	fmt.Printf("Found %d products in local database:\n\n", len(products))
	fmt.Printf("%-36s  %-40s  %-10s  %-8s\n", "ID", "Name", "Price", "Active")
	fmt.Println(strings.Repeat("-", 100))

	count := 0
	for _, p := range products {
		if count >= limit {
			fmt.Printf("\n... and %d more (use -limit to see more)\n", len(products)-limit)
			break
		}

		name := p.Name
		if len(name) > 40 {
			name = name[:37] + "..."
		}

		price := fmt.Sprintf("$%.2f", float64(p.PriceCents)/100)

		active := "No"
		if p.IsActive.Valid && p.IsActive.Bool {
			active = "Yes"
		}

		fmt.Printf("%-36s  %-40s  %-10s  %-8s\n", p.ID, name, price, active)
		count++
	}
}

func runSync(productID string, all bool, dryRun bool, limit int) {
	if productID == "" && !all {
		fmt.Println("Error: Either -product <id> or -all is required")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  sync sync -product <id>      Sync a single product")
		fmt.Println("  sync sync -all               Sync all products")
		fmt.Println("  sync sync -all -dry-run      Preview sync")
		os.Exit(1)
	}

	client := sync.NewClient()
	if !client.IsConfigured() {
		log.Fatalf("Error: PRODUCTION_API_KEY environment variable is not set")
	}

	store := getStore()
	defer store.Close()

	ctx := context.Background()

	if productID != "" {
		syncSingleProduct(ctx, store, client, productID, dryRun)
	} else if all {
		syncAllProducts(ctx, store, client, dryRun, limit)
	}
}

func syncSingleProduct(ctx context.Context, store *storage.Storage, client *sync.Client, productID string, dryRun bool) {
	product, err := store.Queries.GetProduct(ctx, productID)
	if err != nil {
		log.Fatalf("Failed to get product: %v", err)
	}

	images, err := store.Queries.GetProductImages(ctx, productID)
	if err != nil {
		log.Fatalf("Failed to get product images: %v", err)
	}

	if dryRun {
		fmt.Println("=== DRY RUN ===")
		fmt.Printf("Would sync product:\n")
		fmt.Printf("  ID: %s\n", product.ID)
		fmt.Printf("  Name: %s\n", product.Name)
		fmt.Printf("  Price: $%.2f\n", float64(product.PriceCents)/100)
		fmt.Printf("  Images: %d\n", len(images))
		return
	}

	fmt.Printf("Syncing product: %s\n", product.Name)

	req := buildProductRequest(product)

	imagePaths := make([]string, 0, len(images))
	for _, img := range images {
		path := filepath.Join("public/images/products", img.ImageUrl)
		if _, err := os.Stat(path); err == nil {
			imagePaths = append(imagePaths, path)
		}
	}

	result, err := client.SyncProduct(ctx, req, imagePaths)
	if err != nil {
		log.Fatalf("Failed to sync product: %v", err)
	}

	fmt.Printf("Success! Product %s (%s)\n", result.Action, result.ProductID)
	fmt.Printf("  Images uploaded: %d\n", len(result.Images))
}

func syncAllProducts(ctx context.Context, store *storage.Storage, client *sync.Client, dryRun bool, limit int) {
	products, err := store.Queries.ListAllProducts(ctx)
	if err != nil {
		log.Fatalf("Failed to list products: %v", err)
	}

	if len(products) == 0 {
		fmt.Println("No products to sync.")
		return
	}

	if len(products) > limit {
		products = products[:limit]
	}

	if dryRun {
		fmt.Println("=== DRY RUN ===")
		fmt.Printf("Would sync %d products:\n\n", len(products))
		for _, p := range products {
			images, _ := store.Queries.GetProductImages(ctx, p.ID)
			fmt.Printf("  - %s (%d images)\n", p.Name, len(images))
		}
		return
	}

	fmt.Printf("Syncing %d products to production...\n\n", len(products))

	var created, updated, failed int

	for i, product := range products {
		fmt.Printf("[%d/%d] Syncing: %s... ", i+1, len(products), product.Name)

		images, err := store.Queries.GetProductImages(ctx, product.ID)
		if err != nil {
			fmt.Printf("FAILED (get images: %v)\n", err)
			failed++
			continue
		}

		req := buildProductRequest(product)

		imagePaths := make([]string, 0, len(images))
		for _, img := range images {
			path := filepath.Join("public/images/products", img.ImageUrl)
			if _, err := os.Stat(path); err == nil {
				imagePaths = append(imagePaths, path)
			}
		}

		result, err := client.SyncProduct(ctx, req, imagePaths)
		if err != nil {
			fmt.Printf("FAILED (%v)\n", err)
			failed++
			continue
		}

		if result.Action == "created" {
			created++
			fmt.Printf("CREATED (%d images)\n", len(result.Images))
		} else {
			updated++
			fmt.Printf("UPDATED (%d images)\n", len(result.Images))
		}

		time.Sleep(100 * time.Millisecond)
	}

	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Created: %d\n", created)
	fmt.Printf("Updated: %d\n", updated)
	fmt.Printf("Failed:  %d\n", failed)
	fmt.Printf("Total:   %d\n", created+updated+failed)
}

func buildProductRequest(p db.Product) sync.ProductRequest {
	req := sync.ProductRequest{
		Name:          p.Name,
		Slug:          p.Slug,
		PriceCents:    p.PriceCents,
		StockQuantity: 999,
		IsActive:      true,
		IsFeatured:    false,
		IsPremium:     false,
		IsNew:         false,
	}

	if p.Description.Valid {
		req.Description = p.Description.String
	}
	if p.ShortDescription.Valid {
		req.ShortDescription = p.ShortDescription.String
	}
	if p.CategoryID.Valid {
		req.CategoryID = p.CategoryID.String
	}
	if p.Sku.Valid {
		req.SKU = p.Sku.String
	}
	if p.WeightGrams.Valid {
		req.WeightGrams = p.WeightGrams.Int64
	}
	if p.LeadTimeDays.Valid {
		req.LeadTimeDays = p.LeadTimeDays.Int64
	}
	if p.IsActive.Valid {
		req.IsActive = p.IsActive.Bool
	}
	if p.IsFeatured.Valid {
		req.IsFeatured = p.IsFeatured.Bool
	}
	if p.IsPremium.Valid {
		req.IsPremium = p.IsPremium.Bool
	}
	if p.Disclaimer.Valid {
		req.Disclaimer = p.Disclaimer.String
	}
	if p.SeoTitle.Valid {
		req.SEOTitle = p.SeoTitle.String
	}
	if p.SeoDescription.Valid {
		req.SEODescription = p.SeoDescription.String
	}
	if p.SeoKeywords.Valid {
		req.SEOKeywords = p.SeoKeywords.String
	}
	if p.OgImageUrl.Valid {
		req.OGImageURL = p.OgImageUrl.String
	}
	if p.SourceUrl.Valid {
		req.SourceURL = p.SourceUrl.String
	}
	if p.SourcePlatform.Valid {
		req.SourcePlatform = p.SourcePlatform.String
	}
	if p.DesignerName.Valid {
		req.DesignerName = p.DesignerName.String
	}

	return req
}

func runTest() {
	client := sync.NewClient()

	fmt.Printf("Testing connection to: %s\n", client.GetBaseURL())

	if !client.IsConfigured() {
		fmt.Println("Error: PRODUCTION_API_KEY is not set")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.TestConnection(ctx); err != nil {
		fmt.Printf("Connection failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Connection successful!")
}
