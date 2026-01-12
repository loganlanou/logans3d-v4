package handlers

import (
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/internal/auth"
	"github.com/loganlanou/logans3d-v4/storage"
	"github.com/loganlanou/logans3d-v4/storage/db"
)

type APIProductsHandler struct {
	store *storage.Storage
}

func NewAPIProductsHandler(store *storage.Storage) *APIProductsHandler {
	return &APIProductsHandler{store: store}
}

type CreateProductRequest struct {
	Name             string   `json:"name"`
	Slug             string   `json:"slug"`
	Description      string   `json:"description"`
	ShortDescription string   `json:"short_description"`
	PriceCents       int64    `json:"price_cents"`
	CategoryID       string   `json:"category_id"`
	SKU              string   `json:"sku"`
	StockQuantity    int64    `json:"stock_quantity"`
	WeightGrams      int64    `json:"weight_grams"`
	LeadTimeDays     int64    `json:"lead_time_days"`
	IsActive         bool     `json:"is_active"`
	IsFeatured       bool     `json:"is_featured"`
	IsPremium        bool     `json:"is_premium"`
	IsNew            bool     `json:"is_new"`
	Disclaimer       string   `json:"disclaimer"`
	SEOTitle         string   `json:"seo_title"`
	SEODescription   string   `json:"seo_description"`
	SEOKeywords      string   `json:"seo_keywords"`
	OGImageURL       string   `json:"og_image_url"`
	SourceURL        string   `json:"source_url"`
	SourcePlatform   string   `json:"source_platform"`
	DesignerName     string   `json:"designer_name"`
	ReleaseDate      *string  `json:"release_date"`
	Tags             []string `json:"tags"`
}

type ProductResponse struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Slug             string     `json:"slug"`
	Description      string     `json:"description"`
	ShortDescription string     `json:"short_description"`
	PriceCents       int64      `json:"price_cents"`
	CategoryID       string     `json:"category_id"`
	SKU              string     `json:"sku"`
	StockQuantity    int64      `json:"stock_quantity"`
	WeightGrams      int64      `json:"weight_grams"`
	LeadTimeDays     int64      `json:"lead_time_days"`
	IsActive         bool       `json:"is_active"`
	IsFeatured       bool       `json:"is_featured"`
	IsPremium        bool       `json:"is_premium"`
	IsNew            bool       `json:"is_new"`
	SourceURL        string     `json:"source_url,omitempty"`
	SourcePlatform   string     `json:"source_platform,omitempty"`
	DesignerName     string     `json:"designer_name,omitempty"`
	ReleaseDate      *time.Time `json:"release_date,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (h *APIProductsHandler) ListProducts(c echo.Context) error {
	apiKey := auth.GetAPIKeyInfo(c.Request().Context())
	if apiKey == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "API key required")
	}

	if !apiKey.HasPermission("products:read") {
		return echo.NewHTTPError(http.StatusForbidden, "Permission denied: products:read required")
	}

	products, err := h.store.Queries.ListAllProducts(c.Request().Context())
	if err != nil {
		slog.Error("failed to list products", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to list products")
	}

	response := make([]ProductResponse, len(products))
	for i, p := range products {
		response[i] = productToResponse(p)
	}

	return c.JSON(http.StatusOK, response)
}

func (h *APIProductsHandler) GetProduct(c echo.Context) error {
	apiKey := auth.GetAPIKeyInfo(c.Request().Context())
	if apiKey == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "API key required")
	}

	if !apiKey.HasPermission("products:read") {
		return echo.NewHTTPError(http.StatusForbidden, "Permission denied: products:read required")
	}

	id := c.Param("id")
	product, err := h.store.Queries.GetProduct(c.Request().Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "Product not found")
		}
		slog.Error("failed to get product", "error", err, "id", id)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get product")
	}

	return c.JSON(http.StatusOK, productToResponse(product))
}

func (h *APIProductsHandler) GetProductBySourceURL(c echo.Context) error {
	apiKey := auth.GetAPIKeyInfo(c.Request().Context())
	if apiKey == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "API key required")
	}

	if !apiKey.HasPermission("products:read") {
		return echo.NewHTTPError(http.StatusForbidden, "Permission denied: products:read required")
	}

	sourceURL := c.QueryParam("source_url")
	if sourceURL == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "source_url query parameter required")
	}

	product, err := h.store.Queries.GetProductBySourceURL(c.Request().Context(), sql.NullString{String: sourceURL, Valid: true})
	if err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "Product not found")
		}
		slog.Error("failed to get product by source URL", "error", err, "source_url", sourceURL)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get product")
	}

	return c.JSON(http.StatusOK, productToResponse(product))
}

func (h *APIProductsHandler) CreateProduct(c echo.Context) error {
	apiKey := auth.GetAPIKeyInfo(c.Request().Context())
	if apiKey == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "API key required")
	}

	if !apiKey.HasPermission("products:write") {
		return echo.NewHTTPError(http.StatusForbidden, "Permission denied: products:write required")
	}

	var req CreateProductRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Name is required")
	}
	if req.CategoryID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Category ID is required")
	}

	if req.Slug == "" {
		req.Slug = generateSlug(req.Name)
	}

	id := uuid.New().String()

	var releaseDate sql.NullTime
	if req.ReleaseDate != nil && *req.ReleaseDate != "" {
		t, err := time.Parse(time.RFC3339, *req.ReleaseDate)
		if err != nil {
			t, err = time.Parse("2006-01-02", *req.ReleaseDate)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid release_date format. Use RFC3339 or YYYY-MM-DD")
			}
		}
		releaseDate = sql.NullTime{Time: t, Valid: true}
	}

	params := db.CreateProductWithSourceParams{
		ID:               id,
		Name:             req.Name,
		Slug:             req.Slug,
		Description:      sql.NullString{String: req.Description, Valid: req.Description != ""},
		ShortDescription: sql.NullString{String: req.ShortDescription, Valid: req.ShortDescription != ""},
		PriceCents:       req.PriceCents,
		CategoryID:       sql.NullString{String: req.CategoryID, Valid: true},
		Sku:              sql.NullString{String: req.SKU, Valid: req.SKU != ""},
		StockQuantity:    sql.NullInt64{Int64: req.StockQuantity, Valid: true},
		HasVariants:      sql.NullBool{Bool: false, Valid: true},
		WeightGrams:      sql.NullInt64{Int64: req.WeightGrams, Valid: req.WeightGrams > 0},
		LeadTimeDays:     sql.NullInt64{Int64: req.LeadTimeDays, Valid: req.LeadTimeDays > 0},
		IsActive:         sql.NullBool{Bool: req.IsActive, Valid: true},
		IsFeatured:       sql.NullBool{Bool: req.IsFeatured, Valid: true},
		IsPremium:        sql.NullBool{Bool: req.IsPremium, Valid: true},
		IsNew:            sql.NullBool{Bool: req.IsNew, Valid: true},
		Disclaimer:       sql.NullString{String: req.Disclaimer, Valid: req.Disclaimer != ""},
		SeoTitle:         sql.NullString{String: req.SEOTitle, Valid: req.SEOTitle != ""},
		SeoDescription:   sql.NullString{String: req.SEODescription, Valid: req.SEODescription != ""},
		SeoKeywords:      sql.NullString{String: req.SEOKeywords, Valid: req.SEOKeywords != ""},
		OgImageUrl:       sql.NullString{String: req.OGImageURL, Valid: req.OGImageURL != ""},
		SourceUrl:        sql.NullString{String: req.SourceURL, Valid: req.SourceURL != ""},
		SourcePlatform:   sql.NullString{String: req.SourcePlatform, Valid: req.SourcePlatform != ""},
		DesignerName:     sql.NullString{String: req.DesignerName, Valid: req.DesignerName != ""},
		ReleaseDate:      releaseDate,
	}

	product, err := h.store.Queries.CreateProductWithSource(c.Request().Context(), params)
	if err != nil {
		slog.Error("failed to create product", "error", err, "name", req.Name)
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return echo.NewHTTPError(http.StatusConflict, "Product with this name or slug already exists")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create product")
	}

	if len(req.Tags) > 0 {
		for _, tagID := range req.Tags {
			err := h.store.Queries.AddProductTag(c.Request().Context(), db.AddProductTagParams{
				ProductID: id,
				TagID:     tagID,
			})
			if err != nil {
				slog.Warn("failed to add tag to product", "error", err, "product_id", id, "tag_id", tagID)
			}
		}
	}

	return c.JSON(http.StatusCreated, productToResponse(product))
}

func (h *APIProductsHandler) DeleteProduct(c echo.Context) error {
	apiKey := auth.GetAPIKeyInfo(c.Request().Context())
	if apiKey == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "API key required")
	}

	if !apiKey.HasPermission("products:write") {
		return echo.NewHTTPError(http.StatusForbidden, "Permission denied: products:write required")
	}

	id := c.Param("id")

	_, err := h.store.Queries.GetProduct(c.Request().Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "Product not found")
		}
		slog.Error("failed to get product for deletion", "error", err, "id", id)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete product")
	}

	err = h.store.Queries.DeleteProduct(c.Request().Context(), id)
	if err != nil {
		slog.Error("failed to delete product", "error", err, "id", id)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete product")
	}

	return c.NoContent(http.StatusNoContent)
}

// UpdateProduct updates an existing product via the API
func (h *APIProductsHandler) UpdateProduct(c echo.Context) error {
	apiKey := auth.GetAPIKeyInfo(c.Request().Context())
	if apiKey == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "API key required")
	}

	if !apiKey.HasPermission("products:write") {
		return echo.NewHTTPError(http.StatusForbidden, "Permission denied: products:write required")
	}

	id := c.Param("id")

	// Check product exists
	existing, err := h.store.Queries.GetProduct(c.Request().Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "Product not found")
		}
		slog.Error("failed to get product for update", "error", err, "id", id)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get product")
	}

	var req CreateProductRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	// Use existing values as defaults if not provided
	if req.Name == "" {
		req.Name = existing.Name
	}
	if req.Slug == "" {
		req.Slug = existing.Slug
	}
	if req.CategoryID == "" && existing.CategoryID.Valid {
		req.CategoryID = existing.CategoryID.String
	}

	params := db.UpdateProductParams{
		Name:             req.Name,
		Slug:             req.Slug,
		Description:      sql.NullString{String: req.Description, Valid: req.Description != ""},
		ShortDescription: sql.NullString{String: req.ShortDescription, Valid: req.ShortDescription != ""},
		PriceCents:       req.PriceCents,
		CategoryID:       sql.NullString{String: req.CategoryID, Valid: req.CategoryID != ""},
		Sku:              sql.NullString{String: req.SKU, Valid: req.SKU != ""},
		StockQuantity:    sql.NullInt64{Int64: req.StockQuantity, Valid: true},
		HasVariants:      existing.HasVariants,
		WeightGrams:      sql.NullInt64{Int64: req.WeightGrams, Valid: req.WeightGrams > 0},
		LeadTimeDays:     sql.NullInt64{Int64: req.LeadTimeDays, Valid: req.LeadTimeDays > 0},
		IsActive:         sql.NullBool{Bool: req.IsActive, Valid: true},
		IsFeatured:       sql.NullBool{Bool: req.IsFeatured, Valid: true},
		IsPremium:        sql.NullBool{Bool: req.IsPremium, Valid: true},
		Disclaimer:       sql.NullString{String: req.Disclaimer, Valid: req.Disclaimer != ""},
		SeoTitle:         sql.NullString{String: req.SEOTitle, Valid: req.SEOTitle != ""},
		SeoDescription:   sql.NullString{String: req.SEODescription, Valid: req.SEODescription != ""},
		SeoKeywords:      sql.NullString{String: req.SEOKeywords, Valid: req.SEOKeywords != ""},
		OgImageUrl:       sql.NullString{String: req.OGImageURL, Valid: req.OGImageURL != ""},
		ID:               id,
	}

	product, err := h.store.Queries.UpdateProduct(c.Request().Context(), params)
	if err != nil {
		slog.Error("failed to update product", "error", err, "id", id)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update product")
	}

	// Update source info if provided
	if req.SourceURL != "" || req.SourcePlatform != "" || req.DesignerName != "" {
		var releaseDate sql.NullTime
		if req.ReleaseDate != nil && *req.ReleaseDate != "" {
			t, err := time.Parse(time.RFC3339, *req.ReleaseDate)
			if err != nil {
				t, _ = time.Parse("2006-01-02", *req.ReleaseDate)
			}
			releaseDate = sql.NullTime{Time: t, Valid: true}
		}

		err = h.store.Queries.UpdateProductSource(c.Request().Context(), db.UpdateProductSourceParams{
			SourceUrl:      sql.NullString{String: req.SourceURL, Valid: req.SourceURL != ""},
			SourcePlatform: sql.NullString{String: req.SourcePlatform, Valid: req.SourcePlatform != ""},
			DesignerName:   sql.NullString{String: req.DesignerName, Valid: req.DesignerName != ""},
			ReleaseDate:    releaseDate,
			ID:             id,
		})
		if err != nil {
			slog.Warn("failed to update product source info", "error", err, "id", id)
		}
	}

	// Update tags if provided
	if len(req.Tags) > 0 {
		// Remove existing tags first
		_ = h.store.Queries.ClearProductTags(c.Request().Context(), id)
		for _, tagID := range req.Tags {
			err := h.store.Queries.AddProductTag(c.Request().Context(), db.AddProductTagParams{
				ProductID: id,
				TagID:     tagID,
			})
			if err != nil {
				slog.Warn("failed to add tag to product", "error", err, "product_id", id, "tag_id", tagID)
			}
		}
	}

	slog.Info("product updated via API", "id", id, "name", product.Name)

	return c.JSON(http.StatusOK, productToResponse(product))
}

func (h *APIProductsHandler) ListCategories(c echo.Context) error {
	apiKey := auth.GetAPIKeyInfo(c.Request().Context())
	if apiKey == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "API key required")
	}

	categories, err := h.store.Queries.ListCategories(c.Request().Context())
	if err != nil {
		slog.Error("failed to list categories", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to list categories")
	}

	return c.JSON(http.StatusOK, categories)
}

func (h *APIProductsHandler) ListTags(c echo.Context) error {
	apiKey := auth.GetAPIKeyInfo(c.Request().Context())
	if apiKey == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "API key required")
	}

	tags, err := h.store.Queries.ListTags(c.Request().Context())
	if err != nil {
		slog.Error("failed to list tags", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to list tags")
	}

	return c.JSON(http.StatusOK, tags)
}

func (h *APIProductsHandler) AddProductImage(c echo.Context) error {
	apiKey := auth.GetAPIKeyInfo(c.Request().Context())
	if apiKey == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "API key required")
	}

	if !apiKey.HasPermission("products:write") {
		return echo.NewHTTPError(http.StatusForbidden, "Permission denied: products:write required")
	}

	productID := c.Param("id")

	_, err := h.store.Queries.GetProduct(c.Request().Context(), productID)
	if err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "Product not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get product")
	}

	var imageFilename string
	var altText string
	var displayOrder int64
	var isPrimary bool

	// Try multipart file upload first
	file, err := c.FormFile("image")
	if err == nil && file != nil {
		// Handle file upload
		src, err := file.Open()
		if err != nil {
			slog.Error("failed to open uploaded file", "error", err)
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to open uploaded file")
		}
		defer src.Close()

		uploadDir := "public/images/products"
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			slog.Error("failed to create upload directory", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to save image")
		}

		ext := filepath.Ext(file.Filename)
		if ext == "" {
			ext = ".jpg"
		}
		imageFilename = fmt.Sprintf("%s_%d%s", productID, time.Now().UnixNano(), ext)
		filePath := filepath.Join(uploadDir, imageFilename)

		dst, err := os.Create(filePath)
		if err != nil {
			slog.Error("failed to create image file", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to save image")
		}
		defer dst.Close()

		if _, err = io.Copy(dst, src); err != nil {
			os.Remove(filePath)
			slog.Error("failed to write image file", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to save image")
		}

		// Get metadata from form values
		altText = c.FormValue("alt_text")
		if do := c.FormValue("display_order"); do != "" {
			_, _ = fmt.Sscanf(do, "%d", &displayOrder)
		}
		isPrimary = c.FormValue("is_primary") == "true"
	} else {
		// Fall back to JSON body with image_url
		var req struct {
			ImageURL     string `json:"image_url"`
			AltText      string `json:"alt_text"`
			DisplayOrder int64  `json:"display_order"`
			IsPrimary    bool   `json:"is_primary"`
		}
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
		}
		if req.ImageURL == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "Either file upload (image field) or image_url is required")
		}
		imageFilename = req.ImageURL
		altText = req.AltText
		displayOrder = req.DisplayOrder
		isPrimary = req.IsPrimary
	}

	imageID := uuid.New().String()

	if isPrimary {
		_ = h.store.Queries.UnsetAllPrimaryProductImages(c.Request().Context(), productID)
	}

	image, err := h.store.Queries.CreateProductImage(c.Request().Context(), db.CreateProductImageParams{
		ID:           imageID,
		ProductID:    productID,
		ImageUrl:     imageFilename,
		AltText:      sql.NullString{String: altText, Valid: altText != ""},
		DisplayOrder: sql.NullInt64{Int64: displayOrder, Valid: true},
		IsPrimary:    sql.NullBool{Bool: isPrimary, Valid: true},
	})
	if err != nil {
		slog.Error("failed to create product image", "error", err, "product_id", productID)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create product image")
	}

	return c.JSON(http.StatusCreated, image)
}

func productToResponse(p db.Product) ProductResponse {
	resp := ProductResponse{
		ID:         p.ID,
		Name:       p.Name,
		Slug:       p.Slug,
		PriceCents: p.PriceCents,
	}

	if p.Description.Valid {
		resp.Description = p.Description.String
	}
	if p.ShortDescription.Valid {
		resp.ShortDescription = p.ShortDescription.String
	}
	if p.CategoryID.Valid {
		resp.CategoryID = p.CategoryID.String
	}
	if p.Sku.Valid {
		resp.SKU = p.Sku.String
	}
	if p.StockQuantity.Valid {
		resp.StockQuantity = p.StockQuantity.Int64
	}
	if p.WeightGrams.Valid {
		resp.WeightGrams = p.WeightGrams.Int64
	}
	if p.LeadTimeDays.Valid {
		resp.LeadTimeDays = p.LeadTimeDays.Int64
	}
	if p.IsActive.Valid {
		resp.IsActive = p.IsActive.Bool
	}
	if p.IsFeatured.Valid {
		resp.IsFeatured = p.IsFeatured.Bool
	}
	if p.IsPremium.Valid {
		resp.IsPremium = p.IsPremium.Bool
	}
	if p.IsNew.Valid {
		resp.IsNew = p.IsNew.Bool
	}
	if p.SourceUrl.Valid {
		resp.SourceURL = p.SourceUrl.String
	}
	if p.SourcePlatform.Valid {
		resp.SourcePlatform = p.SourcePlatform.String
	}
	if p.DesignerName.Valid {
		resp.DesignerName = p.DesignerName.String
	}
	if p.ReleaseDate.Valid {
		resp.ReleaseDate = &p.ReleaseDate.Time
	}
	if p.CreatedAt.Valid {
		resp.CreatedAt = p.CreatedAt.Time
	}
	if p.UpdatedAt.Valid {
		resp.UpdatedAt = p.UpdatedAt.Time
	}

	return resp
}

func generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		if r == ' ' || r == '-' || r == '_' {
			return '-'
		}
		return -1
	}, slug)

	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")

	slug = fmt.Sprintf("%s-%s", slug, uuid.New().String()[:8])

	return slug
}
