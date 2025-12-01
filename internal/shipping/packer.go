package shipping

import (
	"fmt"
	"log/slog"
	"math"
	"sort"
)

type ItemCounts struct {
	Small  int `json:"small"`
	Medium int `json:"medium"`
	Large  int `json:"large"`
	XL     int `json:"xl"`

	SmallWeightOz  float64 `json:"small_weight_oz,omitempty"`
	MediumWeightOz float64 `json:"medium_weight_oz,omitempty"`
	LargeWeightOz  float64 `json:"large_weight_oz,omitempty"`
	XLWeightOz     float64 `json:"xl_weight_oz,omitempty"`

	SmallMaxDims  DimensionGuard `json:"small_max_dims,omitempty"`
	MediumMaxDims DimensionGuard `json:"medium_max_dims,omitempty"`
	LargeMaxDims  DimensionGuard `json:"large_max_dims,omitempty"`
	XLMaxDims     DimensionGuard `json:"xl_max_dims,omitempty"`
}

type PackingSolution struct {
	Boxes      []BoxSelection `json:"boxes"`
	TotalCost  float64        `json:"total_cost"`
	TotalBoxes int            `json:"total_boxes"`
	Valid      bool           `json:"valid"`
	Error      string         `json:"error,omitempty"`
}

type BoxSelection struct {
	Box                  Box        `json:"box"`
	Quantity             int        `json:"quantity"`
	SmallUnits           int        `json:"small_units"`
	Weight               float64    `json:"weight"`
	BoxCost              float64    `json:"box_cost"`
	PackingMaterialsCost float64    `json:"packing_materials_cost"`
	ItemCounts           ItemCounts `json:"item_counts"` // Track what items are in this box
}

type Packer struct {
	config *ShippingConfig
}

func NewPacker(config *ShippingConfig) *Packer {
	return &Packer{config: config}
}

func (p *Packer) categoryWeight(category string, count int, override float64) float64 {
	if count <= 0 {
		return 0
	}
	if override > 0 {
		return override
	}
	if weight, exists := p.config.Packing.ItemWeights[category]; exists {
		return weight.AvgOz * float64(count)
	}
	return 0
}

func (p *Packer) categoryGuard(category string, provided DimensionGuard) DimensionGuard {
	guard := p.config.Packing.DimensionGuard[category]
	if provided.L > 0 && provided.W > 0 && provided.H > 0 {
		guard = provided
	}
	return guard
}

func (p *Packer) SmallUnits(counts ItemCounts) int {
	eq := p.config.Packing.Equivalences
	return counts.Small +
		eq["medium"]*counts.Medium +
		eq["large"]*counts.Large +
		eq["xlarge"]*counts.XL
}

func (p *Packer) Capacity(box Box) int {
	vol := box.L * box.W * box.H
	return int(math.Floor((vol * p.config.Packing.FillRatio) / p.config.Packing.UnitVolumeIn3))
}

func (p *Packer) EstimateWeight(box Box, counts ItemCounts) float64 {
	// Start with box weight
	totalWeight := box.BoxWeightOz

	slog.Debug("EstimateWeight: Evaluating candidate box",
		"box_sku", box.SKU,
		"box_name", box.Name,
		"box_weight_oz", box.BoxWeightOz)

	// Add item weights based on actual categories and counts
	var itemWeightBreakdown []interface{}
	totalItemWeight := 0.0

	addCategoryWeight := func(category string, count int, override float64) {
		if count == 0 {
			return
		}
		weight := p.categoryWeight(category, count, override)
		if weight <= 0 {
			return
		}
		totalWeight += weight
		totalItemWeight += weight
		itemWeightBreakdown = append(itemWeightBreakdown,
			category+"_items", count,
			category+"_total_oz", weight)
	}

	addCategoryWeight("small", counts.Small, counts.SmallWeightOz)
	addCategoryWeight("medium", counts.Medium, counts.MediumWeightOz)
	addCategoryWeight("large", counts.Large, counts.LargeWeightOz)
	addCategoryWeight("xlarge", counts.XL, counts.XLWeightOz)

	if len(itemWeightBreakdown) > 0 {
		slog.Debug("EstimateWeight: Item weights", itemWeightBreakdown...)
	}

	// Add packing materials
	materials := p.config.Packing.PackingMaterials
	totalItems := counts.Small + counts.Medium + counts.Large + counts.XL

	// Bubble wrap per item
	bubbleWrapWeight := materials.BubbleWrapPerItemOz * float64(totalItems)
	totalWeight += bubbleWrapWeight

	// Base packing materials per box
	packingPaperWeight := materials.PackingPaperPerBoxOz
	tapeLabelsWeight := materials.TapeAndLabelsPerBoxOz
	airPillowsWeight := materials.AirPillowsPerBoxOz

	totalWeight += packingPaperWeight
	totalWeight += tapeLabelsWeight
	totalWeight += airPillowsWeight

	totalPackingMaterials := bubbleWrapWeight + packingPaperWeight + tapeLabelsWeight + airPillowsWeight

	slog.Debug("EstimateWeight: Packing materials",
		"total_items", totalItems,
		"bubble_wrap_oz", bubbleWrapWeight,
		"packing_paper_oz", packingPaperWeight,
		"tape_labels_oz", tapeLabelsWeight,
		"air_pillows_oz", airPillowsWeight,
		"total_packing_materials_oz", totalPackingMaterials)

	slog.Debug("EstimateWeight: Final weight breakdown",
		"box_sku", box.SKU,
		"box_weight_oz", box.BoxWeightOz,
		"items_weight_oz", totalItemWeight,
		"packing_materials_oz", totalPackingMaterials,
		"total_weight_oz", totalWeight,
		"total_weight_lbs", totalWeight/16.0)

	return totalWeight
}

// EstimateWeightLegacy provides backward compatibility with the old signature
func (p *Packer) EstimateWeightLegacy(box Box, smallUnits int) float64 {
	return box.BoxWeightOz + (p.config.Packing.UnitWeightOz * float64(smallUnits))
}

func (p *Packer) dimensionsOK(box Box, counts ItemCounts) bool {
	boxDims := []float64{box.L, box.W, box.H}
	sort.Float64s(boxDims)

	checkCategory := func(guardDims DimensionGuard, count int) bool {
		if count == 0 {
			return true
		}
		catDims := []float64{guardDims.L, guardDims.W, guardDims.H}
		sort.Float64s(catDims)

		for i := 0; i < 3; i++ {
			if catDims[i] > boxDims[i] {
				return false
			}
		}
		return true
	}

	return checkCategory(p.categoryGuard("small", counts.SmallMaxDims), counts.Small) &&
		checkCategory(p.categoryGuard("medium", counts.MediumMaxDims), counts.Medium) &&
		checkCategory(p.categoryGuard("large", counts.LargeMaxDims), counts.Large) &&
		checkCategory(p.categoryGuard("xlarge", counts.XLMaxDims), counts.XL)
}

func (p *Packer) candidateBoxes(counts ItemCounts) []Box {
	need := p.SmallUnits(counts)
	var candidates []Box

	for _, box := range p.config.Boxes {
		capacity := p.Capacity(box)
		if capacity >= need && p.dimensionsOK(box, counts) {
			candidates = append(candidates, box)
		}
	}

	return candidates
}

func (p *Packer) PackSingleBox(counts ItemCounts) *PackingSolution {
	candidates := p.candidateBoxes(counts)

	if len(candidates) == 0 {
		return &PackingSolution{
			Valid: false,
			Error: "no single box can fit all items",
		}
	}

	slog.Debug("PackSingleBox: Evaluating candidate boxes",
		"num_candidates", len(candidates),
		"evaluating_for_single_box", true)

	smallUnits := p.SmallUnits(counts)
	var bestSolution *PackingSolution
	bestCost := math.Inf(1)

	for _, box := range candidates {
		weight := p.EstimateWeight(box, counts)
		boxCost := box.UnitCostUSD
		materialsCost := p.config.Packing.PackingMaterials.HandlingFeePerBoxUSD

		selection := BoxSelection{
			Box:                  box,
			Quantity:             1,
			SmallUnits:           smallUnits,
			Weight:               weight,
			BoxCost:              boxCost,
			PackingMaterialsCost: materialsCost,
			ItemCounts:           counts,
		}

		totalCost := boxCost + materialsCost

		solution := &PackingSolution{
			Boxes:      []BoxSelection{selection},
			TotalCost:  totalCost,
			TotalBoxes: 1,
			Valid:      true,
		}

		if totalCost < bestCost {
			slog.Debug("PackSingleBox: New best candidate",
				"box_sku", box.SKU,
				"box_cost_usd", boxCost,
				"materials_cost_usd", materialsCost,
				"total_cost_usd", totalCost,
				"weight_oz", weight,
				"previous_best_cost", bestCost)
			bestCost = totalCost
			bestSolution = solution
		}
	}

	return bestSolution
}

func (p *Packer) PackMultipleBoxes(counts ItemCounts) *PackingSolution {
	totalUnits := p.SmallUnits(counts)
	if totalUnits == 0 {
		return &PackingSolution{Valid: false, Error: "no items to pack"}
	}

	// For multi-box packing, we'll use a simplified approach:
	// Fill the largest box optimally, then recursively pack the remainder
	return p.packRecursively(counts, 0)
}

func (p *Packer) packRecursively(counts ItemCounts, depth int) *PackingSolution {
	if depth > 10 {
		return &PackingSolution{Valid: false, Error: "too many boxes required (>10)"}
	}

	totalUnits := p.SmallUnits(counts)
	if totalUnits == 0 {
		return &PackingSolution{Boxes: []BoxSelection{}, TotalCost: 0, TotalBoxes: 0, Valid: true}
	}

	// Try to pack everything in a single box first
	singleBoxSolution := p.PackSingleBox(counts)
	if singleBoxSolution.Valid {
		return singleBoxSolution
	}

	// Find the largest box that can fit at least some items
	boxes := make([]Box, len(p.config.Boxes))
	copy(boxes, p.config.Boxes)
	sort.Slice(boxes, func(i, j int) bool {
		return p.Capacity(boxes[i]) > p.Capacity(boxes[j])
	})

	for _, box := range boxes {
		capacity := p.Capacity(box)
		if capacity <= 0 {
			continue
		}

		// Try to fill this box optimally
		boxCounts, remainingCounts := p.distributeItemsToBox(counts, capacity)

		if p.SmallUnits(boxCounts) == 0 {
			continue // This box can't fit anything
		}

		weight := p.EstimateWeight(box, boxCounts)
		materialsCost := p.config.Packing.PackingMaterials.HandlingFeePerBoxUSD

		selection := BoxSelection{
			Box:                  box,
			Quantity:             1,
			SmallUnits:           p.SmallUnits(boxCounts),
			Weight:               weight,
			BoxCost:              box.UnitCostUSD,
			PackingMaterialsCost: materialsCost,
			ItemCounts:           boxCounts,
		}

		// Recursively pack the remaining items
		remainingSolution := p.packRecursively(remainingCounts, depth+1)
		if !remainingSolution.Valid {
			continue // Try next box
		}

		// Combine solutions
		allBoxes := append([]BoxSelection{selection}, remainingSolution.Boxes...)
		totalCost := box.UnitCostUSD + materialsCost + remainingSolution.TotalCost

		return &PackingSolution{
			Boxes:      allBoxes,
			TotalCost:  totalCost,
			TotalBoxes: len(allBoxes),
			Valid:      true,
		}
	}

	return &PackingSolution{Valid: false, Error: "unable to pack items in available boxes"}
}

// distributeItemsToBox optimally distributes items to fill a box up to its capacity
func (p *Packer) distributeItemsToBox(counts ItemCounts, capacity int) (boxCounts ItemCounts, remaining ItemCounts) {
	// Prioritize larger items first to minimize wasted space
	remaining = counts

	equivalences := p.config.Packing.Equivalences
	remainingCapacity := capacity

	// Compute average weights per item so we can split across boxes
	avgSmall := 0.0
	if counts.Small > 0 && counts.SmallWeightOz > 0 {
		avgSmall = counts.SmallWeightOz / float64(counts.Small)
	}
	avgMedium := 0.0
	if counts.Medium > 0 && counts.MediumWeightOz > 0 {
		avgMedium = counts.MediumWeightOz / float64(counts.Medium)
	}
	avgLarge := 0.0
	if counts.Large > 0 && counts.LargeWeightOz > 0 {
		avgLarge = counts.LargeWeightOz / float64(counts.Large)
	}
	avgXL := 0.0
	if counts.XL > 0 && counts.XLWeightOz > 0 {
		avgXL = counts.XLWeightOz / float64(counts.XL)
	}

	// Pack XL items first
	if remaining.XL > 0 && equivalences["xlarge"] <= remainingCapacity {
		xlToPack := remainingCapacity / equivalences["xlarge"]
		if xlToPack > remaining.XL {
			xlToPack = remaining.XL
		}
		boxCounts.XL = xlToPack
		remaining.XL -= xlToPack
		remainingCapacity -= xlToPack * equivalences["xlarge"]
	}

	// Pack Large items
	if remaining.Large > 0 && equivalences["large"] <= remainingCapacity {
		largeToPack := remainingCapacity / equivalences["large"]
		if largeToPack > remaining.Large {
			largeToPack = remaining.Large
		}
		boxCounts.Large = largeToPack
		remaining.Large -= largeToPack
		remainingCapacity -= largeToPack * equivalences["large"]
	}

	// Pack Medium items
	if remaining.Medium > 0 && equivalences["medium"] <= remainingCapacity {
		mediumToPack := remainingCapacity / equivalences["medium"]
		if mediumToPack > remaining.Medium {
			mediumToPack = remaining.Medium
		}
		boxCounts.Medium = mediumToPack
		remaining.Medium -= mediumToPack
		remainingCapacity -= mediumToPack * equivalences["medium"]
	}

	// Pack Small items
	if remaining.Small > 0 && remainingCapacity > 0 {
		smallToPack := remainingCapacity
		if smallToPack > remaining.Small {
			smallToPack = remaining.Small
		}
		boxCounts.Small = smallToPack
		remaining.Small -= smallToPack
	}

	// Carry forward dimension guards for reporting
	boxCounts.SmallMaxDims = counts.SmallMaxDims
	boxCounts.MediumMaxDims = counts.MediumMaxDims
	boxCounts.LargeMaxDims = counts.LargeMaxDims
	boxCounts.XLMaxDims = counts.XLMaxDims
	remaining.SmallMaxDims = counts.SmallMaxDims
	remaining.MediumMaxDims = counts.MediumMaxDims
	remaining.LargeMaxDims = counts.LargeMaxDims
	remaining.XLMaxDims = counts.XLMaxDims

	// Split weights proportionally based on what we packed
	boxCounts.SmallWeightOz = avgSmall * float64(boxCounts.Small)
	boxCounts.MediumWeightOz = avgMedium * float64(boxCounts.Medium)
	boxCounts.LargeWeightOz = avgLarge * float64(boxCounts.Large)
	boxCounts.XLWeightOz = avgXL * float64(boxCounts.XL)

	remaining.SmallWeightOz = math.Max(counts.SmallWeightOz-boxCounts.SmallWeightOz, 0)
	remaining.MediumWeightOz = math.Max(counts.MediumWeightOz-boxCounts.MediumWeightOz, 0)
	remaining.LargeWeightOz = math.Max(counts.LargeWeightOz-boxCounts.LargeWeightOz, 0)
	remaining.XLWeightOz = math.Max(counts.XLWeightOz-boxCounts.XLWeightOz, 0)

	return boxCounts, remaining
}

func (p *Packer) Pack(counts ItemCounts) *PackingSolution {
	totalSmallUnits := p.SmallUnits(counts)

	slog.Debug("Pack: Starting packing calculation",
		"small", counts.Small,
		"medium", counts.Medium,
		"large", counts.Large,
		"xl", counts.XL,
		"total_small_units", totalSmallUnits)

	if totalSmallUnits == 0 {
		return &PackingSolution{
			Valid: false,
			Error: "no items to pack",
		}
	}

	singleBoxSolution := p.PackSingleBox(counts)
	if singleBoxSolution.Valid {
		slog.Debug("Pack: Single box solution found",
			"box_sku", singleBoxSolution.Boxes[0].Box.SKU,
			"box_name", singleBoxSolution.Boxes[0].Box.Name,
			"weight_oz", singleBoxSolution.Boxes[0].Weight,
			"box_cost", singleBoxSolution.TotalCost)
		return singleBoxSolution
	}

	slog.Debug("Pack: Single box solution not possible, trying multi-box")
	multiBoxSolution := p.PackMultipleBoxes(counts)

	if multiBoxSolution.Valid {
		slog.Debug("Pack: Multi-box solution found",
			"num_boxes", multiBoxSolution.TotalBoxes,
			"total_cost", multiBoxSolution.TotalCost)
		for i, box := range multiBoxSolution.Boxes {
			slog.Debug("Pack: Multi-box solution - box details",
				"box_index", i,
				"box_sku", box.Box.SKU,
				"box_name", box.Box.Name,
				"weight_oz", box.Weight,
				"items_small", box.ItemCounts.Small,
				"items_medium", box.ItemCounts.Medium,
				"items_large", box.ItemCounts.Large,
				"items_xl", box.ItemCounts.XL)
		}
	} else {
		slog.Debug("Pack: No valid packing solution found", "error", multiBoxSolution.Error)
	}

	return multiBoxSolution
}

func (p *Packer) ValidateItemDimensions(category string, length, width, height float64) error {
	guard, exists := p.config.Packing.DimensionGuard[category]
	if !exists {
		return fmt.Errorf("unknown category: %s", category)
	}

	if length > guard.L || width > guard.W || height > guard.H {
		return fmt.Errorf("item dimensions (%gx%gx%g) exceed %s category limits (%gx%gx%g)",
			length, width, height, category, guard.L, guard.W, guard.H)
	}

	return nil
}
