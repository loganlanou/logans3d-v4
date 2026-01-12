package importer

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Cults3DScraper scrapes products from Cults3D
type Cults3DScraper struct {
	client *HTTPClient
}

// NewCults3DScraper creates a new Cults3D scraper
func NewCults3DScraper() *Cults3DScraper {
	return &Cults3DScraper{
		client: NewHTTPClient(30), // 30 requests per minute
	}
}

// Name returns the scraper name
func (s *Cults3DScraper) Name() string {
	return "Cults3D"
}

// Platform returns the platform slug
func (s *Cults3DScraper) Platform() string {
	return "cults3d"
}

// FetchDesignerProducts fetches all product URLs from a designer's page
func (s *Cults3DScraper) FetchDesignerProducts(ctx context.Context, designerURL string) ([]string, error) {
	var allURLs []string
	page := 1

	for {
		pageURL := fmt.Sprintf("%s?page=%d", designerURL, page)
		slog.Debug("fetching designer page", "url", pageURL, "page", page)

		body, err := s.client.Get(ctx, pageURL)
		if err != nil {
			return nil, fmt.Errorf("fetch page %d: %w", page, err)
		}

		urls := s.extractProductURLs(string(body))
		if len(urls) == 0 {
			break // No more products
		}

		allURLs = append(allURLs, urls...)
		slog.Debug("found products on page", "page", page, "count", len(urls), "total", len(allURLs))

		// Check if there's a next page
		if !s.hasNextPage(string(body), page) {
			break
		}

		page++
	}

	return allURLs, nil
}

// FetchProduct fetches full product details from a product URL
func (s *Cults3DScraper) FetchProduct(ctx context.Context, productURL string) (*ScrapedProduct, error) {
	slog.Debug("fetching product", "url", productURL)

	body, err := s.client.Get(ctx, productURL)
	if err != nil {
		return nil, fmt.Errorf("fetch product: %w", err)
	}

	html := string(body)

	product := &ScrapedProduct{
		SourceURL: productURL,
		Platform:  "cults3d",
		RawHTML:   html,
		ScrapedAt: time.Now(),
	}

	// Extract name
	product.Name = s.extractName(html)
	if product.Name == "" {
		return nil, fmt.Errorf("could not extract product name from %s", productURL)
	}

	// Extract description
	product.Description = s.extractDescription(html)

	// Extract price
	product.OriginalPriceCents = s.extractPrice(html)

	// Extract release date
	product.ReleaseDate = s.extractReleaseDate(html)

	// Extract image URLs
	product.ImageURLs = s.extractImageURLs(html)

	// Extract tags
	product.Tags = s.extractTags(html)

	// Extract designer slug from URL
	product.DesignerSlug = s.extractDesignerSlug(productURL)

	return product, nil
}

// extractProductURLs extracts product URLs from a listing page
func (s *Cults3DScraper) extractProductURLs(html string) []string {
	// Pattern: href="/en/3d-model/category/product-name"
	re := regexp.MustCompile(`href="(/en/3d-model/[^"]+)"`)
	matches := re.FindAllStringSubmatch(html, -1)

	seen := make(map[string]bool)
	var urls []string

	for _, match := range matches {
		path := match[1]
		fullURL := "https://cults3d.com" + path

		// Skip if already seen
		if seen[fullURL] {
			continue
		}
		seen[fullURL] = true

		// Skip pages that aren't actual products
		if strings.Contains(path, "/comments") ||
			strings.Contains(path, "/reviews") ||
			strings.Contains(path, "/download") {
			continue
		}

		urls = append(urls, fullURL)
	}

	return urls
}

// hasNextPage checks if there's a next page link
func (s *Cults3DScraper) hasNextPage(html string, currentPage int) bool {
	nextPage := currentPage + 1
	// Look for link to next page
	pattern := fmt.Sprintf(`page=%d`, nextPage)
	return strings.Contains(html, pattern)
}

// extractName extracts the product name from the page
func (s *Cults3DScraper) extractName(html string) string {
	// Try to find the h1 title
	re := regexp.MustCompile(`<h1[^>]*>([^<]+)</h1>`)
	if matches := re.FindStringSubmatch(html); len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// Fallback: try og:title
	re = regexp.MustCompile(`<meta\s+property="og:title"\s+content="([^"]+)"`)
	if matches := re.FindStringSubmatch(html); len(matches) > 1 {
		// Remove " STL file" suffix if present
		title := strings.TrimSuffix(matches[1], " STL file")
		return strings.TrimSpace(title)
	}

	return ""
}

// extractDescription extracts the product description
func (s *Cults3DScraper) extractDescription(html string) string {
	// Look for the description section
	re := regexp.MustCompile(`(?s)<div[^>]*class="[^"]*description[^"]*"[^>]*>(.*?)</div>`)
	if matches := re.FindStringSubmatch(html); len(matches) > 1 {
		// Strip HTML tags
		desc := regexp.MustCompile(`<[^>]+>`).ReplaceAllString(matches[1], "")
		desc = strings.TrimSpace(desc)
		// Limit length
		if len(desc) > 2000 {
			desc = desc[:2000]
		}
		return desc
	}

	// Fallback: og:description
	re = regexp.MustCompile(`<meta\s+property="og:description"\s+content="([^"]+)"`)
	if matches := re.FindStringSubmatch(html); len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	return ""
}

// extractPrice extracts the price in cents
func (s *Cults3DScraper) extractPrice(html string) int {
	// Pattern: US$ X.XX or $X.XX
	re := regexp.MustCompile(`(?:US\$|USD|\$)\s*(\d+)\.(\d{2})`)
	if matches := re.FindStringSubmatch(html); len(matches) > 2 {
		dollars, _ := strconv.Atoi(matches[1])
		cents, _ := strconv.Atoi(matches[2])
		return dollars*100 + cents
	}

	// Try JSON schema price
	re = regexp.MustCompile(`"price"\s*:\s*"?(\d+(?:\.\d+)?)"?`)
	if matches := re.FindStringSubmatch(html); len(matches) > 1 {
		price, _ := strconv.ParseFloat(matches[1], 64)
		return int(price * 100)
	}

	return 0
}

// extractReleaseDate extracts the publication date
func (s *Cults3DScraper) extractReleaseDate(html string) *time.Time {
	// Pattern: Publication date or datePublished
	// Example: "December 21, 2025 17:00 (UTC)"
	re := regexp.MustCompile(`(?:Publication date|Published)[:\s]+(\w+\s+\d+,\s+\d{4})`)
	if matches := re.FindStringSubmatch(html); len(matches) > 1 {
		t, err := time.Parse("January 2, 2006", matches[1])
		if err == nil {
			return &t
		}
	}

	// Try datePublished in JSON
	re = regexp.MustCompile(`"datePublished"\s*:\s*"([^"]+)"`)
	if matches := re.FindStringSubmatch(html); len(matches) > 1 {
		t, err := time.Parse(time.RFC3339, matches[1])
		if err == nil {
			return &t
		}
		// Try simpler format
		t, err = time.Parse("2006-01-02", matches[1])
		if err == nil {
			return &t
		}
	}

	return nil
}

// extractImageURLs extracts image URLs from the page
func (s *Cults3DScraper) extractImageURLs(html string) []string {
	// Look for images.cults3d.com URLs
	re := regexp.MustCompile(`https://images\.cults3d\.com/[^"'\s]+`)
	matches := re.FindAllString(html, -1)

	seen := make(map[string]bool)
	var urls []string

	for _, url := range matches {
		// Skip small thumbnails (look for larger versions)
		if strings.Contains(url, "/113x113") {
			continue
		}

		// Normalize URL
		url = strings.Split(url, "?")[0] // Remove query params

		if seen[url] {
			continue
		}
		seen[url] = true

		urls = append(urls, url)
	}

	// Limit to first 10 images
	if len(urls) > 10 {
		urls = urls[:10]
	}

	return urls
}

// extractTags extracts tags/categories from the page
func (s *Cults3DScraper) extractTags(html string) []string {
	var tags []string
	seen := make(map[string]bool)

	// Look for tag links
	re := regexp.MustCompile(`href="/en/tags/([^"]+)"`)
	for _, match := range re.FindAllStringSubmatch(html, -1) {
		tag := strings.ReplaceAll(match[1], "-", " ")
		tag = strings.ToLower(tag)
		if !seen[tag] {
			seen[tag] = true
			tags = append(tags, tag)
		}
	}

	// Also look for common keywords in the content
	keywords := []string{"articulated", "print in place", "no support", "flexi", "flexible"}
	lowerHTML := strings.ToLower(html)
	for _, kw := range keywords {
		if strings.Contains(lowerHTML, kw) && !seen[kw] {
			seen[kw] = true
			tags = append(tags, kw)
		}
	}

	return tags
}

// extractDesignerSlug extracts the designer slug from a product URL
func (s *Cults3DScraper) extractDesignerSlug(productURL string) string {
	// Products often have the designer name at the end: /product-name-designername
	// But this isn't reliable, so we'll set it later from the designer config
	return ""
}
