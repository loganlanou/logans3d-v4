package service

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"

	"github.com/loganlanou/logans3d-v4/storage/db"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"github.com/jung-kurt/gofpdf"
	qrcode "github.com/skip2/go-qrcode"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/math/fixed"
)

const (
	templatePath   = "public/images/gift-certificates/template.png"
	outputDir      = "data/gift-certificates" // Private directory, not publicly accessible
	montserratPath = "public/fonts/Montserrat-Bold.ttf"

	// Default base URL for production
	defaultBaseURL = "https://www.logans3dcreations.com"

	// Layout positions (for 1024x1024 template)
	// Adjusted for Logan's 3D template design
	// Orange ribbon vertical center is around Y=460, text baseline needs to be lower
	amountY        = 540 // Y position for amount baseline - centered on orange ribbon
	amountX        = 512 // Centered X for amount
	dateY          = 900 // Y position for date
	dateX          = 80  // X position for date
	refY           = 940 // Y position for reference
	refX           = 80  // X position for reference
	qrBoxX         = 780 // X position of white box left edge
	qrBoxY         = 770 // Y position of white box top edge
	qrBoxW         = 220 // White box width
	qrBoxH         = 220 // White box height
	qrSize         = 190 // QR code size (larger to fill white box better)
	idY            = 765 // Y position for partial ID (above white box with small margin)
	idX            = 890 // X position for partial ID (centered above QR box)
	disclaimerY    = 990 // Y position for disclaimer at bottom
	disclaimerX    = 80  // Left-aligned X for disclaimer (same as date/reference)
	disclaimerText = "Treat like cash. Not responsible for lost or stolen certificates. Only original is valid."
)

// getBaseURL returns the base URL from environment or default
func getBaseURL() string {
	if url := os.Getenv("BASE_URL"); url != "" {
		return url
	}
	return defaultBaseURL
}

// GenerateGiftCertificateImages generates PNG and PDF images for a gift certificate
func GenerateGiftCertificateImages(cert db.GiftCertificate) (pngPath, pdfPath string, err error) {
	baseURL := getBaseURL()

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Load template image
	templateFile, err := os.Open(templatePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to open template: %w", err)
	}
	defer templateFile.Close()

	templateImg, err := png.Decode(templateFile)
	if err != nil {
		return "", "", fmt.Errorf("failed to decode template: %w", err)
	}

	// Create a new RGBA image from the template
	bounds := templateImg.Bounds()
	img := image.NewRGBA(bounds)
	draw.Draw(img, bounds, templateImg, image.Point{}, draw.Src)

	// Load Montserrat Bold font for amount
	montserratData, err := os.ReadFile(montserratPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read Montserrat font: %w", err)
	}
	montserratFont, err := truetype.Parse(montserratData)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse Montserrat font: %w", err)
	}

	// Load regular font for date/reference
	regularFont, err := truetype.Parse(goregular.TTF)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse regular font: %w", err)
	}

	// Draw amount on the ribbon (large, Montserrat Bold, white)
	amountText := fmt.Sprintf("$%.2f", cert.Amount)
	drawCenteredText(img, amountText, amountX, amountY, montserratFont, 144, color.White)

	// Draw date issued
	if cert.IssuedAt.Valid {
		dateText := fmt.Sprintf("Date Issued: %s", cert.IssuedAt.Time.Format("01/02/2006"))
		drawText(img, dateText, dateX, dateY, regularFont, 24, color.White)
	}

	// Draw reference if present
	if cert.Reference.Valid && cert.Reference.String != "" {
		refText := fmt.Sprintf("Reference: %s", cert.Reference.String)
		drawText(img, refText, refX, refY, regularFont, 24, color.White)
	}

	// Draw partial ID above QR code for fallback identification
	// Format: first 4 chars ... last 4 chars (e.g., "8616...2800")
	partialID := formatPartialID(cert.ID)
	drawCenteredText(img, partialID, idX, idY, regularFont, 18, color.White)

	// Generate QR code with base URL
	verifyURL := fmt.Sprintf("%s/gift-certificates/verify/%s", baseURL, cert.ID)
	qr, err := qrcode.New(verifyURL, qrcode.Medium)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate QR code: %w", err)
	}
	qrImg := qr.Image(qrSize)

	// Draw QR code centered in the white box
	qrBounds := qrImg.Bounds()
	// Center the QR code within the white box
	qrX := qrBoxX + (qrBoxW-qrBounds.Dx())/2
	qrY := qrBoxY + (qrBoxH-qrBounds.Dy())/2
	draw.Draw(img, image.Rect(qrX, qrY, qrX+qrBounds.Dx(), qrY+qrBounds.Dy()), qrImg, image.Point{}, draw.Over)

	// Draw disclaimer at the bottom (small, left-aligned with date/reference, muted color)
	disclaimerColor := color.RGBA{180, 180, 180, 255} // Light gray
	drawText(img, disclaimerText, disclaimerX, disclaimerY, regularFont, 14, disclaimerColor)

	// Save PNG
	pngPath = filepath.Join(outputDir, fmt.Sprintf("%s.png", cert.ID))
	pngFile, err := os.Create(pngPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to create PNG file: %w", err)
	}
	defer pngFile.Close()

	if err := png.Encode(pngFile, img); err != nil {
		return "", "", fmt.Errorf("failed to encode PNG: %w", err)
	}

	// Generate PDF (simple wrapper around PNG for now)
	pdfPath, err = generatePDF(pngPath, cert.ID)
	if err != nil {
		// PDF generation failed, but PNG succeeded
		// Return PNG path only
		return pngPath, "", nil
	}

	return pngPath, pdfPath, nil
}

// drawText draws text at the specified position
func drawText(img *image.RGBA, text string, x, y int, f *truetype.Font, size float64, c color.Color) {
	ctx := freetype.NewContext()
	ctx.SetDPI(72)
	ctx.SetFont(f)
	ctx.SetFontSize(size)
	ctx.SetClip(img.Bounds())
	ctx.SetDst(img)
	ctx.SetSrc(image.NewUniform(c))
	ctx.SetHinting(font.HintingFull)

	pt := freetype.Pt(x, y)
	_, _ = ctx.DrawString(text, pt)
}

// drawCenteredText draws text centered at the specified X position
func drawCenteredText(img *image.RGBA, text string, centerX, y int, f *truetype.Font, size float64, c color.Color) {
	// Calculate text width
	face := truetype.NewFace(f, &truetype.Options{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})

	var width fixed.Int26_6
	for _, r := range text {
		advance, ok := face.GlyphAdvance(r)
		if ok {
			width += advance
		}
	}

	// Calculate starting X position
	x := centerX - width.Round()/2

	drawText(img, text, x, y, f, size, c)
}

// formatPartialID returns a truncated ID showing first 4 and last 4 characters
// This provides fallback identification if QR code is damaged while preventing ID guessing
func formatPartialID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:4] + "..." + id[len(id)-4:]
}

// generatePDF creates a PDF containing the PNG image
func generatePDF(pngPath, certID string) (string, error) {
	pdf := gofpdf.New("P", "mm", "Letter", "")
	pdf.AddPage()

	// Letter size is 215.9mm x 279.4mm
	// Center the 1024x1024 image (which we'll scale to fit nicely)
	// Use 150mm width for a good print size
	imgWidth := 150.0
	imgHeight := 150.0 // Square image

	// Center horizontally: (215.9 - 150) / 2 = ~33mm
	x := (215.9 - imgWidth) / 2
	// Position from top with some margin
	y := 30.0

	// Register and add the PNG image
	pdf.RegisterImageOptions(pngPath, gofpdf.ImageOptions{ImageType: "PNG"})
	pdf.ImageOptions(pngPath, x, y, imgWidth, imgHeight, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")

	// Save PDF
	pdfPath := filepath.Join(outputDir, fmt.Sprintf("%s.pdf", certID))
	err := pdf.OutputFileAndClose(pdfPath)
	if err != nil {
		return "", fmt.Errorf("failed to create PDF: %w", err)
	}

	return pdfPath, nil
}
