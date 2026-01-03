package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"

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

	return admin.ImporterDesignerDetail(c, *designer, products, stats).Render(ctx, c.Response().Writer)
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
	// TODO: Implement import logic - create products from scraped data
	// This will:
	// 1. Get unimported products for the designer
	// 2. Create product records in the products table
	// 3. Download and save images
	// 4. Mark products as imported

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Import functionality coming soon",
	})
}
