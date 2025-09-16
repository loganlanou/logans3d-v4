# Shipping Integration & Packaging Optimization System
## Product Requirements Document

## Executive Summary

This document outlines the requirements for implementing a comprehensive shipping integration and packaging optimization system for a 3D printing e-commerce business. The system will optimize shipping costs through intelligent packaging decisions and provide customers with multiple shipping options while enabling automated label printing.

## Business Context

### Problem Statement
- Shipping costs often equal or exceed product costs for 3D printed items
- Manual packaging decisions are inefficient and suboptimal
- Need for real-time shipping rate comparison for customers
- Requirement for automated shipping label generation
- Box size optimization is critical due to dimensional weight pricing

### Success Metrics
- Reduce shipping costs by 20% through optimal packaging
- Provide 3+ shipping options to customers at checkout
- Automate 95% of shipping label generation
- Reduce packaging waste by 30%

## Research Findings

### Shipping API Provider Recommendation: EasyPost

**Rationale:**
- Free tier supports up to 120,000 shipments annually
- Official Go SDK with comprehensive documentation at https://github.com/EasyPost/easypost-go
- Supports 100+ carriers including USPS, FedEx, UPS, DHL
- Up to 83% discounted shipping rates
- Active developer community and documentation at https://docs.easypost.com

**Alternative:** Shippo (if budget constraints exist)
- Free 30 labels/month, then $0.07/label
- 85+ carrier integration
- 4-6 hour integration time

**Why Not ShipStation:** 2025 policy changes require $99.99/month minimum for API access

### Box Purchasing Strategy Research
- Volume purchasing can save ~20% on packaging costs
- Standardization reduces operational complexity
- 3PL partnerships often provide better rates than direct purchasing
- Sustainability requirements: 85% of consumers expect eco-friendly packaging
- 82% willing to pay premium for sustainable packaging

## Technical Requirements

### 1. Shipping API Integration

#### Core Interfaces Needed
```go
// Core interfaces needed
type ShippingProvider interface {
    GetRates(shipment Shipment) ([]ShippingRate, error)
    PurchaseLabel(rateID string) (ShippingLabel, error)
    TrackPackage(trackingNumber string) (TrackingInfo, error)
}

type ShippingRate struct {
    Carrier      string
    Service      string
    Rate         float64
    DeliveryDays int
    RateID       string
}

type Shipment struct {
    FromAddress Address
    ToAddress   Address
    Parcel      Parcel
}

type Parcel struct {
    Length float64 // inches
    Width  float64 // inches
    Height float64 // inches
    Weight float64 // ounces
}
```

### 2. Item Size Classification System

#### Size Categories Configuration
```yaml
item_sizes:
  small:
    base_weight: 2.5      # ounces
    dimensions:
      length: 2.0         # inches
      width: 2.0
      height: 1.0
    packing_weight: 0.5

  medium:
    base_weight: 7.5      # 3x small
    dimensions:
      length: 3.0
      width: 3.0
      height: 2.0
    packing_weight: 0.8

  large:
    base_weight: 15.0     # 2x medium
    dimensions:
      length: 4.5
      width: 4.5
      height: 3.0
    packing_weight: 1.2

  extra_large:
    base_weight: 45.0     # 3x large
    dimensions:
      length: 6.0
      width: 6.0
      height: 4.5
    packing_weight: 2.0
```

#### Conversion Rules
- 3 Small items = 1 Medium item (volume and weight)
- 2 Medium items = 1 Large item
- 3 Large items = 1 Extra Large item

#### Product-Specific Shipping Overrides
Some products require special packing and cannot be consolidated with other items:

```yaml
product_shipping_overrides:
  # Product SKU or ID with custom shipping requirements
  "DELICATE-MINIATURE-SET":
    weight: 4.2               # ounces - exact weight
    dimensions:
      length: 8.0             # inches - exact dimensions
      width: 6.0
      height: 3.0
    requires_own_box: true    # Cannot be combined with other items
    box_type: "custom_padded" # Specific box type required
    packing_instructions: "Wrap each piece individually in bubble wrap"

  "LARGE-PROTOTYPE-MODEL":
    weight: 12.5
    dimensions:
      length: 14.0
      width: 10.0
      height: 8.0
    requires_own_box: true
    box_type: "large_reinforced"
    fragile: true
    packing_instructions: "Double-wall box with foam padding on all sides"

### 3. Packaging Optimization Algorithm

#### Core Algorithm Flow
```go
type OptimizationResult struct {
    Packages []Package
    TotalCost float64
    TotalWeight float64
}

type Package struct {
    BoxType         string
    Items           []ItemGroup
    Weight          float64
    Dimensions      Dimensions
    SpecialHandling string  // Packing instructions for override items
    Fragile         bool    // Special handling flag
}

func OptimizePackaging(order []OrderItem, config ShippingConfig) OptimizationResult {
    // Step 1: Separate override items from regular items
    overrideItems, regularItems := separateOverrideItems(order, config.ProductOverrides)

    // Step 2: Create packages for override items (each gets own box)
    overridePackages := createOverridePackages(overrideItems, config)

    // Step 3: Consolidate regular items using conversion rules
    consolidatedItems := consolidateItems(regularItems, config.ConversionRules)

    // Step 4: Generate packaging combinations for regular items
    combinations := generatePackagingCombinations(consolidatedItems, config.BoxSizes)

    // Step 5: Evaluate combinations and combine with override packages
    regularResult := evaluateCombinations(combinations, config)

    // Step 6: Combine override and regular packages
    return combineResults(overridePackages, regularResult)
}

func separateOverrideItems(items []OrderItem, overrides map[string]ProductShippingOverride) ([]OrderItem, []OrderItem) {
    var overrideItems, regularItems []OrderItem

    for _, item := range items {
        if override, exists := overrides[item.ProductID]; exists && override.RequiresOwnBox {
            overrideItems = append(overrideItems, item)
        } else {
            regularItems = append(regularItems, item)
        }
    }

    return overrideItems, regularItems
}

func createOverridePackages(items []OrderItem, config ShippingConfig) []Package {
    var packages []Package

    for _, item := range items {
        override := config.ProductOverrides[item.ProductID]

        // Each override item gets its own package
        for i := 0; i < item.Quantity; i++ {
            package := Package{
                BoxType: override.BoxType,
                Items: []ItemGroup{{
                    ProductID: item.ProductID,
                    Quantity: 1,
                }},
                Weight: override.Weight + getBoxWeight(override.BoxType, config),
                Dimensions: override.Dimensions,
                SpecialHandling: override.PackingInstructions,
                Fragile: override.Fragile,
            }
            packages = append(packages, package)
        }
    }

    return packages
}

func consolidateItems(items []OrderItem, rules ConversionRules) []ConsolidatedItem {
    counts := make(map[string]int)

    // Count items by size
    for _, item := range items {
        counts[item.Size] += item.Quantity
    }

    // Apply bottom-up consolidation
    // Small → Medium
    mediumFromSmall := counts["small"] / rules.SmallToMedium
    counts["medium"] += mediumFromSmall
    counts["small"] %= rules.SmallToMedium

    // Medium → Large
    largeFromMedium := counts["medium"] / rules.MediumToLarge
    counts["large"] += largeFromMedium
    counts["medium"] %= rules.MediumToLarge

    // Large → Extra Large
    extraLargeFromLarge := counts["large"] / rules.LargeToExtraLarge
    counts["extra_large"] += extraLargeFromLarge
    counts["large"] %= rules.LargeToExtraLarge

    return countsToItems(counts)
}
```

#### Optimization Factors
1. **Total Shipping Cost** (primary factor)
2. **Packaging Material Cost**
3. **Dimensional Weight Considerations**
4. **Carrier-Specific Rate Optimization**

### 4. Box Configuration System

#### Standard Box Sizes
```yaml
box_configs:
  small_box:
    internal_dims: {length: 6, width: 4, height: 2}
    max_weight: 16
    cost_per_unit: 0.45

  medium_box:
    internal_dims: {length: 8, width: 6, height: 4}
    max_weight: 32
    cost_per_unit: 0.65

  large_box:
    internal_dims: {length: 12, width: 9, height: 6}
    max_weight: 64
    cost_per_unit: 0.85

  extra_large_box:
    internal_dims: {length: 16, width: 12, height: 8}
    max_weight: 128
    cost_per_unit: 1.15
```

## Implementation Plan

### Phase 1: Foundation (Week 1-2)
1. Set up EasyPost account and API integration
2. Implement basic shipping rate retrieval
3. Create item size configuration system
4. Build basic packaging calculator

### Phase 2: Core Optimization (Week 3-4)
1. Implement item consolidation algorithm
2. Build multi-box optimization logic
3. Add carrier rate comparison
4. Create packaging cost calculator

### Phase 3: Integration (Week 5-6)
1. Integrate with existing checkout system
2. Add shipping option selection UI
3. Implement label generation and printing
4. Add order tracking capabilities

### Phase 4: Optimization (Week 7-8)
1. Add A/B testing for packaging decisions
2. Implement cost analytics and reporting
3. Fine-tune algorithm parameters
4. Add international shipping support

## Technical Architecture

### Database Schema
```sql
-- Shipping configurations
CREATE TABLE shipping_configs (
    id INTEGER PRIMARY KEY,
    item_size VARCHAR(20),
    base_weight DECIMAL(8,2),
    length DECIMAL(8,2),
    width DECIMAL(8,2),
    height DECIMAL(8,2),
    packing_weight DECIMAL(8,2)
);

-- Product shipping overrides
CREATE TABLE product_shipping_overrides (
    id INTEGER PRIMARY KEY,
    product_id VARCHAR(100) UNIQUE,
    weight DECIMAL(8,2),
    length DECIMAL(8,2),
    width DECIMAL(8,2),
    height DECIMAL(8,2),
    requires_own_box BOOLEAN DEFAULT false,
    box_type VARCHAR(50),
    packing_instructions TEXT,
    fragile BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_product_id (product_id)
);

-- Box configurations
CREATE TABLE box_configs (
    id INTEGER PRIMARY KEY,
    box_type VARCHAR(50),
    length DECIMAL(8,2),
    width DECIMAL(8,2),
    height DECIMAL(8,2),
    max_weight DECIMAL(8,2),
    cost_per_unit DECIMAL(8,4)
);

-- Shipping rates cache
CREATE TABLE shipping_rates_cache (
    id INTEGER PRIMARY KEY,
    rate_hash VARCHAR(64),
    carrier VARCHAR(50),
    service VARCHAR(100),
    rate DECIMAL(8,2),
    delivery_days INTEGER,
    created_at TIMESTAMP,
    INDEX idx_hash (rate_hash),
    INDEX idx_created (created_at)
);

-- Orders and shipments
CREATE TABLE order_shipments (
    id INTEGER PRIMARY KEY,
    order_id INTEGER,
    tracking_number VARCHAR(100),
    carrier VARCHAR(50),
    service VARCHAR(100),
    cost DECIMAL(8,2),
    label_url VARCHAR(500),
    created_at TIMESTAMP
);
```

### Configuration Management
- YAML-based configuration files for easy updates
- Environment-specific settings (dev/staging/prod)
- Hot-reload capability for shipping configurations
- Version control for configuration changes

### Configuration File Structure
```go
type ItemSizeConfig struct {
    Size        string  // "small", "medium", "large", "extra_large"
    BaseWeight  float64 // Weight in ounces
    BaseDims    Dimensions
    PackingWeight float64 // Additional weight for packaging materials
}

type Dimensions struct {
    Length float64 // inches
    Width  float64 // inches
    Height float64 // inches
}

type ShippingConfig struct {
    ItemSizes       []ItemSizeConfig
    BoxSizes        []BoxConfig
    ConversionRules ConversionMatrix
    ProductOverrides map[string]ProductShippingOverride
}

type ProductShippingOverride struct {
    ProductID           string
    Weight              float64     // Exact weight in ounces
    Dimensions          Dimensions  // Exact dimensions
    RequiresOwnBox      bool       // Cannot be combined with other items
    BoxType             string     // Specific box type required
    PackingInstructions string     // Special packing instructions
    Fragile             bool       // Fragile item flag
}

type BoxConfig struct {
    BoxType      string
    Dimensions   Dimensions
    MaxWeight    float64
    BoxWeight    float64 // Empty box weight
    CostPer100   float64 // Cost per 100 boxes
}

type ConversionMatrix struct {
    SmallToMedium      int // 3
    MediumToLarge      int // 2
    LargeToExtraLarge  int // 3
}
```

## Business Requirements

### Customer Experience
1. **Checkout Integration**: Display 3+ shipping options with estimated delivery dates
2. **Real-time Rates**: Calculate shipping costs dynamically based on destination
3. **Tracking Integration**: Provide tracking numbers and status updates
4. **Mobile Optimization**: Ensure shipping selection works on mobile devices

### Operational Requirements
1. **Label Printing**: Automated PDF label generation for thermal printers
2. **Batch Processing**: Support for printing multiple labels at once
3. **Error Handling**: Graceful handling of API failures with fallback options
4. **Audit Trail**: Log all shipping decisions and costs for analysis

### Cost Management
1. **Rate Shopping**: Always select most cost-effective option
2. **Carrier Diversification**: Avoid over-dependence on single carrier
3. **Analytics Dashboard**: Track shipping costs and optimization savings
4. **Seasonal Adjustments**: Account for peak season rate changes

## Smart Box Selection Rules

1. **Dimensional Weight Check**: Compare actual weight vs. dimensional weight for each carrier
2. **Multi-Package vs. Single Package**: Compare cost of multiple smaller boxes vs. one larger box
3. **Fragility Consideration**: Prefer smaller boxes for fragile items to reduce movement
4. **Carrier-Specific Optimization**: Different carriers have different sweet spots for dimensions

## Edge Case Handling

- **Oversized Items**: Items that don't fit standard consolidation rules
- **Mixed Orders**: Orders with non-consolidatable combinations
- **Weight Limits**: When consolidated weight exceeds box capacity
- **International Shipping**: Different dimensional weight rules
- **Override Items in Mixed Orders**: Orders containing both override and regular items
- **Multiple Override Items**: Orders with multiple different override products
- **Custom Box Requirements**: Override items requiring specialized packaging not in standard inventory

## Risk Mitigation

### Technical Risks
- **API Rate Limits**: Implement caching and request queuing
- **Carrier API Downtime**: Multi-carrier failover system
- **Package Size Miscalculation**: Add manual override capabilities
- **Weight Accuracy**: Regular calibration of weight estimates

### Business Risks
- **Shipping Cost Volatility**: Regular rate updates and alerts
- **International Regulations**: Compliance checking for global shipping
- **Customer Dissatisfaction**: Clear shipping policy communication
- **Inventory Impact**: Integration with inventory management system

## Success Measurement

### Key Performance Indicators (KPIs)
1. **Average Shipping Cost per Order**: Target 15% reduction
2. **Customer Shipping Option Selection**: Track preference patterns
3. **Label Generation Success Rate**: Target 99.5% automation
4. **Packaging Efficiency**: Measure void fill percentage reduction

### Analytics Implementation
- Daily shipping cost reports
- Weekly packaging optimization analysis
- Monthly carrier performance comparison
- Quarterly ROI assessment on shipping optimization

## API Integration Details

### EasyPost Setup Steps
1. Create account at https://easypost.com
2. Obtain API keys (test and production)
3. Install Go SDK: `go get github.com/EasyPost/easypost-go`
4. Implement rate shopping workflow
5. Set up webhook endpoints for tracking updates

### Sample Integration Code
```go
import "github.com/EasyPost/easypost-go"

client := easypost.New("your-api-key")

// Create shipment and get rates
shipment, err := client.CreateShipment(&easypost.Shipment{
    ToAddress: toAddr,
    FromAddress: fromAddr,
    Parcel: parcel,
})

// Buy cheapest rate
rate := shipment.LowestRate()
shipment, err = client.BuyShipment(shipment.ID, rate.ID)

// Print label
labelURL := shipment.PostageLabel.LabelURL
```

---

This comprehensive PRD provides all the technical specifications, business requirements, and implementation guidance needed to build an intelligent shipping integration and packaging optimization system for your 3D printing business.