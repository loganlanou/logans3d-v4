package shipping

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		modifyFunc  func(*ShippingConfig)
		wantErr     bool
		errContains string
	}{
		{
			name:       "valid_default_config",
			modifyFunc: nil,
			wantErr:    false,
		},
		// UnitVolumeIn3 validation
		{
			name: "zero_unit_volume",
			modifyFunc: func(c *ShippingConfig) {
				c.Packing.UnitVolumeIn3 = 0
			},
			wantErr:     true,
			errContains: "unit_volume_in3 must be positive",
		},
		{
			name: "negative_unit_volume",
			modifyFunc: func(c *ShippingConfig) {
				c.Packing.UnitVolumeIn3 = -10
			},
			wantErr:     true,
			errContains: "unit_volume_in3 must be positive",
		},
		// FillRatio validation
		{
			name: "zero_fill_ratio",
			modifyFunc: func(c *ShippingConfig) {
				c.Packing.FillRatio = 0
			},
			wantErr:     true,
			errContains: "fill_ratio must be between 0 and 1",
		},
		{
			name: "negative_fill_ratio",
			modifyFunc: func(c *ShippingConfig) {
				c.Packing.FillRatio = -0.5
			},
			wantErr:     true,
			errContains: "fill_ratio must be between 0 and 1",
		},
		{
			name: "fill_ratio_greater_than_one",
			modifyFunc: func(c *ShippingConfig) {
				c.Packing.FillRatio = 1.5
			},
			wantErr:     true,
			errContains: "fill_ratio must be between 0 and 1",
		},
		// Box validation
		{
			name: "no_boxes_configured",
			modifyFunc: func(c *ShippingConfig) {
				c.Boxes = []Box{}
			},
			wantErr:     true,
			errContains: "at least one box must be configured",
		},
		{
			name: "box_with_zero_length",
			modifyFunc: func(c *ShippingConfig) {
				c.Boxes[0].L = 0
			},
			wantErr:     true,
			errContains: "box 0: dimensions must be positive",
		},
		{
			name: "box_with_negative_width",
			modifyFunc: func(c *ShippingConfig) {
				c.Boxes[0].W = -5
			},
			wantErr:     true,
			errContains: "box 0: dimensions must be positive",
		},
		{
			name: "box_with_negative_weight",
			modifyFunc: func(c *ShippingConfig) {
				c.Boxes[0].BoxWeightOz = -1
			},
			wantErr:     true,
			errContains: "box 0: weight cannot be negative",
		},
		{
			name: "box_with_negative_cost",
			modifyFunc: func(c *ShippingConfig) {
				c.Boxes[0].UnitCostUSD = -0.50
			},
			wantErr:     true,
			errContains: "box 0: cost cannot be negative",
		},
		// ItemWeights validation - Bug regression test (dfc1b8b)
		{
			name: "missing_small_category_in_ItemWeights",
			modifyFunc: func(c *ShippingConfig) {
				delete(c.Packing.ItemWeights, "small")
			},
			wantErr:     true,
			errContains: "item_weights missing required category: small",
		},
		{
			name: "missing_medium_category_in_ItemWeights",
			modifyFunc: func(c *ShippingConfig) {
				delete(c.Packing.ItemWeights, "medium")
			},
			wantErr:     true,
			errContains: "item_weights missing required category: medium",
		},
		{
			name: "missing_large_category_in_ItemWeights",
			modifyFunc: func(c *ShippingConfig) {
				delete(c.Packing.ItemWeights, "large")
			},
			wantErr:     true,
			errContains: "item_weights missing required category: large",
		},
		{
			name: "missing_xlarge_category_in_ItemWeights",
			modifyFunc: func(c *ShippingConfig) {
				delete(c.Packing.ItemWeights, "xlarge")
			},
			wantErr:     true,
			errContains: "item_weights missing required category: xlarge",
		},
		{
			name: "zero_weight_for_small_category",
			modifyFunc: func(c *ShippingConfig) {
				iw := c.Packing.ItemWeights["small"]
				iw.AvgOz = 0
				c.Packing.ItemWeights["small"] = iw
			},
			wantErr:     true,
			errContains: "item_weights[small].avg_oz must be positive",
		},
		{
			name: "negative_weight_for_medium_category",
			modifyFunc: func(c *ShippingConfig) {
				iw := c.Packing.ItemWeights["medium"]
				iw.AvgOz = -5
				c.Packing.ItemWeights["medium"] = iw
			},
			wantErr:     true,
			errContains: "item_weights[medium].avg_oz must be positive",
		},
		// DimensionGuard validation - Bug regression test (dfc1b8b)
		{
			name: "missing_small_category_in_DimensionGuard",
			modifyFunc: func(c *ShippingConfig) {
				delete(c.Packing.DimensionGuard, "small")
			},
			wantErr:     true,
			errContains: "dimension_guard_in missing required category: small",
		},
		{
			name: "missing_medium_category_in_DimensionGuard",
			modifyFunc: func(c *ShippingConfig) {
				delete(c.Packing.DimensionGuard, "medium")
			},
			wantErr:     true,
			errContains: "dimension_guard_in missing required category: medium",
		},
		{
			name: "zero_dimension_in_DimensionGuard",
			modifyFunc: func(c *ShippingConfig) {
				dg := c.Packing.DimensionGuard["small"]
				dg.L = 0
				c.Packing.DimensionGuard["small"] = dg
			},
			wantErr:     true,
			errContains: "dimension_guard_in[small] dimensions must be positive",
		},
		{
			name: "negative_dimension_in_DimensionGuard",
			modifyFunc: func(c *ShippingConfig) {
				dg := c.Packing.DimensionGuard["large"]
				dg.W = -10
				c.Packing.DimensionGuard["large"] = dg
			},
			wantErr:     true,
			errContains: "dimension_guard_in[large] dimensions must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := CreateDefaultConfig()
			if tt.modifyFunc != nil {
				tt.modifyFunc(config)
			}

			err := validateConfig(config)

			if tt.wantErr {
				require.Error(t, err, "Expected validation error for: %s", tt.name)
				assert.Contains(t, err.Error(), tt.errContains,
					"Error should contain %q, got: %v", tt.errContains, err)
			} else {
				assert.NoError(t, err, "Config should be valid")
			}
		})
	}
}

func TestCreateDefaultConfig(t *testing.T) {
	config := CreateDefaultConfig()

	t.Run("should_be_valid", func(t *testing.T) {
		err := validateConfig(config)
		assert.NoError(t, err, "Default config should pass validation")
	})

	t.Run("should_have_all_required_ItemWeights", func(t *testing.T) {
		requiredCategories := []string{"small", "medium", "large", "xlarge"}
		for _, cat := range requiredCategories {
			iw, exists := config.Packing.ItemWeights[cat]
			assert.True(t, exists, "ItemWeights should have category: %s", cat)
			assert.Greater(t, iw.AvgOz, 0.0, "ItemWeights[%s].AvgOz should be positive", cat)
		}
	})

	t.Run("should_have_all_required_DimensionGuard", func(t *testing.T) {
		requiredCategories := []string{"small", "medium", "large", "xlarge"}
		for _, cat := range requiredCategories {
			dg, exists := config.Packing.DimensionGuard[cat]
			assert.True(t, exists, "DimensionGuard should have category: %s", cat)
			assert.Greater(t, dg.L, 0.0, "DimensionGuard[%s].L should be positive", cat)
			assert.Greater(t, dg.W, 0.0, "DimensionGuard[%s].W should be positive", cat)
			assert.Greater(t, dg.H, 0.0, "DimensionGuard[%s].H should be positive", cat)
		}
	})

	t.Run("should_have_boxes", func(t *testing.T) {
		assert.NotEmpty(t, config.Boxes, "Default config should have at least one box")
	})

	t.Run("should_have_valid_packing_config", func(t *testing.T) {
		assert.Greater(t, config.Packing.UnitVolumeIn3, 0.0)
		assert.Greater(t, config.Packing.FillRatio, 0.0)
		assert.LessOrEqual(t, config.Packing.FillRatio, 1.0)
	})
}
