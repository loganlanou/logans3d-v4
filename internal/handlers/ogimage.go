package handlers

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/internal/ogimage"
	"github.com/loganlanou/logans3d-v4/storage"
	"github.com/loganlanou/logans3d-v4/storage/db"
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
// Supports variant query params: ?color={styleId}&size={sizeId}
// Add ?refresh=true to force regeneration (bypass cache)
func (h *OGImageHandler) HandleGenerateOGImage(c echo.Context) error {
	ctx := c.Request().Context()
	productID := c.Param("product_id")

	if productID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Product ID is required")
	}

	// Parse query params
	colorID := c.QueryParam("color")   // style ID
	sizeID := c.QueryParam("size")     // size ID
	refresh := c.QueryParam("refresh") // force regeneration

	// Get product details
	product, err := h.storage.Queries.GetProduct(ctx, productID)
	if err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "Product not found")
		}
		slog.Error("failed to get product", "error", err, "product_id", productID)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load product")
	}

	// If variant params provided, generate variant-specific OG image
	if colorID != "" && sizeID != "" {
		return h.generateVariantOGImage(c, product, colorID, sizeID, refresh == "true")
	}

	// Standard product OG image generation (no variant)
	return h.generateProductOGImage(c, product, refresh == "true")
}

// generateProductOGImage generates the standard product OG image (no variant)
func (h *OGImageHandler) generateProductOGImage(c echo.Context, product db.Product, forceRefresh bool) error {
	ctx := c.Request().Context()

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
	images, err := h.storage.Queries.GetProductImages(ctx, product.ID)
	if err != nil {
		slog.Debug("failed to get product images", "error", err, "product_id", product.ID)
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
	ogImageFilename := fmt.Sprintf("product-%s.png", product.ID)
	ogImagePath := filepath.Join("public", "og-images", ogImageFilename)

	// Check if OG image exists and is recent (skip if forceRefresh)
	if !forceRefresh {
		if info, err := os.Stat(ogImagePath); err == nil {
			productUpdated := product.UpdatedAt
			if productUpdated.Valid {
				ogImageCreated := info.ModTime()
				if productUpdated.Time.Before(ogImageCreated) {
					return c.File(ogImagePath)
				}
			} else {
				if time.Since(info.ModTime()) < 7*24*time.Hour {
					return c.File(ogImagePath)
				}
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
		slog.Error("failed to generate OG image", "error", err, "product_id", product.ID)
		return h.serveDefaultOGImage(c)
	}

	return c.File(ogImagePath)
}

// generateVariantOGImage generates an OG image for a specific variant (style + size)
func (h *OGImageHandler) generateVariantOGImage(c echo.Context, product db.Product, styleID, sizeID string, forceRefresh bool) error {
	ctx := c.Request().Context()

	// Get style details
	style, err := h.storage.Queries.GetProductStyle(ctx, styleID)
	if err != nil {
		slog.Debug("style not found, falling back to product OG", "style_id", styleID)
		return h.generateProductOGImage(c, product, forceRefresh)
	}

	// Verify style belongs to this product
	if style.ProductID != product.ID {
		slog.Debug("style doesn't belong to product, falling back", "style_id", styleID, "product_id", product.ID)
		return h.generateProductOGImage(c, product, forceRefresh)
	}

	// Get size details
	size, err := h.storage.Queries.GetSize(ctx, sizeID)
	if err != nil {
		slog.Debug("size not found, falling back to product OG", "size_id", sizeID)
		return h.generateProductOGImage(c, product, forceRefresh)
	}

	// Get SKU for price calculation
	sku, err := h.storage.Queries.GetSkuByStyleAndSize(ctx, db.GetSkuByStyleAndSizeParams{
		ProductID:      product.ID,
		ProductStyleID: styleID,
		SizeID:         sizeID,
	})
	if err != nil {
		slog.Debug("SKU not found, falling back to product OG", "product_id", product.ID, "style_id", styleID, "size_id", sizeID)
		return h.generateProductOGImage(c, product, forceRefresh)
	}

	// Get style's primary image
	styleImage, err := h.storage.Queries.GetPrimaryStyleImage(ctx, styleID)
	imageFile := "default.jpg"
	if err == nil && styleImage.ImageUrl != "" {
		imageFile = filepath.Join("styles", styleImage.ImageUrl)
	} else {
		// Fallback to product's primary image
		images, err := h.storage.Queries.GetProductImages(ctx, product.ID)
		if err == nil && len(images) > 0 {
			for _, img := range images {
				if img.IsPrimary.Valid && img.IsPrimary.Bool {
					imageFile = img.ImageUrl
					break
				}
			}
			if imageFile == "default.jpg" {
				imageFile = images[0].ImageUrl
			}
		}
	}

	// Calculate final price
	priceAdjustment := int64(0)
	if sku.PriceAdjustmentCents.Valid {
		priceAdjustment = sku.PriceAdjustmentCents.Int64
	}
	finalPriceCents := product.PriceCents + priceAdjustment

	// Build paths - variant-specific cache filename
	productImagePath := filepath.Join("public", "images", "products", imageFile)
	ogImageFilename := fmt.Sprintf("product-%s-%s-%s.png", product.ID, styleID, sizeID)
	ogImagePath := filepath.Join("public", "og-images", ogImageFilename)

	// Check if variant OG image exists and is recent (skip if forceRefresh)
	if !forceRefresh {
		if info, err := os.Stat(ogImagePath); err == nil {
			if time.Since(info.ModTime()) < 7*24*time.Hour {
				return c.File(ogImagePath)
			}
		}
	}

	// Generate variant OG image
	productInfo := ogimage.ProductInfo{
		Name:      product.Name,
		ImagePath: productImagePath,
	}

	variantInfo := ogimage.VariantInfo{
		StyleName:  style.Name,
		SizeName:   size.DisplayName,
		PriceCents: finalPriceCents,
	}

	err = ogimage.GenerateVariantOGImage(productInfo, variantInfo, ogImagePath)
	if err != nil {
		slog.Error("failed to generate variant OG image", "error", err, "product_id", product.ID, "style_id", styleID, "size_id", sizeID)
		return h.serveDefaultOGImage(c)
	}

	return c.File(ogImagePath)
}

// serveDefaultOGImage serves the default OG image as fallback
func (h *OGImageHandler) serveDefaultOGImage(c echo.Context) error {
	defaultOGPath := filepath.Join("public", "og-images", "default.png")
	if _, err := os.Stat(defaultOGPath); err == nil {
		return c.File(defaultOGPath)
	}
	return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate OG image")
}

// HandleGenerateMultiVariantOGImage generates an OG image showing all product variants in a grid
// Route: GET /api/og-image/multi/:product_id
// Add ?refresh=true to force regeneration
func (h *OGImageHandler) HandleGenerateMultiVariantOGImage(c echo.Context) error {
	ctx := c.Request().Context()
	productID := c.Param("product_id")

	if productID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Product ID is required")
	}

	refresh := c.QueryParam("refresh") == "true"

	// Get product details
	product, err := h.storage.Queries.GetProduct(ctx, productID)
	if err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "Product not found")
		}
		slog.Error("failed to get product", "error", err, "product_id", productID)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load product")
	}

	// Check if product has variants
	if !product.HasVariants.Valid || !product.HasVariants.Bool {
		// No variants, fall back to standard product OG image
		return h.generateProductOGImage(c, product, refresh)
	}

	// Build cache path
	ogImageFilename := fmt.Sprintf("product-%s-multi.png", product.ID)
	ogImagePath := filepath.Join("public", "og-images", ogImageFilename)

	// Check cache (skip if refresh)
	if !refresh {
		if info, err := os.Stat(ogImagePath); err == nil {
			if time.Since(info.ModTime()) < 7*24*time.Hour {
				return c.File(ogImagePath)
			}
		}
	}

	// Get price range
	priceRange, err := h.storage.Queries.GetProductPriceRange(ctx, productID)
	if err != nil {
		slog.Debug("failed to get price range, using base price", "error", err)
	}

	// Format price range - MinPrice/MaxPrice are interface{} from sqlc aggregate functions
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
	styleImages, err := h.storage.Queries.GetAllStylePrimaryImages(ctx, productID)
	if err != nil {
		slog.Debug("failed to get style images", "error", err)
	}

	// Build image paths (up to 4 for grid)
	var imagePaths []string
	var styleNames []string
	for i, img := range styleImages {
		if i >= 4 {
			break
		}
		imagePath := filepath.Join("public", "images", "products", "styles", img.ImageUrl)
		imagePaths = append(imagePaths, imagePath)
		styleNames = append(styleNames, img.StyleName)
	}

	// If no style images, try product images
	if len(imagePaths) == 0 {
		images, err := h.storage.Queries.GetProductImages(ctx, productID)
		if err == nil && len(images) > 0 {
			for i, img := range images {
				if i >= 4 {
					break
				}
				imagePath := filepath.Join("public", "images", "products", img.ImageUrl)
				imagePaths = append(imagePaths, imagePath)
			}
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

	err = ogimage.GenerateMultiVariantOGImage(info, ogImagePath)
	if err != nil {
		slog.Error("failed to generate multi-variant OG image", "error", err, "product_id", product.ID)
		return h.serveDefaultOGImage(c)
	}

	return c.File(ogImagePath)
}

// HandleDownloadCarouselImages generates a ZIP file containing individual style images
// for posting as an Instagram carousel (up to 10 images)
// Route: GET /api/carousel/:product_id
func (h *OGImageHandler) HandleDownloadCarouselImages(c echo.Context) error {
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

	// Get all style primary images
	styleImages, err := h.storage.Queries.GetAllStylePrimaryImages(ctx, productID)
	if err != nil {
		slog.Error("failed to get style images", "error", err, "product_id", productID)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load style images")
	}

	if len(styleImages) == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "No style images found for this product")
	}

	// Limit to 10 images (Instagram carousel max)
	maxImages := 10
	if len(styleImages) > maxImages {
		styleImages = styleImages[:maxImages]
	}

	// Create ZIP file in memory
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// Add each style image to the ZIP
	for i, img := range styleImages {
		imagePath := filepath.Join("public", "images", "products", "styles", img.ImageUrl)

		// Read the source image
		imageData, err := os.ReadFile(imagePath)
		if err != nil {
			slog.Debug("failed to read style image, skipping", "error", err, "path", imagePath)
			continue
		}

		// Create a safe filename: 01_StyleName.ext
		ext := filepath.Ext(img.ImageUrl)
		safeName := strings.ReplaceAll(img.StyleName, " ", "_")
		safeName = strings.ReplaceAll(safeName, "/", "-")
		filename := fmt.Sprintf("%02d_%s%s", i+1, safeName, ext)

		// Add to ZIP
		writer, err := zipWriter.Create(filename)
		if err != nil {
			slog.Debug("failed to create zip entry", "error", err, "filename", filename)
			continue
		}

		_, err = writer.Write(imageData)
		if err != nil {
			slog.Debug("failed to write to zip", "error", err, "filename", filename)
			continue
		}
	}

	// Close the ZIP writer
	if err := zipWriter.Close(); err != nil {
		slog.Error("failed to close zip writer", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create ZIP file")
	}

	// Generate filename for download
	safeProductName := strings.ReplaceAll(product.Name, " ", "_")
	safeProductName = strings.ReplaceAll(safeProductName, "/", "-")
	zipFilename := fmt.Sprintf("%s_instagram_carousel.zip", safeProductName)

	// Set headers for download
	c.Response().Header().Set("Content-Type", "application/zip")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", zipFilename))
	c.Response().Header().Set("Content-Length", fmt.Sprintf("%d", buf.Len()))

	return c.Blob(http.StatusOK, "application/zip", buf.Bytes())
}
