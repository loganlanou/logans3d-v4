package sync

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultTimeout = 60 * time.Second
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient() *Client {
	baseURL := os.Getenv("PRODUCTION_API_URL")
	if baseURL == "" {
		baseURL = "https://logans3dcreations.com"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	apiKey := os.Getenv("PRODUCTION_API_KEY")

	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

func (c *Client) IsConfigured() bool {
	return c.apiKey != ""
}

func (c *Client) GetBaseURL() string {
	return c.baseURL
}

type ProductRequest struct {
	Name             string   `json:"name"`
	Slug             string   `json:"slug,omitempty"`
	Description      string   `json:"description,omitempty"`
	ShortDescription string   `json:"short_description,omitempty"`
	PriceCents       int64    `json:"price_cents"`
	CategoryID       string   `json:"category_id"`
	SKU              string   `json:"sku,omitempty"`
	StockQuantity    int64    `json:"stock_quantity"`
	WeightGrams      int64    `json:"weight_grams,omitempty"`
	LeadTimeDays     int64    `json:"lead_time_days,omitempty"`
	IsActive         bool     `json:"is_active"`
	IsFeatured       bool     `json:"is_featured"`
	IsPremium        bool     `json:"is_premium"`
	IsNew            bool     `json:"is_new"`
	Disclaimer       string   `json:"disclaimer,omitempty"`
	SEOTitle         string   `json:"seo_title,omitempty"`
	SEODescription   string   `json:"seo_description,omitempty"`
	SEOKeywords      string   `json:"seo_keywords,omitempty"`
	OGImageURL       string   `json:"og_image_url,omitempty"`
	SourceURL        string   `json:"source_url,omitempty"`
	SourcePlatform   string   `json:"source_platform,omitempty"`
	DesignerName     string   `json:"designer_name,omitempty"`
	ReleaseDate      *string  `json:"release_date,omitempty"`
	Tags             []string `json:"tags,omitempty"`
}

type ProductResponse struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Slug             string     `json:"slug"`
	Description      string     `json:"description"`
	ShortDescription string     `json:"short_description"`
	PriceCents       int64      `json:"price_cents"`
	CategoryID       string     `json:"category_id"`
	SKU              string     `json:"sku"`
	StockQuantity    int64      `json:"stock_quantity"`
	WeightGrams      int64      `json:"weight_grams"`
	LeadTimeDays     int64      `json:"lead_time_days"`
	IsActive         bool       `json:"is_active"`
	IsFeatured       bool       `json:"is_featured"`
	IsPremium        bool       `json:"is_premium"`
	IsNew            bool       `json:"is_new"`
	SourceURL        string     `json:"source_url,omitempty"`
	SourcePlatform   string     `json:"source_platform,omitempty"`
	DesignerName     string     `json:"designer_name,omitempty"`
	ReleaseDate      *time.Time `json:"release_date,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type ImageResponse struct {
	ID           string `json:"id"`
	ProductID    string `json:"product_id"`
	ImageURL     string `json:"image_url"`
	AltText      string `json:"alt_text"`
	DisplayOrder int64  `json:"display_order"`
	IsPrimary    bool   `json:"is_primary"`
}

type SyncResult struct {
	Action    string           `json:"action"` // "created" or "updated"
	ProductID string           `json:"product_id"`
	Product   *ProductResponse `json:"product"`
	Images    []ImageResponse  `json:"images,omitempty"`
	Error     string           `json:"error,omitempty"`
}

type APIError struct {
	Message string `json:"message"`
}

func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader, contentType string) (*http.Response, error) {
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	return c.httpClient.Do(req)
}

func (c *Client) GetProductBySourceURL(ctx context.Context, sourceURL string) (*ProductResponse, error) {
	path := "/api/products/by-source?source_url=" + sourceURL
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil, "")
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var product ProductResponse
	if err := json.NewDecoder(resp.Body).Decode(&product); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &product, nil
}

func (c *Client) CreateProduct(ctx context.Context, req ProductRequest) (*ProductResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/api/products", bytes.NewReader(body), "application/json")
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var product ProductResponse
	if err := json.NewDecoder(resp.Body).Decode(&product); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &product, nil
}

func (c *Client) UpdateProduct(ctx context.Context, productID string, req ProductRequest) (*ProductResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	path := "/api/products/" + productID
	resp, err := c.doRequest(ctx, http.MethodPut, path, bytes.NewReader(body), "application/json")
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var product ProductResponse
	if err := json.NewDecoder(resp.Body).Decode(&product); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &product, nil
}

func (c *Client) UploadImage(ctx context.Context, productID, imagePath string, displayOrder int, isPrimary bool) (*ImageResponse, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("open image file: %w", err)
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("image", filepath.Base(imagePath))
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("copy file content: %w", err)
	}

	_ = writer.WriteField("display_order", fmt.Sprintf("%d", displayOrder))
	if isPrimary {
		_ = writer.WriteField("is_primary", "true")
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	path := "/api/products/" + productID + "/images"
	resp, err := c.doRequest(ctx, http.MethodPost, path, &body, writer.FormDataContentType())
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var image ImageResponse
	if err := json.NewDecoder(resp.Body).Decode(&image); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &image, nil
}

func (c *Client) SyncProduct(ctx context.Context, req ProductRequest, imagePaths []string) (*SyncResult, error) {
	if !c.IsConfigured() {
		return nil, fmt.Errorf("sync client not configured: PRODUCTION_API_KEY not set")
	}

	result := &SyncResult{}

	existing, err := c.GetProductBySourceURL(ctx, req.SourceURL)
	if err != nil {
		slog.Error("failed to check for existing product", "error", err, "source_url", req.SourceURL)
		return nil, fmt.Errorf("check existing product: %w", err)
	}

	var product *ProductResponse
	if existing != nil {
		slog.Debug("product exists, updating", "product_id", existing.ID, "source_url", req.SourceURL)
		product, err = c.UpdateProduct(ctx, existing.ID, req)
		if err != nil {
			slog.Error("failed to update product", "error", err, "product_id", existing.ID)
			return nil, fmt.Errorf("update product: %w", err)
		}
		result.Action = "updated"
	} else {
		slog.Debug("product does not exist, creating", "source_url", req.SourceURL)
		product, err = c.CreateProduct(ctx, req)
		if err != nil {
			slog.Error("failed to create product", "error", err, "name", req.Name)
			return nil, fmt.Errorf("create product: %w", err)
		}
		result.Action = "created"
	}

	result.ProductID = product.ID
	result.Product = product

	for i, imagePath := range imagePaths {
		isPrimary := i == 0
		image, err := c.UploadImage(ctx, product.ID, imagePath, i, isPrimary)
		if err != nil {
			slog.Warn("failed to upload image", "error", err, "path", imagePath, "product_id", product.ID)
			continue
		}
		result.Images = append(result.Images, *image)
	}

	slog.Info("product synced to production",
		"action", result.Action,
		"product_id", result.ProductID,
		"name", product.Name,
		"images_uploaded", len(result.Images),
	)

	return result, nil
}

func (c *Client) TestConnection(ctx context.Context) error {
	if !c.IsConfigured() {
		return fmt.Errorf("PRODUCTION_API_KEY not set")
	}

	resp, err := c.doRequest(ctx, http.MethodGet, "/api/categories", nil, "")
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid API key")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
