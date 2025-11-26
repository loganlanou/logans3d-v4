package handlers

import (
	"database/sql"
	"io"
	"log/slog"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/internal/utils"
	"github.com/loganlanou/logans3d-v4/storage/db"
	"github.com/loganlanou/logans3d-v4/views/admin"
)

func buildProductSkuViews(rows []db.GetProductSkusRow) []admin.ProductSkuView {
	results := make([]admin.ProductSkuView, 0, len(rows))

	for _, row := range rows {
		view := admin.ProductSkuView{
			ID:                   row.ID,
			SKU:                  row.Sku,
			Style:                row.StyleName,
			Size:                 row.SizeDisplayName,
			PriceAdjustmentCents: int64FromNull(row.PriceAdjustmentCents),
			Stock:                int64FromNull(row.StockQuantity),
			Active:               row.IsActive.Bool,
			StylePrimaryImage:    row.StylePrimaryImage,
		}
		results = append(results, view)
	}
	return results
}

// HandleCreateProductStyle lets admins add a product style with images
func (h *AdminHandler) HandleCreateProductStyle(c echo.Context) error {
	ctx := c.Request().Context()
	productID := c.Param("id")
	if productID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "product id is required")
	}

	product, err := h.storage.Queries.GetProduct(ctx, productID)
	if err != nil {
		slog.Error("product not found", "error", err, "product_id", productID)
		return echo.NewHTTPError(http.StatusBadRequest, "product not found")
	}

	styleName := strings.TrimSpace(c.FormValue("style_name"))
	if styleName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "style name is required")
	}
	isPrimary := c.FormValue("is_primary") == "on" || c.FormValue("is_primary") == "true" || c.FormValue("is_primary") == "1"

	form, err := c.MultipartForm()
	if err != nil {
		slog.Error("failed to read uploaded files", "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, "failed to read uploaded files")
	}
	files := form.File["style_images"]
	if len(files) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "at least one image upload is required")
	}

	// Clear other primaries if this will be primary
	if isPrimary {
		if err := h.storage.Queries.ClearPrimaryProductStyles(ctx, productID); err != nil {
			slog.Error("failed to clear primary styles", "error", err)
		}
	}

	// Create the product style
	styleID := uuid.New().String()
	if _, err := h.storage.Queries.CreateProductStyle(ctx, db.CreateProductStyleParams{
		ID:           styleID,
		ProductID:    productID,
		Name:         styleName,
		IsPrimary:    sql.NullBool{Bool: isPrimary, Valid: true},
		DisplayOrder: sql.NullInt64{Int64: 0, Valid: true},
	}); err != nil {
		slog.Error("failed to create style", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create style")
	}

	// Upload and attach images
	for idx, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			slog.Error("failed to open uploaded file", "error", err)
			return echo.NewHTTPError(http.StatusBadRequest, "failed to open uploaded file")
		}
		defer file.Close()

		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
		if ext == "" {
			ext = ".jpg"
		}
		filename := uuid.New().String() + ext
		destDir := filepath.Join("public", "images", "products", "styles")
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			slog.Error("failed to prepare upload directory", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to prepare upload directory")
		}
		destPath := filepath.Join(destDir, filename)

		dst, err := os.Create(destPath)
		if err != nil {
			slog.Error("failed to save uploaded file", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to save uploaded file")
		}
		if _, err := io.Copy(dst, file); err != nil {
			dst.Close()
			slog.Error("failed to write uploaded file", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to write uploaded file")
		}
		dst.Close()

		isFirstImage := idx == 0
		if _, err := h.storage.Queries.CreateProductStyleImage(ctx, db.CreateProductStyleImageParams{
			ID:             uuid.New().String(),
			ProductStyleID: styleID,
			ImageUrl:       filename,
			IsPrimary:      sql.NullBool{Bool: isFirstImage, Valid: true},
			DisplayOrder:   sql.NullInt64{Int64: int64(idx), Valid: true},
		}); err != nil {
			slog.Error("failed to attach style image", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to attach style image")
		}
	}

	// Ensure product is marked as variant-enabled
	_ = h.storage.Queries.SetProductVariantsFlag(ctx, db.SetProductVariantsFlagParams{
		HasVariants: sql.NullBool{Bool: true, Valid: true},
		ID:          productID,
	})

	// Auto-generate SKUs for all enabled product sizes
	productSizes, err := h.storage.Queries.GetProductSizeConfigs(ctx, productID)
	if err == nil && len(productSizes) > 0 {
		baseSku := product.Sku.String
		if baseSku == "" {
			baseSku = strings.ToUpper(product.Slug)
		}

		styleCode := normalizeVariantCode(styleName)

		for _, sizeConfig := range productSizes {
			// Check if SKU already exists for this style+size
			existingSku, err := h.storage.Queries.GetProductSkuByStyleAndSize(ctx, db.GetProductSkuByStyleAndSizeParams{
				ProductID:      productID,
				ProductStyleID: styleID,
				SizeID:         sizeConfig.SizeID,
			})
			if err == nil && existingSku.ID != "" {
				continue
			}

			priceAdjustment := int64(0)
			if sizeConfig.PriceAdjustmentCents.Valid {
				priceAdjustment = sizeConfig.PriceAdjustmentCents.Int64
			} else {
				priceAdjustment = sizeConfig.ChartDefaultAdjustment
			}

			skuCode := utils.GenerateSKU(baseSku, styleCode, sizeConfig.SizeName)
			if err := utils.ValidateSKU(ctx, h.storage.Queries, skuCode); err != nil {
				slog.Debug("SKU already exists, skipping", "sku", skuCode)
				continue
			}

			skuID := uuid.New().String()

			if _, err := h.storage.Queries.CreateProductSku(ctx, db.CreateProductSkuParams{
				ID:                   skuID,
				ProductID:            productID,
				ProductStyleID:       styleID,
				SizeID:               sizeConfig.SizeID,
				Sku:                  skuCode,
				PriceAdjustmentCents: sql.NullInt64{Int64: priceAdjustment, Valid: true},
				StockQuantity:        sql.NullInt64{Int64: 0, Valid: true},
				IsActive:             sql.NullBool{Bool: true, Valid: true},
			}); err != nil {
				slog.Error("failed to create auto-generated SKU", "error", err, "sku", skuCode)
				continue
			}
		}
	}

	// Check if this is an HTMX request
	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Style added successfully!", "type": "success"}}`)
		c.Response().Header().Set("HX-Refresh", "true")
		return c.NoContent(http.StatusOK)
	}

	return c.Redirect(http.StatusSeeOther, "/admin/product/edit?id="+productID+"#variants")
}

// HandleDeleteProductStyle removes a style and its images/SKUs
func (h *AdminHandler) HandleDeleteProductStyle(c echo.Context) error {
	ctx := c.Request().Context()
	styleID := c.Param("styleId")
	productID := c.QueryParam("product_id")

	if styleID == "" || productID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing style or product")
	}

	// Delete SKUs for this style first
	if err := h.storage.Queries.DeleteProductSkusByStyle(ctx, styleID); err != nil {
		slog.Error("failed to delete SKUs for style", "error", err, "style_id", styleID)
	}

	// Delete the style (cascades to images)
	if err := h.storage.Queries.DeleteProductStyle(ctx, styleID); err != nil {
		slog.Error("failed to delete style", "error", err, "style_id", styleID)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete style")
	}

	return c.Redirect(http.StatusSeeOther, "/admin/product/edit?id="+productID+"#variants")
}

// HandleSetPrimaryStyle sets a style as the primary for a product
func (h *AdminHandler) HandleSetPrimaryStyle(c echo.Context) error {
	ctx := c.Request().Context()
	styleID := c.Param("styleId")
	productID := c.QueryParam("product_id")

	if styleID == "" || productID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing style or product")
	}

	style, err := h.storage.Queries.GetProductStyle(ctx, styleID)
	if err != nil {
		slog.Error("style not found", "error", err, "style_id", styleID)
		return echo.NewHTTPError(http.StatusBadRequest, "style not found")
	}

	if err := h.storage.Queries.ClearPrimaryProductStyles(ctx, style.ProductID); err != nil {
		slog.Error("failed to clear existing primaries", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to clear existing primaries")
	}

	if err := h.storage.Queries.SetPrimaryProductStyle(ctx, styleID); err != nil {
		slog.Error("failed to set primary style", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to set primary style")
	}

	return c.Redirect(http.StatusSeeOther, "/admin/product/edit?id="+productID+"#variants")
}

// HandleSetPrimaryStyleImage marks a specific style image as primary
func (h *AdminHandler) HandleSetPrimaryStyleImage(c echo.Context) error {
	ctx := c.Request().Context()
	imageID := c.Param("imageId")
	productID := c.QueryParam("product_id")

	if imageID == "" || productID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing image or product")
	}

	img, err := h.storage.Queries.GetProductStyleImage(ctx, imageID)
	if err != nil {
		slog.Error("image not found", "error", err, "image_id", imageID)
		return echo.NewHTTPError(http.StatusBadRequest, "image not found")
	}

	if err := h.storage.Queries.ClearPrimaryStyleImages(ctx, img.ProductStyleID); err != nil {
		slog.Error("failed to clear existing primary", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to clear existing primary")
	}
	if err := h.storage.Queries.SetPrimaryStyleImage(ctx, imageID); err != nil {
		slog.Error("failed to set primary", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to set primary")
	}

	return c.Redirect(http.StatusSeeOther, "/admin/product/edit?id="+productID+"#variants")
}

// HandleDeleteStyleImage removes a style image
func (h *AdminHandler) HandleDeleteStyleImage(c echo.Context) error {
	ctx := c.Request().Context()
	imageID := c.Param("imageId")
	productID := c.QueryParam("product_id")

	if imageID == "" || productID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing image or product")
	}

	if err := h.storage.Queries.DeleteProductStyleImage(ctx, imageID); err != nil {
		slog.Error("failed to delete style image", "error", err, "image_id", imageID)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete image")
	}

	return c.Redirect(http.StatusSeeOther, "/admin/product/edit?id="+productID+"#variants")
}

// HandleSaveProductSizes saves the product-level size configuration
func (h *AdminHandler) HandleSaveProductSizes(c echo.Context) error {
	ctx := c.Request().Context()
	productID := c.Param("id")
	if productID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "product id is required")
	}

	if _, err := h.storage.Queries.GetProduct(ctx, productID); err != nil {
		slog.Error("product not found", "error", err, "product_id", productID)
		return echo.NewHTTPError(http.StatusBadRequest, "product not found")
	}

	allSizes, err := h.storage.Queries.GetSizeCharts(ctx)
	if err != nil {
		slog.Error("failed to load sizes", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load sizes")
	}

	formValues, err := c.FormParams()
	if err != nil {
		slog.Error("failed to parse form", "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, "failed to parse form")
	}

	selectedSizes := make(map[string]bool)
	for _, sizeID := range formValues["size_enabled"] {
		selectedSizes[sizeID] = true
	}

	for displayOrder, size := range allSizes {
		if selectedSizes[size.SizeID] {
			var priceAdjustment sql.NullInt64
			if adjustmentStr := c.FormValue("price_adjustment_" + size.SizeID); adjustmentStr != "" {
				if cents, err := parseCurrencyToCents(adjustmentStr); err == nil {
					priceAdjustment = sql.NullInt64{Int64: cents, Valid: true}
				}
			}

			if _, err := h.storage.Queries.UpsertProductSizeConfig(ctx, db.UpsertProductSizeConfigParams{
				ID:                   uuid.New().String(),
				ProductID:            productID,
				SizeID:               size.SizeID,
				PriceAdjustmentCents: priceAdjustment,
				IsEnabled:            sql.NullBool{Bool: true, Valid: true},
				DisplayOrder:         sql.NullInt64{Int64: int64(displayOrder), Valid: true},
			}); err != nil {
				slog.Error("failed to save size config", "error", err)
				return echo.NewHTTPError(http.StatusInternalServerError, "failed to save size config")
			}
		} else {
			_ = h.storage.Queries.DeleteProductSizeConfig(ctx, db.DeleteProductSizeConfigParams{
				ProductID: productID,
				SizeID:    size.SizeID,
			})
		}
	}

	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Product sizes saved!", "type": "success"}}`)
		return c.NoContent(http.StatusOK)
	}

	return c.Redirect(http.StatusSeeOther, "/admin/product/edit?id="+productID+"#variants")
}

// HandleCreateProductSKU creates a SKU with style + size (manual mode)
func (h *AdminHandler) HandleCreateProductSKU(c echo.Context) error {
	ctx := c.Request().Context()
	productID := c.Param("id")
	if productID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "product id is required")
	}

	styleID := c.FormValue("style_id")
	sizeID := c.FormValue("size_id")
	priceAdjustmentStr := c.FormValue("price_adjustment")
	stockStr := c.FormValue("stock_quantity")
	customSku := c.FormValue("sku")
	isActive := c.FormValue("is_active") != "off"

	product, err := h.storage.Queries.GetProduct(ctx, productID)
	if err != nil {
		slog.Error("product not found", "error", err, "product_id", productID)
		return echo.NewHTTPError(http.StatusBadRequest, "product not found")
	}

	style, err := h.storage.Queries.GetProductStyle(ctx, styleID)
	if err != nil {
		slog.Error("invalid style selection", "error", err, "style_id", styleID)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid style selection")
	}

	size, err := h.storage.Queries.GetSize(ctx, sizeID)
	if err != nil {
		slog.Error("invalid size selection", "error", err, "size_id", sizeID)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid size selection")
	}

	priceAdjustment := int64(0)
	if priceAdjustmentStr != "" {
		if v, err := parseCurrencyToCents(priceAdjustmentStr); err == nil {
			priceAdjustment = v
		}
	}

	stockQuantity := int64(0)
	if stockStr != "" {
		if v, err := strconv.ParseInt(stockStr, 10, 64); err == nil {
			stockQuantity = v
		}
	}

	skuCode := customSku
	if skuCode == "" {
		baseSku := product.Sku.String
		if baseSku == "" {
			baseSku = strings.ToUpper(product.Slug)
		}
		styleCode := normalizeVariantCode(style.Name)
		skuCode = utils.GenerateSKU(baseSku, styleCode, size.Name)
	}

	if err := utils.ValidateSKU(ctx, h.storage.Queries, skuCode); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	skuID := uuid.New().String()

	if _, err := h.storage.Queries.CreateProductSku(ctx, db.CreateProductSkuParams{
		ID:                   skuID,
		ProductID:            productID,
		ProductStyleID:       styleID,
		SizeID:               sizeID,
		Sku:                  skuCode,
		PriceAdjustmentCents: sql.NullInt64{Int64: priceAdjustment, Valid: true},
		StockQuantity:        sql.NullInt64{Int64: stockQuantity, Valid: true},
		IsActive:             sql.NullBool{Bool: isActive, Valid: true},
	}); err != nil {
		slog.Error("failed to create SKU", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create SKU")
	}

	// Ensure product is marked as variant-enabled
	if err := h.storage.Queries.SetProductVariantsFlag(ctx, db.SetProductVariantsFlagParams{
		HasVariants: sql.NullBool{Bool: true, Valid: true},
		ID:          productID,
	}); err != nil {
		slog.Error("failed to update product variant flag", "error", err)
	}

	return c.Redirect(http.StatusSeeOther, "/admin/product/edit?id="+productID)
}

// HandleDeleteProductSku deletes a SKU
func (h *AdminHandler) HandleDeleteProductSku(c echo.Context) error {
	ctx := c.Request().Context()
	skuID := c.Param("skuId")
	productID := c.QueryParam("product_id")

	if skuID == "" || productID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing sku or product")
	}

	if err := h.storage.Queries.DeleteProductSku(ctx, skuID); err != nil {
		slog.Error("failed to delete SKU", "error", err, "sku_id", skuID)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete SKU")
	}

	return c.Redirect(http.StatusSeeOther, "/admin/product/edit?id="+productID)
}

func int64FromNull(n sql.NullInt64) int64 {
	if n.Valid {
		return n.Int64
	}
	return 0
}

func float64FromNull(n sql.NullFloat64) float64 {
	if n.Valid {
		return n.Float64
	}
	return 0
}

func parseCurrencyToCents(value string) (int64, error) {
	val, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, err
	}
	return int64(math.Round(val * 100)), nil
}

func normalizeVariantCode(value string) string {
	if value == "" {
		return ""
	}
	slug := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
			return unicode.ToLower(r)
		}
		if unicode.IsSpace(r) {
			return '-'
		}
		if r == '_' {
			return '-'
		}
		return -1
	}, strings.TrimSpace(value))

	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "style"
	}
	return slug
}

func parseFloat64(value string) float64 {
	if value == "" {
		return 0
	}
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return v
}

// ============================================
// Style Panel handlers (for admin UI)
// ============================================

// HandleGetStylePanel returns the style detail panel HTML
func (h *AdminHandler) HandleGetStylePanel(c echo.Context) error {
	ctx := c.Request().Context()
	styleID := c.Param("styleId")
	productID := c.QueryParam("product_id")

	if styleID == "" || productID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing style or product")
	}

	style, err := h.storage.Queries.GetProductStyle(ctx, styleID)
	if err != nil {
		slog.Error("style not found", "error", err, "style_id", styleID)
		return echo.NewHTTPError(http.StatusNotFound, "style not found")
	}

	images, err := h.storage.Queries.GetProductStyleImages(ctx, styleID)
	if err != nil {
		slog.Error("failed to load style images", "error", err)
		images = []db.ProductStyleImage{}
	}

	skus, err := h.storage.Queries.GetStyleSkus(ctx, styleID)
	if err != nil {
		slog.Error("failed to load style SKUs", "error", err)
		skus = []db.GetStyleSkusRow{}
	}

	availableSizes, err := h.storage.Queries.GetAvailableSizesForStyle(ctx, db.GetAvailableSizesForStyleParams{
		ProductID:      productID,
		ProductStyleID: styleID,
	})
	if err != nil {
		slog.Error("failed to load available sizes", "error", err)
		availableSizes = []db.Size{}
	}

	panelData := admin.StylePanelData{
		ProductID:      productID,
		Style:          style,
		Images:         images,
		Skus:           skus,
		AvailableSizes: availableSizes,
	}

	return Render(c, admin.StylePanel(panelData))
}

// HandleSetPrimaryStyleImageFromPanel sets a style image as primary and returns refreshed panel + OOB card
func (h *AdminHandler) HandleSetPrimaryStyleImageFromPanel(c echo.Context) error {
	ctx := c.Request().Context()
	imageID := c.Param("imageId")
	productID := c.QueryParam("product_id")
	styleID := c.QueryParam("style_id")

	if imageID == "" || productID == "" || styleID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing image, product, or style")
	}

	img, err := h.storage.Queries.GetProductStyleImage(ctx, imageID)
	if err != nil {
		slog.Error("image not found", "error", err, "image_id", imageID)
		return echo.NewHTTPError(http.StatusBadRequest, "image not found")
	}

	if err := h.storage.Queries.ClearPrimaryStyleImages(ctx, img.ProductStyleID); err != nil {
		slog.Error("failed to clear existing primary", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to clear existing primary")
	}
	if err := h.storage.Queries.SetPrimaryStyleImage(ctx, imageID); err != nil {
		slog.Error("failed to set primary", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to set primary")
	}

	c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Primary image updated", "type": "success"}}`)

	return h.renderStylePanelWithOOBCard(c, productID, styleID)
}

// renderStylePanelWithOOBCard is a helper that renders both the style panel and an OOB style card update
func (h *AdminHandler) renderStylePanelWithOOBCard(c echo.Context, productID, styleID string) error {
	ctx := c.Request().Context()

	style, err := h.storage.Queries.GetProductStyle(ctx, styleID)
	if err != nil {
		slog.Error("style not found", "error", err, "style_id", styleID)
		return echo.NewHTTPError(http.StatusNotFound, "style not found")
	}

	images, err := h.storage.Queries.GetProductStyleImages(ctx, styleID)
	if err != nil {
		slog.Error("failed to load style images", "error", err)
		images = []db.ProductStyleImage{}
	}

	skus, err := h.storage.Queries.GetStyleSkus(ctx, styleID)
	if err != nil {
		slog.Error("failed to load style SKUs", "error", err)
		skus = []db.GetStyleSkusRow{}
	}

	availableSizes, err := h.storage.Queries.GetAvailableSizesForStyle(ctx, db.GetAvailableSizesForStyleParams{
		ProductID:      productID,
		ProductStyleID: styleID,
	})
	if err != nil {
		slog.Error("failed to load available sizes", "error", err)
		availableSizes = []db.Size{}
	}

	// Get the primary image URL for the card
	primaryImage, err := h.storage.Queries.GetPrimaryStyleImage(ctx, styleID)
	primaryImageURL := ""
	if err == nil {
		primaryImageURL = primaryImage.ImageUrl
	}

	panelData := admin.StylePanelData{
		ProductID:      productID,
		Style:          style,
		Images:         images,
		Skus:           skus,
		AvailableSizes: availableSizes,
	}

	cardData := admin.StyleCardData{
		StyleID:      styleID,
		ProductID:    productID,
		Name:         style.Name,
		IsPrimary:    style.IsPrimary.Valid && style.IsPrimary.Bool,
		PrimaryImage: primaryImageURL,
		ImageCount:   len(images),
	}

	return Render(c, admin.StylePanelWithOOBCard(panelData, cardData))
}

// HandleDeleteStyleImageFromPanel deletes a style image and returns refreshed panel + OOB card
func (h *AdminHandler) HandleDeleteStyleImageFromPanel(c echo.Context) error {
	ctx := c.Request().Context()
	imageID := c.Param("imageId")
	productID := c.QueryParam("product_id")
	styleID := c.QueryParam("style_id")

	if imageID == "" || productID == "" || styleID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing image, product, or style")
	}

	img, err := h.storage.Queries.GetProductStyleImage(ctx, imageID)
	if err != nil {
		slog.Error("image not found", "error", err, "image_id", imageID)
		return echo.NewHTTPError(http.StatusBadRequest, "image not found")
	}

	// Delete the file from disk
	filePath := filepath.Join("public", "images", "products", "styles", img.ImageUrl)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		slog.Error("failed to delete image file", "error", err, "path", filePath)
	}

	// Delete from database
	if err := h.storage.Queries.DeleteProductStyleImage(ctx, imageID); err != nil {
		slog.Error("failed to delete style image", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete image")
	}

	c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Image deleted", "type": "success"}}`)

	return h.renderStylePanelWithOOBCard(c, productID, styleID)
}

// HandleAddStyleImages uploads additional images to an existing style
func (h *AdminHandler) HandleAddStyleImages(c echo.Context) error {
	ctx := c.Request().Context()
	styleID := c.Param("styleId")
	productID := c.QueryParam("product_id")

	if styleID == "" || productID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing style or product")
	}

	style, err := h.storage.Queries.GetProductStyle(ctx, styleID)
	if err != nil {
		slog.Error("style not found", "error", err, "style_id", styleID)
		return echo.NewHTTPError(http.StatusNotFound, "style not found")
	}

	form, err := c.MultipartForm()
	if err != nil {
		slog.Error("failed to parse multipart form", "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, "failed to parse form")
	}

	files := form.File["images"]
	if len(files) == 0 {
		c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "No images selected", "type": "error"}}`)
		c.SetParamNames("styleId")
		c.SetParamValues(styleID)
		return h.HandleGetStylePanel(c)
	}

	uploadDir := filepath.Join("public", "images", "products", "styles")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		slog.Error("failed to create upload directory", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create upload directory")
	}

	// Check if style already has images (to determine if new ones should be primary)
	existingImages, _ := h.storage.Queries.GetProductStyleImages(ctx, styleID)
	hasPrimary := false
	for _, img := range existingImages {
		if img.IsPrimary.Valid && img.IsPrimary.Bool {
			hasPrimary = true
			break
		}
	}

	for i, file := range files {
		src, err := file.Open()
		if err != nil {
			slog.Error("failed to open uploaded file", "error", err)
			continue
		}

		ext := filepath.Ext(file.Filename)
		newFilename := uuid.New().String() + ext
		dstPath := filepath.Join(uploadDir, newFilename)

		dst, err := os.Create(dstPath)
		if err != nil {
			src.Close()
			slog.Error("failed to create destination file", "error", err)
			continue
		}

		if _, err := io.Copy(dst, src); err != nil {
			src.Close()
			dst.Close()
			slog.Error("failed to copy file", "error", err)
			continue
		}
		src.Close()
		dst.Close()

		// First image becomes primary if no existing primary
		isPrimary := !hasPrimary && i == 0

		_, err = h.storage.Queries.CreateProductStyleImage(ctx, db.CreateProductStyleImageParams{
			ID:             uuid.New().String(),
			ProductStyleID: style.ID,
			ImageUrl:       newFilename,
			IsPrimary:      sql.NullBool{Bool: isPrimary, Valid: true},
			DisplayOrder:   sql.NullInt64{Int64: int64(len(existingImages) + i), Valid: true},
		})
		if err != nil {
			slog.Error("failed to save style image", "error", err)
			continue
		}
	}

	c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Images added", "type": "success"}}`)

	return h.renderStylePanelWithOOBCard(c, productID, styleID)
}

// HandleUpdateSkuPrice updates just the price adjustment for a SKU
func (h *AdminHandler) HandleUpdateSkuPrice(c echo.Context) error {
	ctx := c.Request().Context()
	skuID := c.Param("skuId")
	productID := c.QueryParam("product_id")
	styleID := c.QueryParam("style_id")

	if skuID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing SKU ID")
	}

	priceStr := c.FormValue("price_adjustment")
	priceCents, err := parseCurrencyToCents(priceStr)
	if err != nil {
		slog.Error("invalid price format", "error", err, "value", priceStr)
		c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Invalid price format", "type": "error"}}`)
		return c.NoContent(http.StatusBadRequest)
	}

	if err := h.storage.Queries.UpdateSkuPrice(ctx, db.UpdateSkuPriceParams{
		PriceAdjustmentCents: sql.NullInt64{Int64: priceCents, Valid: true},
		ID:                   skuID,
	}); err != nil {
		slog.Error("failed to update SKU price", "error", err, "sku_id", skuID)
		c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Failed to update price", "type": "error"}}`)
		return c.NoContent(http.StatusInternalServerError)
	}

	c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Price updated", "type": "success"}}`)

	// If we have style_id and product_id, refresh the panel
	if styleID != "" && productID != "" {
		return h.HandleGetStylePanel(c)
	}
	return c.NoContent(http.StatusOK)
}

// HandleUpdateSkuStock updates just the stock quantity for a SKU
func (h *AdminHandler) HandleUpdateSkuStock(c echo.Context) error {
	ctx := c.Request().Context()
	skuID := c.Param("skuId")
	productID := c.QueryParam("product_id")
	styleID := c.QueryParam("style_id")

	if skuID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing SKU ID")
	}

	stockStr := c.FormValue("stock_quantity")
	stock, err := strconv.ParseInt(stockStr, 10, 64)
	if err != nil {
		slog.Error("invalid stock format", "error", err, "value", stockStr)
		c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Invalid stock format", "type": "error"}}`)
		return c.NoContent(http.StatusBadRequest)
	}

	if err := h.storage.Queries.UpdateSkuStock(ctx, db.UpdateSkuStockParams{
		StockQuantity: sql.NullInt64{Int64: stock, Valid: true},
		ID:            skuID,
	}); err != nil {
		slog.Error("failed to update SKU stock", "error", err, "sku_id", skuID)
		c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Failed to update stock", "type": "error"}}`)
		return c.NoContent(http.StatusInternalServerError)
	}

	c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Stock updated", "type": "success"}}`)

	// If we have style_id and product_id, refresh the panel
	if styleID != "" && productID != "" {
		return h.HandleGetStylePanel(c)
	}
	return c.NoContent(http.StatusOK)
}

// HandleToggleSkuActive toggles the active status of a SKU
func (h *AdminHandler) HandleToggleSkuActive(c echo.Context) error {
	ctx := c.Request().Context()
	skuID := c.Param("skuId")

	if skuID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing SKU ID")
	}

	isActiveStr := c.FormValue("is_active")
	isActive := isActiveStr == "true" || isActiveStr == "on" || isActiveStr == "1"

	if err := h.storage.Queries.UpdateSkuActive(ctx, db.UpdateSkuActiveParams{
		IsActive: sql.NullBool{Bool: isActive, Valid: true},
		ID:       skuID,
	}); err != nil {
		slog.Error("failed to update SKU active status", "error", err, "sku_id", skuID)
		c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Failed to update status", "type": "error"}}`)
		return c.NoContent(http.StatusInternalServerError)
	}

	statusText := "deactivated"
	if isActive {
		statusText = "activated"
	}
	c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "SKU `+statusText+`", "type": "success"}}`)
	return c.NoContent(http.StatusOK)
}

// HandleAddStyleSku adds a single SKU for a specific size to a style
func (h *AdminHandler) HandleAddStyleSku(c echo.Context) error {
	ctx := c.Request().Context()
	styleID := c.Param("styleId")
	productID := c.QueryParam("product_id")
	sizeID := c.FormValue("size_id")

	if styleID == "" || productID == "" || sizeID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing required parameters")
	}

	product, err := h.storage.Queries.GetProduct(ctx, productID)
	if err != nil {
		slog.Error("product not found", "error", err, "product_id", productID)
		return echo.NewHTTPError(http.StatusBadRequest, "product not found")
	}

	style, err := h.storage.Queries.GetProductStyle(ctx, styleID)
	if err != nil {
		slog.Error("style not found", "error", err, "style_id", styleID)
		return echo.NewHTTPError(http.StatusBadRequest, "style not found")
	}

	size, err := h.storage.Queries.GetSize(ctx, sizeID)
	if err != nil {
		slog.Error("size not found", "error", err, "size_id", sizeID)
		return echo.NewHTTPError(http.StatusBadRequest, "size not found")
	}

	// Get price adjustment from size chart
	sizeChart, _ := h.storage.Queries.GetSizeChart(ctx, sizeID)
	priceAdjustment := int64(0)
	if sizeChart.DefaultPriceAdjustmentCents.Valid {
		priceAdjustment = sizeChart.DefaultPriceAdjustmentCents.Int64
	}

	// Generate SKU code
	baseSku := product.Sku.String
	if baseSku == "" {
		baseSku = strings.ToUpper(product.Slug)
	}
	styleCode := normalizeVariantCode(style.Name)
	skuCode := utils.GenerateSKU(baseSku, styleCode, size.Name)

	if err := utils.ValidateSKU(ctx, h.storage.Queries, skuCode); err != nil {
		c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "SKU already exists", "type": "error"}}`)
		return h.HandleGetStylePanel(c)
	}

	skuID := uuid.New().String()
	if _, err := h.storage.Queries.CreateProductSku(ctx, db.CreateProductSkuParams{
		ID:                   skuID,
		ProductID:            productID,
		ProductStyleID:       styleID,
		SizeID:               sizeID,
		Sku:                  skuCode,
		PriceAdjustmentCents: sql.NullInt64{Int64: priceAdjustment, Valid: true},
		StockQuantity:        sql.NullInt64{Int64: 0, Valid: true},
		IsActive:             sql.NullBool{Bool: true, Valid: true},
	}); err != nil {
		slog.Error("failed to create SKU", "error", err)
		c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Failed to create SKU", "type": "error"}}`)
		return h.HandleGetStylePanel(c)
	}

	c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "SKU created", "type": "success"}}`)
	return h.HandleGetStylePanel(c)
}

// HandleDeleteSkuFromPanel deletes a SKU and returns refreshed panel
func (h *AdminHandler) HandleDeleteSkuFromPanel(c echo.Context) error {
	ctx := c.Request().Context()
	skuID := c.Param("skuId")
	productID := c.QueryParam("product_id")
	styleID := c.QueryParam("style_id")

	if skuID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing SKU ID")
	}

	if err := h.storage.Queries.DeleteProductSku(ctx, skuID); err != nil {
		slog.Error("failed to delete SKU", "error", err, "sku_id", skuID)
		c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Failed to delete SKU", "type": "error"}}`)
		if styleID != "" && productID != "" {
			return h.HandleGetStylePanel(c)
		}
		return c.NoContent(http.StatusInternalServerError)
	}

	c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "SKU deleted", "type": "success"}}`)

	// Refresh the panel
	if styleID != "" && productID != "" {
		return h.HandleGetStylePanel(c)
	}
	return c.NoContent(http.StatusOK)
}

// HandleSetPrimaryStyleFromPanel sets a style as primary and refreshes the page
func (h *AdminHandler) HandleSetPrimaryStyleFromPanel(c echo.Context) error {
	ctx := c.Request().Context()
	styleID := c.Param("styleId")
	productID := c.QueryParam("product_id")

	if styleID == "" || productID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing style or product")
	}

	style, err := h.storage.Queries.GetProductStyle(ctx, styleID)
	if err != nil {
		slog.Error("style not found", "error", err, "style_id", styleID)
		return echo.NewHTTPError(http.StatusBadRequest, "style not found")
	}

	if err := h.storage.Queries.ClearPrimaryProductStyles(ctx, style.ProductID); err != nil {
		slog.Error("failed to clear existing primaries", "error", err)
	}

	if err := h.storage.Queries.SetPrimaryProductStyle(ctx, styleID); err != nil {
		slog.Error("failed to set primary style", "error", err)
		c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Failed to set primary", "type": "error"}}`)
		return c.NoContent(http.StatusInternalServerError)
	}

	c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Primary style updated", "type": "success"}}`)
	c.Response().Header().Set("HX-Refresh", "true")
	return c.NoContent(http.StatusOK)
}

// HandleDeleteStyleFromPanel deletes a style and refreshes the page
func (h *AdminHandler) HandleDeleteStyleFromPanel(c echo.Context) error {
	ctx := c.Request().Context()
	styleID := c.Param("styleId")
	productID := c.QueryParam("product_id")

	if styleID == "" || productID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing style or product")
	}

	// Delete SKUs for this style first
	if err := h.storage.Queries.DeleteProductSkusByStyle(ctx, styleID); err != nil {
		slog.Error("failed to delete SKUs for style", "error", err, "style_id", styleID)
	}

	// Delete the style (cascades to images)
	if err := h.storage.Queries.DeleteProductStyle(ctx, styleID); err != nil {
		slog.Error("failed to delete style", "error", err, "style_id", styleID)
		c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Failed to delete style", "type": "error"}}`)
		return c.NoContent(http.StatusInternalServerError)
	}

	c.Response().Header().Set("HX-Trigger", `{"showToast": {"message": "Style deleted", "type": "success"}}`)
	c.Response().Header().Set("HX-Refresh", "true")
	return c.NoContent(http.StatusOK)
}
