package shipping

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to create a mock rate
func mockRate(rateID, shipmentID, carrier, service string, price float64, days int) Rate {
	return Rate{
		RateID:          rateID,
		ShipmentID:      shipmentID,
		CarrierNickname: carrier,
		ServiceType:     service,
		ShippingAmount:  Amount{Amount: price, Currency: "usd"},
		DeliveryDays:    days,
	}
}

// Helper to create a mock box selection
func mockBoxSelection(boxCost, handlingCost float64) BoxSelection {
	return BoxSelection{
		Box:                  Box{SKU: "TEST-BOX", Name: "Test Box", L: 10, W: 8, H: 6},
		BoxCost:              boxCost,
		PackingMaterialsCost: handlingCost,
	}
}

func TestAggregateRates_SingleBox(t *testing.T) {
	boxRates := []BoxRatesResult{
		{
			BoxSelection: mockBoxSelection(0.50, 1.50),
			Rates: []Rate{
				mockRate("rate-usps", "shp-usps", "USPS", "Ground Advantage", 5.00, 5),
				mockRate("rate-ups", "shp-ups", "UPS", "Ground", 6.00, 4),
				mockRate("rate-fedex", "shp-fedex", "FedEx", "Ground", 5.50, 3),
			},
		},
	}

	solution := &PackingSolution{
		Boxes:      []BoxSelection{mockBoxSelection(0.50, 1.50)},
		TotalBoxes: 1,
	}

	options := AggregateRates(boxRates, solution, "price_then_days")

	require.Len(t, options, 3, "Should have 3 shipping options")

	// Verify costs are calculated correctly
	for _, opt := range options {
		expectedTotal := opt.Price + opt.BoxCost + opt.HandlingCost
		assert.Equal(t, expectedTotal, opt.TotalCost, "TotalCost should equal Price + BoxCost + HandlingCost")
		assert.Len(t, opt.AllRateIDs, 1, "Single box should have 1 rate ID")
		assert.Len(t, opt.AllShipmentIDs, 1, "Single box should have 1 shipment ID")
	}
}

func TestAggregateRates_MultiBox_AllCovered(t *testing.T) {
	t.Log("Bug regression test: Multi-box orders should aggregate costs across all boxes")

	boxRates := []BoxRatesResult{
		{
			BoxSelection: mockBoxSelection(0.50, 1.50),
			Rates: []Rate{
				mockRate("usps-box1", "shp-box1-usps", "USPS", "Ground", 5.00, 5),
				mockRate("ups-box1", "shp-box1-ups", "UPS", "Ground", 6.00, 4),
			},
		},
		{
			BoxSelection: mockBoxSelection(0.60, 1.50),
			Rates: []Rate{
				mockRate("usps-box2", "shp-box2-usps", "USPS", "Ground", 5.50, 6),
				mockRate("ups-box2", "shp-box2-ups", "UPS", "Ground", 6.50, 5),
			},
		},
	}

	solution := &PackingSolution{
		Boxes:      []BoxSelection{mockBoxSelection(0.50, 1.50), mockBoxSelection(0.60, 1.50)},
		TotalBoxes: 2,
	}

	options := AggregateRates(boxRates, solution, "price_then_days")

	require.Len(t, options, 2, "Should have 2 shipping options (USPS and UPS)")

	// Find USPS option and verify aggregation
	var uspsOpt *ShippingOption
	for i := range options {
		if options[i].CarrierName == "USPS" {
			uspsOpt = &options[i]
			break
		}
	}
	require.NotNil(t, uspsOpt, "Should have USPS option")

	// Verify USPS aggregation
	assert.Equal(t, 10.50, uspsOpt.Price, "Price should be sum: 5.00 + 5.50")
	assert.Equal(t, 1.10, uspsOpt.BoxCost, "BoxCost should be sum: 0.50 + 0.60")
	assert.Equal(t, 3.00, uspsOpt.HandlingCost, "HandlingCost should be sum: 1.50 + 1.50")
	assert.Equal(t, 14.60, uspsOpt.TotalCost, "TotalCost should be 10.50 + 1.10 + 3.00")
	assert.Len(t, uspsOpt.AllRateIDs, 2, "Should have 2 rate IDs for 2-box order")
	assert.Len(t, uspsOpt.AllShipmentIDs, 2, "Should have 2 shipment IDs for 2-box order")
}

func TestAggregateRates_MultiBox_PartialCoverage(t *testing.T) {
	t.Log("Bug regression test: Carrier missing rate for one box should be EXCLUDED entirely")

	boxRates := []BoxRatesResult{
		{
			BoxSelection: mockBoxSelection(0.50, 1.50),
			Rates: []Rate{
				mockRate("usps-box1", "shp-box1-usps", "USPS", "Ground", 5.00, 5),
				mockRate("ups-box1", "shp-box1-ups", "UPS", "Ground", 6.00, 4),
			},
		},
		{
			BoxSelection: mockBoxSelection(0.60, 1.50),
			Rates: []Rate{
				// UPS missing for box 2 - only USPS available
				mockRate("usps-box2", "shp-box2-usps", "USPS", "Ground", 5.50, 6),
			},
		},
	}

	solution := &PackingSolution{
		Boxes:      []BoxSelection{mockBoxSelection(0.50, 1.50), mockBoxSelection(0.60, 1.50)},
		TotalBoxes: 2,
	}

	options := AggregateRates(boxRates, solution, "price_then_days")

	// Should only have USPS since UPS doesn't cover all boxes
	require.Len(t, options, 1, "Should only have 1 option (UPS excluded)")
	assert.Equal(t, "USPS", options[0].CarrierName, "Only USPS should be available")
	assert.Equal(t, 10.50, options[0].Price, "Price should be sum: 5.00 + 5.50")
}

func TestAggregateRates_MultiBox_NoCoverage(t *testing.T) {
	t.Log("Bug regression test: No carrier covering all boxes should return empty options")

	boxRates := []BoxRatesResult{
		{
			BoxSelection: mockBoxSelection(0.50, 1.50),
			Rates: []Rate{
				mockRate("usps-box1", "shp-box1", "USPS", "Ground", 5.00, 5),
			},
		},
		{
			BoxSelection: mockBoxSelection(0.60, 1.50),
			Rates: []Rate{
				// Different carrier for box 2 - no carrier covers both
				mockRate("ups-box2", "shp-box2", "UPS", "Ground", 6.00, 4),
			},
		},
	}

	solution := &PackingSolution{
		Boxes:      []BoxSelection{mockBoxSelection(0.50, 1.50), mockBoxSelection(0.60, 1.50)},
		TotalBoxes: 2,
	}

	options := AggregateRates(boxRates, solution, "price_then_days")

	assert.Len(t, options, 0, "Should have no options when no carrier covers all boxes")
}

func TestAggregateRates_DeliveryDays_MaxAcrossBoxes(t *testing.T) {
	t.Log("Multi-box orders should use MAX delivery days since boxes ship in parallel")

	boxRates := []BoxRatesResult{
		{
			BoxSelection: mockBoxSelection(0.50, 1.50),
			Rates: []Rate{
				mockRate("usps-box1", "shp-box1", "USPS", "Ground", 5.00, 3), // 3 days
			},
		},
		{
			BoxSelection: mockBoxSelection(0.60, 1.50),
			Rates: []Rate{
				mockRate("usps-box2", "shp-box2", "USPS", "Ground", 5.50, 5), // 5 days
			},
		},
	}

	solution := &PackingSolution{
		Boxes:      []BoxSelection{mockBoxSelection(0.50, 1.50), mockBoxSelection(0.60, 1.50)},
		TotalBoxes: 2,
	}

	options := AggregateRates(boxRates, solution, "price_then_days")

	require.Len(t, options, 1)
	assert.Equal(t, 5, options[0].DeliveryDays, "Should use max delivery days (5, not 3)")
}

func TestAggregateRates_AllRateIDs_Populated(t *testing.T) {
	t.Log("Multi-box orders must track all rate IDs for multi-box label purchase")

	boxRates := []BoxRatesResult{
		{
			BoxSelection: mockBoxSelection(0.50, 1.50),
			Rates: []Rate{
				mockRate("rate-box1", "shp-box1", "USPS", "Ground", 5.00, 3),
			},
		},
		{
			BoxSelection: mockBoxSelection(0.60, 1.50),
			Rates: []Rate{
				mockRate("rate-box2", "shp-box2", "USPS", "Ground", 5.50, 5),
			},
		},
	}

	solution := &PackingSolution{
		Boxes:      []BoxSelection{mockBoxSelection(0.50, 1.50), mockBoxSelection(0.60, 1.50)},
		TotalBoxes: 2,
	}

	options := AggregateRates(boxRates, solution, "price_then_days")

	require.Len(t, options, 1)
	assert.Equal(t, []string{"rate-box1", "rate-box2"}, options[0].AllRateIDs,
		"AllRateIDs should contain rate IDs from both boxes")
	assert.Equal(t, []string{"shp-box1", "shp-box2"}, options[0].AllShipmentIDs,
		"AllShipmentIDs should contain shipment IDs from both boxes")

	// Primary IDs should be first box for backward compatibility
	assert.Equal(t, "rate-box1", options[0].RateID, "Primary RateID should be first box")
	assert.Equal(t, "shp-box1", options[0].ShipmentID, "Primary ShipmentID should be first box")
}

func TestAggregateRates_CostCorrectlySummed(t *testing.T) {
	t.Log("Bug regression test: TotalCost must be sum of all components across all boxes")

	boxRates := []BoxRatesResult{
		{
			BoxSelection: BoxSelection{
				Box:                  Box{SKU: "BOX-1"},
				BoxCost:              0.50,
				PackingMaterialsCost: 1.50,
			},
			Rates: []Rate{
				mockRate("rate-1", "shp-1", "USPS", "Ground", 5.00, 3),
			},
		},
		{
			BoxSelection: BoxSelection{
				Box:                  Box{SKU: "BOX-2"},
				BoxCost:              0.60,
				PackingMaterialsCost: 1.50,
			},
			Rates: []Rate{
				mockRate("rate-2", "shp-2", "USPS", "Ground", 5.50, 4),
			},
		},
	}

	solution := &PackingSolution{
		Boxes:      []BoxSelection{boxRates[0].BoxSelection, boxRates[1].BoxSelection},
		TotalBoxes: 2,
	}

	options := AggregateRates(boxRates, solution, "price_then_days")

	require.Len(t, options, 1)

	// Verify each component
	assert.Equal(t, 10.50, options[0].Price, "Price = 5.00 + 5.50")
	assert.Equal(t, 1.10, options[0].BoxCost, "BoxCost = 0.50 + 0.60")
	assert.Equal(t, 3.00, options[0].HandlingCost, "HandlingCost = 1.50 + 1.50")
	assert.Equal(t, 14.60, options[0].TotalCost, "TotalCost = 10.50 + 1.10 + 3.00")
}

func TestAggregateRates_EmptyInput(t *testing.T) {
	options := AggregateRates(nil, nil, "price_then_days")
	assert.Nil(t, options, "Empty input should return nil")

	options = AggregateRates([]BoxRatesResult{}, nil, "price_then_days")
	assert.Nil(t, options, "Empty slice should return nil")
}

func TestAggregateRates_BoxWithNoRates(t *testing.T) {
	boxRates := []BoxRatesResult{
		{
			BoxSelection: mockBoxSelection(0.50, 1.50),
			Rates: []Rate{
				mockRate("rate-1", "shp-1", "USPS", "Ground", 5.00, 3),
			},
		},
		{
			BoxSelection: mockBoxSelection(0.60, 1.50),
			Rates:        []Rate{}, // No rates for this box
		},
	}

	solution := &PackingSolution{
		Boxes:      []BoxSelection{mockBoxSelection(0.50, 1.50), mockBoxSelection(0.60, 1.50)},
		TotalBoxes: 2,
	}

	options := AggregateRates(boxRates, solution, "price_then_days")

	assert.Len(t, options, 0, "Should have no options when a box has no rates")
}

func TestSortShippingOptions(t *testing.T) {
	options := []ShippingOption{
		{CarrierName: "FedEx", TotalCost: 15.00, DeliveryDays: 2},
		{CarrierName: "USPS", TotalCost: 10.00, DeliveryDays: 5},
		{CarrierName: "UPS", TotalCost: 12.00, DeliveryDays: 3},
		{CarrierName: "DHL", TotalCost: 10.00, DeliveryDays: 4}, // Same price as USPS
	}

	t.Run("price_then_days", func(t *testing.T) {
		sorted := SortShippingOptions(options, "price_then_days")

		assert.Equal(t, "DHL", sorted[0].CarrierName, "DHL should be first (10.00, 4 days)")
		assert.Equal(t, "USPS", sorted[1].CarrierName, "USPS should be second (10.00, 5 days)")
		assert.Equal(t, "UPS", sorted[2].CarrierName, "UPS should be third (12.00)")
		assert.Equal(t, "FedEx", sorted[3].CarrierName, "FedEx should be last (15.00)")
	})

	t.Run("days_then_price", func(t *testing.T) {
		sorted := SortShippingOptions(options, "days_then_price")

		assert.Equal(t, "FedEx", sorted[0].CarrierName, "FedEx should be first (2 days)")
		assert.Equal(t, "UPS", sorted[1].CarrierName, "UPS should be second (3 days)")
		assert.Equal(t, "DHL", sorted[2].CarrierName, "DHL should be third (4 days)")
		assert.Equal(t, "USPS", sorted[3].CarrierName, "USPS should be last (5 days)")
	})

	t.Run("default_price_only", func(t *testing.T) {
		sorted := SortShippingOptions(options, "")

		assert.Equal(t, 10.00, sorted[0].TotalCost, "Cheapest should be first")
		assert.Equal(t, 10.00, sorted[1].TotalCost, "Second cheapest")
		assert.Equal(t, 12.00, sorted[2].TotalCost, "Third")
		assert.Equal(t, 15.00, sorted[3].TotalCost, "Most expensive last")
	})

	t.Run("empty_slice", func(t *testing.T) {
		sorted := SortShippingOptions([]ShippingOption{}, "price_then_days")
		assert.Empty(t, sorted)
	})
}
