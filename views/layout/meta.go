package layout

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/storage/db"
)

// PageMeta contains all metadata for a page (SEO, Open Graph, Twitter, Schema.org)
type PageMeta struct {
	// Basic HTML meta
	Title        string
	Description  string
	Keywords     []string
	CanonicalURL string

	// Open Graph
	OGType        string // "website" or "product"
	OGTitle       string
	OGDescription string
	OGImageURL    string // MUST be absolute URL
	OGURL         string // MUST be absolute URL
	OGSiteName    string

	// Twitter Cards
	TwitterCard        string // "summary_large_image"
	TwitterTitle       string
	TwitterDescription string
	TwitterImageURL    string // MUST be absolute URL
	TwitterSite        string // "@handle"

	// Facebook
	FacebookAppID  string
	FacebookPageID string

	// Internal state
	SiteURL    string // e.g., "https://www.logans3dcreations.com"
	Product    *db.Product
	Categories []db.Category

	// Schema.org JSON-LD (pre-computed)
	ProductSchemaJSON string
}

// NewPageMeta creates a PageMeta with site-wide defaults
// Call this first, then chain .FromProduct() or other modifiers
func NewPageMeta(c echo.Context, queries *db.Queries) PageMeta {
	ctx := context.Background()

	// Load site config from database (with fallback defaults)
	siteURL := getConfigValue(ctx, queries, "site_url", "https://www.logans3dcreations.com")
	siteName := getConfigValue(ctx, queries, "site_name", "Logan's 3D Creations")
	defaultDescription := getConfigValue(ctx, queries, "site_description", "Custom 3D printed collectibles, dinosaurs, and more")
	defaultOGImage := getConfigValue(ctx, queries, "default_og_image", "/public/images/social/default-og.jpg")
	twitterHandle := getConfigValue(ctx, queries, "twitter_handle", "")
	facebookPageID := getConfigValue(ctx, queries, "facebook_page_id", "")
	facebookAppID := getConfigValue(ctx, queries, "facebook_app_id", "")

	// Build canonical URL from request path
	canonicalURL := BuildAbsoluteURL(siteURL, c.Request().URL.Path)

	return PageMeta{
		// HTML meta defaults
		Title:        siteName,
		Description:  defaultDescription,
		Keywords:     []string{"3D printing", "collectibles", "custom", "dinosaurs"},
		CanonicalURL: canonicalURL,

		// Open Graph defaults
		OGType:        "website",
		OGTitle:       siteName,
		OGDescription: defaultDescription,
		OGImageURL:    BuildAbsoluteURL(siteURL, defaultOGImage),
		OGURL:         canonicalURL,
		OGSiteName:    siteName,

		// Twitter defaults
		TwitterCard:        "summary_large_image",
		TwitterTitle:       siteName,
		TwitterDescription: defaultDescription,
		TwitterImageURL:    BuildAbsoluteURL(siteURL, defaultOGImage),
		TwitterSite:        twitterHandle,

		// Facebook
		FacebookPageID: facebookPageID,
		FacebookAppID:  facebookAppID,

		// Internal
		SiteURL: siteURL,
	}
}

// FromProduct updates PageMeta with product-specific information
func (pm PageMeta) FromProduct(product db.Product) PageMeta {
	// Use SEO overrides if available, otherwise fallback to product fields
	title := product.SeoTitle.String
	if title == "" {
		title = product.Name
	}

	description := product.SeoDescription.String
	if description == "" {
		if product.Description.Valid {
			description = product.Description.String
		} else if product.ShortDescription.Valid {
			description = product.ShortDescription.String
		}
	}

	// Keywords: Use custom or auto-generate
	var keywords []string
	if product.SeoKeywords.Valid && product.SeoKeywords.String != "" {
		keywords = strings.Split(product.SeoKeywords.String, ",")
		for i := range keywords {
			keywords[i] = strings.TrimSpace(keywords[i])
		}
	} else {
		// Auto-generate from product name
		keywords = []string{
			"3D printed",
			product.Name,
			"collectible",
			"custom printing",
		}
	}

	// Update all title fields
	pm.Title = title + " - " + pm.OGSiteName
	pm.OGTitle = title
	pm.TwitterTitle = title

	// Update all description fields
	pm.Description = description
	pm.OGDescription = description
	pm.TwitterDescription = description

	// Update keywords
	pm.Keywords = keywords

	// Build product-specific URLs
	productURL := fmt.Sprintf("%s/shop/product/%s", pm.SiteURL, product.Slug)
	pm.CanonicalURL = productURL
	pm.OGURL = productURL

	// Update OG type for product
	pm.OGType = "product"

	// Handle custom OG image if set
	if product.OgImageUrl.Valid && product.OgImageUrl.String != "" {
		pm.OGImageURL = BuildAbsoluteURL(pm.SiteURL, product.OgImageUrl.String)
		pm.TwitterImageURL = pm.OGImageURL
	}

	// Store product for schema generation
	pm.Product = &product

	return pm
}

// WithProductImage sets the product image for OG/Twitter
// Call after FromProduct() with the primary product image
func (pm PageMeta) WithProductImage(imageFilename string) PageMeta {
	if imageFilename != "" {
		imageURL := fmt.Sprintf("/public/images/products/%s", imageFilename)
		absoluteURL := BuildAbsoluteURL(pm.SiteURL, imageURL)

		// Only set if not already overridden by custom og_image_url
		if pm.Product != nil && !pm.Product.OgImageUrl.Valid {
			pm.OGImageURL = absoluteURL
			pm.TwitterImageURL = absoluteURL
		}
	}
	return pm
}

// WithCategories adds category information for breadcrumbs and schema
func (pm PageMeta) WithCategories(categories []db.Category) PageMeta {
	pm.Categories = categories

	// Re-generate product schema with categories if product exists
	if pm.Product != nil {
		pm.ProductSchemaJSON = pm.generateProductSchemaJSON(*pm.Product, pm.Categories)
	}

	return pm
}

// WithOGImage overrides the OG image URL
func (pm PageMeta) WithOGImage(imageURL string) PageMeta {
	absoluteURL := BuildAbsoluteURL(pm.SiteURL, imageURL)
	pm.OGImageURL = absoluteURL
	pm.TwitterImageURL = absoluteURL
	return pm
}

// KeywordsString returns keywords as a comma-separated string
func (pm PageMeta) KeywordsString() string {
	return strings.Join(pm.Keywords, ", ")
}

// BuildAbsoluteURL constructs an absolute URL from a path
func BuildAbsoluteURL(siteURL, path string) string {
	// Handle empty path
	if path == "" {
		return siteURL
	}

	// Handle already absolute URLs
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}

	// Remove trailing slash from site URL
	siteURL = strings.TrimRight(siteURL, "/")

	// Ensure path starts with /
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return siteURL + path
}

// CanonicalURL helper for building canonical URL from context
func CanonicalURL(c echo.Context, path string) string {
	siteURL := "https://www.logans3dcreations.com" // Could load from config
	return BuildAbsoluteURL(siteURL, path)
}

// generateProductSchemaJSON generates Schema.org Product JSON-LD
func (pm PageMeta) generateProductSchemaJSON(product db.Product, categories []db.Category) string {
	data := pm.ProductSchemaData()
	if data == nil {
		return ""
	}

	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(bytes)
}

// ProductSchemaData returns the product schema as a map for JSON-LD
func (pm PageMeta) ProductSchemaData() map[string]interface{} {
	if pm.Product == nil {
		return nil
	}

	product := pm.Product

	// Build category string from breadcrumb
	category := ""
	if len(pm.Categories) > 0 {
		category = pm.Categories[len(pm.Categories)-1].Name
	}

	// Build offers section
	availability := "https://schema.org/InStock"
	if product.StockQuantity.Valid && product.StockQuantity.Int64 <= 0 {
		availability = "https://schema.org/OutOfStock"
	}

	offers := map[string]interface{}{
		"@type":         "Offer",
		"url":           pm.OGURL,
		"priceCurrency": "USD",
		"price":         fmt.Sprintf("%.2f", float64(product.PriceCents)/100.0),
		"availability":  availability,
	}

	schema := map[string]interface{}{
		"@context":    "https://schema.org/",
		"@type":       "Product",
		"name":        product.Name,
		"description": pm.Description,
		"brand": map[string]interface{}{
			"@type": "Brand",
			"name":  "Logan's 3D Creations",
		},
		"offers": offers,
	}

	// Add SKU if available
	if product.Sku.Valid {
		schema["sku"] = product.Sku.String
	}

	// Add image if available
	if pm.OGImageURL != "" {
		schema["image"] = pm.OGImageURL
	}

	// Add category if available
	if category != "" {
		schema["category"] = category
	}

	return schema
}

// OrganizationSchemaData returns site-wide Organization schema
func (pm PageMeta) OrganizationSchemaData() map[string]interface{} {
	schema := map[string]interface{}{
		"@context": "https://schema.org",
		"@type":    "Organization",
		"name":     pm.OGSiteName,
		"url":      pm.SiteURL,
	}

	// Add logo if available
	logoPath := "/public/images/social/logo-square.jpg"
	schema["logo"] = BuildAbsoluteURL(pm.SiteURL, logoPath)

	// Add social media URLs if available (currently commented out - add when handles are provided)
	// sameAs := []string{}
	// if pm.TwitterSite != "" {
	//     sameAs = append(sameAs, "https://twitter.com/" + strings.TrimPrefix(pm.TwitterSite, "@"))
	// }
	// if pm.FacebookPageID != "" {
	//     sameAs = append(sameAs, "https://www.facebook.com/..." + pm.FacebookPageID)
	// }
	// if len(sameAs) > 0 {
	//     schema["sameAs"] = sameAs
	// }

	return schema
}

// getConfigValue retrieves a config value from the database with a fallback default
func getConfigValue(ctx context.Context, queries *db.Queries, key string, defaultValue string) string {
	value, err := queries.GetSiteConfig(ctx, key)
	if err != nil || value == "" {
		return defaultValue
	}
	return value
}
