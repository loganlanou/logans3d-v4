package shipping

import (
	"log/slog"
	"sort"
)

// BoxRatesResult holds the rates returned for a single box in a multi-box order.
type BoxRatesResult struct {
	BoxSelection BoxSelection
	Rates        []Rate
}

// rateKey uniquely identifies a carrier/service combination for aggregation.
type rateKey struct {
	CarrierName string
	ServiceName string
}

// aggregatedRate accumulates rate data across multiple boxes for a single carrier/service.
type aggregatedRate struct {
	RateIDs       []string // Rate IDs for each box (needed to purchase all shipments)
	ShipmentIDs   []string // Shipment IDs for each box
	BoxCount      int      // Number of boxes with rates for this carrier/service
	CarrierName   string
	ServiceName   string
	TotalPrice    float64 // Sum of shipping costs across all boxes
	Currency      string
	DeliveryDays  int    // Max delivery days across boxes (they ship in parallel)
	EstimatedDate string // Latest estimated date
	TotalBoxCost  float64
	TotalHandling float64
}

// AggregateRates combines rates from multiple boxes into shipping options.
// Only includes carrier/service combinations that have rates for ALL boxes.
// This prevents undercharging by ensuring every box in the order is covered.
//
// Parameters:
//   - boxRates: slice of rates per box, with box cost/handling info
//   - packingSolution: the packing solution (for box count and primary box SKU)
//   - sortPreference: "price_then_days", "days_then_price", or default (price only)
//
// Returns a slice of ShippingOption with aggregated costs across all boxes.
func AggregateRates(boxRates []BoxRatesResult, packingSolution *PackingSolution, sortPreference string) []ShippingOption {
	if len(boxRates) == 0 {
		return nil
	}

	totalBoxesRequired := len(boxRates)
	ratesByService := make(map[rateKey]*aggregatedRate)

	// Aggregate rates across all boxes
	for boxIdx, br := range boxRates {
		for _, rate := range br.Rates {
			key := rateKey{CarrierName: rate.CarrierNickname, ServiceName: rate.ServiceType}

			if existing, ok := ratesByService[key]; ok {
				// Add this box's costs to the existing aggregate
				existing.RateIDs = append(existing.RateIDs, rate.RateID)
				existing.ShipmentIDs = append(existing.ShipmentIDs, rate.ShipmentID)
				existing.BoxCount++
				existing.TotalPrice += rate.ShippingAmount.Amount
				existing.TotalBoxCost += br.BoxSelection.BoxCost
				existing.TotalHandling += br.BoxSelection.PackingMaterialsCost
				// Use max delivery days (boxes ship in parallel)
				if rate.DeliveryDays > existing.DeliveryDays {
					existing.DeliveryDays = rate.DeliveryDays
					existing.EstimatedDate = rate.EstimatedDate
				}
			} else {
				// First box for this carrier/service
				ratesByService[key] = &aggregatedRate{
					RateIDs:       []string{rate.RateID},
					ShipmentIDs:   []string{rate.ShipmentID},
					BoxCount:      1,
					CarrierName:   rate.CarrierNickname,
					ServiceName:   rate.ServiceType,
					TotalPrice:    rate.ShippingAmount.Amount,
					Currency:      rate.ShippingAmount.Currency,
					DeliveryDays:  rate.DeliveryDays,
					EstimatedDate: rate.EstimatedDate,
					TotalBoxCost:  br.BoxSelection.BoxCost,
					TotalHandling: br.BoxSelection.PackingMaterialsCost,
				}
			}
		}
		_ = boxIdx // suppress unused variable warning for debug logging
	}

	// Convert aggregated rates to shipping options
	// CRITICAL: Only include rates that cover ALL boxes to prevent undercharging
	var options []ShippingOption
	for key, agg := range ratesByService {
		if agg.BoxCount != totalBoxesRequired {
			slog.Debug("AggregateRates: Skipping rate with incomplete box coverage",
				"carrier", agg.CarrierName,
				"service", agg.ServiceName,
				"boxes_covered", agg.BoxCount,
				"boxes_required", totalBoxesRequired)
			continue
		}

		totalCost := agg.TotalPrice + agg.TotalBoxCost + agg.TotalHandling

		// Determine primary box SKU (first box in the packing solution)
		primaryBoxSKU := ""
		if packingSolution != nil && len(packingSolution.Boxes) > 0 {
			primaryBoxSKU = packingSolution.Boxes[0].Box.SKU
		}

		option := ShippingOption{
			RateID:          agg.RateIDs[0], // Primary rate ID (first box) for backward compatibility
			ShipmentID:      agg.ShipmentIDs[0],
			AllRateIDs:      agg.RateIDs,     // All rate IDs for multi-box label purchase
			AllShipmentIDs:  agg.ShipmentIDs, // All shipment IDs for multi-box label purchase
			CarrierName:     key.CarrierName,
			ServiceName:     key.ServiceName,
			Price:           agg.TotalPrice,
			Currency:        agg.Currency,
			DeliveryDays:    agg.DeliveryDays,
			EstimatedDate:   agg.EstimatedDate,
			BoxSKU:          primaryBoxSKU,
			BoxCost:         agg.TotalBoxCost,
			HandlingCost:    agg.TotalHandling,
			TotalCost:       totalCost,
			PackingSolution: packingSolution,
		}

		options = append(options, option)
	}

	return SortShippingOptions(options, sortPreference)
}

// SortShippingOptions sorts shipping options based on the given preference.
// Supported preferences:
//   - "price_then_days": sort by price first, then delivery days
//   - "days_then_price": sort by delivery days first, then price
//   - default: sort by price only
func SortShippingOptions(options []ShippingOption, sortPreference string) []ShippingOption {
	if len(options) == 0 {
		return options
	}

	sorted := make([]ShippingOption, len(options))
	copy(sorted, options)

	switch sortPreference {
	case "price_then_days":
		sort.Slice(sorted, func(i, j int) bool {
			if sorted[i].TotalCost == sorted[j].TotalCost {
				return sorted[i].DeliveryDays < sorted[j].DeliveryDays
			}
			return sorted[i].TotalCost < sorted[j].TotalCost
		})
	case "days_then_price":
		sort.Slice(sorted, func(i, j int) bool {
			if sorted[i].DeliveryDays == sorted[j].DeliveryDays {
				return sorted[i].TotalCost < sorted[j].TotalCost
			}
			return sorted[i].DeliveryDays < sorted[j].DeliveryDays
		})
	default:
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].TotalCost < sorted[j].TotalCost
		})
	}

	return sorted
}
