package handlers

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/internal/ogimage"
	"github.com/loganlanou/logans3d-v4/storage"
)

type OGImageHandler struct {
	storage *storage.Storage
}

func NewOGImageHandler(storage *storage.Storage) *OGImageHandler {
	return &OGImageHandler{
		storage: storage,
	}
}

// HandleGenerateOGImage generates an Open Graph image for a product
func (h *OGImageHandler) HandleGenerateOGImage(c echo.Context) error {
	ctx := c.Request().Context()
	productID := c.Param("product_id")

	if productID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Product ID is required")
	}

	// Get product details
	product, err := h.storage.Queries.GetProduct(ctx, productID)
	if err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "Product not found")
		}
		slog.Error("failed to get product", "error", err, "product_id", productID)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load product")
	}

	// Get category name
	categoryName := "Products"
	if product.CategoryID.Valid {
		category, err := h.storage.Queries.GetCategory(ctx, product.CategoryID.String)
		if err != nil {
			slog.Debug("failed to get category", "error", err, "category_id", product.CategoryID.String)
		} else {
			categoryName = category.Name
		}
	}

	// Get primary product image
	images, err := h.storage.Queries.GetProductImages(ctx, productID)
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

	// Check if OG image exists and is recent
	if info, err := os.Stat(ogImagePath); err == nil {
		// Image exists, check if it needs regeneration
		// Regenerate if product was updated after OG image was created
		productUpdated := product.UpdatedAt
		if productUpdated.Valid {
			ogImageCreated := info.ModTime()
			if productUpdated.Time.Before(ogImageCreated) {
				// OG image is up to date, serve it
				return c.File(ogImagePath)
			}
		} else {
			// No updated_at, assume OG image is good if less than 7 days old
			if time.Since(info.ModTime()) < 7*24*time.Hour {
				return c.File(ogImagePath)
			}
		}
	}

	// Generate new OG image
	productInfo := ogimage.ProductInfo{
		Name:         product.Name,
		CategoryName: categoryName,
		ImagePath:    productImagePath,
	}

	err = ogimage.GenerateOGImage(productInfo, ogImagePath)
	if err != nil {
		slog.Error("failed to generate OG image", "error", err, "product_id", productID)

		// Try to serve default OG image as fallback
		defaultOGPath := filepath.Join("public", "og-images", "default.png")
		if _, err := os.Stat(defaultOGPath); err == nil {
			return c.File(defaultOGPath)
		}

		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate OG image")
	}

	// Serve the generated image
	return c.File(ogImagePath)
}
