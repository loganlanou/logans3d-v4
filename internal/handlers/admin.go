package handlers

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/internal/email"
	"github.com/loganlanou/logans3d-v4/internal/shipping"
	"github.com/loganlanou/logans3d-v4/internal/types"
	"github.com/loganlanou/logans3d-v4/storage"
	"github.com/loganlanou/logans3d-v4/storage/db"
	"github.com/loganlanou/logans3d-v4/views/admin"
)


type AdminHandler struct {
	storage         *storage.Storage
	shippingService *shipping.ShippingService
	emailService    *email.Service
}

func NewAdminHandler(storage *storage.Storage, shippingService *shipping.ShippingService, emailService *email.Service) *AdminHandler {
	return &AdminHandler{
		storage:         storage,
		shippingService: shippingService,
		emailService:    emailService,
	}
}

func (h *AdminHandler) HandleAdminDashboard(c echo.Context) error {
	// Get query parameters for filtering and sorting
	categoryFilter := c.QueryParam("category")
	featuredFilter := c.QueryParam("featured")
	premiumFilter := c.QueryParam("premium")
	newFilter := c.QueryParam("new")
	statusFilter := c.QueryParam("status")
	sortBy := c.QueryParam("sort")
	sortOrder := c.QueryParam("order")

	// Default sort by name ascending if no sort specified
	if sortBy == "" {
		sortBy = "name"
		sortOrder = "asc"
	}

	// Get all products (admin needs to see inactive products too)
	products, err := h.storage.Queries.ListAllProducts(c.Request().Context())
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch products")
	}

	productsWithImages := h.buildProductsWithImages(c.Request().Context(), products)

	// Apply filters
	filteredProducts := filterProducts(productsWithImages, categoryFilter, featuredFilter, premiumFilter, newFilter, statusFilter)

	// Apply sorting
	sortedProducts := sortProducts(filteredProducts, sortBy, sortOrder)

	// Get all categories for filter dropdown
	categories, err := h.storage.Queries.ListCategories(c.Request().Context())
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch categories")
	}

	return Render(c, admin.Dashboard(c, sortedProducts, categories, categoryFilter, featuredFilter, premiumFilter, newFilter, statusFilter, sortBy, sortOrder))
}

func filterProducts(products []types.ProductWithImage, categoryFilter, featuredFilter, premiumFilter, newFilter, statusFilter string) []types.ProductWithImage {
	filtered := make([]types.ProductWithImage, 0, len(products))

	for _, p := range products {
		// Category filter
		if categoryFilter != "" && categoryFilter != "all" {
			if !p.Product.CategoryID.Valid || p.Product.CategoryID.String != categoryFilter {
				continue
			}
		}

		// New filter
		if newFilter == "true" {
			if !p.IsNew {
				continue
			}
		}

		// Featured filter
		if featuredFilter == "true" {
			isFeatured := p.Product.IsFeatured.Valid && p.Product.IsFeatured.Bool
			if !isFeatured {
				continue
			}
		}

		// Premium filter
		if premiumFilter == "true" {
			isPremium := p.Product.IsPremium.Valid && p.Product.IsPremium.Bool
			if !isPremium {
				continue
			}
		}

		// Status filter - only filter when checkbox is checked
		if statusFilter == "inactive" {
			isActive := p.Product.IsActive.Valid && p.Product.IsActive.Bool
			if isActive {
				continue
			}
		}

		filtered = append(filtered, p)
	}

	return filtered
}

func sortProducts(products []types.ProductWithImage, sortBy, sortOrder string) []types.ProductWithImage {
	if sortBy == "" {
		return products
	}

	// Create a copy to avoid modifying the original slice
	sorted := make([]types.ProductWithImage, len(products))
	copy(sorted, products)

	// Default to ascending if not specified
	if sortOrder == "" {
		sortOrder = "asc"
	}

	// Sort based on the field
	switch sortBy {
	case "name":
		if sortOrder == "asc" {
			// Sort by name ascending
			for i := 0; i < len(sorted); i++ {
				for j := i + 1; j < len(sorted); j++ {
					if sorted[i].Product.Name > sorted[j].Product.Name {
						sorted[i], sorted[j] = sorted[j], sorted[i]
					}
				}
			}
		} else {
			// Sort by name descending
			for i := 0; i < len(sorted); i++ {
				for j := i + 1; j < len(sorted); j++ {
					if sorted[i].Product.Name < sorted[j].Product.Name {
						sorted[i], sorted[j] = sorted[j], sorted[i]
					}
				}
			}
		}
	case "price":
		if sortOrder == "asc" {
			// Sort by price ascending
			for i := 0; i < len(sorted); i++ {
				for j := i + 1; j < len(sorted); j++ {
					if sorted[i].Product.PriceCents > sorted[j].Product.PriceCents {
						sorted[i], sorted[j] = sorted[j], sorted[i]
					}
				}
			}
		} else {
			// Sort by price descending
			for i := 0; i < len(sorted); i++ {
				for j := i + 1; j < len(sorted); j++ {
					if sorted[i].Product.PriceCents < sorted[j].Product.PriceCents {
						sorted[i], sorted[j] = sorted[j], sorted[i]
					}
				}
			}
		}
	}

	return sorted
}

func (h *AdminHandler) HandleCategoriesTab(c echo.Context) error {
	filter := c.QueryParam("filter")
	if filter == "" {
		filter = "all"
	}

	// Get all categories first
	allCategories, err := h.storage.Queries.ListCategories(c.Request().Context())
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch categories")
	}

	// Filter categories based on the filter parameter
	var categories []db.Category
	switch filter {
	case "root":
		for _, cat := range allCategories {
			if !cat.ParentID.Valid {
				categories = append(categories, cat)
			}
		}
	case "subcategories":
		for _, cat := range allCategories {
			if cat.ParentID.Valid {
				categories = append(categories, cat)
			}
		}
	case "empty":
		// Get products to check which categories are empty
		products, err := h.storage.Queries.ListProducts(c.Request().Context())
		if err != nil {
			return c.String(http.StatusInternalServerError, "Failed to fetch products")
		}
		productsWithImages := h.buildProductsWithImages(c.Request().Context(), products)

		for _, cat := range allCategories {
			productCount := 0
			for _, p := range productsWithImages {
				if p.Product.CategoryID.Valid && p.Product.CategoryID.String == cat.ID {
					productCount++
				}
			}
			if productCount == 0 {
				categories = append(categories, cat)
			}
		}
	default: // "all"
		categories = allCategories
	}

	// Get products for statistics (always get all products for accurate counts)
	products, err := h.storage.Queries.ListProducts(c.Request().Context())
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch products")
	}

	productsWithImages := h.buildProductsWithImages(c.Request().Context(), products)

	return Render(c, admin.CategoriesTab(c, productsWithImages, categories, filter))
}


func (h *AdminHandler) buildProductsWithImages(ctx context.Context, products []db.Product) []types.ProductWithImage {
	// Get products with their primary images
	productsWithImages := make([]types.ProductWithImage, 0, len(products))
	for _, product := range products {
		images, err := h.storage.Queries.GetProductImages(ctx, product.ID)
		if err != nil {
			// Continue without image if there's an error
			// Use database is_new column
			isNew := product.IsNew.Valid && product.IsNew.Bool

			// Check if product is discontinued (inactive)
			isDiscontinued := !product.IsActive.Valid || !product.IsActive.Bool

			productsWithImages = append(productsWithImages, types.ProductWithImage{
				Product:       product,
				ImageURL:      "",
				IsNew:         isNew,
				IsDiscontinued: isDiscontinued,
			})
			continue
		}

		imageURL := ""
		if len(images) > 0 {
			// Get the primary image or the first image
			var rawImageURL string
			for _, img := range images {
				if img.IsPrimary.Valid && img.IsPrimary.Bool {
					rawImageURL = img.ImageUrl
					break
				}
			}
			// If no primary image found, use the first one
			if rawImageURL == "" {
				rawImageURL = images[0].ImageUrl
			}

			// Build the full path from the filename
			if rawImageURL != "" {
				// Database should only contain filenames
				// Always build the full path here
				imageURL = "/public/images/products/" + rawImageURL
			}
		}

		// Use database is_new column
		isNew := product.IsNew.Valid && product.IsNew.Bool

		// Check if product is discontinued (inactive)
		isDiscontinued := !product.IsActive.Valid || !product.IsActive.Bool

		productsWithImages = append(productsWithImages, types.ProductWithImage{
			Product:       product,
			ImageURL:      imageURL,
			IsNew:         isNew,
			IsDiscontinued: isDiscontinued,
		})
	}

	return productsWithImages
}

func (h *AdminHandler) HandleProductForm(c echo.Context) error {
	productID := c.QueryParam("id")
	var product *db.Product
	var productImages []db.ProductImage

	if productID != "" {
		p, err := h.storage.Queries.GetProduct(c.Request().Context(), productID)
		if err != nil && err != sql.ErrNoRows {
			return c.String(http.StatusInternalServerError, "Failed to fetch product")
		}
		if err == nil {
			product = &p

			// Get product images
			productImages, err = h.storage.Queries.GetProductImages(c.Request().Context(), productID)
			if err != nil {
				// Log error but don't fail
				fmt.Printf("Failed to fetch product images: %v\n", err)
			}
		}
	}

	categories, err := h.storage.Queries.ListCategories(c.Request().Context())
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch categories")
	}

	return Render(c, admin.ProductForm(c, product, categories, productImages))
}

func (h *AdminHandler) HandleCreateProduct(c echo.Context) error {
	name := c.FormValue("name")
	description := c.FormValue("description")
	shortDescription := c.FormValue("short_description")
	priceStr := c.FormValue("price")
	categoryID := c.FormValue("category_id")
	sku := c.FormValue("sku")
	stockQuantityStr := c.FormValue("stock_quantity")
	isPremiumCollectionStr := c.FormValue("is_premium_collection")
	
	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid price")
	}
	
	stockQuantity := int64(0)
	if stockQuantityStr != "" {
		sq, err := strconv.ParseInt(stockQuantityStr, 10, 64)
		if err == nil {
			stockQuantity = sq
		}
	}

	slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	productID := uuid.New().String()
	isPremiumCollection := isPremiumCollectionStr == "on" || isPremiumCollectionStr == "true"

	params := db.CreateProductParams{
		ID:               productID,
		Name:             name,
		Slug:             slug,
		Description:      sql.NullString{String: description, Valid: description != ""},
		ShortDescription: sql.NullString{String: shortDescription, Valid: shortDescription != ""},
		PriceCents:       int64(price * 100),
		CategoryID:       sql.NullString{String: categoryID, Valid: categoryID != ""},
		Sku:              sql.NullString{String: sku, Valid: sku != ""},
		StockQuantity:    sql.NullInt64{Int64: stockQuantity, Valid: true},
		WeightGrams:      sql.NullInt64{Valid: false},
		LeadTimeDays:     sql.NullInt64{Valid: false},
		IsActive:         sql.NullBool{Bool: true, Valid: true},
		IsFeatured:       sql.NullBool{Bool: false, Valid: true},
		IsPremium:        sql.NullBool{Bool: isPremiumCollection, Valid: true},
	}

	_, err = h.storage.Queries.CreateProduct(c.Request().Context(), params)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to create product: " + err.Error())
	}

	// Handle image upload
	file, err := c.FormFile("image")
	if err == nil && file != nil {
		src, err := file.Open()
		if err == nil {
			defer src.Close()

			// Create images directory if it doesn't exist
			uploadDir := "public/images/products"
			if err := os.MkdirAll(uploadDir, 0755); err != nil {
				return c.String(http.StatusInternalServerError, "Failed to create images directory")
			}

			// Generate unique filename
			ext := filepath.Ext(file.Filename)
			filename := fmt.Sprintf("%s_%d%s", productID, time.Now().Unix(), ext)
			filepath := filepath.Join(uploadDir, filename)

			// Create destination file
			dst, err := os.Create(filepath)
			if err != nil {
				return c.String(http.StatusInternalServerError, "Failed to save image")
			}
			defer dst.Close()

			// Copy file
			if _, err = io.Copy(dst, src); err != nil {
				return c.String(http.StatusInternalServerError, "Failed to save image")
			}

			// Save only the filename to database
			// The view layer will build the full path
			imageFilename := filename

			// Save to database - this will be the primary image
			imageParams := db.CreateProductImageParams{
				ID:           uuid.New().String(),
				ProductID:    productID,
				ImageUrl:     imageFilename,
				AltText:      sql.NullString{String: name, Valid: true},
				DisplayOrder: sql.NullInt64{Int64: 0, Valid: true},
				IsPrimary:    sql.NullBool{Bool: true, Valid: true},
			}

			_, err = h.storage.Queries.CreateProductImage(c.Request().Context(), imageParams)
			if err != nil {
				// Log error but don't fail the product creation
				fmt.Printf("Failed to save product image to database: %v\n", err)
			}
		}
	}

	return c.Redirect(http.StatusSeeOther, "/admin")
}

func (h *AdminHandler) HandleUpdateProduct(c echo.Context) error {
	productID := c.Param("id")
	
	name := c.FormValue("name")
	description := c.FormValue("description")
	shortDescription := c.FormValue("short_description")
	priceStr := c.FormValue("price")
	categoryID := c.FormValue("category_id")
	sku := c.FormValue("sku")
	stockQuantityStr := c.FormValue("stock_quantity")
	isActiveStr := c.FormValue("is_active")
	isPremiumCollectionStr := c.FormValue("is_premium_collection")
	
	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid price")
	}
	
	stockQuantity := int64(0)
	if stockQuantityStr != "" {
		sq, err := strconv.ParseInt(stockQuantityStr, 10, 64)
		if err == nil {
			stockQuantity = sq
		}
	}

	isActive := isActiveStr == "on" || isActiveStr == "true"
	isPremiumCollection := isPremiumCollectionStr == "on" || isPremiumCollectionStr == "true"

	slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))

	// Get current product to preserve is_featured
	currentProduct, err := h.storage.Queries.GetProduct(c.Request().Context(), productID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to get product")
	}

	params := db.UpdateProductParams{
		ID:               productID,
		Name:             name,
		Slug:             slug,
		Description:      sql.NullString{String: description, Valid: description != ""},
		ShortDescription: sql.NullString{String: shortDescription, Valid: shortDescription != ""},
		PriceCents:       int64(price * 100),
		CategoryID:       sql.NullString{String: categoryID, Valid: categoryID != ""},
		Sku:              sql.NullString{String: sku, Valid: sku != ""},
		StockQuantity:    sql.NullInt64{Int64: stockQuantity, Valid: true},
		WeightGrams:      sql.NullInt64{Valid: false},
		LeadTimeDays:     sql.NullInt64{Valid: false},
		IsActive:         sql.NullBool{Bool: isActive, Valid: true},
		IsFeatured:       currentProduct.IsFeatured,
		IsPremium:        sql.NullBool{Bool: isPremiumCollection, Valid: true},
	}

	_, err = h.storage.Queries.UpdateProduct(c.Request().Context(), params)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to update product: " + err.Error())
	}

	// Handle image upload
	file, err := c.FormFile("image")
	if err == nil && file != nil {
		src, err := file.Open()
		if err == nil {
			defer src.Close()

			uploadDir := "public/images/products"
			if err := os.MkdirAll(uploadDir, 0755); err != nil {
				return c.String(http.StatusInternalServerError, "Failed to create images directory")
			}

			ext := filepath.Ext(file.Filename)
			filename := fmt.Sprintf("%s_%d%s", productID, time.Now().Unix(), ext)
			filepath := filepath.Join(uploadDir, filename)

			dst, err := os.Create(filepath)
			if err != nil {
				return c.String(http.StatusInternalServerError, "Failed to save image")
			}
			defer dst.Close()

			if _, err = io.Copy(dst, src); err != nil {
				return c.String(http.StatusInternalServerError, "Failed to save image")
			}

			// Save only the filename to database
			// The view layer will build the full path
			imageFilename := filename

			// Check if there are existing images
			existingImages, err := h.storage.Queries.GetProductImages(c.Request().Context(), productID)
			isPrimary := len(existingImages) == 0 // First image is primary

			// Get next display order
			displayOrder := int64(len(existingImages))

			// Save new image to database
			imageParams := db.CreateProductImageParams{
				ID:           uuid.New().String(),
				ProductID:    productID,
				ImageUrl:     imageFilename,
				AltText:      sql.NullString{String: name, Valid: true},
				DisplayOrder: sql.NullInt64{Int64: displayOrder, Valid: true},
				IsPrimary:    sql.NullBool{Bool: isPrimary, Valid: true},
			}

			_, err = h.storage.Queries.CreateProductImage(c.Request().Context(), imageParams)
			if err != nil {
				// Log error but don't fail the product update
				fmt.Printf("Failed to save product image to database: %v\n", err)
			}
		}
	}

	return c.Redirect(http.StatusSeeOther, "/admin")
}

func (h *AdminHandler) HandleDeleteProduct(c echo.Context) error {
	productID := c.Param("id")

	err := h.storage.Queries.DeleteProduct(c.Request().Context(), productID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to delete product")
	}

	return c.Redirect(http.StatusSeeOther, "/admin")
}

func (h *AdminHandler) HandleDeleteProductImage(c echo.Context) error {
	imageID := c.Param("imageId")
	productID := c.QueryParam("product_id")

	// Get the image before deleting to remove file
	image, err := h.storage.Queries.GetProductImages(c.Request().Context(), productID)
	if err == nil {
		// Find and delete the file
		for _, img := range image {
			if img.ID == imageID {
				uploadDir := "public/images/products"
				filepath := filepath.Join(uploadDir, img.ImageUrl)
				os.Remove(filepath)
				break
			}
		}
	}

	err = h.storage.Queries.DeleteProductImage(c.Request().Context(), imageID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to delete image")
	}

	if productID != "" {
		return c.Redirect(http.StatusSeeOther, "/admin/product/edit?id="+productID)
	}
	return c.Redirect(http.StatusSeeOther, "/admin")
}

func (h *AdminHandler) HandleSetPrimaryProductImage(c echo.Context) error {
	imageID := c.Param("imageId")
	productID := c.QueryParam("product_id")

	err := h.storage.Queries.SetPrimaryProductImage(c.Request().Context(), db.SetPrimaryProductImageParams{
		ID:        imageID,
		ProductID: productID,
	})
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to set primary image")
	}

	if productID != "" {
		return c.Redirect(http.StatusSeeOther, "/admin/product/edit?id="+productID)
	}
	return c.Redirect(http.StatusSeeOther, "/admin")
}

func (h *AdminHandler) HandleToggleProductFeatured(c echo.Context) error {
	productID := c.Param("id")

	product, err := h.storage.Queries.ToggleProductFeatured(c.Request().Context(), productID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to toggle featured status",
		})
	}

	isFeatured := product.IsFeatured.Valid && product.IsFeatured.Bool

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"is_featured": isFeatured,
	})
}

func (h *AdminHandler) HandleToggleProductPremium(c echo.Context) error {
	productID := c.Param("id")

	product, err := h.storage.Queries.ToggleProductPremium(c.Request().Context(), productID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to toggle premium status",
		})
	}

	isPremium := product.IsPremium.Valid && product.IsPremium.Bool

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"is_premium": isPremium,
	})
}

func (h *AdminHandler) HandleToggleProductActive(c echo.Context) error {
	productID := c.Param("id")

	product, err := h.storage.Queries.ToggleProductActive(c.Request().Context(), productID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to toggle active status",
		})
	}

	isActive := product.IsActive.Valid && product.IsActive.Bool

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"is_active": isActive,
	})
}

func (h *AdminHandler) HandleToggleProductNew(c echo.Context) error {
	productID := c.Param("id")

	product, err := h.storage.Queries.ToggleProductNew(c.Request().Context(), productID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to toggle new status",
		})
	}

	isNew := product.IsNew.Valid && product.IsNew.Bool

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"is_new": isNew,
	})
}

func (h *AdminHandler) HandleProductSearch(c echo.Context) error {
	query := c.QueryParam("q")
	if query == "" {
		return c.JSON(http.StatusOK, []map[string]interface{}{})
	}

	// Get all products and filter by query
	products, err := h.storage.Queries.ListAllProducts(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to search products",
		})
	}

	productsWithImages := h.buildProductsWithImages(c.Request().Context(), products)

	// Filter products by name (case insensitive)
	var results []map[string]interface{}
	queryLower := strings.ToLower(query)

	for _, p := range productsWithImages {
		if strings.Contains(strings.ToLower(p.Product.Name), queryLower) {
			results = append(results, map[string]interface{}{
				"id":    p.Product.ID,
				"name":  p.Product.Name,
				"slug":  p.Product.Slug,
				"price": float64(p.Product.PriceCents) / 100,
				"image": p.ImageURL,
			})

			// Limit to 10 results
			if len(results) >= 10 {
				break
			}
		}
	}

	return c.JSON(http.StatusOK, results)
}

// Category Management Functions

func (h *AdminHandler) HandleCategoryForm(c echo.Context) error {
	categoryID := c.QueryParam("id")
	var category *db.Category

	if categoryID != "" {
		cat, err := h.storage.Queries.GetCategory(c.Request().Context(), categoryID)
		if err != nil && err != sql.ErrNoRows {
			return c.String(http.StatusInternalServerError, "Failed to fetch category")
		}
		if err == nil {
			category = &cat
		}
	}

	allCategories, err := h.storage.Queries.ListCategories(c.Request().Context())
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch categories")
	}

	return Render(c, admin.CategoryForm(c, category, allCategories))
}

func (h *AdminHandler) HandleCreateCategory(c echo.Context) error {
	name := c.FormValue("name")
	description := c.FormValue("description")
	parentID := c.FormValue("parent_id")
	displayOrderStr := c.FormValue("display_order")
	
	slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	categoryID := uuid.New().String()
	
	var displayOrder sql.NullInt64
	if displayOrderStr != "" {
		if order, err := strconv.ParseInt(displayOrderStr, 10, 64); err == nil {
			displayOrder = sql.NullInt64{Int64: order, Valid: true}
		}
	}

	params := db.CreateCategoryParams{
		ID:           categoryID,
		Name:         name,
		Slug:         slug,
		Description:  sql.NullString{String: description, Valid: description != ""},
		ParentID:     sql.NullString{String: parentID, Valid: parentID != ""},
		DisplayOrder: displayOrder,
	}

	_, err := h.storage.Queries.CreateCategory(c.Request().Context(), params)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to create category: " + err.Error())
	}

	return c.Redirect(http.StatusSeeOther, "/admin")
}

func (h *AdminHandler) HandleUpdateCategory(c echo.Context) error {
	categoryID := c.Param("id")
	
	name := c.FormValue("name")
	description := c.FormValue("description")
	parentID := c.FormValue("parent_id")
	displayOrderStr := c.FormValue("display_order")
	
	slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	
	var displayOrder sql.NullInt64
	if displayOrderStr != "" {
		if order, err := strconv.ParseInt(displayOrderStr, 10, 64); err == nil {
			displayOrder = sql.NullInt64{Int64: order, Valid: true}
		}
	}

	params := db.UpdateCategoryParams{
		ID:           categoryID,
		Name:         name,
		Slug:         slug,
		Description:  sql.NullString{String: description, Valid: description != ""},
		ParentID:     sql.NullString{String: parentID, Valid: parentID != ""},
		DisplayOrder: displayOrder,
	}

	_, err := h.storage.Queries.UpdateCategory(c.Request().Context(), params)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to update category: " + err.Error())
	}

	return c.Redirect(http.StatusSeeOther, "/admin")
}

func (h *AdminHandler) HandleDeleteCategory(c echo.Context) error {
	categoryID := c.Param("id")
	
	err := h.storage.Queries.DeleteCategory(c.Request().Context(), categoryID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to delete category")
	}

	return c.Redirect(http.StatusSeeOther, "/admin")
}

// Developer Window Functions

var appStartTime = time.Now()

func (h *AdminHandler) HandleDeveloperDashboard(c echo.Context) error {
	// Get system information
	sysInfo := types.SystemInfo{
		AppName:      "Logan's 3D Creations v4",
		Version:      "4.0.0",
		Environment:  os.Getenv("ENVIRONMENT"),
		StartTime:    appStartTime,
		Uptime:       time.Since(appStartTime).String(),
		GoVersion:    runtime.Version(),
		Architecture: runtime.GOARCH,
		OS:           runtime.GOOS,
		PID:          os.Getpid(),
		DBPath:       os.Getenv("DB_PATH"),
		Port:         os.Getenv("PORT"),
	}

	// Get database stats
	dbStats := types.DatabaseStats{}

	// Count products
	products, err := h.storage.Queries.ListProducts(c.Request().Context())
	if err == nil {
		dbStats.ProductCount = int64(len(products))
	}

	// Count categories
	categories, err := h.storage.Queries.ListCategories(c.Request().Context())
	if err == nil {
		dbStats.CategoryCount = int64(len(categories))
	}

	// Get memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memStats := types.MemoryStats{
		Alloc:        m.Alloc,
		TotalAlloc:   m.TotalAlloc,
		Sys:          m.Sys,
		NumGC:        m.NumGC,
		Goroutines:   runtime.NumGoroutine(),
		AllocMB:      float64(m.Alloc) / 1024 / 1024,
		SysMB:        float64(m.Sys) / 1024 / 1024,
	}

	// Get database file size
	if stat, err := os.Stat(sysInfo.DBPath); err == nil {
		dbStats.DatabaseSize = fmt.Sprintf("%.2f MB", float64(stat.Size())/1024/1024)
	}

	return Render(c, admin.DevOverview(c, sysInfo, dbStats, memStats))
}

func (h *AdminHandler) HandleDevSystem(c echo.Context) error {
	// Get system information
	sysInfo := types.SystemInfo{
		AppName:      "Logan's 3D Creations v4",
		Version:      "4.0.0",
		Environment:  os.Getenv("ENVIRONMENT"),
		StartTime:    appStartTime,
		Uptime:       time.Since(appStartTime).String(),
		GoVersion:    runtime.Version(),
		Architecture: runtime.GOARCH,
		OS:           runtime.GOOS,
		CPUCount:     fmt.Sprintf("%d", runtime.NumCPU()),
		PID:          os.Getpid(),
		DBPath:       os.Getenv("DB_PATH"),
		Port:         os.Getenv("PORT"),
	}

	return Render(c, admin.DevSystem(c, sysInfo))
}

func (h *AdminHandler) HandleDevDatabase(c echo.Context) error {
	// Get system information
	sysInfo := types.SystemInfo{
		AppName:      "Logan's 3D Creations v4",
		Version:      "4.0.0",
		Environment:  os.Getenv("ENVIRONMENT"),
		StartTime:    appStartTime,
		Uptime:       time.Since(appStartTime).String(),
		GoVersion:    runtime.Version(),
		Architecture: runtime.GOARCH,
		OS:           runtime.GOOS,
		PID:          os.Getpid(),
		DBPath:       os.Getenv("DB_PATH"),
		Port:         os.Getenv("PORT"),
	}

	// Get database stats
	dbStats := types.DatabaseStats{}

	// Count products
	products, err := h.storage.Queries.ListProducts(c.Request().Context())
	if err == nil {
		dbStats.ProductCount = int64(len(products))
	}

	// Count categories
	categories, err := h.storage.Queries.ListCategories(c.Request().Context())
	if err == nil {
		dbStats.CategoryCount = int64(len(categories))
	}

	// Count users
	users, err := h.storage.Queries.ListUsers(c.Request().Context())
	if err == nil {
		dbStats.UserCount = int64(len(users))
	}

	// Get database file size
	if stat, err := os.Stat(sysInfo.DBPath); err == nil {
		dbStats.DatabaseSize = fmt.Sprintf("%.2f MB", float64(stat.Size())/1024/1024)
	}

	return Render(c, admin.DevDatabase(c, sysInfo, dbStats))
}

func (h *AdminHandler) HandleDevMemory(c echo.Context) error {
	// Get memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memStats := types.MemoryStats{
		Alloc:        m.Alloc,
		TotalAlloc:   m.TotalAlloc,
		Sys:          m.Sys,
		HeapAlloc:    m.HeapAlloc,
		HeapSys:      m.HeapSys,
		HeapIdle:     m.HeapIdle,
		HeapInuse:    m.HeapInuse,
		HeapReleased: m.HeapReleased,
		NumGC:        m.NumGC,
		LastGC:       time.Unix(0, int64(m.LastGC)).Format("2006-01-02 15:04:05"),
		Goroutines:   runtime.NumGoroutine(),
		AllocMB:      float64(m.Alloc) / 1024 / 1024,
		SysMB:        float64(m.Sys) / 1024 / 1024,
	}

	return Render(c, admin.DevMemory(c, memStats))
}

func (h *AdminHandler) HandleDevLogs(c echo.Context) error {
	// Get system information
	sysInfo := types.SystemInfo{
		AppName:      "Logan's 3D Creations v4",
		Version:      "4.0.0",
		Environment:  os.Getenv("ENVIRONMENT"),
		StartTime:    appStartTime,
		Uptime:       time.Since(appStartTime).String(),
		GoVersion:    runtime.Version(),
		Architecture: runtime.GOARCH,
		OS:           runtime.GOOS,
		PID:          os.Getpid(),
		DBPath:       os.Getenv("DB_PATH"),
		Port:         os.Getenv("PORT"),
	}

	return Render(c, admin.DevLogs(c, sysInfo))
}

func (h *AdminHandler) HandleDevConfig(c echo.Context) error {
	// Get system information
	sysInfo := types.SystemInfo{
		AppName:      "Logan's 3D Creations v4",
		Version:      "4.0.0",
		Environment:  os.Getenv("ENVIRONMENT"),
		StartTime:    appStartTime,
		Uptime:       time.Since(appStartTime).String(),
		GoVersion:    runtime.Version(),
		Architecture: runtime.GOARCH,
		OS:           runtime.GOOS,
		PID:          os.Getpid(),
		DBPath:       os.Getenv("DB_PATH"),
		Port:         os.Getenv("PORT"),
	}

	return Render(c, admin.DevConfig(c, sysInfo))
}

func (h *AdminHandler) HandleLogStream(c echo.Context) error {
	logPath := os.Getenv("LOG_FILE_PATH")
	if logPath == "" {
		logPath = "./tmp/air-combined.log" // Fallback to default
	}

	// Check if log file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Log file not found"})
	}

	// Set SSE headers
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().Header().Set("X-Accel-Buffering", "no")

	// Open the log file
	file, err := os.Open(logPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Seek to end of file
	file.Seek(0, io.SeekEnd)

	// Create a ticker for polling
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	// Channel to signal when client disconnects
	notify := c.Request().Context().Done()

	flusher, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming unsupported")
	}

	reader := bufio.NewReader(file)

	for {
		select {
		case <-notify:
			// Client disconnected
			return nil
		case <-ticker.C:
			// Read new lines
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					if err == io.EOF {
						break
					}
					return err
				}

				// Send line as SSE
				fmt.Fprintf(c.Response().Writer, "data: %s\n\n", line)
				flusher.Flush()
			}
		}
	}
}

func (h *AdminHandler) HandleLogTail(c echo.Context) error {
	logPath := os.Getenv("LOG_FILE_PATH")
	if logPath == "" {
		logPath = "./tmp/air-combined.log" // Fallback to default
	}
	lines := 100 // Default to last 100 lines

	if linesParam := c.QueryParam("lines"); linesParam != "" {
		if n, err := strconv.Atoi(linesParam); err == nil && n > 0 {
			lines = n
		}
	}

	// Check if log file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"lines": []string{},
			"message": "Log file not found",
		})
	}

	// Read the file
	content, err := os.ReadFile(logPath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Split into lines and get last N
	allLines := strings.Split(string(content), "\n")
	startIndex := len(allLines) - lines
	if startIndex < 0 {
		startIndex = 0
	}

	tailLines := allLines[startIndex:]

	return c.JSON(http.StatusOK, map[string]interface{}{
		"lines": tailLines,
		"total": len(allLines),
	})
}

func (h *AdminHandler) HandleLogClear(c echo.Context) error {
	// We won't actually delete the log file (Air needs it), just return success
	// The UI will clear its display
	return c.JSON(http.StatusOK, map[string]string{
		"message": "Log display cleared (file preserved)",
	})
}

func (h *AdminHandler) HandleSystemInfo(c echo.Context) error {
	info := map[string]interface{}{
		"timestamp": time.Now(),
		"system": map[string]interface{}{
			"go_version":    runtime.Version(),
			"architecture":  runtime.GOARCH,
			"os":           runtime.GOOS,
			"pid":          os.Getpid(),
			"goroutines":   runtime.NumGoroutine(),
		},
		"environment": map[string]interface{}{
			"db_path":     os.Getenv("DB_PATH"),
			"port":        os.Getenv("PORT"),
			"environment": os.Getenv("ENVIRONMENT"),
		},
	}
	
	return c.JSON(http.StatusOK, info)
}

func (h *AdminHandler) HandleMemoryStats(c echo.Context) error {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	stats := map[string]interface{}{
		"timestamp": time.Now(),
		"memory": map[string]interface{}{
			"alloc_mb":     float64(m.Alloc) / 1024 / 1024,
			"sys_mb":       float64(m.Sys) / 1024 / 1024,
			"total_alloc":  m.TotalAlloc,
			"num_gc":       m.NumGC,
			"goroutines":   runtime.NumGoroutine(),
		},
	}
	
	return c.JSON(http.StatusOK, stats)
}

func (h *AdminHandler) HandleDatabaseInfo(c echo.Context) error {
	stats := map[string]interface{}{
		"timestamp": time.Now(),
	}
	
	// Count products
	if products, err := h.storage.Queries.ListProducts(c.Request().Context()); err == nil {
		stats["product_count"] = len(products)
	}
	
	// Count categories
	if categories, err := h.storage.Queries.ListCategories(c.Request().Context()); err == nil {
		stats["category_count"] = len(categories)
	}
	
	// Get database file size
	if dbPath := os.Getenv("DB_PATH"); dbPath != "" {
		if stat, err := os.Stat(dbPath); err == nil {
			stats["database_size_bytes"] = stat.Size()
			stats["database_size_mb"] = float64(stat.Size()) / 1024 / 1024
		}
	}
	
	return c.JSON(http.StatusOK, stats)
}

func (h *AdminHandler) HandleConfigInfo(c echo.Context) error {
	config := map[string]interface{}{
		"timestamp": time.Now(),
		"environment_variables": map[string]string{
			"DB_PATH":     os.Getenv("DB_PATH"),
			"PORT":        os.Getenv("PORT"),
			"ENVIRONMENT": os.Getenv("ENVIRONMENT"),
		},
		"runtime": map[string]interface{}{
			"start_time": appStartTime,
			"uptime":     time.Since(appStartTime).String(),
		},
	}
	
	return c.JSON(http.StatusOK, config)
}

func (h *AdminHandler) HandleGarbageCollect(c echo.Context) error {
	runtime.GC()
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	result := map[string]interface{}{
		"timestamp": time.Now(),
		"message":   "Garbage collection completed",
		"memory": map[string]interface{}{
			"alloc_mb": float64(m.Alloc) / 1024 / 1024,
			"sys_mb":   float64(m.Sys) / 1024 / 1024,
			"num_gc":   m.NumGC,
		},
	}
	
	return c.JSON(http.StatusOK, result)
}

// Orders Management Functions

func (h *AdminHandler) HandleOrdersList(c echo.Context) error {
	status := c.QueryParam("status")

	var orders []db.Order
	var err error

	if status != "" {
		orders, err = h.storage.Queries.ListOrdersByStatus(c.Request().Context(), sql.NullString{String: status, Valid: true})
	} else {
		orders, err = h.storage.Queries.ListOrders(c.Request().Context())
	}

	if err != nil {
		slog.Error("failed to fetch orders", "error", err, "status_filter", status)
		return c.String(http.StatusInternalServerError, "Failed to fetch orders")
	}

	return Render(c, admin.OrdersList(c, orders))
}

func (h *AdminHandler) HandleOrderSearch(c echo.Context) error {
	query := c.QueryParam("q")
	if query == "" {
		return c.JSON(http.StatusOK, []map[string]interface{}{})
	}

	// Get all orders
	orders, err := h.storage.Queries.ListOrders(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to search orders",
		})
	}

	// Filter orders by customer name (case insensitive)
	var results []map[string]interface{}
	queryLower := strings.ToLower(query)

	for _, order := range orders {
		if strings.Contains(strings.ToLower(order.CustomerName), queryLower) {
			// Get item count for this order
			orderItems, _ := h.storage.Queries.GetOrderItems(c.Request().Context(), order.ID)
			itemCount := len(orderItems)

			// Format date
			var dateStr string
			if order.CreatedAt.Valid {
				dateStr = order.CreatedAt.Time.Format("Jan 2, 2006")
			}

			results = append(results, map[string]interface{}{
				"id":             order.ID,
				"order_number":   order.ID[:8],
				"customer_name":  order.CustomerName,
				"customer_email": order.CustomerEmail,
				"total_cents":    order.TotalCents,
				"status":         order.Status.String,
				"created_at":     dateStr,
				"item_count":     itemCount,
			})

			// Limit to 10 results
			if len(results) >= 10 {
				break
			}
		}
	}

	return c.JSON(http.StatusOK, results)
}

func (h *AdminHandler) HandleOrderDetail(c echo.Context) error {
	orderID := c.Param("id")
	ctx := c.Request().Context()

	order, err := h.storage.Queries.GetOrder(ctx, orderID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.String(http.StatusNotFound, "Order not found")
		}
		return c.String(http.StatusInternalServerError, "Failed to fetch order")
	}

	orderItems, err := h.storage.Queries.GetOrderItems(ctx, orderID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch order items")
	}

	shippingSelection, err := h.storage.Queries.GetOrderShippingSelection(ctx, orderID)
	if err != nil && err != sql.ErrNoRows {
		slog.Error("failed to fetch shipping selection", "error", err, "order_id", orderID)
	}

	return Render(c, admin.OrderDetail(c, order, orderItems, shippingSelection))
}

func (h *AdminHandler) HandleUpdateOrderStatus(c echo.Context) error {
	orderID := c.Param("id")

	// Try to read from JSON body first (for AJAX requests)
	var requestBody struct {
		Status         string `json:"status"`
		Carrier        string `json:"carrier"`
		TrackingNumber string `json:"tracking_number"`
		TrackingURL    string `json:"tracking_url"`
	}

	var status string
	if err := c.Bind(&requestBody); err == nil && requestBody.Status != "" {
		status = requestBody.Status
	} else {
		// Fallback to form value (for form submissions)
		status = c.FormValue("status")
	}

	if status == "" {
		return c.String(http.StatusBadRequest, "Status is required")
	}

	ctx := c.Request().Context()

	// Update order status
	_, err := h.storage.Queries.UpdateOrderStatus(ctx, db.UpdateOrderStatusParams{
		ID:     orderID,
		Status: sql.NullString{String: status, Valid: true},
	})
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to update order status")
	}

	// If changing to shipped and tracking info provided, update tracking
	if status == "shipped" && requestBody.TrackingNumber != "" {
		_, err := h.storage.Queries.UpdateOrderTracking(ctx, db.UpdateOrderTrackingParams{
			ID:             orderID,
			TrackingNumber: sql.NullString{String: requestBody.TrackingNumber, Valid: true},
			TrackingUrl:    sql.NullString{String: requestBody.TrackingURL, Valid: requestBody.TrackingURL != ""},
			Carrier:        sql.NullString{String: requestBody.Carrier, Valid: requestBody.Carrier != ""},
		})
		if err != nil {
			return c.String(http.StatusInternalServerError, "Failed to update tracking information")
		}
	}

	// Return JSON for AJAX requests
	if c.Request().Header.Get("Content-Type") == "application/json" {
		return c.JSON(http.StatusOK, map[string]string{"status": "success"})
	}

	return c.Redirect(http.StatusSeeOther, "/admin/orders")
}

// HandleGetOrderTrackingLookup retrieves tracking info from EasyPost for an order
func (h *AdminHandler) HandleGetOrderTrackingLookup(c echo.Context) error {
	orderID := c.Param("id")
	ctx := c.Request().Context()

	// Get order
	order, err := h.storage.Queries.GetOrder(ctx, orderID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Order not found"})
	}

	// Check if order has EasyPost shipment ID
	if !order.EasypostShipmentID.Valid || order.EasypostShipmentID.String == "" {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"has_shipment": false,
			"message":      "No EasyPost shipment linked to this order",
		})
	}

	// Get tracking info from EasyPost
	tracking, err := h.shippingService.GetShipmentTracking(order.EasypostShipmentID.String)
	if err != nil {
		slog.Error("failed to get tracking from EasyPost", "error", err, "shipment_id", order.EasypostShipmentID.String)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve tracking information"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"has_shipment":    true,
		"shipment_id":     order.EasypostShipmentID.String,
		"tracking_number": tracking.TrackingNumber,
		"carrier":         tracking.Carrier,
		"tracking_url":    tracking.TrackingURL,
	})
}

// HandleGetOrderShippingRates retrieves current shipping rates for an order's EasyPost shipment
func (h *AdminHandler) HandleGetOrderShippingRates(c echo.Context) error {
	orderID := c.Param("id")
	ctx := c.Request().Context()

	// Get order
	order, err := h.storage.Queries.GetOrder(ctx, orderID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Order not found"})
	}

	// Check if order has EasyPost shipment ID
	if !order.EasypostShipmentID.Valid || order.EasypostShipmentID.String == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "No EasyPost shipment linked to this order"})
	}

	// Get refreshed rates from EasyPost
	rates, err := h.shippingService.RefreshShipmentRates(order.EasypostShipmentID.String)
	if err != nil {
		slog.Error("failed to refresh rates from EasyPost", "error", err, "shipment_id", order.EasypostShipmentID.String)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve shipping rates"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"shipment_id": order.EasypostShipmentID.String,
		"rates":       rates,
	})
}

// HandleBuyShippingLabel purchases a shipping label from EasyPost
func (h *AdminHandler) HandleBuyShippingLabel(c echo.Context) error {
	orderID := c.Param("id")
	ctx := c.Request().Context()

	// Parse request body
	var req struct {
		RateID string `json:"rate_id"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if req.RateID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Rate ID is required"})
	}

	// Get order
	order, err := h.storage.Queries.GetOrder(ctx, orderID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Order not found"})
	}

	// Check if order has EasyPost shipment ID
	if !order.EasypostShipmentID.Valid || order.EasypostShipmentID.String == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "No EasyPost shipment linked to this order"})
	}

	// Check if label already purchased
	if order.EasypostLabelUrl.Valid && order.EasypostLabelUrl.String != "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":     "Label already purchased for this order",
			"label_url": order.EasypostLabelUrl.String,
		})
	}

	// Buy shipping label from EasyPost
	label, err := h.shippingService.CreateLabelFromShipment(order.EasypostShipmentID.String, req.RateID)
	if err != nil {
		slog.Error("failed to buy label from EasyPost", "error", err,
			"shipment_id", order.EasypostShipmentID.String,
			"rate_id", req.RateID)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to purchase shipping label"})
	}

	// Update order with label URL, tracking info, and set status to shipped
	carrier := label.ServiceCode
	if label.CarrierID != "" {
		carrier = label.CarrierID
	}

	_, updateErr := h.storage.Queries.UpdateOrderLabel(ctx, db.UpdateOrderLabelParams{
		ID:                orderID,
		EasypostLabelUrl:  sql.NullString{String: label.LabelDownload.Hrefs.PDF, Valid: true},
		TrackingNumber:    sql.NullString{String: label.TrackingNumber, Valid: true},
		Carrier:           sql.NullString{String: carrier, Valid: true},
		Status:            sql.NullString{String: "shipped", Valid: true},
	})
	if updateErr != nil {
		slog.Error("failed to update order with label info", "error", updateErr, "order_id", orderID)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Label purchased but failed to update order"})
	}

	slog.Info("shipping label purchased and order updated",
		"order_id", orderID,
		"tracking_number", label.TrackingNumber,
		"carrier", carrier)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":         true,
		"label_url":       label.LabelDownload.Hrefs.PDF,
		"tracking_number": label.TrackingNumber,
		"carrier":         carrier,
		"status":          "shipped",
	})
}

// Quotes Management Functions

func (h *AdminHandler) HandleQuotesList(c echo.Context) error {
	status := c.QueryParam("status")

	var quotes []db.QuoteRequest
	var err error

	if status != "" {
		quotes, err = h.storage.Queries.ListQuoteRequestsByStatus(c.Request().Context(), sql.NullString{String: status, Valid: true})
	} else {
		quotes, err = h.storage.Queries.ListQuoteRequests(c.Request().Context())
	}

	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch quote requests")
	}

	return Render(c, admin.QuotesList(c, quotes))
}

func (h *AdminHandler) HandleQuoteDetail(c echo.Context) error {
	quoteID := c.Param("id")

	quote, err := h.storage.Queries.GetQuoteRequest(c.Request().Context(), quoteID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.String(http.StatusNotFound, "Quote request not found")
		}
		return c.String(http.StatusInternalServerError, "Failed to fetch quote request")
	}

	return Render(c, admin.QuoteDetail(c, quote))
}

func (h *AdminHandler) HandleUpdateQuote(c echo.Context) error {
	quoteID := c.Param("id")
	
	status := c.FormValue("status")
	adminNotes := c.FormValue("admin_notes")
	quotedPriceStr := c.FormValue("quoted_price")
	
	var quotedPriceCents sql.NullInt64
	if quotedPriceStr != "" {
		if price, err := strconv.ParseFloat(quotedPriceStr, 64); err == nil {
			quotedPriceCents = sql.NullInt64{Int64: int64(price * 100), Valid: true}
		}
	}
	
	// Get existing quote to preserve other fields
	existingQuote, err := h.storage.Queries.GetQuoteRequest(c.Request().Context(), quoteID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch existing quote")
	}
	
	params := db.UpdateQuoteRequestParams{
		ID:                  quoteID,
		CustomerName:        existingQuote.CustomerName,
		CustomerEmail:       existingQuote.CustomerEmail,
		CustomerPhone:       existingQuote.CustomerPhone,
		ProjectDescription:  existingQuote.ProjectDescription,
		Quantity:            existingQuote.Quantity,
		MaterialPreference:  existingQuote.MaterialPreference,
		FinishPreference:    existingQuote.FinishPreference,
		DeadlineDate:        existingQuote.DeadlineDate,
		BudgetRange:         existingQuote.BudgetRange,
		Status:              sql.NullString{String: status, Valid: true},
		AdminNotes:          sql.NullString{String: adminNotes, Valid: adminNotes != ""},
		QuotedPriceCents:    quotedPriceCents,
	}
	
	_, err = h.storage.Queries.UpdateQuoteRequest(c.Request().Context(), params)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to update quote request")
	}
	
	return c.Redirect(http.StatusSeeOther, "/admin/quotes")
}

// Events Management Functions

func (h *AdminHandler) HandleEventsList(c echo.Context) error {
	filter := c.QueryParam("filter")

	var events []db.Event
	var err error

	switch filter {
	case "upcoming":
		events, err = h.storage.Queries.ListUpcomingEvents(c.Request().Context())
	case "active":
		events, err = h.storage.Queries.ListActiveEvents(c.Request().Context())
	case "past":
		events, err = h.storage.Queries.ListPastEvents(c.Request().Context())
	default:
		events, err = h.storage.Queries.ListEvents(c.Request().Context())
	}

	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch events")
	}

	return Render(c, admin.EventsList(c, events))
}

func (h *AdminHandler) HandleEventForm(c echo.Context) error {
	eventID := c.QueryParam("id")
	var event *db.Event

	if eventID != "" {
		e, err := h.storage.Queries.GetEvent(c.Request().Context(), eventID)
		if err != nil && err != sql.ErrNoRows {
			return c.String(http.StatusInternalServerError, "Failed to fetch event")
		}
		if err == nil {
			event = &e
		}
	}

	return Render(c, admin.EventForm(c, event))
}

func (h *AdminHandler) HandleCreateEvent(c echo.Context) error {
	title := c.FormValue("title")
	description := c.FormValue("description")
	location := c.FormValue("location")
	address := c.FormValue("address")
	startDateStr := c.FormValue("start_date")
	endDateStr := c.FormValue("end_date")
	url := c.FormValue("url")
	isActiveStr := c.FormValue("is_active")
	
	if title == "" || startDateStr == "" {
		return c.String(http.StatusBadRequest, "Title and start date are required")
	}
	
	startDate, err := time.Parse("2006-01-02T15:04", startDateStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid start date format")
	}
	
	var endDate sql.NullTime
	if endDateStr != "" {
		if ed, err := time.Parse("2006-01-02T15:04", endDateStr); err == nil {
			endDate = sql.NullTime{Time: ed, Valid: true}
		}
	}
	
	isActive := isActiveStr == "on" || isActiveStr == "true"
	eventID := uuid.New().String()
	
	params := db.CreateEventParams{
		ID:          eventID,
		Title:       title,
		Description: sql.NullString{String: description, Valid: description != ""},
		Location:    sql.NullString{String: location, Valid: location != ""},
		Address:     sql.NullString{String: address, Valid: address != ""},
		StartDate:   startDate,
		EndDate:     endDate,
		Url:         sql.NullString{String: url, Valid: url != ""},
		IsActive:    sql.NullBool{Bool: isActive, Valid: true},
	}
	
	_, err = h.storage.Queries.CreateEvent(c.Request().Context(), params)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to create event: "+err.Error())
	}
	
	return c.Redirect(http.StatusSeeOther, "/admin/events")
}

func (h *AdminHandler) HandleUpdateEvent(c echo.Context) error {
	eventID := c.Param("id")
	
	title := c.FormValue("title")
	description := c.FormValue("description")
	location := c.FormValue("location")
	address := c.FormValue("address")
	startDateStr := c.FormValue("start_date")
	endDateStr := c.FormValue("end_date")
	url := c.FormValue("url")
	isActiveStr := c.FormValue("is_active")
	
	if title == "" || startDateStr == "" {
		return c.String(http.StatusBadRequest, "Title and start date are required")
	}
	
	startDate, err := time.Parse("2006-01-02T15:04", startDateStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid start date format")
	}
	
	var endDate sql.NullTime
	if endDateStr != "" {
		if ed, err := time.Parse("2006-01-02T15:04", endDateStr); err == nil {
			endDate = sql.NullTime{Time: ed, Valid: true}
		}
	}
	
	isActive := isActiveStr == "on" || isActiveStr == "true"
	
	params := db.UpdateEventParams{
		ID:          eventID,
		Title:       title,
		Description: sql.NullString{String: description, Valid: description != ""},
		Location:    sql.NullString{String: location, Valid: location != ""},
		Address:     sql.NullString{String: address, Valid: address != ""},
		StartDate:   startDate,
		EndDate:     endDate,
		Url:         sql.NullString{String: url, Valid: url != ""},
		IsActive:    sql.NullBool{Bool: isActive, Valid: true},
	}
	
	_, err = h.storage.Queries.UpdateEvent(c.Request().Context(), params)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to update event: "+err.Error())
	}
	
	return c.Redirect(http.StatusSeeOther, "/admin/events")
}

func (h *AdminHandler) HandleDeleteEvent(c echo.Context) error {
	eventID := c.Param("id")

	err := h.storage.Queries.DeleteEvent(c.Request().Context(), eventID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to delete event")
	}

	return c.Redirect(http.StatusSeeOther, "/admin/events")
}

// Shipping Management Functions

func (h *AdminHandler) HandleShippingTab(c echo.Context) error {
	// Get all boxes (including inactive for admin view)
	boxes, err := h.storage.Queries.ListAllBoxCatalog(c.Request().Context())
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch boxes")
	}

	return Render(c, admin.ShippingTab(c, boxes))
}

func (h *AdminHandler) HandleBoxForm(c echo.Context) error {
	// Check for SKU in path parameter (for edit) or query param (for new)
	sku := c.Param("sku")
	if sku == "" {
		sku = c.QueryParam("sku")
	}

	var box *db.BoxCatalog

	if sku != "" {
		// Load box by SKU for editing
		b, err := h.storage.Queries.GetBoxBySKU(c.Request().Context(), sku)
		if err != nil && err != sql.ErrNoRows {
			return c.String(http.StatusInternalServerError, "Failed to fetch box")
		}
		if err == nil {
			box = &b
		}
	}

	return Render(c, admin.BoxForm(c, box))
}

func (h *AdminHandler) HandleCreateBox(c echo.Context) error {
	sku := c.FormValue("sku")
	name := c.FormValue("name")
	lengthStr := c.FormValue("length_inches")
	widthStr := c.FormValue("width_inches")
	heightStr := c.FormValue("height_inches")
	weightStr := c.FormValue("box_weight_oz")
	costStr := c.FormValue("unit_cost_usd")
	isActiveStr := c.FormValue("is_active")

	if sku == "" || name == "" {
		return c.String(http.StatusBadRequest, "SKU and name are required")
	}

	length, err := strconv.ParseFloat(lengthStr, 64)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid length")
	}

	width, err := strconv.ParseFloat(widthStr, 64)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid width")
	}

	height, err := strconv.ParseFloat(heightStr, 64)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid height")
	}

	weight, err := strconv.ParseFloat(weightStr, 64)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid weight")
	}

	cost, err := strconv.ParseFloat(costStr, 64)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid cost")
	}

	isActive := isActiveStr == "on" || isActiveStr == "true"
	boxID := uuid.New().String()

	params := db.CreateBoxCatalogItemParams{
		ID:            boxID,
		Sku:           sku,
		Name:          name,
		LengthInches:  length,
		WidthInches:   width,
		HeightInches:  height,
		BoxWeightOz:   weight,
		UnitCostUsd:   cost,
		IsActive:      sql.NullBool{Bool: isActive, Valid: true},
	}

	_, err = h.storage.Queries.CreateBoxCatalogItem(c.Request().Context(), params)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to create box: "+err.Error())
	}

	return c.Redirect(http.StatusSeeOther, "/admin/shipping/boxes")
}

func (h *AdminHandler) HandleUpdateBox(c echo.Context) error {
	sku := c.Param("sku")

	name := c.FormValue("name")
	lengthStr := c.FormValue("length_inches")
	widthStr := c.FormValue("width_inches")
	heightStr := c.FormValue("height_inches")
	weightStr := c.FormValue("box_weight_oz")
	costStr := c.FormValue("unit_cost_usd")
	isActiveStr := c.FormValue("is_active")

	if name == "" {
		return c.String(http.StatusBadRequest, "Name is required")
	}

	length, err := strconv.ParseFloat(lengthStr, 64)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid length")
	}

	width, err := strconv.ParseFloat(widthStr, 64)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid width")
	}

	height, err := strconv.ParseFloat(heightStr, 64)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid height")
	}

	weight, err := strconv.ParseFloat(weightStr, 64)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid weight")
	}

	cost, err := strconv.ParseFloat(costStr, 64)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid cost")
	}

	isActive := isActiveStr == "on" || isActiveStr == "true"

	params := db.UpdateBoxCatalogItemParams{
		Sku:          sku,
		Name:         name,
		LengthInches: length,
		WidthInches:  width,
		HeightInches: height,
		BoxWeightOz:  weight,
		UnitCostUsd:  cost,
		IsActive:     sql.NullBool{Bool: isActive, Valid: true},
	}

	_, err = h.storage.Queries.UpdateBoxCatalogItem(c.Request().Context(), params)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to update box: "+err.Error())
	}

	return c.Redirect(http.StatusSeeOther, "/admin/shipping/boxes")
}

func (h *AdminHandler) HandleDeleteBox(c echo.Context) error {
	sku := c.Param("sku")

	err := h.storage.Queries.DeleteBoxCatalogItem(c.Request().Context(), sku)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to delete box")
	}

	return c.Redirect(http.StatusSeeOther, "/admin/shipping/boxes")
}

func (h *AdminHandler) HandleShippingConfig(c echo.Context) error {
	// Load shipping configuration from database
	config, err := shipping.LoadShippingConfigFromDB(c.Request().Context(), h.storage.Queries)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to load shipping configuration: "+err.Error())
	}

	return Render(c, admin.ShippingConfig(c, config))
}

func (h *AdminHandler) HandleSaveShippingConfig(c echo.Context) error {
	ctx := c.Request().Context()

	// Load current config
	config, err := shipping.LoadShippingConfigFromDB(ctx, h.storage.Queries)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to load current config: "+err.Error())
	}

	// Parse item weights for each size category
	sizes := []string{"small", "medium", "large", "xlarge"}
	for _, size := range sizes {
		minGrams, _ := strconv.ParseFloat(c.FormValue(fmt.Sprintf("item_weights_%s_min_grams", size)), 64)
		maxGrams, _ := strconv.ParseFloat(c.FormValue(fmt.Sprintf("item_weights_%s_max_grams", size)), 64)
		avgGrams, _ := strconv.ParseFloat(c.FormValue(fmt.Sprintf("item_weights_%s_avg_grams", size)), 64)
		avgOz, _ := strconv.ParseFloat(c.FormValue(fmt.Sprintf("item_weights_%s_avg_oz", size)), 64)

		config.Packing.ItemWeights[size] = shipping.ItemWeights{
			MinGrams: minGrams,
			MaxGrams: maxGrams,
			AvgGrams: avgGrams,
			AvgOz:    avgOz,
		}
	}

	// Parse packing materials
	config.Packing.PackingMaterials.BubbleWrapPerItemOz, _ = strconv.ParseFloat(c.FormValue("bubble_wrap_per_item_oz"), 64)
	config.Packing.PackingMaterials.PackingPaperPerBoxOz, _ = strconv.ParseFloat(c.FormValue("packing_paper_per_box_oz"), 64)
	config.Packing.PackingMaterials.TapeAndLabelsPerBoxOz, _ = strconv.ParseFloat(c.FormValue("tape_and_labels_per_box_oz"), 64)
	config.Packing.PackingMaterials.AirPillowsPerBoxOz, _ = strconv.ParseFloat(c.FormValue("air_pillows_per_box_oz"), 64)
	config.Packing.PackingMaterials.HandlingFeePerBoxUSD, _ = strconv.ParseFloat(c.FormValue("handling_fee_per_box_usd"), 64)

	// Parse dimension guards
	for _, size := range sizes {
		L, _ := strconv.ParseFloat(c.FormValue(fmt.Sprintf("dimension_guard_%s_L", size)), 64)
		W, _ := strconv.ParseFloat(c.FormValue(fmt.Sprintf("dimension_guard_%s_W", size)), 64)
		H, _ := strconv.ParseFloat(c.FormValue(fmt.Sprintf("dimension_guard_%s_H", size)), 64)

		config.Packing.DimensionGuard[size] = shipping.DimensionGuard{
			L: L,
			W: W,
			H: H,
		}
	}

	// Parse other settings
	config.Packing.UnitVolumeIn3, _ = strconv.ParseFloat(c.FormValue("unit_volume_in3"), 64)
	config.Packing.FillRatio, _ = strconv.ParseFloat(c.FormValue("fill_ratio"), 64)

	// Marshal the updated config to JSON
	configJSON, err := json.Marshal(config)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to marshal config: "+err.Error())
	}

	// Update in database
	_, err = h.storage.Queries.UpdateShippingConfig(ctx, string(configJSON))
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to save config: "+err.Error())
	}

	// Reload configuration in the running shipping service
	if h.shippingService != nil {
		h.shippingService.UpdateConfig(config)
	}

	return c.Redirect(http.StatusSeeOther, "/admin/shipping/config")
}

func (h *AdminHandler) HandleShippingSettings(c echo.Context) error {
	// Load shipping configuration from database
	config, err := shipping.LoadShippingConfigFromDB(c.Request().Context(), h.storage.Queries)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to load shipping configuration: "+err.Error())
	}

	return Render(c, admin.ShippingSettings(c, config))
}

func (h *AdminHandler) HandleSaveShippingSettings(c echo.Context) error {
	ctx := c.Request().Context()

	// Load current config
	config, err := shipping.LoadShippingConfigFromDB(ctx, h.storage.Queries)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to load current config: "+err.Error())
	}

	// Update USPS ship-from address
	config.Shipping.ShipFromUSPS.Name = c.FormValue("ship_from_usps_name")
	config.Shipping.ShipFromUSPS.Phone = c.FormValue("ship_from_usps_phone")
	config.Shipping.ShipFromUSPS.AddressLine1 = c.FormValue("ship_from_usps_address_line1")
	config.Shipping.ShipFromUSPS.CityLocality = c.FormValue("ship_from_usps_city_locality")
	config.Shipping.ShipFromUSPS.StateProvince = c.FormValue("ship_from_usps_state_province")
	config.Shipping.ShipFromUSPS.PostalCode = c.FormValue("ship_from_usps_postal_code")
	config.Shipping.ShipFromUSPS.CountryCode = c.FormValue("ship_from_usps_country_code")

	// Update Other carriers ship-from address
	config.Shipping.ShipFromOther.Name = c.FormValue("ship_from_other_name")
	config.Shipping.ShipFromOther.Phone = c.FormValue("ship_from_other_phone")
	config.Shipping.ShipFromOther.AddressLine1 = c.FormValue("ship_from_other_address_line1")
	config.Shipping.ShipFromOther.CityLocality = c.FormValue("ship_from_other_city_locality")
	config.Shipping.ShipFromOther.StateProvince = c.FormValue("ship_from_other_state_province")
	config.Shipping.ShipFromOther.PostalCode = c.FormValue("ship_from_other_postal_code")
	config.Shipping.ShipFromOther.CountryCode = c.FormValue("ship_from_other_country_code")

	// Update rate preferences
	presentTopN, err := strconv.Atoi(c.FormValue("present_top_n"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid present_top_n value")
	}
	config.Shipping.RatePreferences.PresentTopN = presentTopN
	config.Shipping.RatePreferences.Sort = c.FormValue("sort")

	// Update label format
	config.Shipping.Labels.Format = c.FormValue("label_format")

	// Marshal the updated config to JSON
	configJSON, err := json.Marshal(config)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to marshal config: "+err.Error())
	}

	// Update in database
	_, err = h.storage.Queries.UpdateShippingConfig(ctx, string(configJSON))
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to save config: "+err.Error())
	}

	// Reload configuration in the running shipping service
	if h.shippingService != nil {
		h.shippingService.UpdateConfig(config)
	}

	return c.Redirect(http.StatusSeeOther, "/admin/shipping/settings")
}

// Email Preview Handlers

func (h *AdminHandler) HandleEmailPreview(c echo.Context) error {
	// Create sample order data
	sampleData := createSampleOrderData()

	// Render both email templates
	customerHTML, err := email.RenderCustomerOrderEmail(sampleData)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to render customer email template: "+err.Error())
	}

	adminHTML, err := email.RenderAdminOrderEmail(sampleData)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to render admin email template: "+err.Error())
	}

	// Create sample contact request data
	contactData := createSampleContactRequestData()
	contactHTML, err := email.RenderContactRequestEmail(contactData)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to render contact request email template: "+err.Error())
	}

	return Render(c, admin.EmailPreview(c, customerHTML, adminHTML, contactHTML))
}

func (h *AdminHandler) HandleEmailPreviewCustomer(c echo.Context) error {
	// Create sample order data
	sampleData := createSampleOrderData()

	// Render the customer email template
	html, err := email.RenderCustomerOrderEmail(sampleData)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to render email template: "+err.Error())
	}

	return c.HTML(http.StatusOK, html)
}

func (h *AdminHandler) HandleEmailPreviewAdmin(c echo.Context) error {
	// Create sample order data
	sampleData := createSampleOrderData()

	// Render the admin email template
	html, err := email.RenderAdminOrderEmail(sampleData)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to render email template: "+err.Error())
	}

	return c.HTML(http.StatusOK, html)
}

// createSampleOrderData creates sample order data for email preview
func createSampleOrderData() *email.OrderData {
	return &email.OrderData{
		OrderID:       "SAMPLE-12345",
		CustomerName:  "John Doe",
		CustomerEmail: "john.doe@example.com",
		OrderDate:     "October 22, 2025 at 9:30 PM",
		Items: []email.OrderItem{
			{
				ProductName:  "Pachycephalosaurus",
				ProductImage: "pachycephalosaurus.jpg",
				Quantity:     2,
				PriceCents:   2999, // $29.99
				TotalCents:   5998, // $59.98
			},
			{
				ProductName:  "Crystal Dragon with Wings",
				ProductImage: "crystal_dragon_with_wings.jpeg",
				Quantity:     1,
				PriceCents:   1999, // $19.99
				TotalCents:   1999, // $19.99
			},
		},
		SubtotalCents: 7997,  // $79.97
		TaxCents:      547,   // $5.47
		ShippingCents: 750,   // $7.50
		TotalCents:    9294,  // $92.94
		ShippingAddress: email.Address{
			Name:       "John Doe",
			Line1:      "123 Main Street",
			Line2:      "Apt 4B",
			City:       "Springfield",
			State:      "IL",
			PostalCode: "62701",
			Country:    "US",
		},
		BillingAddress: email.Address{
			Name:       "John Doe",
			Line1:      "123 Main Street",
			Line2:      "Apt 4B",
			City:       "Springfield",
			State:      "IL",
			PostalCode: "62701",
			Country:    "US",
		},
		PaymentIntentID: "pi_1234567890abcdef",
	}
}

// createSampleContactRequestData creates sample contact request data for email preview
func createSampleContactRequestData() *email.ContactRequestData {
	return &email.ContactRequestData{
		ID:                  "01SAMPLE1234567890",
		FirstName:           "Jane",
		LastName:            "Smith",
		Email:               "jane.smith@example.com",
		Phone:               "(555) 123-4567",
		Subject:             "General Inquiry",
		Message:             "Hello, I'm interested in learning more about your custom 3D printing services. I have a unique project in mind and would love to discuss the possibilities with your team. Can you provide information about pricing and turnaround times? Thank you!",
		NewsletterSubscribe: true,
		IPAddress:           "192.168.1.100",
		UserAgent:           "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
		Referrer:            "https://www.google.com",
		SubmittedAt:         "October 23, 2025 at 1:30 AM",
	}
}

func (h *AdminHandler) HandleSendTestEmail(c echo.Context) error {
	var request struct {
		Email string `json:"email"`
		Type  string `json:"type"`
	}

	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if request.Email == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Email address is required",
		})
	}

	if request.Type != "customer" && request.Type != "admin" && request.Type != "contact" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid email type",
		})
	}

	// Send the appropriate email
	var err error
	if request.Type == "contact" {
		// Create sample contact request data
		contactData := createSampleContactRequestData()
		contactData.Email = request.Email
		err = h.emailService.SendContactRequestNotification(contactData)
	} else {
		// Create sample order data
		sampleData := createSampleOrderData()
		sampleData.CustomerEmail = request.Email

		if request.Type == "customer" {
			err = h.emailService.SendOrderConfirmation(sampleData)
		} else {
			err = h.emailService.SendOrderNotificationToAdmin(sampleData)
		}
	}

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to send test email: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Test email sent successfully to %s", request.Email),
	})
}
// HandleContactsList displays all contact requests
func (h *AdminHandler) HandleContactsList(c echo.Context) error {
	ctx := c.Request().Context()

	// Get filter parameters
	status := c.QueryParam("status")
	priority := c.QueryParam("priority")
	subject := c.QueryParam("subject")
	search := c.QueryParam("search")

	// Check if "show_all" parameter is set (used when clearing filters)
	showAll := c.QueryParam("show_all")

	var contacts []db.ContactRequest
	var err error

	if search != "" {
		searchPattern := "%" + search + "%"
		contacts, err = h.storage.Queries.SearchContactRequests(ctx, db.SearchContactRequestsParams{
			FirstName: searchPattern,
			LastName:  searchPattern,
			Email:     sql.NullString{String: searchPattern, Valid: true},
			Message:   searchPattern,
			Limit:     100,
			Offset:    0,
		})
	} else if status != "" || priority != "" || subject != "" {
		var statusParam, priorityParam, subjectParam interface{}
		if status != "" {
			statusParam = status
		}
		if priority != "" {
			priorityParam = priority
		}
		if subject != "" {
			subjectParam = subject
		}

		contacts, err = h.storage.Queries.FilterContactRequests(ctx, db.FilterContactRequestsParams{
			Status:     statusParam,
			Priority:   priorityParam,
			Subject:    subjectParam,
			AssignedTo: nil,
			Offset:     0,
			Limit:      100,
		})
	} else if showAll == "true" {
		// Explicitly show all contacts (used when clearing filters)
		contacts, err = h.storage.Queries.ListContactRequests(ctx, db.ListContactRequestsParams{
			Limit:  100,
			Offset: 0,
		})
	} else {
		// Default: show only active requests (exclude resolved and spam)
		contacts, err = h.storage.Queries.ListActiveContactRequests(ctx, db.ListActiveContactRequestsParams{
			Limit:  100,
			Offset: 0,
		})
	}

	if err != nil {
		slog.Error("failed to fetch contact requests", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to load contact requests")
	}

	// Get stats
	stats, err := h.storage.Queries.GetContactRequestStats(ctx)
	if err != nil {
		slog.Error("failed to fetch contact stats", "error", err)
	}

	return Render(c, admin.ContactsList(c, contacts, stats, status, priority, subject, search))
}

// HandleContactsTable returns just the contacts table for HTMX updates
func (h *AdminHandler) HandleContactsTable(c echo.Context) error {
	ctx := c.Request().Context()

	// Get filter parameters
	status := c.QueryParam("status")
	priority := c.QueryParam("priority")
	subject := c.QueryParam("subject")
	search := c.QueryParam("search")

	// Check if "show_all" parameter is set (used when clearing filters)
	showAll := c.QueryParam("show_all")

	var contacts []db.ContactRequest
	var err error

	if search != "" {
		searchPattern := "%" + search + "%"
		contacts, err = h.storage.Queries.SearchContactRequests(ctx, db.SearchContactRequestsParams{
			FirstName: searchPattern,
			LastName:  searchPattern,
			Email:     sql.NullString{String: searchPattern, Valid: true},
			Message:   searchPattern,
			Limit:     100,
			Offset:    0,
		})
	} else if status != "" || priority != "" || subject != "" {
		var statusParam, priorityParam, subjectParam interface{}
		if status != "" {
			statusParam = status
		}
		if priority != "" {
			priorityParam = priority
		}
		if subject != "" {
			subjectParam = subject
		}

		contacts, err = h.storage.Queries.FilterContactRequests(ctx, db.FilterContactRequestsParams{
			Status:     statusParam,
			Priority:   priorityParam,
			Subject:    subjectParam,
			AssignedTo: nil,
			Offset:     0,
			Limit:      100,
		})
	} else if showAll == "true" {
		// Explicitly show all contacts (used when clearing filters)
		contacts, err = h.storage.Queries.ListContactRequests(ctx, db.ListContactRequestsParams{
			Limit:  100,
			Offset: 0,
		})
	} else {
		// Default: show only active requests (exclude resolved and spam)
		contacts, err = h.storage.Queries.ListActiveContactRequests(ctx, db.ListActiveContactRequestsParams{
			Limit:  100,
			Offset: 0,
		})
	}

	if err != nil {
		slog.Error("failed to fetch contact requests", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to load contact requests")
	}

	return Render(c, admin.ContactsTable(contacts))
}

// HandleContactDetail displays a single contact request
func (h *AdminHandler) HandleContactDetail(c echo.Context) error {
	ctx := c.Request().Context()
	id := c.Param("id")

	contact, err := h.storage.Queries.GetContactRequest(ctx, id)
	if err != nil {
		slog.Error("failed to fetch contact request", "error", err, "id", id)
		return c.String(http.StatusNotFound, "Contact request not found")
	}

	return Render(c, admin.ContactDetail(c, contact))
}

// HandleUpdateContactStatus updates the status of a contact request
func (h *AdminHandler) HandleUpdateContactStatus(c echo.Context) error {
	ctx := c.Request().Context()
	id := c.Param("id")
	status := c.FormValue("status")

	if status == "" {
		return c.String(http.StatusBadRequest, "Status is required")
	}

	err := h.storage.Queries.UpdateContactRequestStatus(ctx, db.UpdateContactRequestStatusParams{
		Status: sql.NullString{String: status, Valid: true},
		ID:     id,
	})

	if err != nil {
		slog.Error("failed to update contact status", "error", err, "id", id)
		return c.String(http.StatusInternalServerError, "Failed to update status")
	}

	// Return updated contact or redirect
	return c.Redirect(http.StatusSeeOther, "/admin/contacts/"+id)
}

// HandleUpdateContactPriority updates the priority of a contact request
func (h *AdminHandler) HandleUpdateContactPriority(c echo.Context) error {
	ctx := c.Request().Context()
	id := c.Param("id")
	priority := c.FormValue("priority")

	if priority == "" {
		return c.String(http.StatusBadRequest, "Priority is required")
	}

	err := h.storage.Queries.UpdateContactRequestPriority(ctx, db.UpdateContactRequestPriorityParams{
		Priority: sql.NullString{String: priority, Valid: true},
		ID:       id,
	})

	if err != nil {
		slog.Error("failed to update contact priority", "error", err, "id", id)
		return c.String(http.StatusInternalServerError, "Failed to update priority")
	}

	return c.Redirect(http.StatusSeeOther, "/admin/contacts/"+id)
}

// HandleAddContactNotes adds notes to a contact request
func (h *AdminHandler) HandleAddContactNotes(c echo.Context) error {
	ctx := c.Request().Context()
	id := c.Param("id")
	notes := c.FormValue("notes")

	if notes == "" {
		return c.String(http.StatusBadRequest, "Notes are required")
	}

	err := h.storage.Queries.AddContactRequestResponse(ctx, db.AddContactRequestResponseParams{
		ResponseNotes: sql.NullString{String: notes, Valid: true},
		ID:            id,
	})

	if err != nil {
		slog.Error("failed to add contact notes", "error", err, "id", id)
		return c.String(http.StatusInternalServerError, "Failed to add notes")
	}

	return c.Redirect(http.StatusSeeOther, "/admin/contacts/"+id)
}

func (h *AdminHandler) HandleDeleteContactNotes(c echo.Context) error {
	ctx := c.Request().Context()
	id := c.Param("id")

	err := h.storage.Queries.AddContactRequestResponse(ctx, db.AddContactRequestResponseParams{
		ResponseNotes: sql.NullString{String: "", Valid: false},
		ID:            id,
	})

	if err != nil {
		slog.Error("failed to delete contact notes", "error", err, "id", id)
		return c.String(http.StatusInternalServerError, "Failed to delete notes")
	}

	return c.Redirect(http.StatusSeeOther, "/admin/contacts/"+id)
}
