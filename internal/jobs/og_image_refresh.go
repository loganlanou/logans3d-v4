package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"

	"github.com/loganlanou/logans3d-v4/internal/ogimage"
	"github.com/loganlanou/logans3d-v4/storage"
)

const (
	// MaxConcurrentOGGenerations limits how many OG images can be generated simultaneously
	MaxConcurrentOGGenerations = 5
)

type OGImageRefresher struct {
	storage     *storage.Storage
	aiGenerator *ogimage.AIGenerator
}

func NewOGImageRefresher(storage *storage.Storage) *OGImageRefresher {
	return NewOGImageRefresherWithAI(storage, "")
}

func NewOGImageRefresherWithAI(storage *storage.Storage, geminiAPIKey string) *OGImageRefresher {
	r := &OGImageRefresher{
		storage: storage,
	}
	if geminiAPIKey != "" {
		r.aiGenerator = ogimage.NewAIGenerator(geminiAPIKey)
	}
	return r
}

// Start begins the OG image refresh process in a background goroutine
func (r *OGImageRefresher) Start(ctx context.Context) {
	go r.refreshAllOGImages(ctx)
}

// refreshAllOGImages regenerates all OG images for products
func (r *OGImageRefresher) refreshAllOGImages(ctx context.Context) {
	slog.Info("starting OG image refresh job")
	startTime := time.Now()

	// Get all active products
	products, err := r.storage.Queries.ListProducts(ctx)
	if err != nil {
		slog.Error("failed to get products for OG refresh", "error", err)
		return
	}

	if len(products) == 0 {
		slog.Info("no products found for OG image refresh")
		return
	}

	// Ensure output directory exists
	ogDir := filepath.Join("public", "og-images")
	if err := os.MkdirAll(ogDir, 0755); err != nil {
		slog.Error("failed to create OG images directory", "error", err, "dir", ogDir)
		return
	}

	// Create semaphore to limit concurrent goroutines
	sem := semaphore.NewWeighted(MaxConcurrentOGGenerations)
	var wg sync.WaitGroup

	var successCount, errorCount int
	var mu sync.Mutex

	for _, product := range products {
		wg.Add(1)

		go func(productID string, productName string, hasVariants bool) {
			defer wg.Done()

			// Acquire semaphore
			if err := sem.Acquire(ctx, 1); err != nil {
				slog.Debug("context cancelled while waiting for semaphore", "error", err)
				return
			}
			defer sem.Release(1)

			// Check if context is cancelled
			if ctx.Err() != nil {
				return
			}

			var generated bool
			var genErr error

			if hasVariants {
				generated, genErr = r.generateMultiVariantOG(ctx, productID, productName)
			} else {
				generated, genErr = r.generateProductOG(ctx, productID, productName)
			}

			mu.Lock()
			if genErr != nil {
				errorCount++
			} else if generated {
				successCount++
			}
			mu.Unlock()
		}(product.ID, product.Name, product.HasVariants.Valid && product.HasVariants.Bool)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	duration := time.Since(startTime)
	slog.Info("OG image refresh job completed",
		"total_products", len(products),
		"generated", successCount,
		"errors", errorCount,
		"duration", duration,
	)
}

// generateProductOG generates the standard product OG image
func (r *OGImageRefresher) generateProductOG(ctx context.Context, productID, productName string) (bool, error) {
	// Get product details
	product, err := r.storage.Queries.GetProduct(ctx, productID)
	if err != nil {
		slog.Debug("failed to get product for OG generation", "error", err, "product_id", productID)
		return false, err
	}

	// Get category name
	categoryName := "Products"
	if product.CategoryID.Valid {
		category, err := r.storage.Queries.GetCategory(ctx, product.CategoryID.String)
		if err == nil {
			categoryName = category.Name
		}
	}

	// Get primary product image
	images, err := r.storage.Queries.GetProductImages(ctx, productID)
	if err != nil {
		slog.Debug("failed to get product images", "error", err, "product_id", productID)
	}

	primaryImageFile := "default.jpg"
	if len(images) > 0 {
		for _, img := range images {
			if img.IsPrimary.Valid && img.IsPrimary.Bool {
				primaryImageFile = img.ImageUrl
				break
			}
		}
		if primaryImageFile == "default.jpg" && len(images) > 0 {
			primaryImageFile = images[0].ImageUrl
		}
	}

	// Build paths
	productImagePath := filepath.Join("public", "images", "products", primaryImageFile)
	ogImageFilename := fmt.Sprintf("product-%s.png", productID)
	ogImagePath := filepath.Join("public", "og-images", ogImageFilename)

	// Check if source image exists
	if _, err := os.Stat(productImagePath); os.IsNotExist(err) {
		slog.Debug("product image not found, skipping OG generation", "product_id", productID, "image_path", productImagePath)
		return false, nil
	}

	// Check if OG image already exists and is recent (less than 7 days old)
	if info, err := os.Stat(ogImagePath); err == nil {
		if time.Since(info.ModTime()) < 7*24*time.Hour {
			slog.Debug("OG image already exists and is recent, skipping", "product_id", productID)
			return false, nil
		}
	}

	// Generate OG image
	productInfo := ogimage.ProductInfo{
		Name:         product.Name,
		CategoryName: categoryName,
		ImagePath:    productImagePath,
	}

	if err := ogimage.GenerateOGImage(productInfo, ogImagePath); err != nil {
		slog.Debug("failed to generate product OG image", "error", err, "product_id", productID)
		return false, err
	}

	slog.Debug("generated product OG image", "product", productName, "output", ogImagePath)
	return true, nil
}

// generateMultiVariantOG generates a multi-variant grid OG image
func (r *OGImageRefresher) generateMultiVariantOG(ctx context.Context, productID, productName string) (bool, error) {
	// Get product details
	product, err := r.storage.Queries.GetProduct(ctx, productID)
	if err != nil {
		slog.Debug("failed to get product for multi-variant OG generation", "error", err, "product_id", productID)
		return false, err
	}

	// Get price range
	priceRange, err := r.storage.Queries.GetProductPriceRange(ctx, productID)
	if err != nil {
		slog.Debug("failed to get price range", "error", err, "product_id", productID)
	}

	// Format price range
	var priceRangeStr string
	minPrice, minOk := priceRange.MinPrice.(int64)
	maxPrice, maxOk := priceRange.MaxPrice.(int64)
	if minOk && maxOk && minPrice > 0 {
		if minPrice == maxPrice {
			priceRangeStr = fmt.Sprintf("$%.2f", float64(minPrice)/100)
		} else {
			priceRangeStr = fmt.Sprintf("$%.2f - $%.2f", float64(minPrice)/100, float64(maxPrice)/100)
		}
	} else {
		priceRangeStr = fmt.Sprintf("$%.2f", float64(product.PriceCents)/100)
	}

	// Get style count
	styleCount := int(priceRange.StyleCount)
	if styleCount == 0 {
		styleCount = 1
	}

	// Get all style primary images
	styleImages, err := r.storage.Queries.GetAllStylePrimaryImages(ctx, productID)
	if err != nil {
		slog.Debug("failed to get style images", "error", err, "product_id", productID)
	}

	// Build image paths (up to 4 for grid)
	var imagePaths []string
	var styleNames []string
	for i, img := range styleImages {
		if i >= 4 {
			break
		}
		imagePath := filepath.Join("public", "images", "products", "styles", img.ImageUrl)
		// Check if image exists
		if _, err := os.Stat(imagePath); os.IsNotExist(err) {
			continue
		}
		imagePaths = append(imagePaths, imagePath)
		styleNames = append(styleNames, img.StyleName)
	}

	// If no style images found, try product images
	if len(imagePaths) == 0 {
		images, err := r.storage.Queries.GetProductImages(ctx, productID)
		if err == nil && len(images) > 0 {
			for i, img := range images {
				if i >= 4 {
					break
				}
				imagePath := filepath.Join("public", "images", "products", img.ImageUrl)
				if _, err := os.Stat(imagePath); os.IsNotExist(err) {
					continue
				}
				imagePaths = append(imagePaths, imagePath)
			}
		}
	}

	// Need at least one image
	if len(imagePaths) == 0 {
		slog.Debug("no images found for multi-variant OG", "product_id", productID)
		return false, nil
	}

	// Build output path
	ogImageFilename := fmt.Sprintf("product-%s-multi.png", productID)
	ogImagePath := filepath.Join("public", "og-images", ogImageFilename)

	// Check if OG image already exists and is recent (less than 7 days old)
	if info, err := os.Stat(ogImagePath); err == nil {
		if time.Since(info.ModTime()) < 7*24*time.Hour {
			slog.Debug("multi-variant OG image already exists and is recent, skipping", "product_id", productID)
			return false, nil
		}
	}

	// Generate multi-variant OG image
	info := ogimage.MultiVariantInfo{
		Name:       product.Name,
		StyleCount: styleCount,
		PriceRange: priceRangeStr,
		ImagePaths: imagePaths,
		StyleNames: styleNames,
	}

	// Use AI generator if available, otherwise fall back to grid method
	var genErr error
	if r.aiGenerator != nil {
		genErr = r.aiGenerator.GenerateMultiVariantOGImage(info, ogImagePath)
	} else {
		genErr = ogimage.GenerateMultiVariantOGImage(info, ogImagePath)
	}
	if genErr != nil {
		slog.Debug("failed to generate multi-variant OG image", "error", genErr, "product_id", productID)
		return false, genErr
	}

	slog.Debug("generated multi-variant OG image", "product", productName, "styles", styleCount, "output", ogImagePath)
	return true, nil
}
