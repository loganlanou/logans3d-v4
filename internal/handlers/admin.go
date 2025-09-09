package handlers

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
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

	return Render(c, admin.Dashboard(products, categories))
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