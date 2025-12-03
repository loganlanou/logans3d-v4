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

// VariantInfo contains variant-specific details for OG image generation
type VariantInfo struct {
	StyleName  string // e.g., "Berry", "Rainbow"
	SizeName   string // e.g., "Medium", "Large"
	PriceCents int64  // Final price including adjustments
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

	// Calculate scale factor based on image size (reference: 1200px width for standard OG)
	scaleFactor := float64(imgWidth) / 1200.0
	if scaleFactor < 1.0 {
		scaleFactor = 1.0 // Don't scale down for small images
	}

	// Add semi-transparent bar ONLY behind text (scaled to image size)
	textAreaHeight := int(150 * scaleFactor)
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

	// Draw product name (large, readable) - scaled font size
	dc.SetRGB(1, 1, 1)
	titleFontSize := 48 * scaleFactor
	face := truetype.NewFace(font, &truetype.Options{Size: titleFontSize})
	dc.SetFontFace(face)

	productName := truncateText(product.Name, 30)
	textY := float64(textAreaY) + (50 * scaleFactor)
	dc.DrawStringAnchored(productName, float64(imgWidth)/2, textY, 0.5, 0.5)

	// Draw category badge and CTA - scaled font size
	subtitleFontSize := 28 * scaleFactor
	face = truetype.NewFace(font, &truetype.Options{Size: subtitleFontSize})
	dc.SetFontFace(face)

	categoryBadge := fmt.Sprintf("%s Collection", product.CategoryName)
	ctaText := getCTAText(product.CategoryName, product.Name)
	secondLine := fmt.Sprintf("%s • %s", categoryBadge, ctaText)

	textY += 50 * scaleFactor
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

// GenerateVariantOGImage creates an Open Graph image for a specific variant
func GenerateVariantOGImage(product ProductInfo, variant VariantInfo, outputPath string) error {
	// Load product image (style-specific image)
	productImg, err := gg.LoadImage(product.ImagePath)
	if err != nil {
		slog.Error("failed to load variant image", "error", err, "path", product.ImagePath)
		return fmt.Errorf("load variant image: %w", err)
	}

	// Use original image dimensions
	dc := gg.NewContextForImage(productImg)
	imgWidth := dc.Width()
	imgHeight := dc.Height()

	// Calculate scale factor based on image size (reference: 1200px width for standard OG)
	scaleFactor := float64(imgWidth) / 1200.0
	if scaleFactor < 1.0 {
		scaleFactor = 1.0 // Don't scale down for small images
	}

	// Add semi-transparent bar ONLY behind text (scaled to image size)
	textAreaHeight := int(150 * scaleFactor)
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

	// Draw product name with variant (large, readable) - scaled font size
	dc.SetRGB(1, 1, 1)
	titleFontSize := 44 * scaleFactor
	face := truetype.NewFace(font, &truetype.Options{Size: titleFontSize})
	dc.SetFontFace(face)

	// Format: "Product Name - Color, Size"
	variantTitle := fmt.Sprintf("%s - %s, %s", truncateText(product.Name, 20), variant.StyleName, variant.SizeName)
	textY := float64(textAreaY) + (50 * scaleFactor)
	dc.DrawStringAnchored(variantTitle, float64(imgWidth)/2, textY, 0.5, 0.5)

	// Draw price and CTA - scaled font size
	subtitleFontSize := 28 * scaleFactor
	face = truetype.NewFace(font, &truetype.Options{Size: subtitleFontSize})
	dc.SetFontFace(face)

	priceStr := fmt.Sprintf("$%.2f", float64(variant.PriceCents)/100)
	secondLine := fmt.Sprintf("%s • Shop Now", priceStr)

	textY += 50 * scaleFactor
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

	slog.Debug("generated variant OG image", "product", product.Name, "variant", variantTitle, "output", outputPath)
	return nil
}

// MultiVariantInfo contains data for multi-variant OG image generation
type MultiVariantInfo struct {
	Name       string   // Product name
	StyleCount int      // Total number of styles/colors (could be 24)
	SizeCount  int      // Total number of sizes
	PriceRange string   // "$5.00 - $12.00" or "$5.00" if same
	ImagePaths []string // Up to 4 image paths for 2x2 grid
	StyleNames []string // Names of styles shown in grid
}

// GenerateMultiVariantOGImage creates an OG image with 2x2 grid of variant images
func GenerateMultiVariantOGImage(info MultiVariantInfo, outputPath string) error {
	// Standard OG image size
	const width = 1200
	const height = 630

	dc := gg.NewContext(width, height)

	// Fill background with dark gray
	dc.SetRGB(0.15, 0.15, 0.15)
	dc.Clear()

	// Calculate grid dimensions (2x2)
	gridSize := 2
	cellWidth := width / gridSize
	cellHeight := (height - 120) / gridSize // Leave space for text at bottom

	// Load and draw up to 4 images in a grid
	for i := 0; i < 4 && i < len(info.ImagePaths); i++ {
		img, err := gg.LoadImage(info.ImagePaths[i])
		if err != nil {
			slog.Debug("failed to load grid image", "error", err, "path", info.ImagePaths[i], "index", i)
			continue
		}

		// Calculate position in grid
		row := i / gridSize
		col := i % gridSize
		cellX := col * cellWidth
		cellY := row * cellHeight

		// Calculate scale to fit image in cell while maintaining aspect ratio
		imgWidth := img.Bounds().Dx()
		imgHeight := img.Bounds().Dy()
		scaleX := float64(cellWidth) / float64(imgWidth)
		scaleY := float64(cellHeight) / float64(imgHeight)
		scale := scaleX
		if scaleY < scaleX {
			scale = scaleY
		}

		// Calculate centered position within cell
		scaledWidth := float64(imgWidth) * scale
		scaledHeight := float64(imgHeight) * scale
		offsetX := (float64(cellWidth) - scaledWidth) / 2
		offsetY := (float64(cellHeight) - scaledHeight) / 2

		// Draw scaled image
		dc.Push()
		dc.Translate(float64(cellX)+offsetX, float64(cellY)+offsetY)
		dc.Scale(scale, scale)
		dc.DrawImage(img, 0, 0)
		dc.Pop()
	}

	// Add semi-transparent bar at bottom for text
	textAreaHeight := 120
	textAreaY := height - textAreaHeight
	dc.SetRGBA(0, 0, 0, 0.85)
	dc.DrawRectangle(0, float64(textAreaY), float64(width), float64(textAreaHeight))
	dc.Fill()

	// Add text overlays
	font, err := truetype.Parse(goregular.TTF)
	if err != nil {
		slog.Error("failed to parse font", "error", err)
		return fmt.Errorf("parse font: %w", err)
	}

	// Draw product name (large)
	dc.SetRGB(1, 1, 1)
	face := truetype.NewFace(font, &truetype.Options{Size: 42})
	dc.SetFontFace(face)

	productName := truncateText(info.Name, 35)
	textY := float64(textAreaY) + 45
	dc.DrawStringAnchored(productName, float64(width)/2, textY, 0.5, 0.5)

	// Draw variant info line (colors, sizes, price)
	face = truetype.NewFace(font, &truetype.Options{Size: 28})
	dc.SetFontFace(face)

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

	textY += 45
	dc.DrawStringAnchored(secondLine, float64(width)/2, textY, 0.5, 0.5)

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

	slog.Debug("generated multi-variant OG image", "product", info.Name, "styles", info.StyleCount, "output", outputPath)
	return nil
}
