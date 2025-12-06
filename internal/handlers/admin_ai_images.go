package handlers

import (
	"bytes"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/internal/ogimage"
	"github.com/loganlanou/logans3d-v4/storage"
	"github.com/loganlanou/logans3d-v4/storage/db"
	"github.com/loganlanou/logans3d-v4/views/admin"
)

type AIBackgroundHandler struct {
	storage     *storage.Storage
	aiGenerator *ogimage.AIGenerator
}

func NewAIBackgroundHandler(storage *storage.Storage, geminiAPIKey string) *AIBackgroundHandler {
	h := &AIBackgroundHandler{
		storage: storage,
	}
	if geminiAPIKey != "" {
		h.aiGenerator = ogimage.NewAIGenerator(geminiAPIKey)
	}
	return h
}

// HandleGenerateAIBackground generates a new AI background for a product
// POST /admin/product/:id/generate-background
func (h *AIBackgroundHandler) HandleGenerateAIBackground(c echo.Context) error {
	if h.aiGenerator == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "AI generation not available")
	}

	ctx := c.Request().Context()
	productID := c.Param("id")

	// Get product
	product, err := h.storage.Queries.GetProduct(ctx, productID)
	if err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "Product not found")
		}
		slog.Error("failed to get product", "error", err, "product_id", productID)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load product")
	}

	// Get primary product image
	images, err := h.storage.Queries.GetProductImages(ctx, productID)
	if err != nil || len(images) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "No product images found")
	}

	// Find primary image
	var sourceImage db.ProductImage
	for _, img := range images {
		if img.IsPrimary.Valid && img.IsPrimary.Bool {
			sourceImage = img
			break
		}
	}
	if sourceImage.ID == "" && len(images) > 0 {
		sourceImage = images[0]
	}

	// Build source image path
	sourceImagePath := filepath.Join("public", "images", "products", sourceImage.ImageUrl)

	// Generate pending image filename and path
	pendingID := uuid.New().String()
	pendingFilename := fmt.Sprintf("%s_%d.png", productID, time.Now().Unix())
	pendingDir := filepath.Join("public", "images", "products", "pending")
	pendingPath := filepath.Join(pendingDir, pendingFilename)

	// Ensure pending directory exists
	if err := os.MkdirAll(pendingDir, 0755); err != nil {
		slog.Error("failed to create pending directory", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create pending directory")
	}

	// Generate AI background
	info := ogimage.SingleProductBackgroundInfo{
		Name:      product.Name,
		ImagePath: sourceImagePath,
	}

	modelUsed, err := h.aiGenerator.GenerateSingleProductBackground(info, pendingPath)
	if err != nil {
		slog.Error("failed to generate AI background", "error", err, "product_id", productID)
		return echo.NewHTTPError(http.StatusInternalServerError, "AI generation failed: "+err.Error())
	}

	// Create pending record in database
	_, err = h.storage.Queries.CreatePendingAIImage(ctx, db.CreatePendingAIImageParams{
		ID:                pendingID,
		ProductID:         productID,
		SourceImageUrl:    sourceImage.ImageUrl,
		GeneratedImageUrl: pendingFilename,
		ModelUsed:         sql.NullString{String: modelUsed, Valid: true},
	})
	if err != nil {
		slog.Error("failed to create pending AI image record", "error", err)
		// Clean up the generated file
		os.Remove(pendingPath)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to save pending image")
	}

	// Return updated pending list
	return h.renderPendingBackgrounds(c, productID)
}

// HandleGetPendingBackgrounds returns the list of pending backgrounds for a product
// GET /admin/product/:id/pending-backgrounds
func (h *AIBackgroundHandler) HandleGetPendingBackgrounds(c echo.Context) error {
	productID := c.Param("id")
	return h.renderPendingBackgrounds(c, productID)
}

// HandleApproveBackground approves a pending background and adds it as a product image
// POST /admin/pending-background/:id/approve
func (h *AIBackgroundHandler) HandleApproveBackground(c echo.Context) error {
	ctx := c.Request().Context()
	pendingID := c.Param("id")

	// Get pending image record
	pending, err := h.storage.Queries.GetPendingAIImage(ctx, pendingID)
	if err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "Pending image not found")
		}
		slog.Error("failed to get pending image", "error", err, "pending_id", pendingID)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load pending image")
	}

	// Move file from pending to products directory
	pendingPath := filepath.Join("public", "images", "products", "pending", pending.GeneratedImageUrl)
	newFilename := fmt.Sprintf("ai_%s", pending.GeneratedImageUrl)
	newPath := filepath.Join("public", "images", "products", newFilename)

	if err := os.Rename(pendingPath, newPath); err != nil {
		slog.Error("failed to move pending image", "error", err, "from", pendingPath, "to", newPath)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to move image")
	}

	// Create product image record (not primary by default)
	_, err = h.storage.Queries.CreateProductImage(ctx, db.CreateProductImageParams{
		ID:        uuid.New().String(),
		ProductID: pending.ProductID,
		ImageUrl:  newFilename,
		AltText:   sql.NullString{String: "AI-generated background", Valid: true},
		IsPrimary: sql.NullBool{Bool: false, Valid: true},
	})
	if err != nil {
		slog.Error("failed to create product image", "error", err)
		// Try to move file back
		if renameErr := os.Rename(newPath, pendingPath); renameErr != nil {
			slog.Error("failed to move file back to pending", "error", renameErr, "from", newPath, "to", pendingPath)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to save image")
	}

	// Update pending status to approved
	err = h.storage.Queries.UpdatePendingAIImageStatus(ctx, db.UpdatePendingAIImageStatusParams{
		Status: "approved",
		ID:     pendingID,
	})
	if err != nil {
		slog.Debug("failed to update pending status", "error", err)
	}

	// Return updated pending list AND product images grid (out-of-band swap)
	return h.renderPendingBackgroundsWithImagesGrid(c, pending.ProductID)
}

// HandleRejectBackground rejects and deletes a pending background
// POST /admin/pending-background/:id/reject
func (h *AIBackgroundHandler) HandleRejectBackground(c echo.Context) error {
	ctx := c.Request().Context()
	pendingID := c.Param("id")

	// Get pending image record
	pending, err := h.storage.Queries.GetPendingAIImage(ctx, pendingID)
	if err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "Pending image not found")
		}
		slog.Error("failed to get pending image", "error", err, "pending_id", pendingID)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load pending image")
	}

	// Delete the pending file
	pendingPath := filepath.Join("public", "images", "products", "pending", pending.GeneratedImageUrl)
	if err := os.Remove(pendingPath); err != nil {
		slog.Debug("failed to remove pending file", "error", err, "path", pendingPath)
	}

	// Update pending status to rejected
	err = h.storage.Queries.UpdatePendingAIImageStatus(ctx, db.UpdatePendingAIImageStatusParams{
		Status: "rejected",
		ID:     pendingID,
	})
	if err != nil {
		slog.Debug("failed to update pending status", "error", err)
	}

	// Return updated pending list
	return h.renderPendingBackgrounds(c, pending.ProductID)
}

// renderPendingBackgrounds renders the pending backgrounds list partial
func (h *AIBackgroundHandler) renderPendingBackgrounds(c echo.Context, productID string) error {
	ctx := c.Request().Context()

	pending, err := h.storage.Queries.GetPendingAIImagesByProduct(ctx, productID)
	if err != nil {
		slog.Debug("failed to get pending images", "error", err)
		pending = []db.PendingAiImage{}
	}

	// Return HTML partial for HTMX
	c.Response().Header().Set("Content-Type", "text/html")

	return c.HTML(http.StatusOK, h.buildPendingBackgroundsHTML(pending))
}

// renderPendingBackgroundsWithImagesGrid renders both pending backgrounds and the product images grid
// The images grid is returned as an out-of-band swap so HTMX updates both sections
func (h *AIBackgroundHandler) renderPendingBackgroundsWithImagesGrid(c echo.Context, productID string) error {
	ctx := c.Request().Context()

	// Get pending images
	pending, err := h.storage.Queries.GetPendingAIImagesByProduct(ctx, productID)
	if err != nil {
		slog.Debug("failed to get pending images", "error", err)
		pending = []db.PendingAiImage{}
	}

	// Get product images for the grid
	images, err := h.storage.Queries.GetProductImages(ctx, productID)
	if err != nil {
		slog.Debug("failed to get product images", "error", err)
		images = []db.ProductImage{}
	}

	// Render the product images grid using the templ component
	var gridBuf bytes.Buffer
	if err := admin.ProductImagesGrid(productID, images).Render(ctx, &gridBuf); err != nil {
		slog.Error("failed to render product images grid", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to render images")
	}

	// Build the response: pending backgrounds + out-of-band images grid
	c.Response().Header().Set("Content-Type", "text/html")

	// The pending backgrounds HTML (this is the main swap target)
	html := h.buildPendingBackgroundsHTML(pending)

	// Get the rendered grid HTML and inject the hx-swap-oob attribute
	// The ProductImagesGrid already has id="product-images-grid", so we just need to add the oob attribute
	gridHTML := gridBuf.String()
	// Replace the opening div to add hx-swap-oob
	gridHTML = `<div id="product-images-grid" hx-swap-oob="outerHTML"` + gridHTML[len(`<div id="product-images-grid"`):]
	html += gridHTML

	return c.HTML(http.StatusOK, html)
}

// buildPendingBackgroundsHTML builds the HTML for the pending backgrounds list
func (h *AIBackgroundHandler) buildPendingBackgroundsHTML(pending []db.PendingAiImage) string {
	if len(pending) == 0 {
		return `<p class="text-sm text-muted-foreground">No pending AI backgrounds</p>`
	}

	// Build HTML for pending images
	html := `<div class="grid grid-cols-2 gap-4 mt-4">`
	for _, bg := range pending {
		html += fmt.Sprintf(`
			<div class="relative border border-border rounded-lg overflow-hidden">
				<img src="/public/images/products/pending/%s" alt="Pending AI background" class="w-full h-auto" />
				<div class="absolute bottom-0 inset-x-0 bg-black/60 p-2 flex gap-2">
					<button
						type="button"
						hx-post="/admin/pending-background/%s/approve"
						hx-target="#pending-backgrounds"
						hx-swap="innerHTML"
						class="flex-1 bg-green-600 hover:bg-green-700 text-white rounded px-2 py-1 text-sm font-medium transition-colors"
					>
						✓ Approve
					</button>
					<button
						type="button"
						hx-post="/admin/pending-background/%s/reject"
						hx-target="#pending-backgrounds"
						hx-swap="innerHTML"
						class="flex-1 bg-red-600 hover:bg-red-700 text-white rounded px-2 py-1 text-sm font-medium transition-colors"
					>
						✗ Reject
					</button>
				</div>
				<span class="absolute top-2 right-2 text-xs bg-black/60 text-white px-2 py-1 rounded">
					%s
				</span>
			</div>`,
			bg.GeneratedImageUrl,
			bg.ID,
			bg.ID,
			bg.ModelUsed.String,
		)
	}
	html += `</div>`

	return html
}
