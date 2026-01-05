package importer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ImageDownloader downloads images from URLs and saves them locally
type ImageDownloader struct {
	client    *http.Client
	outputDir string
}

// DownloadedImage represents a successfully downloaded image
type DownloadedImage struct {
	OriginalURL string
	Filename    string
	FilePath    string
}

// NewImageDownloader creates a new image downloader
func NewImageDownloader(outputDir string) *ImageDownloader {
	return &ImageDownloader{
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		outputDir: outputDir,
	}
}

// DownloadImages downloads multiple images and returns the results
func (d *ImageDownloader) DownloadImages(ctx context.Context, imageURLs []string, productID string) ([]DownloadedImage, error) {
	// Ensure output directory exists
	if err := os.MkdirAll(d.outputDir, 0755); err != nil {
		return nil, fmt.Errorf("create output directory: %w", err)
	}

	var downloaded []DownloadedImage

	for i, url := range imageURLs {
		if url == "" {
			continue
		}

		// Rate limit - don't hammer the server
		if i > 0 {
			select {
			case <-time.After(500 * time.Millisecond):
			case <-ctx.Done():
				return downloaded, ctx.Err()
			}
		}

		img, err := d.downloadImage(ctx, url, productID, i)
		if err != nil {
			slog.Error("failed to download image", "error", err, "url", url)
			continue
		}

		downloaded = append(downloaded, *img)
		slog.Debug("downloaded image", "url", url, "filename", img.Filename)
	}

	return downloaded, nil
}

// downloadImage downloads a single image
func (d *ImageDownloader) downloadImage(ctx context.Context, imageURL string, productID string, index int) (*DownloadedImage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers to look like a browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")
	req.Header.Set("Referer", "https://cults3d.com/")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// Determine file extension from Content-Type or URL
	ext := d.getExtension(imageURL, resp.Header.Get("Content-Type"))

	// Generate deterministic filename based on URL hash
	// This ensures the same URL always produces the same filename, preventing duplicate files
	urlHash := sha256.Sum256([]byte(imageURL))
	hashStr := hex.EncodeToString(urlHash[:8]) // Use first 8 bytes = 16 hex chars
	filename := fmt.Sprintf("%s_%s%s", productID, hashStr, ext)
	filePath := filepath.Join(d.outputDir, filename)

	// Create destination file
	dst, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("create file: %w", err)
	}
	defer dst.Close()

	// Copy the image data
	_, err = io.Copy(dst, resp.Body)
	if err != nil {
		os.Remove(filePath) // Clean up on error
		return nil, fmt.Errorf("write file: %w", err)
	}

	return &DownloadedImage{
		OriginalURL: imageURL,
		Filename:    filename,
		FilePath:    filePath,
	}, nil
}

// getExtension determines the file extension
func (d *ImageDownloader) getExtension(url, contentType string) string {
	// Try to get from Content-Type first
	switch {
	case strings.Contains(contentType, "jpeg") || strings.Contains(contentType, "jpg"):
		return ".jpg"
	case strings.Contains(contentType, "png"):
		return ".png"
	case strings.Contains(contentType, "webp"):
		return ".webp"
	case strings.Contains(contentType, "gif"):
		return ".gif"
	}

	// Fall back to URL extension
	urlLower := strings.ToLower(url)
	switch {
	case strings.Contains(urlLower, ".jpg") || strings.Contains(urlLower, ".jpeg"):
		return ".jpg"
	case strings.Contains(urlLower, ".png"):
		return ".png"
	case strings.Contains(urlLower, ".webp"):
		return ".webp"
	case strings.Contains(urlLower, ".gif"):
		return ".gif"
	}

	// Default to jpg
	return ".jpg"
}
