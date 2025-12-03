package ogimage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateMultiVariantOGImageWithAI(t *testing.T) {
	// Skip if no API key is configured
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping AI image generation test")
	}

	// Create temp output directory
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test-ai-og.png")

	info := MultiVariantInfo{
		Name:       "Test Armadillo Lizard",
		StyleCount: 3,
		PriceRange: "$6.00 - $24.00",
		ImagePaths: []string{
			"testdata/variant1.jpg",
			"testdata/variant2.jpg",
			"testdata/variant3.jpg",
		},
		StyleNames: []string{"Amethyst", "Aurora", "Berry"},
	}

	generator := NewAIGenerator(apiKey)
	err := generator.GenerateMultiVariantOGImage(info, outputPath)
	if err != nil {
		t.Fatalf("AI image generation failed: %v", err)
	}

	// Verify output file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("Expected output file to be created")
	}

	// Verify file is not empty
	fileInfo, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("Failed to stat output file: %v", err)
	}
	if fileInfo.Size() == 0 {
		t.Fatal("Output file is empty")
	}
}

func TestAIGeneratorFallsBackOnError(t *testing.T) {
	// Test with invalid API key to trigger fallback
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test-fallback-og.png")

	info := MultiVariantInfo{
		Name:       "Test Product",
		StyleCount: 2,
		PriceRange: "$10.00",
		ImagePaths: []string{},
		StyleNames: []string{},
	}

	generator := NewAIGenerator("invalid-key")
	err := generator.GenerateMultiVariantOGImage(info, outputPath)

	// Should not error - should fall back to grid method
	if err != nil {
		t.Fatalf("Expected fallback to succeed, got error: %v", err)
	}
}

func TestNewAIGenerator(t *testing.T) {
	generator := NewAIGenerator("test-api-key")
	if generator == nil {
		t.Fatal("Expected non-nil generator")
	}
	if generator.apiKey != "test-api-key" {
		t.Errorf("Expected apiKey to be 'test-api-key', got '%s'", generator.apiKey)
	}
}
