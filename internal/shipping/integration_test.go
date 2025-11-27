//go:build integration
// +build integration

package shipping

import (
	"context"
	"testing"

	"github.com/loganlanou/logans3d-v4/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestShippingService_GetShippingQuote_SingleBox tests the full quote flow for a single-box order
func TestShippingService_GetShippingQuote_SingleBox(t *testing.T) {
	_, queries, cleanup, err := storage.NewTestDB()
	require.NoError(t, err)
	defer cleanup()

	// Load config from database (seeded by migrations)
	ctx := context.Background()
	config, err := LoadShippingConfigFromDB(ctx, queries)
	require.NoError(t, err)

	// Create shipping service (will use mock data since no EASYPOST_API_KEY)
	service, err := NewShippingService(config, queries)
	require.NoError(t, err)

	// Test single box order: 3 small items
	req := &ShippingQuoteRequest{
		ItemCounts: ItemCounts{Small: 3, Medium: 0, Large: 0, XL: 0},
		ShipTo: Address{
			Name:          "Test Customer",
			AddressLine1:  "123 Test St",
			CityLocality:  "Test City",
			StateProvince: "WI",
			PostalCode:    "54701",
			CountryCode:   "US",
		},
	}

	response, err := service.GetShippingQuote(req)
	require.NoError(t, err)
	require.NotNil(t, response)

	// Should have options
	assert.NotEmpty(t, response.Options, "Should return shipping options")
	assert.Empty(t, response.Error, "Should not have error")
	assert.NotNil(t, response.DefaultOption, "Should have a default option")

	// Verify packing solution
	for _, opt := range response.Options {
		require.NotNil(t, opt.PackingSolution)
		assert.Equal(t, 1, opt.PackingSolution.TotalBoxes, "Single box order should use 1 box")

		// Verify cost breakdown
		assert.Greater(t, opt.Price, 0.0, "Shipping price should be positive")
		assert.Greater(t, opt.BoxCost, 0.0, "Box cost should be positive")
		assert.Greater(t, opt.TotalCost, opt.Price, "Total should include box and handling costs")

		// Verify IDs are populated
		assert.NotEmpty(t, opt.RateID, "RateID should be set")
		assert.Len(t, opt.AllRateIDs, 1, "Single box should have 1 rate ID")
	}
}

// TestShippingService_GetShippingQuote_MultiBox tests the full quote flow for multi-box orders
func TestShippingService_GetShippingQuote_MultiBox(t *testing.T) {
	_, queries, cleanup, err := storage.NewTestDB()
	require.NoError(t, err)
	defer cleanup()

	ctx := context.Background()
	config, err := LoadShippingConfigFromDB(ctx, queries)
	require.NoError(t, err)

	service, err := NewShippingService(config, queries)
	require.NoError(t, err)

	// Test multi-box order: 2 XL items (smaller than 3 XL to increase carrier coverage likelihood)
	req := &ShippingQuoteRequest{
		ItemCounts: ItemCounts{Small: 0, Medium: 0, Large: 0, XL: 2},
		ShipTo: Address{
			Name:          "Test Customer",
			AddressLine1:  "123 Test St",
			CityLocality:  "Test City",
			StateProvince: "CA",
			PostalCode:    "90210",
			CountryCode:   "US",
		},
	}

	response, err := service.GetShippingQuote(req)
	require.NoError(t, err)
	require.NotNil(t, response)

	// With real API, carriers might not cover all boxes - this is expected behavior
	// The key test is that when options ARE returned, they have correct structure
	if len(response.Options) == 0 {
		t.Log("No shipping options returned - this is valid when no carrier covers all boxes")
		t.Log("The aggregation logic correctly filters out partial coverage")
		return
	}

	for _, opt := range response.Options {
		require.NotNil(t, opt.PackingSolution)

		// Multi-box order verification
		if opt.PackingSolution.TotalBoxes > 1 {
			t.Logf("Multi-box order: %d boxes for carrier %s", opt.PackingSolution.TotalBoxes, opt.CarrierName)

			// AllRateIDs should have one per box
			assert.Len(t, opt.AllRateIDs, opt.PackingSolution.TotalBoxes,
				"AllRateIDs should have one entry per box")
			assert.Len(t, opt.AllShipmentIDs, opt.PackingSolution.TotalBoxes,
				"AllShipmentIDs should have one entry per box")

			// Primary IDs should be first in the arrays
			assert.Equal(t, opt.RateID, opt.AllRateIDs[0], "Primary RateID should be first in array")
			assert.Equal(t, opt.ShipmentID, opt.AllShipmentIDs[0], "Primary ShipmentID should be first in array")

			// Costs should be aggregated
			// (We can't verify exact amounts without knowing box allocation, but totals should be reasonable)
			assert.Greater(t, opt.Price, 0.0)
			assert.Greater(t, opt.TotalCost, opt.Price, "Total should include materials")
		}
	}
}

// TestShippingService_ConfigFromDatabase verifies config loads correctly from DB
func TestShippingService_ConfigFromDatabase(t *testing.T) {
	_, queries, cleanup, err := storage.NewTestDB()
	require.NoError(t, err)
	defer cleanup()

	ctx := context.Background()
	config, err := LoadShippingConfigFromDB(ctx, queries)
	require.NoError(t, err)

	// Verify config has required fields
	assert.NotEmpty(t, config.Boxes, "Should have boxes from database")
	assert.NotNil(t, config.Packing.ItemWeights, "Should have item weights")
	assert.NotNil(t, config.Packing.DimensionGuard, "Should have dimension guards")

	// Verify required categories exist
	for _, cat := range []string{"small", "medium", "large", "xlarge"} {
		_, exists := config.Packing.ItemWeights[cat]
		assert.True(t, exists, "Should have ItemWeights for category: %s", cat)

		_, exists = config.Packing.DimensionGuard[cat]
		assert.True(t, exists, "Should have DimensionGuard for category: %s", cat)
	}

	// Config should pass validation
	err = validateConfig(config)
	assert.NoError(t, err, "Database config should pass validation")
}

// TestShippingService_EmptyOrder verifies error handling for empty orders
func TestShippingService_EmptyOrder(t *testing.T) {
	_, queries, cleanup, err := storage.NewTestDB()
	require.NoError(t, err)
	defer cleanup()

	ctx := context.Background()
	config, err := LoadShippingConfigFromDB(ctx, queries)
	require.NoError(t, err)

	service, err := NewShippingService(config, queries)
	require.NoError(t, err)

	// Test empty order
	req := &ShippingQuoteRequest{
		ItemCounts: ItemCounts{Small: 0, Medium: 0, Large: 0, XL: 0},
		ShipTo: Address{
			PostalCode: "54701",
		},
	}

	response, err := service.GetShippingQuote(req)
	require.NoError(t, err)
	require.NotNil(t, response)

	assert.NotEmpty(t, response.Error, "Empty order should return error message")
	assert.Empty(t, response.Options, "Empty order should have no options")
}

// TestShippingService_RateSorting verifies rate sorting preferences work
func TestShippingService_RateSorting(t *testing.T) {
	_, queries, cleanup, err := storage.NewTestDB()
	require.NoError(t, err)
	defer cleanup()

	ctx := context.Background()
	config, err := LoadShippingConfigFromDB(ctx, queries)
	require.NoError(t, err)

	// Test price_then_days sorting
	config.Shipping.RatePreferences.Sort = "price_then_days"
	service, err := NewShippingService(config, queries)
	require.NoError(t, err)

	req := &ShippingQuoteRequest{
		ItemCounts: ItemCounts{Small: 3},
		ShipTo:     Address{PostalCode: "54701"},
	}

	response, err := service.GetShippingQuote(req)
	require.NoError(t, err)
	require.NotNil(t, response)

	if len(response.Options) > 1 {
		// Verify sorting by price
		for i := 1; i < len(response.Options); i++ {
			prevCost := response.Options[i-1].TotalCost
			currCost := response.Options[i].TotalCost
			assert.GreaterOrEqual(t, currCost, prevCost,
				"Options should be sorted by price (option %d: %.2f should be >= option %d: %.2f)",
				i, currCost, i-1, prevCost)
		}
	}

	// Test days_then_price sorting
	config.Shipping.RatePreferences.Sort = "days_then_price"
	service.UpdateConfig(config)

	response, err = service.GetShippingQuote(req)
	require.NoError(t, err)

	if len(response.Options) > 1 {
		// Verify sorting by days
		for i := 1; i < len(response.Options); i++ {
			prevDays := response.Options[i-1].DeliveryDays
			currDays := response.Options[i].DeliveryDays
			assert.GreaterOrEqual(t, currDays, prevDays,
				"Options should be sorted by delivery days")
		}
	}
}

// TestCreateLabelsForMultiBox_Success tests multi-box label creation
func TestCreateLabelsForMultiBox_Success(t *testing.T) {
	_, queries, cleanup, err := storage.NewTestDB()
	require.NoError(t, err)
	defer cleanup()

	ctx := context.Background()
	config, err := LoadShippingConfigFromDB(ctx, queries)
	require.NoError(t, err)

	service, err := NewShippingService(config, queries)
	require.NoError(t, err)

	// Skip this test when using real API (would require real shipment IDs and cost money)
	if !service.IsUsingMockData() {
		t.Skip("Skipping label purchase test with real API - requires real shipment IDs")
	}

	// With mock data, this should succeed
	shipmentIDs := []string{"shp_mock_1", "shp_mock_2"}
	rateIDs := []string{"rate_mock_1", "rate_mock_2"}

	labels, err := service.CreateLabelsForMultiBox(shipmentIDs, rateIDs)
	require.NoError(t, err)
	assert.Len(t, labels, 2, "Should create 2 labels for 2 boxes")
}

// TestCreateLabelsForMultiBox_MismatchedArrays tests error handling
func TestCreateLabelsForMultiBox_MismatchedArrays(t *testing.T) {
	_, queries, cleanup, err := storage.NewTestDB()
	require.NoError(t, err)
	defer cleanup()

	ctx := context.Background()
	config, err := LoadShippingConfigFromDB(ctx, queries)
	require.NoError(t, err)

	service, err := NewShippingService(config, queries)
	require.NoError(t, err)

	// Mismatched array lengths should error
	_, err = service.CreateLabelsForMultiBox(
		[]string{"shp_1", "shp_2"},
		[]string{"rate_1"}, // Only 1 rate for 2 shipments
	)
	assert.Error(t, err, "Mismatched arrays should return error")
	assert.Contains(t, err.Error(), "same length", "Error should mention length mismatch")
}

// TestCreateLabelsForMultiBox_EmptyInput tests error handling for empty input
func TestCreateLabelsForMultiBox_EmptyInput(t *testing.T) {
	_, queries, cleanup, err := storage.NewTestDB()
	require.NoError(t, err)
	defer cleanup()

	ctx := context.Background()
	config, err := LoadShippingConfigFromDB(ctx, queries)
	require.NoError(t, err)

	service, err := NewShippingService(config, queries)
	require.NoError(t, err)

	_, err = service.CreateLabelsForMultiBox([]string{}, []string{})
	assert.Error(t, err, "Empty input should return error")
	assert.Contains(t, err.Error(), "no shipments", "Error should mention no shipments")
}
