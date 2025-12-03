package ogimage

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// Primary: Nano Banana Pro - best quality
	geminiPrimaryModel = "gemini-3-pro-image-preview"
	// Fallback: Gemini 2.5 Flash - stable availability
	geminiFallbackModel = "gemini-2.5-flash-preview-05-20"

	geminiAPIBase  = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent"
	defaultTimeout = 120 * time.Second
)

type AIGenerator struct {
	apiKey     string
	httpClient *http.Client
}

func NewAIGenerator(apiKey string) *AIGenerator {
	return &AIGenerator{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

type geminiRequest struct {
	Contents         []geminiContent   `json:"contents"`
	GenerationConfig *generationConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text       string      `json:"text,omitempty"`
	InlineData *inlineData `json:"inline_data,omitempty"`
}

type inlineData struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}

type imageConfig struct {
	AspectRatio string `json:"aspectRatio,omitempty"`
}

type generationConfig struct {
	ResponseModalities []string     `json:"responseModalities,omitempty"`
	ImageConfig        *imageConfig `json:"imageConfig,omitempty"`
}

type geminiResponse struct {
	Candidates []candidate `json:"candidates"`
	Error      *apiError   `json:"error,omitempty"`
}

type candidate struct {
	Content contentResponse `json:"content"`
}

type contentResponse struct {
	Parts []partResponse `json:"parts"`
}

type partResponse struct {
	Text       string          `json:"text,omitempty"`
	InlineData *inlineDataResp `json:"inlineData,omitempty"`
}

type inlineDataResp struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// GenerateMultiVariantOGImageWithModel generates an OG image and returns which model was used
func (g *AIGenerator) GenerateMultiVariantOGImageWithModel(info MultiVariantInfo, outputPath string) (modelUsed string, err error) {
	if g.apiKey == "" || g.apiKey == "invalid-key" {
		slog.Debug("AI generator: no valid API key, falling back to grid method")
		return "grid", GenerateMultiVariantOGImage(info, outputPath)
	}

	// Try primary model first (Nano Banana Pro)
	imageData, err := g.callGeminiAPIWithModel(info, geminiPrimaryModel)
	modelUsed = geminiPrimaryModel

	if err != nil {
		slog.Warn("primary model failed, trying fallback", "primary", geminiPrimaryModel, "error", err)

		// Try fallback model (Gemini 2.5 Flash)
		imageData, err = g.callGeminiAPIWithModel(info, geminiFallbackModel)
		modelUsed = geminiFallbackModel

		if err != nil {
			slog.Error("all AI models failed, keeping existing image", "error", err)
			return "", fmt.Errorf("all AI models failed: %w", err)
		}
	}

	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		slog.Error("failed to create output directory", "error", err, "dir", outputDir)
		return "", fmt.Errorf("create output dir: %w", err)
	}

	if err := os.WriteFile(outputPath, imageData, 0644); err != nil {
		slog.Error("failed to write AI-generated image", "error", err, "path", outputPath)
		return "", fmt.Errorf("write image: %w", err)
	}

	slog.Info("generated AI multi-variant OG image", "product", info.Name, "model", modelUsed, "output", outputPath)
	return modelUsed, nil
}

// GenerateMultiVariantOGImage generates an OG image (legacy method, doesn't return model)
func (g *AIGenerator) GenerateMultiVariantOGImage(info MultiVariantInfo, outputPath string) error {
	_, err := g.GenerateMultiVariantOGImageWithModel(info, outputPath)
	return err
}

func (g *AIGenerator) callGeminiAPIWithModel(info MultiVariantInfo, model string) ([]byte, error) {
	parts := []geminiPart{
		{Text: g.buildPrompt(info)},
	}

	for _, imagePath := range info.ImagePaths {
		imageData, mimeType, err := g.loadImageAsBase64(imagePath)
		if err != nil {
			slog.Debug("failed to load image for AI generation", "error", err, "path", imagePath)
			continue
		}
		parts = append(parts, geminiPart{
			InlineData: &inlineData{
				MimeType: mimeType,
				Data:     imageData,
			},
		})
	}

	if len(parts) == 1 {
		return nil, fmt.Errorf("no images loaded for AI generation")
	}

	req := geminiRequest{
		Contents: []geminiContent{
			{Parts: parts},
		},
		GenerationConfig: &generationConfig{
			ResponseModalities: []string{"TEXT", "IMAGE"},
			ImageConfig: &imageConfig{
				AspectRatio: "16:9", // Closest to 1200x630 (1.9:1)
			},
		},
	}

	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := fmt.Sprintf(geminiAPIBase, model)
	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", g.apiKey)

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if geminiResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", geminiResp.Error.Message)
	}

	for _, candidate := range geminiResp.Candidates {
		for _, part := range candidate.Content.Parts {
			if part.InlineData != nil && part.InlineData.Data != "" {
				imageData, err := base64.StdEncoding.DecodeString(part.InlineData.Data)
				if err != nil {
					return nil, fmt.Errorf("decode image: %w", err)
				}
				return imageData, nil
			}
		}
	}

	return nil, fmt.Errorf("no image in API response")
}

func (g *AIGenerator) buildPrompt(info MultiVariantInfo) string {
	styleList := strings.Join(info.StyleNames, ", ")

	// Build the second line text based on available variants
	var line2Text string
	switch {
	case info.StyleCount > 1 && info.SizeCount > 1:
		line2Text = fmt.Sprintf("%d Colors • %d Sizes • %s", info.StyleCount, info.SizeCount, info.PriceRange)
	case info.StyleCount > 1:
		line2Text = fmt.Sprintf("%d Colors • %s", info.StyleCount, info.PriceRange)
	case info.SizeCount > 1:
		line2Text = fmt.Sprintf("%d Sizes • %s", info.SizeCount, info.PriceRange)
	default:
		line2Text = fmt.Sprintf("%s • Shop Now", info.PriceRange)
	}

	// Narrative prompt with explicit text rendering instructions for Nano Banana Pro
	prompt := fmt.Sprintf(`Create a professional e-commerce product image for social media sharing. This must be a wide 16:9 landscape format image.

The image shows %d beautiful 3D printed collectible toy figures of "%s" in the colors: %s. These articulated toys are arranged in a pleasing diagonal composition in the upper portion of the frame, each facing slightly toward the camera as if posing together for a group photo. They should be spread across the width to use the wide format effectively.

Behind the toys is a softly blurred natural environment - imagine a forest floor with moss, small rocks, and scattered leaves, all rendered with beautiful bokeh (f/1.8 depth of field). The background complements the creatures and suggests their natural habitat while keeping the products as the sharp focal point.

The lighting is soft and professional: main light from upper left, gentle rim lighting to separate subjects from background, no harsh shadows. Style is photorealistic, high-end collectible toy photography with rich cinematic color grading.

TEXT BANNER REQUIREMENT:
At the very bottom of the image, add a semi-transparent dark banner (approximately 15-20%% of image height) containing centered white text:
- Line 1 (larger, bold): "%s"
- Line 2 (smaller): "%s"

The text must be crisp, clearly legible, professionally styled like a retail promotional image. Use a clean sans-serif font. IMPORTANT: Ensure correct spelling of all text - copy the exact text provided above character for character.

CRITICAL REQUIREMENTS:
- 16:9 wide landscape aspect ratio
- Products in upper 75%% of image, NOT overlapping the text banner
- Natural thematic background with bokeh - absolutely NO plain white or studio backgrounds
- Text banner at bottom is MANDATORY with the exact text specified above
- NO watermarks, logos, or signatures anywhere
- Preserve the exact colors and appearance of each toy from the input images`,
		len(info.ImagePaths),
		info.Name,
		styleList,
		info.Name,
		line2Text,
	)

	return prompt
}

func (g *AIGenerator) loadImageAsBase64(imagePath string) (string, string, error) {
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return "", "", fmt.Errorf("read image file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(imagePath))
	var mimeType string
	switch ext {
	case ".jpg", ".jpeg":
		mimeType = "image/jpeg"
	case ".png":
		mimeType = "image/png"
	case ".gif":
		mimeType = "image/gif"
	case ".webp":
		mimeType = "image/webp"
	default:
		mimeType = "image/jpeg"
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	return encoded, mimeType, nil
}

// SingleProductBackgroundInfo contains info for generating a single product background
type SingleProductBackgroundInfo struct {
	Name      string
	ImagePath string
}

// GenerateSingleProductBackground generates an AI background for a single product image (no text overlay)
func (g *AIGenerator) GenerateSingleProductBackground(info SingleProductBackgroundInfo, outputPath string) (modelUsed string, err error) {
	if g.apiKey == "" || g.apiKey == "invalid-key" {
		return "", fmt.Errorf("no valid API key for AI generation")
	}

	// Try primary model first (Nano Banana Pro)
	imageData, err := g.callSingleProductAPI(info, geminiPrimaryModel)
	modelUsed = geminiPrimaryModel

	if err != nil {
		slog.Warn("primary model failed for single product, trying fallback", "primary", geminiPrimaryModel, "error", err)

		// Try fallback model (Gemini 2.5 Flash)
		imageData, err = g.callSingleProductAPI(info, geminiFallbackModel)
		modelUsed = geminiFallbackModel

		if err != nil {
			slog.Error("all AI models failed for single product background", "error", err)
			return "", fmt.Errorf("all AI models failed: %w", err)
		}
	}

	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		slog.Error("failed to create output directory", "error", err, "dir", outputDir)
		return "", fmt.Errorf("create output dir: %w", err)
	}

	if err := os.WriteFile(outputPath, imageData, 0644); err != nil {
		slog.Error("failed to write AI-generated background", "error", err, "path", outputPath)
		return "", fmt.Errorf("write image: %w", err)
	}

	slog.Info("generated AI single-product background", "product", info.Name, "model", modelUsed, "output", outputPath)
	return modelUsed, nil
}

func (g *AIGenerator) callSingleProductAPI(info SingleProductBackgroundInfo, model string) ([]byte, error) {
	parts := []geminiPart{
		{Text: g.buildSingleProductPrompt(info)},
	}

	// Load the source image
	imageData, mimeType, err := g.loadImageAsBase64(info.ImagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load source image: %w", err)
	}
	parts = append(parts, geminiPart{
		InlineData: &inlineData{
			MimeType: mimeType,
			Data:     imageData,
		},
	})

	req := geminiRequest{
		Contents: []geminiContent{
			{Parts: parts},
		},
		GenerationConfig: &generationConfig{
			ResponseModalities: []string{"TEXT", "IMAGE"},
			ImageConfig: &imageConfig{
				AspectRatio: "16:9",
			},
		},
	}

	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := fmt.Sprintf(geminiAPIBase, model)
	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", g.apiKey)

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if geminiResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", geminiResp.Error.Message)
	}

	for _, candidate := range geminiResp.Candidates {
		for _, part := range candidate.Content.Parts {
			if part.InlineData != nil && part.InlineData.Data != "" {
				imageBytes, err := base64.StdEncoding.DecodeString(part.InlineData.Data)
				if err != nil {
					return nil, fmt.Errorf("decode image: %w", err)
				}
				return imageBytes, nil
			}
		}
	}

	return nil, fmt.Errorf("no image in API response")
}

func (g *AIGenerator) buildSingleProductPrompt(info SingleProductBackgroundInfo) string {
	prompt := fmt.Sprintf(`Create a professional e-commerce product photograph. This must be a wide 16:9 landscape format image.

The image shows a single 3D printed collectible toy figure of "%s" positioned prominently in the center-right of the frame, facing slightly toward the camera.

Behind the toy is a softly blurred natural environment - imagine a forest floor with moss, small rocks, and scattered leaves, all rendered with beautiful bokeh (f/1.8 depth of field). The background complements the creature and suggests its natural habitat while keeping the product as the sharp focal point.

The lighting is soft and professional: main light from upper left, gentle rim lighting to separate subject from background, no harsh shadows. Style is photorealistic, high-end collectible toy photography with rich cinematic color grading.

CRITICAL REQUIREMENTS:
- 16:9 wide landscape aspect ratio
- Product centered in frame, sharp focus
- Natural thematic background with bokeh - NO plain white or studio backgrounds
- NO text, watermarks, logos, banners, or overlays of any kind
- Preserve the exact colors and appearance of the toy from the input image`,
		info.Name,
	)

	return prompt
}
