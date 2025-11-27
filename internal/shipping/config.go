package shipping

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/loganlanou/logans3d-v4/storage/db"
)

type DimensionGuard struct {
	L float64 `json:"L"`
	W float64 `json:"W"`
	H float64 `json:"H"`
}

type ItemWeights struct {
	MinGrams float64 `json:"min_grams"`
	MaxGrams float64 `json:"max_grams"`
	AvgGrams float64 `json:"avg_grams"`
	AvgOz    float64 `json:"avg_oz"`
}

type PackingMaterials struct {
	BubbleWrapPerItemOz   float64 `json:"bubble_wrap_per_item_oz"`
	PackingPaperPerBoxOz  float64 `json:"packing_paper_per_box_oz"`
	TapeAndLabelsPerBoxOz float64 `json:"tape_and_labels_per_box_oz"`
	AirPillowsPerBoxOz    float64 `json:"air_pillows_per_box_oz"`
	HandlingFeePerBoxUSD  float64 `json:"handling_fee_per_box_usd"`
}

type PackingConfig struct {
	UnitVolumeIn3    float64                   `json:"unit_volume_in3"`
	UnitWeightOz     float64                   `json:"unit_weight_oz"` // Deprecated, use ItemWeights instead
	Equivalences     map[string]int            `json:"equivalences"`
	FillRatio        float64                   `json:"fill_ratio"`
	DimensionGuard   map[string]DimensionGuard `json:"dimension_guard_in"`
	ItemWeights      map[string]ItemWeights    `json:"item_weights"`
	PackingMaterials PackingMaterials          `json:"packing_materials"`
}

type Box struct {
	SKU         string  `json:"sku"`
	Name        string  `json:"name"`
	L           float64 `json:"L"`
	W           float64 `json:"W"`
	H           float64 `json:"H"`
	BoxWeightOz float64 `json:"box_weight_oz"`
	UnitCostUSD float64 `json:"unit_cost_usd"`
}

type ShipFromAddress struct {
	Name                        string `json:"name"`
	Phone                       string `json:"phone"`
	AddressLine1                string `json:"address_line1"`
	CityLocality                string `json:"city_locality"`
	StateProvince               string `json:"state_province"`
	PostalCode                  string `json:"postal_code"`
	CountryCode                 string `json:"country_code"`
	AddressResidentialIndicator string `json:"address_residential_indicator"`
}

type RatePreferences struct {
	PresentTopN int    `json:"present_top_n"`
	Sort        string `json:"sort"`
}

type LabelsConfig struct {
	Format string `json:"format"`
}

type ShippingAPIConfig struct {
	ShipStationAPIVersion string          `json:"shipstation_api_version"`
	APIKeySecretStorage   string          `json:"api_key_secret_storage"`
	ShipFrom              ShipFromAddress `json:"ship_from"`       // Default/fallback address
	ShipFromUSPS          ShipFromAddress `json:"ship_from_usps"`  // USPS origin (Cadott, WI 54727)
	ShipFromOther         ShipFromAddress `json:"ship_from_other"` // Non-USPS origin (Eau Claire, WI 54701)
	DimDivisors           map[string]int  `json:"dim_divisors"`
	RatePreferences       RatePreferences `json:"rate_preferences"`
	Labels                LabelsConfig    `json:"labels"`
}

type ShippingConfig struct {
	Packing  PackingConfig     `json:"packing"`
	Boxes    []Box             `json:"boxes"`
	Shipping ShippingAPIConfig `json:"shipping"`
}

func LoadShippingConfig(configPath string) (*ShippingConfig, error) {
	if configPath == "" {
		configPath = "./config/shipping.json"
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read shipping config: %w", err)
	}

	var config ShippingConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse shipping config: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid shipping config: %w", err)
	}

	return &config, nil
}

// LoadShippingConfigFromDB loads shipping configuration from the database
func LoadShippingConfigFromDB(ctx context.Context, queries *db.Queries) (*ShippingConfig, error) {
	// Get shipping config JSON from database
	configRow, err := queries.GetShippingConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get shipping config from database: %w", err)
	}

	// Parse the JSON config
	var config ShippingConfig
	if err := json.Unmarshal([]byte(configRow.ConfigJson), &config); err != nil {
		return nil, fmt.Errorf("failed to parse shipping config JSON: %w", err)
	}

	// Load active boxes from box_catalog
	boxes, err := queries.ListBoxCatalog(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load box catalog: %w", err)
	}

	// Convert database boxes to config boxes
	config.Boxes = make([]Box, len(boxes))
	for i, dbBox := range boxes {
		config.Boxes[i] = Box{
			SKU:         dbBox.Sku,
			Name:        dbBox.Name,
			L:           dbBox.LengthInches,
			W:           dbBox.WidthInches,
			H:           dbBox.HeightInches,
			BoxWeightOz: dbBox.BoxWeightOz,
			UnitCostUSD: dbBox.UnitCostUsd,
		}
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid shipping config from database: %w", err)
	}

	return &config, nil
}

func validateConfig(config *ShippingConfig) error {
	if config.Packing.UnitVolumeIn3 <= 0 {
		return fmt.Errorf("unit_volume_in3 must be positive")
	}
	if config.Packing.UnitWeightOz <= 0 {
		return fmt.Errorf("unit_weight_oz must be positive")
	}
	if config.Packing.FillRatio <= 0 || config.Packing.FillRatio > 1 {
		return fmt.Errorf("fill_ratio must be between 0 and 1")
	}
	if len(config.Boxes) == 0 {
		return fmt.Errorf("at least one box must be configured")
	}
	for i, box := range config.Boxes {
		if box.L <= 0 || box.W <= 0 || box.H <= 0 {
			return fmt.Errorf("box %d: dimensions must be positive", i)
		}
		if box.BoxWeightOz < 0 {
			return fmt.Errorf("box %d: weight cannot be negative", i)
		}
		if box.UnitCostUSD < 0 {
			return fmt.Errorf("box %d: cost cannot be negative", i)
		}
	}

	// Validate ItemWeights for required categories
	requiredCategories := []string{"small", "medium", "large", "xlarge"}
	for _, cat := range requiredCategories {
		iw, exists := config.Packing.ItemWeights[cat]
		if !exists {
			return fmt.Errorf("item_weights missing required category: %s", cat)
		}
		if iw.AvgOz <= 0 {
			return fmt.Errorf("item_weights[%s].avg_oz must be positive", cat)
		}
	}

	// Validate DimensionGuard for required categories
	for _, cat := range requiredCategories {
		dg, exists := config.Packing.DimensionGuard[cat]
		if !exists {
			return fmt.Errorf("dimension_guard_in missing required category: %s", cat)
		}
		if dg.L <= 0 || dg.W <= 0 || dg.H <= 0 {
			return fmt.Errorf("dimension_guard_in[%s] dimensions must be positive", cat)
		}
	}

	return nil
}

func CreateDefaultConfig() *ShippingConfig {
	return &ShippingConfig{
		Packing: PackingConfig{
			UnitVolumeIn3: 27.0,
			UnitWeightOz:  2.0, // Deprecated, kept for compatibility
			Equivalences: map[string]int{
				"small":  1,
				"medium": 3,
				"large":  6,
				"xlarge": 18,
			},
			FillRatio: 0.80,
			DimensionGuard: map[string]DimensionGuard{
				"small":  {L: 4, W: 4, H: 4},
				"medium": {L: 8, W: 5, H: 5},
				"large":  {L: 20, W: 10, H: 6},
				"xlarge": {L: 24, W: 12, H: 10},
			},
			ItemWeights: map[string]ItemWeights{
				"small": {
					MinGrams: 70,
					MaxGrams: 100,
					AvgGrams: 85,
					AvgOz:    3.0, // 85g ≈ 3.0 oz
				},
				"medium": {
					MinGrams: 180,
					MaxGrams: 220,
					AvgGrams: 200,
					AvgOz:    7.05, // 200g ≈ 7.05 oz
				},
				"large": {
					MinGrams: 350,
					MaxGrams: 500,
					AvgGrams: 425,
					AvgOz:    15.0, // 425g ≈ 15.0 oz
				},
				"xlarge": {
					MinGrams: 800,
					MaxGrams: 1200,
					AvgGrams: 1000,
					AvgOz:    35.3, // 1000g ≈ 35.3 oz (estimated for very large items)
				},
			},
			PackingMaterials: PackingMaterials{
				BubbleWrapPerItemOz:   0.2,  // Small amount of bubble wrap per item
				PackingPaperPerBoxOz:  1.0,  // Base packing paper per box
				TapeAndLabelsPerBoxOz: 0.5,  // Tape and shipping labels
				AirPillowsPerBoxOz:    0.8,  // Air pillows for void fill
				HandlingFeePerBoxUSD:  1.50, // Flat handling fee per box (covers materials + labor)
			},
		},
		Boxes: []Box{
			{SKU: "CXBSS21", Name: "8x6x4", L: 8, W: 6, H: 4, BoxWeightOz: 4.0, UnitCostUSD: 0.38},
			{SKU: "CXBSS24", Name: "10x8x6", L: 10, W: 8, H: 6, BoxWeightOz: 6.0, UnitCostUSD: 0.54},
			{SKU: "CXBSM1294", Name: "12x9x4", L: 12, W: 9, H: 4, BoxWeightOz: 6.0, UnitCostUSD: 0.62},
			{SKU: "MD12126", Name: "12x12x6 (MD)", L: 12, W: 12, H: 6, BoxWeightOz: 8.0, UnitCostUSD: 0.70},
		},
		Shipping: ShippingAPIConfig{
			ShipStationAPIVersion: "v2",
			APIKeySecretStorage:   "env",
			ShipFrom: ShipFromAddress{
				Name:                        "Creswood Corners",
				Phone:                       "715-703-3768",
				AddressLine1:                "25580 County Highway S",
				CityLocality:                "Cadott",
				StateProvince:               "WI",
				PostalCode:                  "54727",
				CountryCode:                 "US",
				AddressResidentialIndicator: "no",
			},
			DimDivisors: map[string]int{
				"usps":  166,
				"ups":   139,
				"fedex": 139,
			},
			RatePreferences: RatePreferences{
				PresentTopN: 3,
				Sort:        "price_then_days",
			},
			Labels: LabelsConfig{
				Format: "pdf",
			},
		},
	}
}

func SaveConfigToFile(config *ShippingConfig, path string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
