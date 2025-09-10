package handlers

import (
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
	"github.com/loganlanou/logans3d-v4/internal/types"
	"github.com/loganlanou/logans3d-v4/storage"
	"github.com/loganlanou/logans3d-v4/storage/db"
	"github.com/loganlanou/logans3d-v4/views/admin"
)


type AdminHandler struct {
	storage *storage.Storage
}

func NewAdminHandler(storage *storage.Storage) *AdminHandler {
	return &AdminHandler{
		storage: storage,
	}
}

func (h *AdminHandler) HandleAdminDashboard(c echo.Context) error {
	products, err := h.storage.Queries.ListProducts(c.Request().Context())
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch products")
	}

	categories, err := h.storage.Queries.ListCategories(c.Request().Context())
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch categories")
	}

	// Get products with their primary images
	productsWithImages := make([]types.ProductWithImage, 0, len(products))
	for _, product := range products {
		images, err := h.storage.Queries.GetProductImages(c.Request().Context(), product.ID)
		if err != nil {
			// Continue without image if there's an error
			productsWithImages = append(productsWithImages, types.ProductWithImage{
				Product:  product,
				ImageURL: "",
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
			
			// Ensure the image URL has the correct path prefix
			if rawImageURL != "" {
				if strings.HasPrefix(rawImageURL, "/public/") {
					imageURL = rawImageURL
				} else {
					imageURL = "/public/images/products/" + rawImageURL
				}
			}
		}

		productsWithImages = append(productsWithImages, types.ProductWithImage{
			Product:  product,
			ImageURL: imageURL,
		})
	}

	return Render(c, admin.Dashboard(productsWithImages, categories))
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

	return Render(c, admin.ProductForm(product, categories))
}

func (h *AdminHandler) HandleCreateProduct(c echo.Context) error {
	name := c.FormValue("name")
	description := c.FormValue("description")
	shortDescription := c.FormValue("short_description")
	priceStr := c.FormValue("price")
	categoryID := c.FormValue("category_id")
	sku := c.FormValue("sku")
	stockQuantityStr := c.FormValue("stock_quantity")
	
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

			// Save image URL to database
			imageURL := "/public/uploads/products/" + filename
			// For now, we'll store the image URL in memory
			// We'll need to add the CreateProductImage query to the SQL files
			_ = imageURL
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
	isFeaturedStr := c.FormValue("is_featured")
	
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
	isFeatured := isFeaturedStr == "on" || isFeaturedStr == "true"

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
		IsActive:         sql.NullBool{Bool: isActive, Valid: true},
		IsFeatured:       sql.NullBool{Bool: isFeatured, Valid: true},
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

			// Save new image URL to database
			imageURL := "/public/uploads/products/" + filename
			// For now, we'll store the image URL in memory
			// We'll need to add the CreateProductImage query to the SQL files
			_ = imageURL
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

	return Render(c, admin.DeveloperDashboard(sysInfo, dbStats, memStats))
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