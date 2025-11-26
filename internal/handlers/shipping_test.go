package handlers

import (
	"context"
	"database/sql"
	"testing"

	"github.com/loganlanou/logans3d-v4/storage/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ValidShippingCategories defines the expected shipping category values
// These must match the CASE statements in storage/queries/shipping.sql
var ValidShippingCategories = []string{"small", "medium", "large", "xlarge"}

// TestSizeChartShippingCategories verifies that all size charts have valid shipping categories
// The shipping calculator expects: small, medium, large, xlarge
// Invalid values (like "First" or "Priority") will cause items to fall through all CASE statements
func TestSizeChartShippingCategories(t *testing.T) {
	_, queries, cleanup := NewTestDB()
	defer cleanup()
	ctx := context.Background()

	// Get all size charts
	charts, err := queries.GetSizeCharts(ctx)
	require.NoError(t, err, "should be able to get size charts")

	// Verify each chart has a valid shipping category
	for _, chart := range charts {
		if chart.DefaultShippingClass.Valid && chart.DefaultShippingClass.String != "" {
			category := chart.DefaultShippingClass.String
			assert.Contains(t, ValidShippingCategories, category,
				"size chart %s has invalid shipping category %q, expected one of %v",
				chart.SizeID, category, ValidShippingCategories)
		}
	}
}

// TestShippingCategoryValues documents the valid shipping category values
// This test serves as documentation for the expected values
func TestShippingCategoryValues(t *testing.T) {
	// These are the only valid shipping categories
	// They map to box sizes in the shipping packer configuration
	validCategories := map[string]string{
		"small":  "Small items - First Class eligible",
		"medium": "Medium items - may need Priority",
		"large":  "Large items - Priority required",
		"xlarge": "Extra large items - Priority/special handling",
	}

	// Invalid categories that should NOT be used
	invalidCategories := []string{
		"First",    // Legacy value - use "small" or "medium" instead
		"Priority", // Legacy value - use "large" or "xlarge" instead
		"unknown",  // Fallback value when category is missing
	}

	// Document valid categories
	for category, description := range validCategories {
		assert.Contains(t, ValidShippingCategories, category,
			"category %q (%s) should be in valid categories", category, description)
	}

	// Ensure invalid categories are not in valid list
	for _, invalid := range invalidCategories {
		assert.NotContains(t, ValidShippingCategories, invalid,
			"category %q should NOT be a valid shipping category", invalid)
	}
}

// TestProductShippingCategory verifies products can have valid shipping categories set
func TestProductShippingCategory(t *testing.T) {
	_, queries, cleanup := NewTestDB()
	defer cleanup()
	ctx := context.Background()

	// Create a test product first
	product, err := queries.CreateProduct(ctx, db.CreateProductParams{
		ID:          "test-product-shipping",
		Name:        "Test Product",
		Slug:        "test-product-shipping",
		Description: sql.NullString{String: "Test product for shipping", Valid: true},
		PriceCents:  1000,
		WeightGrams: sql.NullInt64{Int64: 100, Valid: true},
	})
	require.NoError(t, err)

	// Update shipping category using the dedicated query
	updated, err := queries.UpdateProductShippingCategory(ctx, db.UpdateProductShippingCategoryParams{
		ID:               product.ID,
		ShippingCategory: sql.NullString{String: "small", Valid: true},
	})
	require.NoError(t, err)

	// Verify the shipping category was saved correctly
	assert.Equal(t, "small", updated.ShippingCategory.String)
	assert.Contains(t, ValidShippingCategories, updated.ShippingCategory.String)
}
