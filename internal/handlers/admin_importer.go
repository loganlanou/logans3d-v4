package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/internal/importer"
	"github.com/loganlanou/logans3d-v4/internal/ogimage"
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

	// Get filter from query params (default to unimported)
	filter := c.QueryParam("filter")
	if filter == "" {
		filter = "unimported"
	}

	// Get stats
	stats := admin.DesignerStats{
		Designer: *designer,
	}

	totalCount, err := h.storage.Queries.CountScrapedProductsByDesigner(ctx, slug)
	if err == nil {
		stats.ScrapedCount = totalCount
	}

	unimported, err := h.storage.Queries.CountUnimportedNonSkippedByDesigner(ctx, slug)
	if err == nil {
		stats.UnimportedCount = unimported
	}

	// Get last scraped time from most recent job
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

	// Get filtered count
	filterCount, err := h.storage.Queries.CountScrapedProductsByDesignerFiltered(ctx, db.CountScrapedProductsByDesignerFilteredParams{
		DesignerSlug: slug,
		Column2:      filter,
	})
	if err != nil {
		slog.Error("failed to count filtered products", "error", err, "designer", slug, "filter", filter)
		filterCount = 0
	}

	// Get scraped products with filter
	products, err := h.storage.Queries.ListScrapedProductsByDesignerFiltered(ctx, db.ListScrapedProductsByDesignerFilteredParams{
		DesignerSlug: slug,
		Column2:      filter,
		Limit:        100,
		Offset:       0,
	})
	if err != nil {
		slog.Error("failed to list scraped products", "error", err, "designer", slug, "filter", filter)
		products = []db.ScrapedProduct{}
	}

	// Get categories for import selector
	categories, err := h.storage.Queries.ListCategories(ctx)
	if err != nil {
		slog.Error("failed to list categories", "error", err)
		categories = []db.Category{}
	}

	data := admin.ImporterDesignerData{
		Designer:    *designer,
		Products:    products,
		Stats:       stats,
		Categories:  categories,
		Filter:      filter,
		TotalCount:  totalCount,
		FilterCount: filterCount,
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
		if failErr := h.storage.Queries.FailImportJob(ctx, db.FailImportJobParams{
			ErrorMessage: sql.NullString{String: err.Error(), Valid: true},
			ID:           jobID,
		}); failErr != nil {
			slog.Error("failed to mark job as failed", "error", failErr, "job_id", jobID)
		}
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
		if failErr := h.storage.Queries.FailImportJob(ctx, db.FailImportJobParams{
			ErrorMessage: sql.NullString{String: err.Error(), Valid: true},
			ID:           jobID,
		}); failErr != nil {
			slog.Error("failed to mark job as failed", "error", failErr, "job_id", jobID)
		}
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
		// Create default import settings for batch import
		// (no variants - batch import creates simple products)
		settings := &ImportSettings{
			CategoryID:      categoryID,
			BasePriceCents:  1500, // Default $15
			Sizes:           []string{},
			SizeAdjustments: make(map[string]int64),
			IsNew:           true,
			IsPremium:       false,
			IsFeatured:      false,
		}
		// Use AI price if available, otherwise markup original price
		if scraped.AiPriceCents.Valid && scraped.AiPriceCents.Int64 > 0 {
			settings.BasePriceCents = scraped.AiPriceCents.Int64
		} else if scraped.OriginalPriceCents.Valid && scraped.OriginalPriceCents.Int64 > 0 {
			settings.BasePriceCents = scraped.OriginalPriceCents.Int64 * 2
			if settings.BasePriceCents < 1000 {
				settings.BasePriceCents = 1000
			}
		}

		err := h.importProduct(ctx, scraped, settings, designer.Name, downloader)
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
func (h *AdminImporterHandler) importProduct(ctx context.Context, scraped db.ScrapedProduct, settings *ImportSettings, designerName string, downloader *importer.ImageDownloader) error {
	productID := uuid.New().String()
	slug := generateProductSlug(scraped.Name)

	// Determine if product has variants (sizes selected)
	hasVariants := len(settings.Sizes) > 0

	// Create product
	params := db.CreateProductParams{
		ID:               productID,
		Name:             scraped.Name,
		Slug:             slug,
		Description:      scraped.Description,
		ShortDescription: sql.NullString{Valid: false},
		PriceCents:       settings.BasePriceCents,
		CategoryID:       sql.NullString{String: settings.CategoryID, Valid: settings.CategoryID != ""},
		Sku:              sql.NullString{Valid: false},
		StockQuantity:    sql.NullInt64{Int64: 100, Valid: true},
		HasVariants:      sql.NullBool{Bool: hasVariants, Valid: true},
		WeightGrams:      sql.NullInt64{Valid: false},
		LeadTimeDays:     sql.NullInt64{Int64: 3, Valid: true},
		IsActive:         sql.NullBool{Bool: true, Valid: true},
		IsFeatured:       sql.NullBool{Bool: settings.IsFeatured, Valid: true},
		IsPremium:        sql.NullBool{Bool: settings.IsPremium, Valid: true},
		Disclaimer:       sql.NullString{Valid: false},
		SeoTitle:         sql.NullString{String: scraped.Name, Valid: true},
		SeoDescription:   scraped.Description,
		SeoKeywords:      sql.NullString{Valid: false},
		OgImageUrl:       sql.NullString{Valid: false},
	}

	_, err := h.storage.Queries.CreateProduct(ctx, params)
	if err != nil {
		slog.Error("failed to create product", "error", err, "name", scraped.Name)
		return fmt.Errorf("create product: %w", err)
	}

	// Update product with source info and is_new flag
	err = h.storage.Queries.UpdateProductSource(ctx, db.UpdateProductSourceParams{
		SourceUrl:      sql.NullString{String: scraped.SourceUrl, Valid: true},
		SourcePlatform: sql.NullString{String: scraped.Platform, Valid: true},
		DesignerName:   sql.NullString{String: designerName, Valid: true},
		ID:             productID,
	})
	if err != nil {
		slog.Error("failed to update product source", "error", err, "product_id", productID)
	}

	// Set is_new flag if enabled
	if settings.IsNew {
		err = h.storage.Queries.UpdateProductIsNew(ctx, db.UpdateProductIsNewParams{
			IsNew: sql.NullBool{Bool: true, Valid: true},
			ID:    productID,
		})
		if err != nil {
			slog.Error("failed to update product is_new", "error", err, "product_id", productID)
		}
	}

	// Get images to import - prefer selected scraped images, fall back to original URLs
	var imagesToImport []string
	scrapedImages, err := h.storage.Queries.ListScrapedProductImages(ctx, scraped.ID)
	if err == nil && len(scrapedImages) > 0 {
		// Use downloaded scraped images that are selected for import
		for _, img := range scrapedImages {
			// For now, use all downloaded images (selection feature can be added later)
			if img.LocalFilename.Valid && img.LocalFilename.String != "" && img.DownloadStatus.Valid && img.DownloadStatus.String == "downloaded" {
				imagesToImport = append(imagesToImport, fmt.Sprintf("scraped/%s", img.LocalFilename.String))
			}
		}
	}

	// If no scraped images, try AI images
	if len(imagesToImport) == 0 {
		aiImages, err := h.storage.Queries.ListScrapedProductAIImages(ctx, scraped.ID)
		if err == nil {
			for _, img := range aiImages {
				if img.LocalFilename != "" {
					imagesToImport = append(imagesToImport, fmt.Sprintf("scraped/ai/%s", img.LocalFilename))
				}
			}
		}
	}

	// If still no images, download from original URLs
	if len(imagesToImport) == 0 && scraped.ImageUrls.Valid && scraped.ImageUrls.String != "" {
		var imageURLs []string
		if err := json.Unmarshal([]byte(scraped.ImageUrls.String), &imageURLs); err == nil && len(imageURLs) > 0 {
			if len(imageURLs) > 5 {
				imageURLs = imageURLs[:5]
			}

			downloaded, err := downloader.DownloadImages(ctx, imageURLs, productID)
			if err != nil {
				slog.Error("failed to download images", "error", err, "product_id", productID)
			}
			for _, img := range downloaded {
				imagesToImport = append(imagesToImport, img.Filename)
			}
		}
	}

	// Create image records
	for i, imgPath := range imagesToImport {
		isPrimary := i == 0
		_, err := h.storage.Queries.CreateProductImage(ctx, db.CreateProductImageParams{
			ID:           uuid.New().String(),
			ProductID:    productID,
			ImageUrl:     imgPath,
			AltText:      sql.NullString{String: scraped.Name, Valid: true},
			DisplayOrder: sql.NullInt64{Int64: int64(i), Valid: true},
			IsPrimary:    sql.NullBool{Bool: isPrimary, Valid: true},
		})
		if err != nil {
			slog.Error("failed to create product image", "error", err, "product_id", productID, "path", imgPath)
		}
	}

	// If product has variants (sizes), create style, size configs, and SKUs
	if hasVariants {
		// Create default style
		styleID := uuid.New().String()
		_, err = h.storage.Queries.CreateProductStyle(ctx, db.CreateProductStyleParams{
			ID:           styleID,
			ProductID:    productID,
			Name:         "Default",
			IsPrimary:    sql.NullBool{Bool: true, Valid: true},
			DisplayOrder: sql.NullInt64{Int64: 0, Valid: true},
		})
		if err != nil {
			slog.Error("failed to create product style", "error", err, "product_id", productID)
		} else {
			// Create style images from product images
			for i, imgPath := range imagesToImport {
				isPrimary := i == 0
				_, err = h.storage.Queries.CreateProductStyleImage(ctx, db.CreateProductStyleImageParams{
					ID:             uuid.New().String(),
					ProductStyleID: styleID,
					ImageUrl:       imgPath,
					IsPrimary:      sql.NullBool{Bool: isPrimary, Valid: true},
					DisplayOrder:   sql.NullInt64{Int64: int64(i), Valid: true},
				})
				if err != nil {
					slog.Error("failed to create product style image", "error", err, "style_id", styleID, "path", imgPath)
				}
			}

			// Create size configs and SKUs for each selected size
			for i, sizeID := range settings.Sizes {
				// Get price adjustment for this size (from form or default to 0)
				priceAdjustment := int64(0)
				if adj, ok := settings.SizeAdjustments[sizeID]; ok {
					priceAdjustment = adj
				}

				// Create size config
				_, err = h.storage.Queries.UpsertProductSizeConfig(ctx, db.UpsertProductSizeConfigParams{
					ID:                   uuid.New().String(),
					ProductID:            productID,
					SizeID:               sizeID,
					PriceAdjustmentCents: sql.NullInt64{Int64: priceAdjustment, Valid: true},
					IsEnabled:            sql.NullBool{Bool: true, Valid: true},
					DisplayOrder:         sql.NullInt64{Int64: int64(i), Valid: true},
				})
				if err != nil {
					slog.Error("failed to create product size config", "error", err, "product_id", productID, "size_id", sizeID)
					continue
				}

				// Create SKU for this style + size combination
				skuCode := fmt.Sprintf("%s-%s", slug, sizeID)
				_, err = h.storage.Queries.CreateProductSku(ctx, db.CreateProductSkuParams{
					ID:                   uuid.New().String(),
					ProductID:            productID,
					ProductStyleID:       styleID,
					SizeID:               sizeID,
					Sku:                  skuCode,
					PriceAdjustmentCents: sql.NullInt64{Int64: priceAdjustment, Valid: true},
					StockQuantity:        sql.NullInt64{Int64: 100, Valid: true},
					IsActive:             sql.NullBool{Bool: true, Valid: true},
				})
				if err != nil {
					slog.Error("failed to create product SKU", "error", err, "product_id", productID, "style_id", styleID, "size_id", sizeID)
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

	slog.Info("imported product",
		"product_id", productID,
		"name", scraped.Name,
		"has_variants", hasVariants,
		"sizes", len(settings.Sizes),
		"images", len(imagesToImport),
	)
	return nil
}

// HandleScrapedProductDetail shows detail page for a scraped product
func (h *AdminImporterHandler) HandleScrapedProductDetail(c echo.Context) error {
	ctx := c.Request().Context()
	productID := c.Param("id")

	// Get the scraped product
	product, err := h.storage.Queries.GetScrapedProduct(ctx, productID)
	if err != nil {
		slog.Error("failed to get scraped product", "error", err, "product_id", productID)
		return echo.NewHTTPError(http.StatusNotFound, "Product not found")
	}

	// Get the designer
	designer := importer.GetDesigner(product.DesignerSlug)
	if designer == nil {
		designer = &importer.Designer{
			Name: product.DesignerSlug,
			Slug: product.DesignerSlug,
		}
	}

	// Get scraped product images
	images, err := h.storage.Queries.ListScrapedProductImages(ctx, productID)
	if err != nil {
		slog.Error("failed to list scraped product images", "error", err, "product_id", productID)
		images = []db.ScrapedProductImage{}
	}

	// Get AI generated images
	aiImages, err := h.storage.Queries.ListScrapedProductAIImages(ctx, productID)
	if err != nil {
		slog.Error("failed to list scraped product AI images", "error", err, "product_id", productID)
		aiImages = []db.ScrapedProductAiImage{}
	}

	// Parse image URLs from JSON and deduplicate
	var imageURLs []string
	if product.ImageUrls.Valid && product.ImageUrls.String != "" && len(images) == 0 {
		var allURLs []string
		if err := json.Unmarshal([]byte(product.ImageUrls.String), &allURLs); err != nil {
			slog.Debug("failed to parse image URLs", "error", err, "product_id", productID)
		} else {
			// Deduplicate - same underlying image may appear in different formats (webp vs jpg)
			imageURLs = deduplicateImageURLs(allURLs)
		}
	}

	// Get categories for import selector
	categories, err := h.storage.Queries.ListCategories(ctx)
	if err != nil {
		slog.Error("failed to list categories", "error", err)
		categories = []db.Category{}
	}

	// Get size charts for size/price selection
	sizeCharts, err := h.storage.Queries.GetSizeCharts(ctx)
	if err != nil {
		slog.Error("failed to get size charts", "error", err)
		sizeCharts = []db.GetSizeChartsRow{}
	}

	// Get previous and next product IDs for navigation
	var prevProductID, nextProductID string
	prevID, err := h.storage.Queries.GetPreviousScrapedProduct(ctx, db.GetPreviousScrapedProductParams{
		DesignerSlug: product.DesignerSlug,
		ID:           productID,
	})
	if err == nil {
		prevProductID = prevID
	}

	nextID, err := h.storage.Queries.GetNextScrapedProduct(ctx, db.GetNextScrapedProductParams{
		DesignerSlug: product.DesignerSlug,
		ID:           productID,
	})
	if err == nil {
		nextProductID = nextID
	}

	data := admin.ScrapedProductDetailData{
		Product:       product,
		Designer:      *designer,
		Images:        images,
		AIImages:      aiImages,
		Categories:    categories,
		ImageURLs:     imageURLs,
		SizeCharts:    sizeCharts,
		PrevProductID: prevProductID,
		NextProductID: nextProductID,
	}

	return admin.ScrapedProductDetail(c, data).Render(ctx, c.Response().Writer)
}

// HandleSkipProduct marks a product as skipped
func (h *AdminImporterHandler) HandleSkipProduct(c echo.Context) error {
	ctx := c.Request().Context()
	productID := c.Param("id")
	reason := c.FormValue("reason")

	err := h.storage.Queries.SkipScrapedProduct(ctx, db.SkipScrapedProductParams{
		SkipReason: sql.NullString{String: reason, Valid: reason != ""},
		ID:         productID,
	})
	if err != nil {
		slog.Error("failed to skip product", "error", err, "product_id", productID)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to skip product")
	}

	c.Response().Header().Set("HX-Trigger", "productUpdated")
	return c.NoContent(http.StatusOK)
}

// HandleUnskipProduct removes skip status from a product
func (h *AdminImporterHandler) HandleUnskipProduct(c echo.Context) error {
	ctx := c.Request().Context()
	productID := c.Param("id")

	err := h.storage.Queries.UnskipScrapedProduct(ctx, productID)
	if err != nil {
		slog.Error("failed to unskip product", "error", err, "product_id", productID)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to unskip product")
	}

	c.Response().Header().Set("HX-Trigger", "productUpdated")
	return c.NoContent(http.StatusOK)
}

// HandleRescrapeProduct re-scrapes a single product
func (h *AdminImporterHandler) HandleRescrapeProduct(c echo.Context) error {
	ctx := c.Request().Context()
	productID := c.Param("id")

	// Get existing product
	product, err := h.storage.Queries.GetScrapedProduct(ctx, productID)
	if err != nil {
		slog.Error("failed to get scraped product", "error", err, "product_id", productID)
		return echo.NewHTTPError(http.StatusNotFound, "Product not found")
	}

	// Re-fetch the product from source
	freshProduct, err := h.scraper.FetchProduct(ctx, product.SourceUrl)
	if err != nil {
		slog.Error("failed to re-fetch product", "error", err, "url", product.SourceUrl)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to re-scrape product")
	}

	// Update the scraped product
	imageURLsJSON, _ := json.Marshal(freshProduct.ImageURLs)
	tagsJSON, _ := json.Marshal(freshProduct.Tags)

	var releaseDate sql.NullTime
	if freshProduct.ReleaseDate != nil {
		releaseDate = sql.NullTime{Time: *freshProduct.ReleaseDate, Valid: true}
	}

	_, err = h.storage.Queries.UpsertScrapedProduct(ctx, db.UpsertScrapedProductParams{
		ID:                 productID,
		DesignerSlug:       product.DesignerSlug,
		Platform:           product.Platform,
		SourceUrl:          product.SourceUrl,
		Name:               freshProduct.Name,
		Description:        sql.NullString{String: freshProduct.Description, Valid: freshProduct.Description != ""},
		OriginalPriceCents: sql.NullInt64{Int64: int64(freshProduct.OriginalPriceCents), Valid: freshProduct.OriginalPriceCents > 0},
		ReleaseDate:        releaseDate,
		ImageUrls:          sql.NullString{String: string(imageURLsJSON), Valid: len(freshProduct.ImageURLs) > 0},
		Tags:               sql.NullString{String: string(tagsJSON), Valid: len(freshProduct.Tags) > 0},
		RawHtml:            sql.NullString{String: freshProduct.RawHTML, Valid: freshProduct.RawHTML != ""},
	})
	if err != nil {
		slog.Error("failed to update scraped product", "error", err, "product_id", productID)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update product")
	}

	slog.Info("re-scraped product", "product_id", productID, "name", freshProduct.Name)

	c.Response().Header().Set("HX-Trigger", "productUpdated")
	return c.NoContent(http.StatusOK)
}

// ImportSettings holds configuration for importing a scraped product
type ImportSettings struct {
	CategoryID      string
	BasePriceCents  int64
	Sizes           []string         // Size IDs to enable
	SizeAdjustments map[string]int64 // SizeID -> adjustment in cents
	IsNew           bool
	IsPremium       bool
	IsFeatured      bool
}

// HandleImportSingleProduct imports a single scraped product
func (h *AdminImporterHandler) HandleImportSingleProduct(c echo.Context) error {
	ctx := c.Request().Context()
	productID := c.Param("id")

	// Parse import settings from form
	settings, err := parseImportSettings(c)
	if err != nil {
		slog.Error("failed to parse import settings", "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid import settings")
	}

	// Get the scraped product
	scraped, err := h.storage.Queries.GetScrapedProduct(ctx, productID)
	if err != nil {
		slog.Error("failed to get scraped product", "error", err, "product_id", productID)
		return echo.NewHTTPError(http.StatusNotFound, "Product not found")
	}

	if scraped.ImportedProductID.Valid {
		return echo.NewHTTPError(http.StatusBadRequest, "Product already imported")
	}

	// Get designer name
	designer := importer.GetDesigner(scraped.DesignerSlug)
	designerName := scraped.DesignerSlug
	if designer != nil {
		designerName = designer.Name
	}

	// Import the product
	downloader := importer.NewImageDownloader("public/images/products")
	err = h.importProduct(ctx, scraped, settings, designerName, downloader)
	if err != nil {
		slog.Error("failed to import product", "error", err, "product_id", productID)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to import product")
	}

	c.Response().Header().Set("HX-Trigger", "productImported")
	return c.NoContent(http.StatusOK)
}

// parseImportSettings parses import settings from form values
func parseImportSettings(c echo.Context) (*ImportSettings, error) {
	settings := &ImportSettings{
		CategoryID:      c.FormValue("category_id"),
		BasePriceCents:  999, // Default $9.99
		Sizes:           []string{},
		SizeAdjustments: make(map[string]int64),
		IsNew:           c.FormValue("is_new") == "true",
		IsPremium:       c.FormValue("is_premium") == "true",
		IsFeatured:      c.FormValue("is_featured") == "true",
	}

	// Parse base price
	if basePriceStr := c.FormValue("base_price_cents"); basePriceStr != "" {
		var basePriceCents int64
		if _, err := fmt.Sscanf(basePriceStr, "%d", &basePriceCents); err == nil && basePriceCents > 0 {
			settings.BasePriceCents = basePriceCents
		}
	}

	// Parse sizes JSON array
	if sizesJSON := c.FormValue("sizes"); sizesJSON != "" {
		if err := json.Unmarshal([]byte(sizesJSON), &settings.Sizes); err != nil {
			slog.Debug("failed to parse sizes JSON", "error", err, "json", sizesJSON)
		}
	}

	// Parse size adjustments JSON object
	if adjustmentsJSON := c.FormValue("size_adjustments"); adjustmentsJSON != "" {
		if err := json.Unmarshal([]byte(adjustmentsJSON), &settings.SizeAdjustments); err != nil {
			slog.Debug("failed to parse size adjustments JSON", "error", err, "json", adjustmentsJSON)
		}
	}

	slog.Debug("parsed import settings",
		"category_id", settings.CategoryID,
		"base_price_cents", settings.BasePriceCents,
		"sizes", settings.Sizes,
		"size_adjustments", settings.SizeAdjustments,
		"is_new", settings.IsNew,
		"is_premium", settings.IsPremium,
		"is_featured", settings.IsFeatured,
	)

	return settings, nil
}

// HandleDownloadImages downloads images for a scraped product
func (h *AdminImporterHandler) HandleDownloadImages(c echo.Context) error {
	ctx := c.Request().Context()
	productID := c.Param("id")

	// Get the scraped product
	product, err := h.storage.Queries.GetScrapedProduct(ctx, productID)
	if err != nil {
		slog.Error("failed to get scraped product", "error", err, "product_id", productID)
		return echo.NewHTTPError(http.StatusNotFound, "Product not found")
	}

	// Parse image URLs from JSON
	var imageURLs []string
	if product.ImageUrls.Valid && product.ImageUrls.String != "" {
		if err := json.Unmarshal([]byte(product.ImageUrls.String), &imageURLs); err != nil {
			slog.Error("failed to parse image URLs", "error", err, "product_id", productID)
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid image URLs")
		}
	}

	if len(imageURLs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "No images to download")
	}

	// Create output directory for scraped images
	outputDir := "public/images/scraped"
	if err := createDirIfNotExists(outputDir); err != nil {
		slog.Error("failed to create scraped images directory", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create directory")
	}

	// Download in background
	go h.downloadProductImages(productID, imageURLs, outputDir)

	c.Response().Header().Set("HX-Trigger", "downloadStarted")
	return c.NoContent(http.StatusAccepted)
}

// downloadProductImages downloads images and updates the database
func (h *AdminImporterHandler) downloadProductImages(productID string, imageURLs []string, outputDir string) {
	ctx := context.Background()

	// First, create or get records for each image URL (upsert to avoid duplicates)
	for i, url := range imageURLs {
		imageID := uuid.New().String()
		// Use upsert query - if URL already exists for this product, returns existing record
		_, err := h.storage.Queries.CreateOrGetScrapedProductImage(ctx, db.CreateOrGetScrapedProductImageParams{
			ID:               imageID,
			ScrapedProductID: productID,
			SourceUrl:        url,
			DownloadStatus:   sql.NullString{String: "pending", Valid: true},
			DisplayOrder:     sql.NullInt64{Int64: int64(i), Valid: true},
		})
		if err != nil {
			slog.Error("failed to create/get scraped product image record", "error", err, "product_id", productID, "url", url)
			continue
		}
	}

	// Now download each image
	downloader := importer.NewImageDownloader(outputDir)
	images, _ := h.storage.Queries.ListScrapedProductImages(ctx, productID)

	for _, img := range images {
		if img.DownloadStatus.Valid && img.DownloadStatus.String == "downloaded" {
			continue // Already downloaded
		}

		downloaded, err := downloader.DownloadImages(ctx, []string{img.SourceUrl}, productID)
		if err != nil || len(downloaded) == 0 {
			errMsg := "unknown error"
			if err != nil {
				errMsg = err.Error()
			}
			h.storage.Queries.UpdateScrapedProductImageStatus(ctx, db.UpdateScrapedProductImageStatusParams{
				DownloadStatus: sql.NullString{String: "failed", Valid: true},
				DownloadError:  sql.NullString{String: errMsg, Valid: true},
				LocalFilename:  sql.NullString{Valid: false},
				ID:             img.ID,
			})
			continue
		}

		// Update with success
		h.storage.Queries.UpdateScrapedProductImageStatus(ctx, db.UpdateScrapedProductImageStatusParams{
			DownloadStatus: sql.NullString{String: "downloaded", Valid: true},
			DownloadError:  sql.NullString{Valid: false},
			LocalFilename:  sql.NullString{String: downloaded[0].Filename, Valid: true},
			ID:             img.ID,
		})
	}

	slog.Info("finished downloading images", "product_id", productID, "total", len(images))
}

// HandleRetryImageDownload retries downloading a failed image
func (h *AdminImporterHandler) HandleRetryImageDownload(c echo.Context) error {
	ctx := c.Request().Context()
	imageID := c.Param("id")

	// Get the image record
	img, err := h.storage.Queries.GetScrapedProductImage(ctx, imageID)
	if err != nil {
		slog.Error("failed to get scraped product image", "error", err, "image_id", imageID)
		return echo.NewHTTPError(http.StatusNotFound, "Image not found")
	}

	// Mark as pending
	err = h.storage.Queries.UpdateScrapedProductImageStatus(ctx, db.UpdateScrapedProductImageStatusParams{
		DownloadStatus: sql.NullString{String: "pending", Valid: true},
		DownloadError:  sql.NullString{Valid: false},
		LocalFilename:  sql.NullString{Valid: false},
		ID:             imageID,
	})
	if err != nil {
		slog.Error("failed to update image status", "error", err, "image_id", imageID)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update status")
	}

	// Download in background
	go func() {
		ctx := context.Background()
		outputDir := "public/images/scraped"
		downloader := importer.NewImageDownloader(outputDir)

		downloaded, err := downloader.DownloadImages(ctx, []string{img.SourceUrl}, img.ScrapedProductID)
		if err != nil || len(downloaded) == 0 {
			errMsg := "unknown error"
			if err != nil {
				errMsg = err.Error()
			}
			h.storage.Queries.UpdateScrapedProductImageStatus(ctx, db.UpdateScrapedProductImageStatusParams{
				DownloadStatus: sql.NullString{String: "failed", Valid: true},
				DownloadError:  sql.NullString{String: errMsg, Valid: true},
				LocalFilename:  sql.NullString{Valid: false},
				ID:             imageID,
			})
			return
		}

		h.storage.Queries.UpdateScrapedProductImageStatus(ctx, db.UpdateScrapedProductImageStatusParams{
			DownloadStatus: sql.NullString{String: "downloaded", Valid: true},
			DownloadError:  sql.NullString{Valid: false},
			LocalFilename:  sql.NullString{String: downloaded[0].Filename, Valid: true},
			ID:             imageID,
		})
	}()

	c.Response().Header().Set("HX-Trigger", "imageRetrying")
	return c.NoContent(http.StatusAccepted)
}

// HandleToggleImageSelection toggles image selection for import
func (h *AdminImporterHandler) HandleToggleImageSelection(c echo.Context) error {
	ctx := c.Request().Context()
	imageID := c.Param("id")

	// Get current state
	img, err := h.storage.Queries.GetScrapedProductImage(ctx, imageID)
	if err != nil {
		slog.Error("failed to get scraped product image", "error", err, "image_id", imageID)
		return echo.NewHTTPError(http.StatusNotFound, "Image not found")
	}

	// Toggle selection
	newSelection := true
	if img.IsSelectedForImport.Valid {
		newSelection = !img.IsSelectedForImport.Bool
	}

	err = h.storage.Queries.UpdateScrapedProductImageSelection(ctx, db.UpdateScrapedProductImageSelectionParams{
		IsSelectedForImport: sql.NullBool{Bool: newSelection, Valid: true},
		ID:                  imageID,
	})
	if err != nil {
		slog.Error("failed to update image selection", "error", err, "image_id", imageID)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update selection")
	}

	return c.NoContent(http.StatusOK)
}

// HandleToggleAIImageSelection toggles AI image selection for import
func (h *AdminImporterHandler) HandleToggleAIImageSelection(c echo.Context) error {
	ctx := c.Request().Context()
	imageID := c.Param("id")

	// Get current state
	img, err := h.storage.Queries.GetScrapedProductAIImage(ctx, imageID)
	if err != nil {
		slog.Error("failed to get scraped product AI image", "error", err, "image_id", imageID)
		return echo.NewHTTPError(http.StatusNotFound, "Image not found")
	}

	// Toggle selection
	newSelection := true
	if img.IsSelectedForImport.Valid {
		newSelection = !img.IsSelectedForImport.Bool
	}

	err = h.storage.Queries.UpdateScrapedProductAIImageSelection(ctx, db.UpdateScrapedProductAIImageSelectionParams{
		IsSelectedForImport: sql.NullBool{Bool: newSelection, Valid: true},
		ID:                  imageID,
	})
	if err != nil {
		slog.Error("failed to update AI image selection", "error", err, "image_id", imageID)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update selection")
	}

	return c.NoContent(http.StatusOK)
}

// HandleGenerateAIImage generates an AI image for a scraped product
func (h *AdminImporterHandler) HandleGenerateAIImage(c echo.Context) error {
	ctx := c.Request().Context()
	productID := c.Param("id")

	// Get the scraped product
	product, err := h.storage.Queries.GetScrapedProduct(ctx, productID)
	if err != nil {
		slog.Error("failed to get scraped product", "error", err, "product_id", productID)
		return echo.NewHTTPError(http.StatusNotFound, "Product not found")
	}

	// Check for Gemini API key
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		slog.Error("GEMINI_API_KEY not set")
		return echo.NewHTTPError(http.StatusBadRequest, "AI generation not configured")
	}

	// Need at least one source image to generate from
	var sourceImagePath string
	var sourceImageID string

	// First check if we have downloaded images
	images, err := h.storage.Queries.ListScrapedProductImages(ctx, productID)
	if err == nil && len(images) > 0 {
		for _, img := range images {
			if img.DownloadStatus.Valid && img.DownloadStatus.String == "downloaded" && img.LocalFilename.Valid {
				sourceImagePath = "public/images/scraped/" + img.LocalFilename.String
				sourceImageID = img.ID
				break
			}
		}
	}

	// If no downloaded images, try to use the first URL directly
	if sourceImagePath == "" {
		var imageURLs []string
		if product.ImageUrls.Valid && product.ImageUrls.String != "" {
			if err := json.Unmarshal([]byte(product.ImageUrls.String), &imageURLs); err == nil && len(imageURLs) > 0 {
				// Download first image temporarily
				outputDir := "public/images/scraped/temp"
				if err := createDirIfNotExists(outputDir); err != nil {
					slog.Error("failed to create temp directory", "error", err)
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create temp directory")
				}
				downloader := importer.NewImageDownloader(outputDir)
				downloaded, err := downloader.DownloadImages(ctx, []string{imageURLs[0]}, productID)
				if err != nil || len(downloaded) == 0 {
					slog.Error("failed to download source image for AI generation", "error", err)
					return echo.NewHTTPError(http.StatusBadRequest, "No source image available for AI generation")
				}
				sourceImagePath = downloaded[0].FilePath
			}
		}
	}

	if sourceImagePath == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "No source image available for AI generation")
	}

	// Generate AI image in background
	go h.generateAIImage(productID, product.Name, sourceImagePath, sourceImageID, apiKey)

	c.Response().Header().Set("HX-Trigger", "aiGenerationStarted")
	return c.NoContent(http.StatusAccepted)
}

// generateAIImage generates an AI image and stores it
func (h *AdminImporterHandler) generateAIImage(productID, productName, sourceImagePath, sourceImageID, apiKey string) {
	ctx := context.Background()

	// Create output directory
	outputDir := "public/images/scraped/ai"
	if err := createDirIfNotExists(outputDir); err != nil {
		slog.Error("failed to create AI output directory", "error", err)
		return
	}

	// Generate unique filename
	outputFilename := fmt.Sprintf("%s_%s_ai.jpg", productID, uuid.New().String()[:8])
	outputPath := outputDir + "/" + outputFilename

	// Create AI generator
	generator := ogimage.NewAIGenerator(apiKey)
	info := ogimage.SingleProductBackgroundInfo{
		Name:      productName,
		ImagePath: sourceImagePath,
	}

	// Generate the image
	modelUsed, err := generator.GenerateSingleProductBackground(info, outputPath)
	if err != nil {
		slog.Error("failed to generate AI image", "error", err, "product_id", productID)
		return
	}

	// Store in database
	aiImageID := uuid.New().String()
	_, err = h.storage.Queries.CreateScrapedProductAIImage(ctx, db.CreateScrapedProductAIImageParams{
		ID:               aiImageID,
		ScrapedProductID: productID,
		SourceImageID:    sql.NullString{String: sourceImageID, Valid: sourceImageID != ""},
		LocalFilename:    outputFilename,
		PromptUsed:       sql.NullString{String: "Single product background generation", Valid: true},
		ModelUsed:        sql.NullString{String: modelUsed, Valid: true},
		Status:           sql.NullString{String: "pending", Valid: true},
		DisplayOrder:     sql.NullInt64{Int64: 0, Valid: true},
	})
	if err != nil {
		slog.Error("failed to store AI image record", "error", err, "product_id", productID)
		return
	}

	slog.Info("generated AI image for scraped product", "product_id", productID, "filename", outputFilename, "model", modelUsed)
}

// createDirIfNotExists creates a directory if it doesn't exist
func createDirIfNotExists(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
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

// deduplicateImageURLs removes duplicate images that appear in different formats
// (e.g., same image in webp and jpg format from CDN)
func deduplicateImageURLs(urls []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, url := range urls {
		// Extract the underlying filename from the URL
		// URLs look like: https://images.cults3d.com/.../filename.jpg
		// or with format filter: https://images.cults3d.com/...format(webp)/.../filename.jpg
		key := extractImageKey(url)
		if key == "" {
			key = url // fallback to full URL if extraction fails
		}

		if !seen[key] {
			seen[key] = true
			result = append(result, url)
		}
	}

	return result
}

// extractImageKey extracts a unique identifier from an image URL
// by finding the underlying filename
func extractImageKey(url string) string {
	// The URLs contain the original filename at the end
	// Find the last path segment after the last /
	lastSlash := strings.LastIndex(url, "/")
	if lastSlash == -1 {
		return ""
	}

	filename := url[lastSlash+1:]
	// Remove query parameters if any
	if idx := strings.Index(filename, "?"); idx != -1 {
		filename = filename[:idx]
	}

	return filename
}
