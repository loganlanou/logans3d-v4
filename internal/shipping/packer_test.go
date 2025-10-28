package shipping

import (
	"testing"
)

func TestSmallUnits(t *testing.T) {
	config := CreateDefaultConfig()
	packer := NewPacker(config)

	tests := []struct {
		name     string
		counts   ItemCounts
		expected int
	}{
		{
			name:     "only small items",
			counts:   ItemCounts{Small: 5, Medium: 0, Large: 0, XL: 0},
			expected: 5,
		},
		{
			name:     "only medium items",
			counts:   ItemCounts{Small: 0, Medium: 2, Large: 0, XL: 0},
			expected: 6, // 2 * 3
		},
		{
			name:     "only large items",
			counts:   ItemCounts{Small: 0, Medium: 0, Large: 1, XL: 0},
			expected: 6, // 1 * 6
		},
		{
			name:     "only XL items",
			counts:   ItemCounts{Small: 0, Medium: 0, Large: 0, XL: 1},
			expected: 18, // 1 * 18
		},
		{
			name:     "mixed items",
			counts:   ItemCounts{Small: 2, Medium: 1, Large: 1, XL: 0},
			expected: 11, // 2 + 3 + 6
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := packer.SmallUnits(tt.counts)
			if result != tt.expected {
				t.Errorf("SmallUnits() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestCapacity(t *testing.T) {
	config := CreateDefaultConfig()
	packer := NewPacker(config)

	tests := []struct {
		name     string
		box      Box
		expected int
	}{
		{
			name:     "8x6x4 box",
			box:      Box{L: 8, W: 6, H: 4},
			expected: 5, // floor((8*6*4*0.8)/27) = floor(153.6/27) = 5
		},
		{
			name:     "10x8x6 box",
			box:      Box{L: 10, W: 8, H: 6},
			expected: 14, // floor((10*8*6*0.8)/27) = floor(384/27) = 14
		},
		{
			name:     "12x12x6 box",
			box:      Box{L: 12, W: 12, H: 6},
			expected: 25, // floor((12*12*6*0.8)/27) = floor(691.2/27) = 25
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := packer.Capacity(tt.box)
			t.Logf("Box %s: volume=%g, capacity calculation=%g, result=%d",
				tt.name, tt.box.L*tt.box.W*tt.box.H,
				(tt.box.L*tt.box.W*tt.box.H*config.Packing.FillRatio)/config.Packing.UnitVolumeIn3,
				result)
			if result != tt.expected {
				t.Errorf("Capacity() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestPackSingleBox(t *testing.T) {
	config := CreateDefaultConfig()
	packer := NewPacker(config)

	tests := []struct {
		name      string
		counts    ItemCounts
		shouldFit bool
		expectBox string // SKU of expected box
	}{
		{
			name:      "small order fits in smallest box",
			counts:    ItemCounts{Small: 3, Medium: 0, Large: 0, XL: 0},
			shouldFit: true,
			expectBox: "CXBSS21", // 8x6x4 box
		},
		{
			name:      "medium order needs bigger box",
			counts:    ItemCounts{Small: 2, Medium: 2, Large: 0, XL: 0},
			shouldFit: true,
			expectBox: "CXBSS24", // 10x8x6 box (8 small units = 2 + 2*3)
		},
		{
			name:      "large order might need multiple boxes",
			counts:    ItemCounts{Small: 0, Medium: 0, Large: 2, XL: 0},
			shouldFit: false, // Large items might not fit due to dimension constraints
		},
		{
			name:      "XL order might need multiple boxes",
			counts:    ItemCounts{Small: 0, Medium: 0, Large: 0, XL: 1},
			shouldFit: false, // XL items might not fit due to dimension constraints
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := packer.PackSingleBox(tt.counts)

			if tt.shouldFit {
				if !result.Valid {
					t.Errorf("PackSingleBox() should have found a solution, got error: %s", result.Error)
					return
				}
				if len(result.Boxes) != 1 {
					t.Errorf("PackSingleBox() should return exactly 1 box, got %d", len(result.Boxes))
					return
				}
				if result.Boxes[0].Box.SKU != tt.expectBox {
					t.Errorf("PackSingleBox() box SKU = %s, want %s", result.Boxes[0].Box.SKU, tt.expectBox)
				}
			} else {
				if result.Valid {
					t.Errorf("PackSingleBox() should have failed but found solution with box %s", result.Boxes[0].Box.SKU)
				}
			}
		})
	}
}

func TestPackMultipleBoxes(t *testing.T) {
	config := CreateDefaultConfig()
	packer := NewPacker(config)

	tests := []struct {
		name     string
		counts   ItemCounts
		maxBoxes int
	}{
		{
			name:     "very large order requiring multiple boxes",
			counts:   ItemCounts{Small: 0, Medium: 0, Large: 0, XL: 3}, // 54 small units
			maxBoxes: 3,                                                // Should need multiple 12x12x6 boxes
		},
		{
			name:     "mixed large order",
			counts:   ItemCounts{Small: 10, Medium: 5, Large: 2, XL: 1}, // 10 + 15 + 12 + 18 = 55 small units
			maxBoxes: 3,                                                 // Should need multiple boxes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := packer.PackMultipleBoxes(tt.counts)

			if !result.Valid {
				t.Errorf("PackMultipleBoxes() failed: %s", result.Error)
				return
			}

			if result.TotalBoxes > tt.maxBoxes {
				t.Errorf("PackMultipleBoxes() used %d boxes, expected <= %d", result.TotalBoxes, tt.maxBoxes)
			}

			// Verify total small units match
			totalPacked := 0
			for _, box := range result.Boxes {
				totalPacked += box.SmallUnits
			}
			expectedTotal := packer.SmallUnits(tt.counts)
			if totalPacked != expectedTotal {
				t.Errorf("PackMultipleBoxes() packed %d small units, expected %d", totalPacked, expectedTotal)
			}
		})
	}
}

func TestPack(t *testing.T) {
	config := CreateDefaultConfig()
	packer := NewPacker(config)

	tests := []struct {
		name    string
		counts  ItemCounts
		wantErr bool
	}{
		{
			name:    "empty order",
			counts:  ItemCounts{Small: 0, Medium: 0, Large: 0, XL: 0},
			wantErr: true,
		},
		{
			name:    "normal small order",
			counts:  ItemCounts{Small: 3, Medium: 1, Large: 0, XL: 0},
			wantErr: false,
		},
		{
			name:    "large order requiring multiple boxes",
			counts:  ItemCounts{Small: 0, Medium: 0, Large: 0, XL: 2},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := packer.Pack(tt.counts)

			if tt.wantErr {
				if result.Valid {
					t.Errorf("Pack() should have failed but succeeded")
				}
			} else {
				if !result.Valid {
					t.Errorf("Pack() failed: %s", result.Error)
				}
			}
		})
	}
}

func TestValidateItemDimensions(t *testing.T) {
	config := CreateDefaultConfig()
	packer := NewPacker(config)

	tests := []struct {
		name     string
		category string
		length   float64
		width    float64
		height   float64
		wantErr  bool
	}{
		{
			name:     "small item within limits",
			category: "small",
			length:   3, width: 3, height: 3,
			wantErr: false,
		},
		{
			name:     "small item exceeds limits",
			category: "small",
			length:   5, width: 3, height: 3,
			wantErr: true,
		},
		{
			name:     "medium item within limits",
			category: "medium",
			length:   7, width: 4, height: 4,
			wantErr: false,
		},
		{
			name:     "invalid category",
			category: "invalid",
			length:   1, width: 1, height: 1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := packer.ValidateItemDimensions(tt.category, tt.length, tt.width, tt.height)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateItemDimensions() should have failed but succeeded")
				}
			} else {
				if err != nil {
					t.Errorf("ValidateItemDimensions() failed: %v", err)
				}
			}
		})
	}
}

func TestEstimateWeightWithActualItems(t *testing.T) {
	config := CreateDefaultConfig()
	packer := NewPacker(config)

	box := Box{L: 10, W: 8, H: 6, BoxWeightOz: 6.0, UnitCostUSD: 0.54}

	tests := []struct {
		name          string
		counts        ItemCounts
		expectedRange struct{ min, max float64 }
	}{
		{
			name:   "small items only",
			counts: ItemCounts{Small: 3, Medium: 0, Large: 0, XL: 0},
			expectedRange: struct{ min, max float64 }{
				min: 6.0 + 3.0*3 + 0.2*3 + 1.0 + 0.5 + 0.8,     // box + items + bubble wrap + materials
				max: 6.0 + 3.0*3 + 0.2*3 + 1.0 + 0.5 + 0.8 + 1, // small tolerance
			},
		},
		{
			name:   "mixed items",
			counts: ItemCounts{Small: 2, Medium: 1, Large: 0, XL: 0},
			expectedRange: struct{ min, max float64 }{
				min: 6.0 + 2*3.0 + 1*7.05 + 0.2*3 + 1.0 + 0.5 + 0.8,
				max: 6.0 + 2*3.0 + 1*7.05 + 0.2*3 + 1.0 + 0.5 + 0.8 + 1,
			},
		},
		{
			name:   "large item",
			counts: ItemCounts{Small: 1, Medium: 0, Large: 1, XL: 0},
			expectedRange: struct{ min, max float64 }{
				min: 6.0 + 1*3.0 + 1*15.0 + 0.2*2 + 1.0 + 0.5 + 0.8,
				max: 6.0 + 1*3.0 + 1*15.0 + 0.2*2 + 1.0 + 0.5 + 0.8 + 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			weight := packer.EstimateWeight(box, tt.counts)

			if weight < tt.expectedRange.min || weight > tt.expectedRange.max {
				t.Errorf("EstimateWeight() = %f, want between %f and %f",
					weight, tt.expectedRange.min, tt.expectedRange.max)
			}

			t.Logf("Weight calculation for %s: %.2f oz", tt.name, weight)
			t.Logf("  Box weight: %.2f oz", box.BoxWeightOz)

			// Log detailed breakdown
			if weights, exists := config.Packing.ItemWeights["small"]; exists && tt.counts.Small > 0 {
				t.Logf("  Small items: %d × %.2f oz = %.2f oz", tt.counts.Small, weights.AvgOz, weights.AvgOz*float64(tt.counts.Small))
			}
			if weights, exists := config.Packing.ItemWeights["medium"]; exists && tt.counts.Medium > 0 {
				t.Logf("  Medium items: %d × %.2f oz = %.2f oz", tt.counts.Medium, weights.AvgOz, weights.AvgOz*float64(tt.counts.Medium))
			}
			if weights, exists := config.Packing.ItemWeights["large"]; exists && tt.counts.Large > 0 {
				t.Logf("  Large items: %d × %.2f oz = %.2f oz", tt.counts.Large, weights.AvgOz, weights.AvgOz*float64(tt.counts.Large))
			}

			totalItems := tt.counts.Small + tt.counts.Medium + tt.counts.Large + tt.counts.XL
			materials := config.Packing.PackingMaterials
			t.Logf("  Bubble wrap: %d items × %.2f oz = %.2f oz", totalItems, materials.BubbleWrapPerItemOz, float64(totalItems)*materials.BubbleWrapPerItemOz)
			t.Logf("  Packing materials: %.2f oz (paper) + %.2f oz (tape/labels) + %.2f oz (air pillows)",
				materials.PackingPaperPerBoxOz, materials.TapeAndLabelsPerBoxOz, materials.AirPillowsPerBoxOz)
		})
	}
}

func TestWeightComparison(t *testing.T) {
	config := CreateDefaultConfig()
	packer := NewPacker(config)

	// Test that our new weight calculation gives more accurate results than the old method
	box := Box{L: 10, W: 8, H: 6, BoxWeightOz: 6.0, UnitCostUSD: 0.54}
	counts := ItemCounts{Small: 2, Medium: 1, Large: 0, XL: 0}
	smallUnits := packer.SmallUnits(counts)

	newWeight := packer.EstimateWeight(box, counts)
	oldWeight := packer.EstimateWeightLegacy(box, smallUnits)

	t.Logf("New weight calculation: %.2f oz", newWeight)
	t.Logf("Old weight calculation: %.2f oz", oldWeight)
	t.Logf("Difference: %.2f oz", newWeight-oldWeight)

	// The new calculation should generally be more detailed and account for materials
	// For this test case: 2 small (6 oz) + 1 medium (7.05 oz) + materials (2.9 oz) + box (6 oz) = ~22 oz
	// Old calculation: 5 small units × 2 oz + box (6 oz) = 16 oz
	if newWeight <= oldWeight {
		t.Errorf("New weight calculation (%f) should typically be higher than old (%f) due to packing materials", newWeight, oldWeight)
	}
}

func TestDistributeItemsToBox(t *testing.T) {
	config := CreateDefaultConfig()
	packer := NewPacker(config)

	tests := []struct {
		name        string
		inputCounts ItemCounts
		capacity    int
		wantPacked  int // total small units that should be packed
	}{
		{
			name:        "pack large items first",
			inputCounts: ItemCounts{Small: 10, Medium: 2, Large: 1, XL: 0},
			capacity:    12, // Can fit 1 large (6) + 1 medium (3) + 3 small (3) = 12
			wantPacked:  12,
		},
		{
			name:        "pack everything when capacity is sufficient",
			inputCounts: ItemCounts{Small: 3, Medium: 1, Large: 0, XL: 0},
			capacity:    10, // Total is 6 small units, capacity is 10
			wantPacked:  6,
		},
		{
			name:        "pack partial when capacity is limited",
			inputCounts: ItemCounts{Small: 0, Medium: 0, Large: 2, XL: 0},
			capacity:    8, // Can only fit 1 large (6 units), not both
			wantPacked:  6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			boxCounts, remaining := packer.distributeItemsToBox(tt.inputCounts, tt.capacity)
			packed := packer.SmallUnits(boxCounts)

			if packed != tt.wantPacked {
				t.Errorf("distributeItemsToBox() packed %d small units, want %d", packed, tt.wantPacked)
			}

			// Verify that packed + remaining equals original
			originalTotal := packer.SmallUnits(tt.inputCounts)
			remainingTotal := packer.SmallUnits(remaining)
			if packed+remainingTotal != originalTotal {
				t.Errorf("Packed (%d) + remaining (%d) != original (%d)", packed, remainingTotal, originalTotal)
			}

			t.Logf("Input: %+v", tt.inputCounts)
			t.Logf("Packed: %+v (total: %d units)", boxCounts, packed)
			t.Logf("Remaining: %+v (total: %d units)", remaining, remainingTotal)
		})
	}
}
