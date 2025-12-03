package ogimage

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font/gofont/goregular"

	_ "image/jpeg"
)

const (
	// Nano Banana Pro - best consistency and text rendering
	geminiAPIEndpoint = "https://generativelanguage.googleapis.com/v1beta/models/gemini-3-pro-image-preview:generateContent"
	defaultTimeout    = 120 * time.Second // Increased for larger model

	// OG image dimensions
	ogWidth  = 1200
	ogHeight = 630
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

func (g *AIGenerator) GenerateMultiVariantOGImage(info MultiVariantInfo, outputPath string) error {
	if g.apiKey == "" || g.apiKey == "invalid-key" {
		slog.Debug("AI generator: no valid API key, falling back to grid method")
		return GenerateMultiVariantOGImage(info, outputPath)
	}

	// Step 1: Get AI-generated image (no text)
	aiImageData, err := g.callGeminiAPI(info)
	if err != nil {
		slog.Error("AI image generation failed, falling back to grid method", "error", err)
		return GenerateMultiVariantOGImage(info, outputPath)
	}

	// Step 2: Overlay text banner programmatically (guaranteed consistent)
	finalImageData, err := g.overlayTextBanner(aiImageData, info)
	if err != nil {
		slog.Error("failed to overlay text banner, falling back to grid method", "error", err)
		return GenerateMultiVariantOGImage(info, outputPath)
	}

	// Step 3: Save the final image
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		slog.Error("failed to create output directory", "error", err, "dir", outputDir)
		return fmt.Errorf("create output dir: %w", err)
	}

	if err := os.WriteFile(outputPath, finalImageData, 0644); err != nil {
		slog.Error("failed to write AI-generated image", "error", err, "path", outputPath)
		return fmt.Errorf("write image: %w", err)
	}

	slog.Info("generated AI multi-variant OG image", "product", info.Name, "output", outputPath, "dimensions", fmt.Sprintf("%dx%d", ogWidth, ogHeight))
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
			ImageConfig: &imageConfig{
				AspectRatio: "16:9", // Closest to 1200x630 (1.9:1)
			},
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

	// Narrative prompt - AI generates image only, we overlay text programmatically
	prompt := fmt.Sprintf(`Imagine a professional product photograph for an e-commerce website showcasing multiple 3D printed collectible toys. The scene features %d color variants of "%s" in the colors %s, arranged in a dynamic diagonal composition across a wide 16:9 landscape frame.

The toys are positioned in the upper two-thirds of the image, each facing slightly toward the camera as if walking together in a friendly group. They should be spread across the width of the frame to take advantage of the wide landscape format.

Behind them, a softly blurred natural environment creates depth and visual interest. Think of a forest floor with moss, small rocks, and fallen leaves - all rendered with beautiful bokeh effect at f/1.8 aperture. The background should complement the creature type and feel like their natural habitat, but remain soft and out of focus so the products stay sharp.

Soft studio lighting comes from the upper left, with gentle rim lighting that separates the subjects from the background. No harsh shadows. The overall style is photorealistic, high-end collectible toy photography with tack-sharp product detail and cinematic color grading.

The bottom portion of the image (roughly the lower 20%%) should continue the blurred background - leave this area clear for a text overlay that will be added separately.

Critical: Do NOT add any text, watermarks, logos, labels, or banners to the image. The image should be purely the product photograph with the thematic background. No white or plain studio backgrounds - the environment must be natural and thematic.`,
		len(info.ImagePaths),
		info.Name,
		styleList,
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

// overlayTextBanner takes the AI-generated image and adds a text banner at the bottom
// This ensures consistent text rendering regardless of AI output
func (g *AIGenerator) overlayTextBanner(imageData []byte, info MultiVariantInfo) ([]byte, error) {
	// Decode the AI-generated image
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, fmt.Errorf("decode AI image: %w", err)
	}

	// Create a new context at exact OG dimensions
	dc := gg.NewContext(ogWidth, ogHeight)

	// Calculate scaling to fit AI image into OG dimensions (cover/crop approach)
	srcWidth := float64(img.Bounds().Dx())
	srcHeight := float64(img.Bounds().Dy())
	scaleX := float64(ogWidth) / srcWidth
	scaleY := float64(ogHeight) / srcHeight

	// Use the larger scale to cover the canvas (may crop edges)
	scale := scaleX
	if scaleY > scaleX {
		scale = scaleY
	}

	// Calculate centered position
	scaledWidth := srcWidth * scale
	scaledHeight := srcHeight * scale
	offsetX := (float64(ogWidth) - scaledWidth) / 2
	offsetY := (float64(ogHeight) - scaledHeight) / 2

	// Draw the scaled/cropped AI image
	dc.Push()
	dc.Translate(offsetX, offsetY)
	dc.Scale(scale, scale)
	dc.DrawImage(img, 0, 0)
	dc.Pop()

	// Add semi-transparent bar at bottom for text
	textAreaHeight := 120.0
	textAreaY := float64(ogHeight) - textAreaHeight
	dc.SetRGBA(0, 0, 0, 0.85)
	dc.DrawRectangle(0, textAreaY, float64(ogWidth), textAreaHeight)
	dc.Fill()

	// Load font
	font, err := truetype.Parse(goregular.TTF)
	if err != nil {
		return nil, fmt.Errorf("parse font: %w", err)
	}

	// Draw product name (large)
	dc.SetRGB(1, 1, 1)
	face := truetype.NewFace(font, &truetype.Options{Size: 42})
	dc.SetFontFace(face)

	productName := truncateText(info.Name, 35)
	textY := textAreaY + 45
	dc.DrawStringAnchored(productName, float64(ogWidth)/2, textY, 0.5, 0.5)

	// Build the second line text based on available variants
	var secondLine string
	switch {
	case info.StyleCount > 1 && info.SizeCount > 1:
		secondLine = fmt.Sprintf("%d Colors • %d Sizes • %s", info.StyleCount, info.SizeCount, info.PriceRange)
	case info.StyleCount > 1:
		secondLine = fmt.Sprintf("%d Colors • %s", info.StyleCount, info.PriceRange)
	case info.SizeCount > 1:
		secondLine = fmt.Sprintf("%d Sizes • %s", info.SizeCount, info.PriceRange)
	default:
		secondLine = fmt.Sprintf("%s • Shop Now", info.PriceRange)
	}

	// Draw variant info line
	face = truetype.NewFace(font, &truetype.Options{Size: 28})
	dc.SetFontFace(face)
	textY += 45
	dc.DrawStringAnchored(secondLine, float64(ogWidth)/2, textY, 0.5, 0.5)

	// Encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, dc.Image()); err != nil {
		return nil, fmt.Errorf("encode final image: %w", err)
	}

	return buf.Bytes(), nil
}
