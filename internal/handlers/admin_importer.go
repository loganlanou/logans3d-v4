package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/internal/importer"
	"github.com/loganlanou/logans3d-v4/storage"
	"github.com/loganlanou/logans3d-v4/storage/db"
	"github.com/loganlanou/logans3d-v4/views/admin"
)

// AdminImporterHandler handles admin importer routes
type AdminImporterHandler struct {
	storage *storage.Storage
	scraper importer.Scraper
}

// NewAdminImporterHandler creates a new admin importer handler
func NewAdminImporterHandler(s *storage.Storage) *AdminImporterHandler {
	return &AdminImporterHandler{
		storage: s,
		scraper: importer.NewCults3DScraper(),
	}
}

// HandleImporterDashboard shows the main importer dashboard
func (h *AdminImporterHandler) HandleImporterDashboard(c echo.Context) error {
	ctx := c.Request().Context()

	// Build designer stats
	var designerStats []admin.DesignerStats
	for _, designer := range importer.Designers {
		stats := admin.DesignerStats{
			Designer: designer,
		}

		// Get scraped product count
		count, err := h.storage.Queries.CountScrapedProductsByDesigner(ctx, designer.Slug)
		if err == nil {
			stats.ScrapedCount = count
		}

		// Get unimported count
		unimported, err := h.storage.Queries.CountUnimportedProductsByDesigner(ctx, designer.Slug)
		if err == nil {
			stats.UnimportedCount = unimported
		}

		// Get last scrape time from most recent job
		for _, src := range designer.Sources {
			job, err := h.storage.Queries.GetLatestJobByDesigner(ctx, db.GetLatestJobByDesignerParams{
				DesignerSlug: designer.Slug,
				Platform:     src.Platform,
			})
			if err == nil && job.CompletedAt.Valid {
				if stats.LastScraped == nil || job.CompletedAt.Time.After(*stats.LastScraped) {
					t := job.CompletedAt.Time
					stats.LastScraped = &t
				}
			}
		}

		designerStats = append(designerStats, stats)
	}

	// Get recent jobs
	jobs, err := h.storage.Queries.ListImportJobs(ctx, db.ListImportJobsParams{
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		slog.Error("failed to list import jobs", "error", err)
		jobs = []db.ImportJob{}
	}

	data := admin.ImporterDashboardData{
		Designers:  designerStats,
		RecentJobs: jobs,
	}

	return admin.ImporterDashboard(c, data).Render(ctx, c.Response().Writer)
}

// HandleImporterDesignerDetail shows detail page for a designer
func (h *AdminImporterHandler) HandleImporterDesignerDetail(c echo.Context) error {
	ctx := c.Request().Context()
	slug := c.Param("slug")

	// Find designer
	designer := importer.GetDesigner(slug)
	if designer == nil {
		return echo.NewHTTPError(http.StatusNotFound, "Designer not found")
	}

	// Get stats
	stats := admin.DesignerStats{
		Designer: *designer,
	}

	count, err := h.storage.Queries.CountScrapedProductsByDesigner(ctx, slug)
	if err == nil {
		stats.ScrapedCount = count
	}

	unimported, err := h.storage.Queries.CountUnimportedProductsByDesigner(ctx, slug)
	if err == nil {
		stats.UnimportedCount = unimported
	}

	// Get scraped products
	products, err := h.storage.Queries.ListScrapedProductsByDesigner(ctx, db.ListScrapedProductsByDesignerParams{
		DesignerSlug: slug,
		Limit:        100,
		Offset:       0,
	})
	if err != nil {
		slog.Error("failed to list scraped products", "error", err, "designer", slug)
		products = []db.ScrapedProduct{}
	}

	// Get categories for import selector
	categories, err := h.storage.Queries.ListCategories(ctx)
	if err != nil {
		slog.Error("failed to list categories", "error", err)
		categories = []db.Category{}
	}

	data := admin.ImporterDesignerData{
		Designer:   *designer,
		Products:   products,
		Stats:      stats,
		Categories: categories,
	}

	return admin.ImporterDesignerDetail(c, data).Render(ctx, c.Response().Writer)
}

// HandleStartScrape starts a scrape job for a designer
func (h *AdminImporterHandler) HandleStartScrape(c echo.Context) error {
	ctx := c.Request().Context()
	slug := c.Param("slug")

	// Find designer
	designer := importer.GetDesigner(slug)
	if designer == nil {
		return echo.NewHTTPError(http.StatusNotFound, "Designer not found")
	}

	// Process each source
	for _, src := range designer.Sources {
		// Create job record
		jobID := uuid.New().String()
		_, err := h.storage.Queries.CreateImportJob(ctx, db.CreateImportJobParams{
			ID:           jobID,
			DesignerSlug: designer.Slug,
			Platform:     src.Platform,
			JobType:      "scrape",
		})
		if err != nil {
			slog.Error("failed to create import job", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to start scrape job")
		}

		// Run scrape in goroutine
		go h.runScrapeJob(jobID, *designer, src)
	}

	// Return success (job started)
	c.Response().Header().Set("HX-Trigger", "jobStarted")
	return c.NoContent(http.StatusAccepted)
}

// runScrapeJob runs the actual scrape operation
func (h *AdminImporterHandler) runScrapeJob(jobID string, designer importer.Designer, src importer.Source) {
	ctx := context.Background()

	slog.Info("starting scrape job", "job_id", jobID, "designer", designer.Slug, "platform", src.Platform)

	// Fetch all product URLs
	productURLs, err := h.scraper.FetchDesignerProducts(ctx, src.URL)
	if err != nil {
		slog.Error("failed to fetch designer products", "error", err, "job_id", jobID)
		h.storage.Queries.FailImportJob(ctx, db.FailImportJobParams{
			ErrorMessage: sql.NullString{String: err.Error(), Valid: true},
			ID:           jobID,
		})
		return
	}

	// Update total count
	err = h.storage.Queries.UpdateImportJobProgress(ctx, db.UpdateImportJobProgressParams{
		ProcessedItems: sql.NullInt64{Int64: 0, Valid: true},
		TotalItems:     sql.NullInt64{Int64: int64(len(productURLs)), Valid: true},
		ID:             jobID,
	})
	if err != nil {
		slog.Error("failed to update job progress", "error", err, "job_id", jobID)
	}

	slog.Info("found products to scrape", "count", len(productURLs), "job_id", jobID)

	// Fetch each product
	for i, productURL := range productURLs {
		product, err := h.scraper.FetchProduct(ctx, productURL)
		if err != nil {
			slog.Error("failed to fetch product", "error", err, "url", productURL, "job_id", jobID)
			continue
		}

		// Set designer slug
		product.DesignerSlug = designer.Slug

		// Convert to database params and upsert
		imageURLsJSON, _ := json.Marshal(product.ImageURLs)
		tagsJSON, _ := json.Marshal(product.Tags)

		var releaseDate sql.NullTime
		if product.ReleaseDate != nil {
			releaseDate = sql.NullTime{Time: *product.ReleaseDate, Valid: true}
		}

		_, err = h.storage.Queries.UpsertScrapedProduct(ctx, db.UpsertScrapedProductParams{
			ID:                 uuid.New().String(),
			DesignerSlug:       product.DesignerSlug,
			Platform:           product.Platform,
			SourceUrl:          product.SourceURL,
			Name:               product.Name,
			Description:        sql.NullString{String: product.Description, Valid: product.Description != ""},
			OriginalPriceCents: sql.NullInt64{Int64: int64(product.OriginalPriceCents), Valid: product.OriginalPriceCents > 0},
			ReleaseDate:        releaseDate,
			ImageUrls:          sql.NullString{String: string(imageURLsJSON), Valid: len(product.ImageURLs) > 0},
			Tags:               sql.NullString{String: string(tagsJSON), Valid: len(product.Tags) > 0},
			RawHtml:            sql.NullString{String: product.RawHTML, Valid: product.RawHTML != ""},
		})
		if err != nil {
			slog.Error("failed to upsert scraped product", "error", err, "url", productURL, "job_id", jobID)
			continue
		}

		// Update progress
		if (i+1)%10 == 0 || i == len(productURLs)-1 {
			err = h.storage.Queries.UpdateImportJobProgress(ctx, db.UpdateImportJobProgressParams{
				ProcessedItems: sql.NullInt64{Int64: int64(i + 1), Valid: true},
				TotalItems:     sql.NullInt64{Int64: int64(len(productURLs)), Valid: true},
				ID:             jobID,
			})
			if err != nil {
				slog.Error("failed to update job progress", "error", err, "job_id", jobID)
			}
		}
	}

	// Mark job complete
	err = h.storage.Queries.CompleteImportJob(ctx, jobID)
	if err != nil {
		slog.Error("failed to complete import job", "error", err, "job_id", jobID)
	}

	slog.Info("scrape job completed", "job_id", jobID, "products", len(productURLs))
}

// HandleImportProducts imports scraped products to the main products table
func (h *AdminImporterHandler) HandleImportProducts(c echo.Context) error {
	ctx := c.Request().Context()
	slug := c.Param("slug")

	// Find designer
	designer := importer.GetDesigner(slug)
	if designer == nil {
		return echo.NewHTTPError(http.StatusNotFound, "Designer not found")
	}

	// Get category ID from form (or use designer default)
	categoryID := c.FormValue("category_id")
	if categoryID == "" {
		// Look up the default category by name
		categories, err := h.storage.Queries.ListCategories(ctx)
		if err == nil {
			for _, cat := range categories {
				if cat.Name == designer.DefaultCategory {
					categoryID = cat.ID
					break
				}
			}
		}
	}

	// Create import job
	jobID := uuid.New().String()
	_, err := h.storage.Queries.CreateImportJob(ctx, db.CreateImportJobParams{
		ID:           jobID,
		DesignerSlug: designer.Slug,
		Platform:     "import",
		JobType:      "import",
	})
	if err != nil {
		slog.Error("failed to create import job", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to start import job")
	}

	// Run import in background
	go h.runImportJob(jobID, *designer, categoryID)

	c.Response().Header().Set("HX-Trigger", "jobStarted")
	return c.NoContent(http.StatusAccepted)
}

// runImportJob runs the actual import operation
func (h *AdminImporterHandler) runImportJob(jobID string, designer importer.Designer, categoryID string) {
	ctx := context.Background()

	slog.Info("starting import job", "job_id", jobID, "designer", designer.Slug, "category_id", categoryID)

	// Get unimported products
	products, err := h.storage.Queries.ListUnimportedProducts(ctx, db.ListUnimportedProductsParams{
		DesignerSlug: designer.Slug,
		Limit:        1000,
		Offset:       0,
	})
	if err != nil {
		slog.Error("failed to list unimported products", "error", err, "job_id", jobID)
		h.storage.Queries.FailImportJob(ctx, db.FailImportJobParams{
			ErrorMessage: sql.NullString{String: err.Error(), Valid: true},
			ID:           jobID,
		})
		return
	}

	// Update total count
	err = h.storage.Queries.UpdateImportJobProgress(ctx, db.UpdateImportJobProgressParams{
		ProcessedItems: sql.NullInt64{Int64: 0, Valid: true},
		TotalItems:     sql.NullInt64{Int64: int64(len(products)), Valid: true},
		ID:             jobID,
	})
	if err != nil {
		slog.Error("failed to update job progress", "error", err, "job_id", jobID)
	}

	slog.Info("found products to import", "count", len(products), "job_id", jobID)

	// Create image downloader
	downloader := importer.NewImageDownloader("public/images/products")

	// Import each product
	imported := 0
	for i, scraped := range products {
		err := h.importProduct(ctx, scraped, categoryID, designer.Name, downloader)
		if err != nil {
			slog.Error("failed to import product", "error", err, "product_id", scraped.ID, "name", scraped.Name)
			continue
		}

		imported++

		// Update progress
		if (i+1)%5 == 0 || i == len(products)-1 {
			err = h.storage.Queries.UpdateImportJobProgress(ctx, db.UpdateImportJobProgressParams{
				ProcessedItems: sql.NullInt64{Int64: int64(i + 1), Valid: true},
				TotalItems:     sql.NullInt64{Int64: int64(len(products)), Valid: true},
				ID:             jobID,
			})
			if err != nil {
				slog.Error("failed to update job progress", "error", err, "job_id", jobID)
			}
		}
	}

	// Mark job complete
	err = h.storage.Queries.CompleteImportJob(ctx, jobID)
	if err != nil {
		slog.Error("failed to complete import job", "error", err, "job_id", jobID)
	}

	slog.Info("import job completed", "job_id", jobID, "total", len(products), "imported", imported)
}

// importProduct imports a single scraped product
func (h *AdminImporterHandler) importProduct(ctx context.Context, scraped db.ScrapedProduct, categoryID, designerName string, downloader *importer.ImageDownloader) error {
	productID := uuid.New().String()
	slug := generateProductSlug(scraped.Name)

	// Determine price (use original or default to $15)
	priceCents := int64(1500)
	if scraped.OriginalPriceCents.Valid && scraped.OriginalPriceCents.Int64 > 0 {
		// Apply markup (2x the original price as a starting point)
		priceCents = scraped.OriginalPriceCents.Int64 * 2
		// Minimum $10
		if priceCents < 1000 {
			priceCents = 1000
		}
	}

	// Create product
	params := db.CreateProductParams{
		ID:               productID,
		Name:             scraped.Name,
		Slug:             slug,
		Description:      scraped.Description,
		ShortDescription: sql.NullString{Valid: false},
		PriceCents:       priceCents,
		CategoryID:       sql.NullString{String: categoryID, Valid: categoryID != ""},
		Sku:              sql.NullString{Valid: false},
		StockQuantity:    sql.NullInt64{Int64: 100, Valid: true},
		HasVariants:      sql.NullBool{Bool: false, Valid: true},
		WeightGrams:      sql.NullInt64{Valid: false},
		LeadTimeDays:     sql.NullInt64{Int64: 3, Valid: true},
		IsActive:         sql.NullBool{Bool: true, Valid: true},
		IsFeatured:       sql.NullBool{Bool: false, Valid: true},
		IsPremium:        sql.NullBool{Bool: false, Valid: true},
		Disclaimer:       sql.NullString{Valid: false},
		SeoTitle:         sql.NullString{String: scraped.Name, Valid: true},
		SeoDescription:   scraped.Description,
		SeoKeywords:      sql.NullString{Valid: false},
		OgImageUrl:       sql.NullString{Valid: false},
	}

	_, err := h.storage.Queries.CreateProduct(ctx, params)
	if err != nil {
		return err
	}

	// Update product with source info
	err = h.storage.Queries.UpdateProductSource(ctx, db.UpdateProductSourceParams{
		SourceUrl:      sql.NullString{String: scraped.SourceUrl, Valid: true},
		SourcePlatform: sql.NullString{String: scraped.Platform, Valid: true},
		DesignerName:   sql.NullString{String: designerName, Valid: true},
		ID:             productID,
	})
	if err != nil {
		slog.Error("failed to update product source", "error", err, "product_id", productID)
	}

	// Download and save images
	if scraped.ImageUrls.Valid && scraped.ImageUrls.String != "" {
		var imageURLs []string
		if err := json.Unmarshal([]byte(scraped.ImageUrls.String), &imageURLs); err == nil && len(imageURLs) > 0 {
			// Limit to first 5 images
			if len(imageURLs) > 5 {
				imageURLs = imageURLs[:5]
			}

			downloaded, err := downloader.DownloadImages(ctx, imageURLs, productID)
			if err != nil {
				slog.Error("failed to download images", "error", err, "product_id", productID)
			}

			// Create image records
			for i, img := range downloaded {
				isPrimary := i == 0
				_, err := h.storage.Queries.CreateProductImage(ctx, db.CreateProductImageParams{
					ID:           uuid.New().String(),
					ProductID:    productID,
					ImageUrl:     img.Filename,
					AltText:      sql.NullString{String: scraped.Name, Valid: true},
					DisplayOrder: sql.NullInt64{Int64: int64(i), Valid: true},
					IsPrimary:    sql.NullBool{Bool: isPrimary, Valid: true},
				})
				if err != nil {
					slog.Error("failed to create product image", "error", err, "product_id", productID, "filename", img.Filename)
				}
			}
		}
	}

	// Mark scraped product as imported
	err = h.storage.Queries.MarkProductImported(ctx, db.MarkProductImportedParams{
		ImportedProductID: sql.NullString{String: productID, Valid: true},
		ID:                scraped.ID,
	})
	if err != nil {
		slog.Error("failed to mark product imported", "error", err, "scraped_id", scraped.ID)
	}

	slog.Info("imported product", "product_id", productID, "name", scraped.Name)
	return nil
}

// generateProductSlug creates a URL-friendly slug from a name
func generateProductSlug(name string) string {
	// Convert to lowercase
	slug := strings.ToLower(name)

	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")

	// Remove special characters
	reg := regexp.MustCompile(`[^a-z0-9-]`)
	slug = reg.ReplaceAllString(slug, "")

	// Remove multiple consecutive hyphens
	reg = regexp.MustCompile(`-+`)
	slug = reg.ReplaceAllString(slug, "-")

	// Trim hyphens from ends
	slug = strings.Trim(slug, "-")

	// Add unique suffix to prevent duplicates
	slug = slug + "-" + uuid.New().String()[:8]

	return slug
}
