package utils

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/loganlanou/logans3d-v4/storage/db"
)

var skuPattern = regexp.MustCompile(`^[A-Z0-9-]+$`)

// GenerateSKU builds a catalog-wide SKU using uppercase codes and hyphens.
// Example: base "TREX", color "red", size "lg" -> "TREX-RED-LG".
func GenerateSKU(baseSKU, colorCode, sizeCode string) string {
	parts := []string{
		normalizeSegment(baseSKU),
		normalizeSegment(colorCode),
		normalizeSegment(sizeCode),
	}

	return strings.Trim(strings.Join(filterEmpty(parts), "-"), "-")
}

// ParseSKU splits a SKU into base, color, and size segments.
func ParseSKU(sku string) (string, string, string, error) {
	normalized := normalizeSegment(sku)
	segments := strings.Split(normalized, "-")
	if len(segments) < 3 {
		return "", "", "", fmt.Errorf("sku %s does not include color and size segments", sku)
	}

	base := strings.Join(segments[:len(segments)-2], "-")
	color := segments[len(segments)-2]
	size := segments[len(segments)-1]

	return base, color, size, nil
}

// ValidateSKU checks format and uniqueness at the database layer.
func ValidateSKU(ctx context.Context, queries *db.Queries, sku string) error {
	normalized := normalizeSegment(sku)
	if normalized == "" {
		return errors.New("sku is required")
	}

	if !skuPattern.MatchString(normalized) {
		return errors.New("sku may only contain letters, numbers, and hyphens")
	}

	exists, err := queries.CheckSkuExists(ctx, normalized)
	if err != nil {
		return fmt.Errorf("failed to validate sku: %w", err)
	}
	if exists > 0 {
		return fmt.Errorf("sku %s already exists", normalized)
	}

	return nil
}

func normalizeSegment(value string) string {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "_", "-")
	normalized = strings.ReplaceAll(normalized, " ", "-")
	normalized = strings.ReplaceAll(normalized, "--", "-")
	return normalized
}

func filterEmpty(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			out = append(out, v)
		}
	}
	return out
}
