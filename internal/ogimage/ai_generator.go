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
	geminiAPIEndpoint = "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash-exp:generateContent"
	defaultTimeout    = 60 * time.Second
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

type generationConfig struct {
	ResponseModalities []string `json:"responseModalities,omitempty"`
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

func (g *AIGenerator) GenerateMultiVariantOGImage(info MultiVariantInfo, outputPath string) error {
	if g.apiKey == "" || g.apiKey == "invalid-key" {
		slog.Debug("AI generator: no valid API key, falling back to grid method")
		return GenerateMultiVariantOGImage(info, outputPath)
	}

	imageData, err := g.callGeminiAPI(info)
	if err != nil {
		slog.Error("AI image generation failed, falling back to grid method", "error", err)
		return GenerateMultiVariantOGImage(info, outputPath)
	}

	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		slog.Error("failed to create output directory", "error", err, "dir", outputDir)
		return fmt.Errorf("create output dir: %w", err)
	}

	if err := os.WriteFile(outputPath, imageData, 0644); err != nil {
		slog.Error("failed to write AI-generated image", "error", err, "path", outputPath)
		return fmt.Errorf("write image: %w", err)
	}

	slog.Info("generated AI multi-variant OG image", "product", info.Name, "output", outputPath)
	return nil
}

func (g *AIGenerator) callGeminiAPI(info MultiVariantInfo) ([]byte, error) {
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
		},
	}

	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", geminiAPIEndpoint, bytes.NewReader(jsonBody))
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

	prompt := fmt.Sprintf(`Professional product photography of 3D printed collectible toys for e-commerce social sharing.

Subject: %d color variants of "%s" (colors: %s)

Composition: Wide 1200x630 landscape format. Toys arranged in a dynamic diagonal line or gentle arc formation in the UPPER 75%% of the image, each facing slightly toward camera, positioned as if walking together or displayed as collectibles.

Environment: Subtle thematic background that complements the creature/toy type - soft out-of-focus natural elements (moss, rocks, leaves) or a gentle gradient suggesting their habitat. Background should be heavily blurred (f/1.8 depth of field) so products remain the sharp focal point.

Lighting: Soft studio lighting from upper left, gentle rim lighting to separate subjects from background, no harsh shadows. Professional commercial photography style.

Style: Photorealistic, high-end collectible toy photography, 8K detail on the products, cinematic color grading with rich but natural tones.

TEXT BANNER (REQUIRED):
Add a semi-transparent dark banner across the BOTTOM of the image (approximately 100-120 pixels tall) containing:
- Line 1 (large, bold, white text, centered): "%s"
- Line 2 (smaller, white text, centered): "Available in %d colors! • %s"

The text must be clearly readable, professionally styled, and centered on the dark banner.

Critical requirements:
- Products must be tack-sharp and the clear focal point
- Products should be in the upper portion, NOT overlapping the text banner
- Background stays subtle and out of focus - never distracting
- NO watermarks, logos, or signatures - ONLY the required text banner
- Maintain exact appearance and colors of each toy from input images
- The text banner is MANDATORY - do not skip it`,
		len(info.ImagePaths),
		info.Name,
		styleList,
		info.Name,
		info.StyleCount,
		info.PriceRange,
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
