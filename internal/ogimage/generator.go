package ogimage

import (
	"fmt"
	"image/png"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font/gofont/goregular"
)

type ProductInfo struct {
	Name         string
	CategoryName string
	ImagePath    string
}

// GenerateOGImage creates an Open Graph image with product photo and text overlay
func GenerateOGImage(product ProductInfo, outputPath string) error {
	// Load product image
	productImg, err := gg.LoadImage(product.ImagePath)
	if err != nil {
		slog.Error("failed to load product image", "error", err, "path", product.ImagePath)
		return fmt.Errorf("load product image: %w", err)
	}

	// Use original image dimensions - NO RESIZE! Let Facebook handle cropping
	dc := gg.NewContextForImage(productImg)
	imgWidth := dc.Width()
	imgHeight := dc.Height()

	// Add semi-transparent bar ONLY behind text (bottom 150px)
	textAreaHeight := 150
	textAreaY := imgHeight - textAreaHeight
	dc.SetRGBA(0, 0, 0, 0.75) // 75% opaque black bar
	dc.DrawRectangle(0, float64(textAreaY), float64(imgWidth), float64(textAreaHeight))
	dc.Fill()

	// Add text overlays within the bar area
	font, err := truetype.Parse(goregular.TTF)
	if err != nil {
		slog.Error("failed to parse font", "error", err)
		return fmt.Errorf("parse font: %w", err)
	}

	// Draw product name (large, readable)
	dc.SetRGB(1, 1, 1)
	face := truetype.NewFace(font, &truetype.Options{Size: 48})
	dc.SetFontFace(face)

	productName := truncateText(product.Name, 30)
	textY := float64(textAreaY) + 50
	dc.DrawStringAnchored(productName, float64(imgWidth)/2, textY, 0.5, 0.5)

	// Draw category badge and CTA
	face = truetype.NewFace(font, &truetype.Options{Size: 28})
	dc.SetFontFace(face)

	categoryBadge := fmt.Sprintf("%s Collection", product.CategoryName)
	ctaText := getCTAText(product.CategoryName, product.Name)
	secondLine := fmt.Sprintf("%s • %s", categoryBadge, ctaText)

	textY += 50
	dc.DrawStringAnchored(secondLine, float64(imgWidth)/2, textY, 0.5, 0.5)

	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		slog.Error("failed to create output directory", "error", err, "dir", outputDir)
		return fmt.Errorf("create output dir: %w", err)
	}

	// Save image
	file, err := os.Create(outputPath)
	if err != nil {
		slog.Error("failed to create output file", "error", err, "path", outputPath)
		return fmt.Errorf("create output file: %w", err)
	}
	defer file.Close()

	if err := png.Encode(file, dc.Image()); err != nil {
		slog.Error("failed to encode PNG", "error", err)
		return fmt.Errorf("encode PNG: %w", err)
	}

	slog.Debug("generated OG image", "product", product.Name, "output", outputPath)
	return nil
}

// truncateText truncates text to maxLength characters
func truncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength-3] + "..."
}

// getCTAText returns appropriate CTA text based on product category
func getCTAText(category, name string) string {
	categoryLower := strings.ToLower(category)
	nameLower := strings.ToLower(name)

	switch {
	case strings.Contains(nameLower, "articulated") || strings.Contains(nameLower, "flexi"):
		return "Customize Yours"
	case strings.Contains(categoryLower, "dinosaur"):
		return "Choose Your Colors"
	case strings.Contains(categoryLower, "custom"):
		return "Pick Your Colors"
	default:
		return "Shop Now"
	}
}
