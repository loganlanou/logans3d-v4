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

### Shipping API Provider Recommendation: ShipEngine/ShipStation

**Rationale:**
- Free developer account with no credit card required for testing
- Official SDKs in multiple languages including Go (https://github.com/ShipEngine/shipengine-go)
- Supports 100+ carriers including USPS, FedEx, UPS, DHL Express
- Discounted shipping rates through carrier partnerships
- Comprehensive OpenAPI 3.0 specification at https://github.com/ShipEngine/shipengine-openapi
- Active developer community and robust documentation at https://shipengine.github.io/shipengine-openapi/

**Key Features:**
- Rate comparison across multiple carriers
- Label creation and management with void/refund capabilities
- Real-time package tracking with webhook support
- Address validation for 160+ countries
- Pickup scheduling and service point lookup
- Custom package type creation

**Why ShipEngine over EasyPost:** Better pricing structure for growing businesses, more comprehensive carrier integrations, and superior webhook/tracking capabilities

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

### 2. Configuration-Driven Data Model

#### Product Class Configuration (YAML)
```yaml
classes:
  S:  { weight_oz: 2.0,  dim_in: [4, 4, 1.5] }
  M:  { weight_oz: 6.0,  dim_in: [6, 6, 3] }
  L:  { weight_oz: 16.0, dim_in: [8, 8, 4] }
  XL: { weight_oz: 36.0, dim_in: [12, 10, 6] }

combine_rules:               # Equivalences apply to volume & weight
  S_to_M:  { S: 3, result: M }
  M_to_L:  { M: 2, result: L }
  L_to_XL: { L: 3, result: XL }

boxes:                        # Actual vendor specifications from The Boxery
  S:
    sku: "BX-SM"
    vendor: "The Boxery"
    vendor_sku: "CXBSS18"
    url: "https://www.theboxery.com/Product.asp?Name=8%27%27x6%27%27x4%27%27+Corrugated+Shipping+Boxes&Product=CXBSS18&d=105505"
    inner_in: [8, 6, 4]
    ect: 32
    pack_qty: 25
    unit_price_usd: 0.38
    bundle_price_usd: 9.50
    volume_in3: 192
    cost_per_in3_usd: 0.001979
    max_wt_lb: 10
    notes: "Standard brown RSC; interior dims; sold in 25-pack."

  M:
    sku: "BX-MD"
    vendor: "The Boxery"
    vendor_sku: "CXBSM1294"
    url: "https://www.theboxery.com/Product.asp?Name=12%27%27x9%27%27x4%27%27+Corrugated+Shipping+Boxes&Product=CXBSM1294&d=1055"
    inner_in: [12, 9, 4]
    ect: 32
    pack_qty: 25
    unit_price_usd: 0.62
    bundle_price_usd: 15.50
    volume_in3: 432
    cost_per_in3_usd: 0.001435
    max_wt_lb: 20
    notes: "Best seller size; interior dims; 25-pack."

  L:
    sku: "BX-LG"
    vendor: "The Boxery"
    vendor_sku: "CXBSM146"
    url: "https://www.theboxery.com/Product.asp?Name=14x10.5x6+Corrugated+Shipping+Boxes&Product=CXBSM146&d=1001"
    inner_in: [14, 10.5, 6]
    ect: 32
    pack_qty: 25
    unit_price_usd: 0.77
    bundle_price_usd: 19.25
    volume_in3: 882
    cost_per_in3_usd: 0.000873
    max_wt_lb: 30
    notes: "Interior dims; 25-pack."

  XL:
    sku: "BX-XL"
    vendor: "The Boxery"
    vendor_sku: "CXBSM18128"
    url: "https://www.theboxery.com/Product.asp?Name=18%27%27x12%27%27x8%27%27+Corrugated+Shipping+Boxes&Product=CXBSM18128&d=1055"
    inner_in: [18, 12, 8]
    ect: 32
    pack_qty: 25
    unit_price_usd: 1.12
    bundle_price_usd: 28.00
    volume_in3: 1728
    cost_per_in3_usd: 0.000648
    max_wt_lb: 40
    notes: "Interior dims; 25-pack."

packing_prefs:
  max_boxes: 6
  allow_softpack: true
  softpack_thickness_in: 0.5
  respect_overrides: true
  evaluate_split_vs_consolidate: true
  prefer_usps_cubic_when_eligible: true
  max_candidate_plans: 5
```

#### Per-SKU Shipping Overrides
Items with special packing that must ship alone or in a fixed box:

```yaml
product_overrides:
  "SKU-DRAGON-SWORD-42":
    requires_separate_box: true
    own_box:
      dim_in: [36, 6, 4]         # L×W×H interior used for rating; outer derived via padding
      weight_lb: 7.5             # includes inserts/foam
      padding_in: 0.25           # added per side → outer ship dims
      max_per_box: 1             # 1 item per parcel
      orientation_lock: true     # no 90° rotation
      fragile: true
      insurance_value_usd: 180.00
      signature_required: false
    fallback_box_sku: "BX-LG"    # optional; used if dim_in omitted

  "SKU-STATUE-XL-001":
    requires_separate_box: true
    own_box:
      box_sku: "BX-XL"
      weight_lb: 12.0
      max_per_box: 1

  "DELICATE-MINIATURE-SET":
    requires_separate_box: true
    own_box:
      dim_in: [8, 6, 3]
      weight_lb: 0.3
      padding_in: 0.5
      max_per_box: 1
      fragile: true
      insurance_value_usd: 45.00
    packing_instructions: "Wrap each piece individually in bubble wrap"
```

#### Order Packing Output Example
```json
{
  "order_id": "ORD-10001",
  "carton_plan": [
    {
      "box_sku": "BX-MD",
      "inner_dim_in": [12, 9, 4],
      "items": [{"class":"S","qty":4},{"class":"M","qty":1}],
      "packed_weight_lb": 2.2,
      "ship_dim_in": [12, 9, 4]
    },
    {
      "box_sku": "BX-XL",
      "items": [{"sku":"SKU-STATUE-XL-001","qty":1}],
      "packed_weight_lb": 12.0,
      "ship_dim_in": [18, 12, 8],
      "override": true
    }
  ]
}
```

### 3. Cartonization Algorithm with Split-vs-Consolidate Evaluation

#### Enhanced Algorithm Flow
```go
type CartonPlan struct {
    OrderID   string
    Packages  []Package
    TotalCost float64
    Strategy  string  // "split-heavy", "consolidated", "softpack-first"
    BoxCount  int
}

type Package struct {
    BoxSKU          string
    InnerDimensions []float64  // [L, W, H] inches
    ShipDimensions  []float64  // Outer dimensions for rating
    Items           []ItemGroup
    PackedWeight    float64    // pounds
    Override        bool       // true if from product override
    SpecialHandling string
    Fragile         bool
    Insurance       float64
    Signature       bool
}

func OptimizeCartonization(order []OrderItem, config ShippingConfig) []CartonPlan {
    // Step 1: Split Overrides First
    overrideItems, regularItems := extractOverrideItems(order, config.ProductOverrides)
    overridePackages := createOverridePackages(overrideItems, config)

    // Step 2: Class Normalization
    normalizedItems := normalizeToClasses(regularItems, config.Classes)

    // Step 3: Equivalence Folding
    consolidatedItems := applyEquivalenceRules(normalizedItems, config.CombineRules)

    // Step 4: Generate Strategy Variants
    strategies := []string{"split-heavy", "consolidated", "softpack-first"}
    var candidatePlans []CartonPlan

    for _, strategy := range strategies {
        packages := generatePackagesByStrategy(consolidatedItems, config, strategy)

        // Always include override packages in every plan
        allPackages := append(overridePackages, packages...)

        if len(allPackages) <= config.PackingPrefs.MaxBoxes {
            plan := CartonPlan{
                OrderID:  order[0].OrderID,
                Packages: allPackages,
                Strategy: strategy,
                BoxCount: len(allPackages),
            }
            candidatePlans = append(candidatePlans, plan)
        }
    }

    // Step 5: Cost Preview per Plan
    for i := range candidatePlans {
        candidatePlans[i].TotalCost = calculatePlanCost(candidatePlans[i], config)
    }

    // Step 6: Choose Best Plans (top 3 for checkout)
    return selectBestPlans(candidatePlans, config.PackingPrefs.MaxCandidatePlans)
}

func generatePackagesByStrategy(items []ConsolidatedItem, config ShippingConfig, strategy string) []Package {
    switch strategy {
    case "split-heavy":
        // Prefer many small/medium cartons (favors USPS Cubic tiers)
        return packSplitHeavy(items, config)
    case "consolidated":
        // Merge into as few large boxes as possible (watch DIM weight)
        return packConsolidated(items, config)
    case "softpack-first":
        // Pack softpacks where eligible before boxes
        return packSoftpackFirst(items, config)
    default:
        return packFirstFitDecreasing(items, config)
    }
}

func calculatePlanCost(plan CartonPlan, config ShippingConfig) float64 {
    totalCost := 0.0

    for _, pkg := range plan.Packages {
        // Calculate billable weight using DIM weight formula
        billableWeight := calculateBillableWeight(pkg, config.DIMDivisors)

        // Get shipping rates from provider
        rates := getShippingRates(pkg, billableWeight)
        totalCost += rates.LowestRate

        // Add box material cost
        boxCost := getBoxCost(pkg.BoxSKU, config)
        totalCost += boxCost
    }

    return totalCost
}

func calculateBillableWeight(pkg Package, dimDivisors map[string]int) float64 {
    // billable_weight = max(actual, ceil((L×W×H)/divisor))
    dims := pkg.ShipDimensions
    dimWeight := math.Ceil((dims[0] * dims[1] * dims[2]) / float64(dimDivisors["default"]))
    return math.Max(pkg.PackedWeight, dimWeight)
}

func applyEquivalenceRules(items []ConsolidatedItem, rules CombineRules) []ConsolidatedItem {
    counts := make(map[string]int)

    // Count items by class
    for _, item := range items {
        counts[item.Class] += item.Quantity
    }

    // Apply combine rules greedily, largest reductions first
    // 3×S → 1×M
    if rule, exists := rules["S_to_M"]; exists {
        converted := counts["S"] / rule.S
        counts["M"] += converted
        counts["S"] %= rule.S
    }

    // 2×M → 1×L
    if rule, exists := rules["M_to_L"]; exists {
        converted := counts["M"] / rule.M
        counts["L"] += converted
        counts["M"] %= rule.M
    }

    // 3×L → 1×XL
    if rule, exists := rules["L_to_XL"]; exists {
        converted := counts["L"] / rule.L
        counts["XL"] += converted
        counts["L"] %= rule.L
    }

    return countsToConsolidatedItems(counts)
}
```

#### Split vs Consolidate Decision Logic
```go
func selectOptimalStrategy(plans []CartonPlan) CartonPlan {
    // Bias toward split-heavy when consolidating crosses DIM threshold
    splitPlan := findPlanByStrategy(plans, "split-heavy")
    consolidatedPlan := findPlanByStrategy(plans, "consolidated")

    if splitPlan != nil && consolidatedPlan != nil {
        // If consolidated plan triggers higher DIM weight bracket, prefer split
        if consolidatedPlan.TotalCost > splitPlan.TotalCost*1.15 {
            return *splitPlan
        }
    }

    // Otherwise choose cheapest plan
    return findCheapestPlan(plans)
}
```

#### Optimization Factors (Updated)
1. **Total Shipping Cost** (primary factor - includes DIM weight calculations)
2. **Split vs Consolidate Strategy** (DIM weight threshold awareness)
3. **USPS Cubic Tier Eligibility** (for small packages)
4. **Packaging Material Cost**
5. **Box Count Constraints** (respect max_boxes preference)

### 4. Checkout Rate Shopping Flow

#### Enhanced Rate Shopping Process
1. **Build 2-3 Candidate Carton Plans** using the cartonization algorithm
2. **For Each Plan**: Call shipping provider API for each parcel with flags:
   - Residential delivery
   - Signature required (based on value thresholds or override flags)
   - Insurance (based on value thresholds or override requirements)
   - Saturday delivery option
3. **Deduplicate by (carrier, service, ETA)** and present top options:
   - **Cheapest**: Lowest total cost across all packages
   - **Best Value**: Balanced cost and delivery time (Pareto optimization)
   - **Fastest**: Shortest max delivery time, with cost as tiebreaker
4. **On Selection**: Persist `rate_id`/`shipment_id` for each package for label purchase

### 5. Fulfillment & Label Management

#### Label Purchase and Printing
- **On Order Capture**: Buy labels for each parcel using stored rate tokens
- **Persist Assets**: Store PDF/ZPL labels, tracking numbers, and rating I/O for audit
- **Packer UI**: Print labels + packing slip; display tracking information

#### Label Refunds and Voids (New Capability)
**Objective**: If an order is canceled or a label is unused, void/refund labels within provider windows and credit back to balance.

**Refund Behavior**:
```go
type LabelRefundService struct {
    provider Provider
    storage  LabelStorage
}

func (s *LabelRefundService) RefundLabels(orderID string) error {
    labels := s.storage.GetLabelsByOrder(orderID)

    for _, label := range labels {
        // Check if pay-on-scan/use (not charged until first scan)
        if s.provider.IsPayOnScan(label) {
            label.Status = "not_charged"
            continue
        }

        // Attempt void/refund within provider window
        result, err := s.provider.VoidLabel(context.Background(), label.ID)
        if err != nil {
            label.RefundStatus = "pending"
            label.RefundReason = err.Error()
        } else {
            label.RefundStatus = result.Status  // "approved", "denied", "pending"
            label.CreditedAmount = result.Amount
            label.VoidBy = result.VoidBy
        }

        s.storage.UpdateLabel(label)
    }

    return nil
}
```

**Nightly Refund Retry Job**:
- Process labels with `refund_status = "pending"`
- Retry refund attempts until approved or expired
- Track void_by deadline per label (strictest cutoff from provider)

**Admin UI Features**:
- **Action**: "Void/Refund Label" with reasons (canceled, address error, duplicate)
- **Display**: Eligibility window, current status, and credited amount
- **Bulk Operations**: Cancel entire order with automatic refund attempts

## Implementation Plan

### Phase 1: Foundation (Week 1-2)
1. Set up ShipEngine account and API integration
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

### ShipEngine Setup Steps
1. Create free developer account at https://www.shipengine.com/docs/getting-started/
2. Obtain API keys (test and production environments)
3. Install Go SDK: `go get github.com/ShipEngine/shipengine-go`
4. Connect carrier accounts (USPS, FedEx, UPS, etc.)
5. Implement rate shopping workflow using `/v1/rates` endpoint
6. Set up webhook endpoints for tracking updates
7. Configure carrier-specific settings and DIM divisors

### 6. Provider Integration Interface

**Primary Provider**: ShipEngine/ShipStation API

**Provider Interface (Go Implementation)**:
```go
type Provider interface {
    GetRates(ctx context.Context, shipments []Shipment, opts RateOpts) ([]Rate, error)
    BuyLabel(ctx context.Context, rateID string) (Label, error)
    VoidLabel(ctx context.Context, labelID string) (VoidResult, error)
    Track(ctx context.Context, trackingNumber string) (Tracking, error)
    ValidateAddress(ctx context.Context, addr Address) (Address, error)
    // Optional capabilities:
    IsPayOnScan(label Label) bool
}

type RateOpts struct {
    Residential   bool
    Signature     bool
    Insurance     float64
    Saturday      bool
    IdempotencyKey string
}

type VoidResult struct {
    Status        string    // "approved", "denied", "pending"
    Amount        float64   // credited amount
    VoidBy        time.Time // deadline for void eligibility
    Reason        string    // denial reason if applicable
}

type Label struct {
    ID            string
    TrackingNumber string
    LabelURL      string
    Cost          float64
    VoidBy        time.Time
    RefundStatus  string
    CreditedAmount float64
}
```

**Key Integration Requirements**:
- **Idempotency**: All purchase/void requests must include idempotency keys
- **DIM Divisors**: Configure per-carrier/service with admin override capability
- **Error Handling**: Graceful degradation with fallback providers
- **Rate Caching**: Cache rates for identical shipment parameters (15-minute TTL)

### Sample ShipEngine Integration
```go
import "github.com/ShipEngine/shipengine-go"

client := shipengine.NewClient("your-api-key")

// Create shipment and get rates for all packages in plan
for _, pkg := range cartonPlan.Packages {
    // Create rate request
    rateRequest := &shipengine.RateRequest{
        RateOptions: shipengine.RateOptions{
            CarrierIds: []string{"carrier-id-1", "carrier-id-2"},
        },
        Shipment: shipengine.Shipment{
            ShipTo: shipengine.Address{
                Name:         toAddr.Name,
                AddressLine1: toAddr.Street1,
                CityLocality: toAddr.City,
                StateProvince: toAddr.State,
                PostalCode:   toAddr.ZIP,
                CountryCode:  toAddr.Country,
            },
            ShipFrom: shipengine.Address{
                Name:         fromAddr.Name,
                AddressLine1: fromAddr.Street1,
                CityLocality: fromAddr.City,
                StateProvince: fromAddr.State,
                PostalCode:   fromAddr.ZIP,
                CountryCode:  fromAddr.Country,
            },
            Packages: []shipengine.Package{
                {
                    Weight: shipengine.Weight{
                        Value: pkg.PackedWeight,
                        Unit:  "pound",
                    },
                    Dimensions: shipengine.Dimensions{
                        Length: pkg.ShipDimensions[0],
                        Width:  pkg.ShipDimensions[1],
                        Height: pkg.ShipDimensions[2],
                        Unit:   "inch",
                    },
                },
            },
        },
    }

    rates, err := client.GetRates(context.Background(), rateRequest)
    if err != nil {
        return err
    }

    // Store rates for checkout selection
    storeRatesForPackage(pkg.ID, rates.RateResponse)
}

// On checkout selection, buy labels
func purchaseLabels(selectedRates []RateSelection) error {
    for _, selection := range selectedRates {
        labelRequest := &shipengine.CreateLabelRequest{
            RateId: selection.RateID,
        }

        label, err := client.CreateLabel(context.Background(), labelRequest)
        if err != nil {
            return err
        }

        // Store label for printing and tracking
        storeLabelForPackage(selection.PackageID, label)
    }
    return nil
}

// Void labels for refunds
func voidLabels(labelIDs []string) error {
    for _, labelID := range labelIDs {
        err := client.VoidLabel(context.Background(), labelID)
        if err != nil {
            return err
        }
    }
    return nil
}
```

## Acceptance Criteria

### 1. Split vs Consolidate Evaluation
- **AC-1.1**: Given a cart that fits in 1×L vs 2×M boxes, system selects the cheaper plan based on actual shipping rates
- **AC-1.2**: Given 1×XL vs 3×S scenario, system prefers split strategy if DIM weight makes XL more expensive
- **AC-1.3**: System respects `max_boxes` preference and excludes plans exceeding this limit
- **AC-1.4**: USPS Cubic tier eligibility is correctly identified and biases toward split-heavy strategy

### 2. Product Shipping Overrides
- **AC-2.1**: An override SKU ships alone with specified dimensions/weight, respecting `max_per_box` limits
- **AC-2.2**: Orientation lock prevents 90° rotations during box fitting calculations
- **AC-2.3**: Override items with `requires_separate_box: true` never consolidate with regular items
- **AC-2.4**: Insurance and signature requirements from overrides propagate to shipping labels

### 3. Label Purchase and Management
- **AC-3.1**: Selected checkout option purchases the exact rated service and generates PDF/ZPL labels
- **AC-3.2**: All label purchase requests include idempotency keys to prevent duplicates
- **AC-3.3**: Labels store complete audit trail: rating inputs/outputs, purchase timestamp, costs
- **AC-3.4**: Packer UI displays labels, packing instructions, and tracking information

### 4. Refunds and Voids
- **AC-4.1**: Canceling an order triggers automatic refund attempts for all associated labels
- **AC-4.2**: Pay-on-scan labels are marked as `not_charged` and skip refund process
- **AC-4.3**: Labels past `void_by` deadline show as ineligible with clear reason messaging
- **AC-4.4**: Nightly job retries pending refunds until approved/expired
- **AC-4.5**: Admin UI shows void eligibility window, status, and credited amounts

### 5. Configuration Management
- **AC-5.1**: YAML configuration changes hot-reload without service restart
- **AC-5.2**: Class equivalence rules (3×S→1×M, 2×M→1×L, 3×L→1×XL) apply correctly
- **AC-5.3**: DIM divisors can be configured per carrier/service with admin overrides
- **AC-5.4**: Box configurations include interior dimensions, weight limits, and costs

## Test Matrix

### Class Mix Testing
- **Single Classes**: S×1 through S×9, M×2, L×3, XL edge cases
- **Mixed Orders**: 2×S + 1×M, 3×S + 2×M + 1×L combinations
- **Softpack vs Box**: Items eligible for softpack vs rigid box requirements
- **Weight Limits**: Orders exceeding individual box weight limits

### Override Product Testing
- **Single Override**: One override item with fixed `box_sku`
- **Custom Dimensions**: Override with `dim_in + padding_in` calculations
- **Multi-Quantity**: Override items with `max_per_box: 2` constraints
- **Insurance/Signature**: Propagation of special handling requirements
- **Mixed Orders**: Override items combined with regular class-based items

### Split vs Consolidate Scenarios
- **1×L vs 2×M**: Compare costs when consolidation vs split strategies differ
- **1×XL vs 3×S**: DIM weight threshold testing
- **Softpack Multiples**: Multiple softpacks vs single box consolidation
- **Max Boxes**: Enforcement of `max_boxes` constraint with plan elimination

### Refund Testing
- **Eligible Windows**: Labels within void deadline
- **Expired Windows**: Labels past void deadline
- **Pay-on-Scan**: Labels not yet charged by carrier
- **Denial Reasons**: Various refund denial scenarios and handling

## Open Questions (Claude to Confirm)

1. **DIM Divisors**: Exact divisors per carrier/service for launch (default 139/166 vs carrier-specific)
2. **Box Catalog**: Final interior dimensions and USPS Cubic tier qualifications
3. **Default Services**: Which services to show at checkout (USPS Ground Advantage/Priority, UPS Ground, FedEx Ground Economy)
4. **Surcharge Configuration**: Default insurance thresholds, signature requirements, Saturday delivery, residential flags
5. **Manual Override UI**: Do we need admin capability to force specific packaging plan/service at checkout?

## Implementation Milestones (Updated)

### M1: Configuration + Cartonization MVP (Weeks 1-3)
- Implement YAML configuration system with class definitions
- Build core cartonization algorithm with equivalence rules
- Create sandbox rate integration with ShipEngine
- Basic override item handling

### M2: Split-vs-Consolidate + Checkout Integration (Weeks 4-6)
- Implement strategy evaluation (split-heavy, consolidated, softpack-first)
- Production rate shopping with 2-3 plan candidates
- Checkout UI for shipping option selection
- DIM weight calculations and carrier-specific divisors

### M3: Label Management + Fulfillment UI (Weeks 7-9)
- Label purchase with idempotency
- Packer UI with label printing and tracking
- Refund/void capability with admin controls
- Audit trail and error handling

### M4: Optimization + USPS Cubic (Weeks 10-12)
- USPS Cubic tier detection and optimization
- Box assortment tuning based on actual order patterns
- Performance optimization and rate caching
- International shipping foundation (if needed)

---

This comprehensive PRD provides all the technical specifications, business requirements, and implementation guidance needed to build an intelligent shipping integration and packaging optimization system with advanced split-vs-consolidate evaluation, product overrides, and label management capabilities.