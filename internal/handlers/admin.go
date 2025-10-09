package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/internal/auth"
	"github.com/loganlanou/logans3d-v4/internal/types"
	"github.com/loganlanou/logans3d-v4/storage"
	"github.com/loganlanou/logans3d-v4/storage/db"
	"github.com/loganlanou/logans3d-v4/views/admin"
)


type AdminHandler struct {
	storage     *storage.Storage
	authService *auth.Service
}

func NewAdminHandler(storage *storage.Storage, authService *auth.Service) *AdminHandler {
	return &AdminHandler{
		storage:     storage,
		authService: authService,
	}
}

func (h *AdminHandler) HandleAdminDashboard(c echo.Context) error {
	authCtx := auth.NewContext(c, h.authService)

	products, err := h.storage.Queries.ListProducts(c.Request().Context())
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch products")
	}

	productsWithImages := h.buildProductsWithImages(c.Request().Context(), products)

	return Render(c, admin.Dashboard(productsWithImages, authCtx))
}

func (h *AdminHandler) HandleCategoriesTab(c echo.Context) error {
	authCtx := auth.NewContext(c, h.authService)

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

	return Render(c, admin.CategoriesTab(productsWithImages, categories, filter, authCtx))
}


func (h *AdminHandler) buildProductsWithImages(ctx context.Context, products []db.Product) []types.ProductWithImage {
	// Calculate cutoff date for "new" items (60 days ago)
	newItemCutoff := time.Now().AddDate(0, 0, -60)

	// Get products with their primary images
	productsWithImages := make([]types.ProductWithImage, 0, len(products))
	for _, product := range products {
		images, err := h.storage.Queries.GetProductImages(ctx, product.ID)
		if err != nil {
			// Continue without image if there's an error
			// Check if product is new (within last 60 days)
			isNew := product.CreatedAt.Valid && product.CreatedAt.Time.After(newItemCutoff)
			
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

		// Check if product is new (within last 60 days)
		isNew := product.CreatedAt.Valid && product.CreatedAt.Time.After(newItemCutoff)
		
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
	authCtx := auth.NewContext(c, h.authService)

	productID := c.QueryParam("id")
	var product *db.Product

	if productID != "" {
		p, err := h.storage.Queries.GetProduct(c.Request().Context(), productID)
		if err != nil && err != sql.ErrNoRows {
			return c.String(http.StatusInternalServerError, "Failed to fetch product")
		}
		if err == nil {
			product = &p
		}
	}

	categories, err := h.storage.Queries.ListCategories(c.Request().Context())
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch categories")
	}

	return Render(c, admin.ProductForm(product, categories, authCtx))
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
		IsFeatured:       sql.NullBool{Bool: isPremiumCollection, Valid: true},
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

			// Create uploads directory if it doesn't exist
			uploadDir := "public/uploads/products"
			if err := os.MkdirAll(uploadDir, 0755); err != nil {
				return c.String(http.StatusInternalServerError, "Failed to create upload directory")
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
			// TODO: Save to database using CreateProductImage
			_ = imageFilename
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
		IsFeatured:       sql.NullBool{Bool: isPremiumCollection, Valid: true},
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

			uploadDir := "public/uploads/products"
			if err := os.MkdirAll(uploadDir, 0755); err != nil {
				return c.String(http.StatusInternalServerError, "Failed to create upload directory")
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
			// TODO: Save to database using UpdateProductImage
			_ = imageFilename
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

// Category Management Functions

func (h *AdminHandler) HandleCategoryForm(c echo.Context) error {
	authCtx := auth.NewContext(c, h.authService)

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

	categories, err := h.storage.Queries.ListCategories(c.Request().Context())
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch categories")
	}

	return Render(c, admin.CategoryForm(category, categories, authCtx))
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
	authCtx := auth.NewContext(c, h.authService)

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

	return Render(c, admin.DeveloperDashboard(sysInfo, dbStats, memStats, authCtx))
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
	authCtx := auth.NewContext(c, h.authService)

	status := c.QueryParam("status")

	var orders []db.Order
	var err error

	if status != "" {
		orders, err = h.storage.Queries.ListOrdersByStatus(c.Request().Context(), sql.NullString{String: status, Valid: true})
	} else {
		orders, err = h.storage.Queries.ListOrders(c.Request().Context())
	}

	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch orders")
	}

	return Render(c, admin.OrdersList(orders, authCtx))
}

func (h *AdminHandler) HandleOrderDetail(c echo.Context) error {
	orderID := c.Param("id")
	
	_, err := h.storage.Queries.GetOrder(c.Request().Context(), orderID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.String(http.StatusNotFound, "Order not found")
		}
		return c.String(http.StatusInternalServerError, "Failed to fetch order")
	}
	
	// For now, just redirect back to orders list
	// In the future, implement order detail view
	return c.Redirect(http.StatusSeeOther, "/admin/orders")
}

func (h *AdminHandler) HandleUpdateOrderStatus(c echo.Context) error {
	orderID := c.Param("id")
	status := c.FormValue("status")
	
	if status == "" {
		return c.String(http.StatusBadRequest, "Status is required")
	}
	
	_, err := h.storage.Queries.UpdateOrderStatus(c.Request().Context(), db.UpdateOrderStatusParams{
		ID:     orderID,
		Status: sql.NullString{String: status, Valid: true},
	})
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to update order status")
	}
	
	return c.Redirect(http.StatusSeeOther, "/admin/orders")
}

// Quotes Management Functions

func (h *AdminHandler) HandleQuotesList(c echo.Context) error {
	authCtx := auth.NewContext(c, h.authService)

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

	return Render(c, admin.QuotesList(quotes, authCtx))
}

func (h *AdminHandler) HandleQuoteDetail(c echo.Context) error {
	authCtx := auth.NewContext(c, h.authService)

	quoteID := c.Param("id")

	quote, err := h.storage.Queries.GetQuoteRequest(c.Request().Context(), quoteID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.String(http.StatusNotFound, "Quote request not found")
		}
		return c.String(http.StatusInternalServerError, "Failed to fetch quote request")
	}

	return Render(c, admin.QuoteDetail(quote, authCtx))
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
	authCtx := auth.NewContext(c, h.authService)

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

	return Render(c, admin.EventsList(events, authCtx))
}

func (h *AdminHandler) HandleEventForm(c echo.Context) error {
	authCtx := auth.NewContext(c, h.authService)

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

	return Render(c, admin.EventForm(event, authCtx))
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