package importer

import (
	"context"
	"time"
)

// Designer represents a 3D model designer with their sources
type Designer struct {
	Name            string
	Slug            string
	Sources         []Source
	DefaultCategory string
}

// Source represents a platform where a designer publishes models
type Source struct {
	Platform string // "cults3d", "mmf"
	URL      string
}

// ScrapedProduct represents a product scraped from a source
type ScrapedProduct struct {
	ID                 string
	DesignerSlug       string
	Platform           string
	SourceURL          string
	Name               string
	Description        string
	OriginalPriceCents int
	ReleaseDate        *time.Time
	ImageURLs          []string
	Tags               []string
	RawHTML            string
	ScrapedAt          time.Time
}

// Scraper interface for fetching products from a source
type Scraper interface {
	// Name returns the scraper name (e.g., "Cults3D")
	Name() string

	// Platform returns the platform slug (e.g., "cults3d")
	Platform() string

	// FetchDesignerProducts fetches all product URLs from a designer's page
	FetchDesignerProducts(ctx context.Context, designerURL string) ([]string, error)

	// FetchProduct fetches full product details from a product URL
	FetchProduct(ctx context.Context, productURL string) (*ScrapedProduct, error)
}

// Designers is the list of configured designers to scrape
var Designers = []Designer{
	{
		Name: "TheDragonsDen",
		Slug: "thedragonsden",
		Sources: []Source{
			{Platform: "cults3d", URL: "https://cults3d.com/en/users/TheDragonsDen/3d-models"},
		},
		DefaultCategory: "Dragons",
	},
	{
		Name: "FlexiFactory",
		Slug: "flexifactory",
		Sources: []Source{
			{Platform: "cults3d", URL: "https://cults3d.com/en/users/FlexiFactory/3d-models"},
		},
		DefaultCategory: "Animals",
	},
	{
		Name: "Cinderwing3D",
		Slug: "cinderwing3d",
		Sources: []Source{
			{Platform: "cults3d", URL: "https://cults3d.com/en/users/Cinderwing3D/3d-models"},
		},
		DefaultCategory: "Dragons",
	},
}

// GetDesigner returns a designer by slug
func GetDesigner(slug string) *Designer {
	for i := range Designers {
		if Designers[i].Slug == slug {
			return &Designers[i]
		}
	}
	return nil
}
