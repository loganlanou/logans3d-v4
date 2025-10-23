package handlers

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
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
	"github.com/loganlanou/logans3d-v4/internal/shipping"
	"github.com/loganlanou/logans3d-v4/internal/types"
	"github.com/loganlanou/logans3d-v4/storage"
	"github.com/loganlanou/logans3d-v4/storage/db"
	"github.com/loganlanou/logans3d-v4/views/admin"
)


type AdminHandler struct {
	storage         *storage.Storage
	shippingService *shipping.ShippingService
}

func NewAdminHandler(storage *storage.Storage, shippingService *shipping.ShippingService) *AdminHandler {
	return &AdminHandler{
		storage:         storage,
		shippingService: shippingService,
	}
}

func (h *AdminHandler) HandleAdminDashboard(c echo.Context) error {
	products, err := h.storage.Queries.ListProducts(c.Request().Context())
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch products")
	}

	productsWithImages := h.buildProductsWithImages(c.Request().Context(), products)

	return Render(c, admin.Dashboard(c, productsWithImages))
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

	return Render(c, admin.ProductForm(c, product, categories))
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
		return c.String(http.StatusInternalServerError, "Failed to fetch orders")
	}

	return Render(c, admin.OrdersList(c, orders))
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