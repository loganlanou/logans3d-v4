# Product Variants Requirements & Design Document

**Date:** 2025-11-20
**Status:** Planning
**Goal:** Implement flexible product variant system with image-based color selection and size options

---

## Overview

Add product variant support to allow customers to select different colors and sizes of products (e.g., T-Rex dinosaur in Red/Large or Blue/Small). The system must be flexible enough to support additional variant types in the future (materials, custom finishes, etc.).

**Key Inspiration:** Amazon's variant selection UI where each color variant shows the actual product image.

---

## Requirements & Decisions

### Functional Requirements

1. **Variant Types (Phase 1):**
   - ✅ Color (with product images)
   - ✅ Size (no images, button selection)
   - ❌ Material/Finish (deferred - all products are PLA for now)
   - ❌ Custom attributes (deferred to future)

2. **Product Types:**
   - Simple products (no variants) - single price, single SKU
   - Variant products - multiple color+size combinations, each with own SKU

3. **Image Handling:**
   - ⚠️ **CRITICAL:** Colors have product images, sizes don't
   - Each color can have multiple images (gallery)
   - One primary image per color (shown in selector)
   - Clicking a color image selects that color AND updates main gallery
   - NOT using color swatches - using actual product photos

4. **Pricing:**
   - Base price stored on product
   - Price adjustments stored on SKU (+$0 for Small/Medium, +$2 for Large, +$5 for XL)
   - Effective price = base price + adjustment

5. **Inventory:**
   - Stock tracked per SKU (color+size combination)
   - Out of stock variants disabled in UI
   - Unavailable combinations hidden or grayed out

6. **SKU Generation:**
   - Format: `{BASE_SKU}-{COLOR_CODE}-{SIZE_CODE}`
   - Example: `TREX-RED-LG`, `DINO-BLU-SM`
   - Auto-generated but allow manual override
   - Must be unique across entire catalog

### Non-Functional Requirements

1. **Data Efficiency:**
   - Minimize redundant data entry
   - Shared data at product level (name, description)
   - Variant-specific data at SKU level (price adjustment, stock)

2. **Flexibility:**
   - Easy to add new colors without schema changes
   - Easy to add new sizes without schema changes
   - Extensible to future variant types (material, finish, etc.)

3. **Admin UX:**
   - Bulk SKU generation (create all color×size combinations at once)
   - Easy image upload per color
   - Visual matrix view of variants

4. **Customer UX:**
   - Amazon-style image grid for color selection
   - Button group for size selection
   - Real-time price and stock updates
   - Disable unavailable options

### Technology Decisions

- **Database:** SQLite with flexible attribute system (not hardcoded columns)
- **Stripe Integration:** Use `price_data` for dynamic variant pricing
- **Frontend:** Alpine.js for variant selection logic
- **UI Components:** TemplUI for admin interface
- **Image Storage:** File system (`/public/images/products/variants/`)

---

## Database Schema Design

### Architecture: Three-Tier Flexible System

**Why:** Allows adding new variant types (material, custom attributes) without schema changes.

### Schema

```sql
-- Variant attribute types (color, size, future: material, finish)
CREATE TABLE variant_attributes (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,              -- 'color', 'size'
    display_name TEXT NOT NULL,             -- 'Color', 'Size'
    has_images BOOLEAN DEFAULT FALSE,       -- TRUE for color, FALSE for size
    display_order INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Attribute values (red, blue, small, large)
CREATE TABLE variant_attribute_values (
    id TEXT PRIMARY KEY,
    attribute_id TEXT NOT NULL REFERENCES variant_attributes(id) ON DELETE CASCADE,
    value TEXT NOT NULL,                    -- 'red', 'small'
    display_name TEXT NOT NULL,             -- 'Red', 'Small'
    hex_color TEXT,                         -- Optional: for future swatch fallback
    display_order INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(attribute_id, value)
);

-- Images for attribute values (colors have images, sizes don't)
CREATE TABLE variant_attribute_images (
    id TEXT PRIMARY KEY,
    attribute_value_id TEXT NOT NULL REFERENCES variant_attribute_values(id) ON DELETE CASCADE,
    image_url TEXT NOT NULL,                -- Filename only: 'trex-red-1.jpg'
    display_order INTEGER DEFAULT 0,
    is_primary BOOLEAN DEFAULT FALSE,       -- First image shown in selector
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Products table updates
ALTER TABLE products ADD COLUMN has_variants BOOLEAN DEFAULT FALSE;

-- Rename existing table for clarity
ALTER TABLE product_variants RENAME TO product_skus;

-- SKUs = purchasable items (specific color+size combinations)
-- Existing columns: id, product_id, name, sku, price_adjustment_cents, stock_quantity
ALTER TABLE product_skus ADD COLUMN is_active BOOLEAN DEFAULT TRUE;

-- Link SKUs to their attribute values
CREATE TABLE product_sku_attributes (
    id TEXT PRIMARY KEY,
    product_sku_id TEXT NOT NULL REFERENCES product_skus(id) ON DELETE CASCADE,
    attribute_id TEXT NOT NULL REFERENCES variant_attributes(id),
    attribute_value_id TEXT NOT NULL REFERENCES variant_attribute_values(id),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(product_sku_id, attribute_id)    -- Each SKU has one value per attribute
);

-- Indexes
CREATE INDEX idx_product_skus_product_id ON product_skus(product_id);
CREATE INDEX idx_product_skus_sku ON product_skus(sku);
CREATE INDEX idx_product_sku_attributes_sku_id ON product_sku_attributes(product_sku_id);
CREATE INDEX idx_variant_attribute_values_attribute_id ON variant_attribute_values(attribute_id);
CREATE INDEX idx_variant_attribute_images_value_id ON variant_attribute_images(attribute_value_id);
```

### Seed Data

```sql
-- Attributes
INSERT INTO variant_attributes (id, name, display_name, has_images, display_order) VALUES
    ('attr_color', 'color', 'Color', TRUE, 1),
    ('attr_size', 'size', 'Size', FALSE, 2);

-- Common colors
INSERT INTO variant_attribute_values (id, attribute_id, value, display_name, hex_color, display_order) VALUES
    ('color_red', 'attr_color', 'red', 'Red', '#FF0000', 1),
    ('color_blue', 'attr_color', 'blue', 'Blue', '#0000FF', 2),
    ('color_green', 'attr_color', 'green', 'Green', '#00FF00', 3),
    ('color_orange', 'attr_color', 'orange', 'Orange', '#FF8800', 4),
    ('color_purple', 'attr_color', 'purple', 'Purple', '#8800FF', 5),
    ('color_black', 'attr_color', 'black', 'Black', '#000000', 6),
    ('color_white', 'attr_color', 'white', 'White', '#FFFFFF', 7);

-- Common sizes
INSERT INTO variant_attribute_values (id, attribute_id, value, display_name, display_order) VALUES
    ('size_sm', 'attr_size', 'small', 'Small', 1),
    ('size_md', 'attr_size', 'medium', 'Medium', 2),
    ('size_lg', 'attr_size', 'large', 'Large', 3),
    ('size_xl', 'attr_size', 'xlarge', 'X-Large', 4);
```

### Data Example

```
Product: T-Rex Dinosaur
├── has_variants: TRUE
├── price_cents: 2000 (base price $20)
└── SKUs:
    ├── TREX-RED-SM
    │   ├── Color: Red (has 3 images: trex-red-1.jpg, trex-red-2.jpg, trex-red-3.jpg)
    │   ├── Size: Small
    │   ├── price_adjustment_cents: 0 → effective price $20.00
    │   └── stock_quantity: 10
    ├── TREX-RED-LG
    │   ├── Color: Red (same 3 images)
    │   ├── Size: Large
    │   ├── price_adjustment_cents: 200 → effective price $22.00
    │   └── stock_quantity: 5
    ├── TREX-BLU-SM
    │   ├── Color: Blue (has 2 images: trex-blue-1.jpg, trex-blue-2.jpg)
    │   ├── Size: Small
    │   ├── price_adjustment_cents: 0 → effective price $20.00
    │   └── stock_quantity: 0 (OUT OF STOCK)
    └── TREX-BLU-LG
        ├── Color: Blue (same 2 images)
        ├── Size: Large
        ├── price_adjustment_cents: 200 → effective price $22.00
        └── stock_quantity: 8
```

---

## Image Handling Strategy

### Key Principles

1. **Images belong to COLOR variants, not SKUs**
   - A color (e.g., Red) has 1+ images
   - All SKUs with that color share the same images
   - Sizes don't have separate images

2. **Image storage in database:**
   - Store ONLY filename: `trex-red-1.jpg`
   - NOT full path: `~~/public/images/products/trex-red-1.jpg~~`
   - Path construction happens in view layer

3. **File system location:**
   - Store at: `./public/images/products/variants/`
   - Served at: `http://localhost:8000/public/images/products/variants/trex-red-1.jpg`

4. **Multiple images per color:**
   - Primary image (is_primary=TRUE): shown in color selector
   - Additional images: shown in main gallery when color selected
   - display_order: controls gallery sequence

### Admin Workflow

1. Admin uploads images for a color (e.g., "Red")
2. System stores filenames in `variant_attribute_images` table
3. First uploaded image marked as primary
4. Images linked to color value, NOT individual SKUs
5. When generating SKUs (Red+Small, Red+Large), they automatically reference the color's images

### Customer Experience

1. Product page loads → shows first color's primary image
2. Color selector displays all colors with their primary images
3. Click a color → main gallery switches to that color's full image set
4. Select size → price updates, but images stay the same

---

## UI/UX Design

### Customer-Facing Product Page

**Layout:**
```
┌─────────────────────────────┬─────────────────────────────┐
│   Main Image Gallery        │   Product Info              │
│   (shows selected color's   │   - Title                   │
│    all images)              │   - Price (updates live)    │
│                             │   - Description             │
│   [◀ Image 1 of 3 ▶]       │                             │
│                             │   Color Selector:           │
│                             │   ┌───┬───┬───┬───┐        │
│                             │   │🖼 │🖼 │🖼 │🖼 │        │
│                             │   │$20│$20│$22│$19│        │
│                             │   └───┴───┴───┴───┘        │
│                             │                             │
│                             │   Size Selector:            │
│                             │   [SM] [MD] [LG] [XL]       │
│                             │                             │
│                             │   Stock: In Stock (5 left)  │
│                             │                             │
│                             │   [Add to Cart]             │
└─────────────────────────────┴─────────────────────────────┘
```

**Color Selector (Amazon-style image grid):**
```html
<div class="grid grid-cols-4 gap-3">
  <!-- Red - Selected -->
  <button class="variant-card active" @click="selectColor('red')">
    <img src="/public/images/products/variants/trex-red-1.jpg" alt="Red" />
    <div class="price">$20.79</div>
    <div class="delivery">FREE Delivery Tomorrow</div>
  </button>

  <!-- Blue - Available -->
  <button class="variant-card" @click="selectColor('blue')">
    <img src="/public/images/products/variants/trex-blue-1.jpg" alt="Blue" />
    <div class="price">$19.19</div>
    <div class="delivery">FREE Delivery Saturday</div>
  </button>

  <!-- Green - Out of Stock -->
  <button class="variant-card disabled" disabled>
    <img src="/public/images/products/variants/trex-green-1.jpg" alt="Green" class="opacity-50" />
    <div class="badge bg-gray-500">Out of Stock</div>
  </button>
</div>
```

**Size Selector (Button group):**
```html
<div class="size-selector" x-show="selectedColor">
  <label>Size</label>
  <div class="flex gap-2">
    <button
      @click="selectSize('small')"
      :class="{'bg-blue-600 text-white': selectedSize === 'small'}"
      :disabled="!isSizeAvailable('small')"
      class="px-4 py-2 border rounded">
      Small
    </button>
    <button
      @click="selectSize('large')"
      :class="{'bg-blue-600 text-white': selectedSize === 'large'}"
      :disabled="!isSizeAvailable('large')"
      class="px-4 py-2 border rounded">
      Large <span class="text-xs ml-1">(+$2.00)</span>
    </button>
  </div>
</div>
```

**Real-time Updates:**
- Price changes when size selected
- Stock status updates per variant
- Add to Cart disabled until both color + size selected
- Main gallery switches when color changes

### Admin Interface

**Product Detail - Variants Section:**

```
┌─────────────────────────────────────────────────────────┐
│  Variants & SKUs                                  [✓ Has Variants] │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  Step 1: Manage Colors & Images                         │
│  ┌─────────────────────────────────────────────────┐   │
│  │ Color    │ Primary Image      │ Images │ SKUs   │   │
│  ├──────────┼───────────────────┼────────┼────────┤   │
│  │ Red      │ [🖼 trex-red-1]   │   3    │   4    │   │
│  │ Blue     │ [🖼 trex-blue-1]  │   2    │   4    │   │
│  │ Green    │ [🖼 trex-green-1] │   1    │   4    │   │
│  └─────────────────────────────────────────────────┘   │
│  [+ Add Color]                                          │
│                                                          │
│  Step 2: Generate/Manage SKUs                           │
│  ┌─────────────────────────────────────────────────┐   │
│  │          │ Small  │ Medium │ Large  │ X-Large  │   │
│  ├──────────┼────────┼────────┼────────┼──────────┤   │
│  │ Red      │ $20/10 │ $20/8  │ $22/5  │ $25/2    │   │
│  │ Blue     │ $20/0  │ $20/3  │ $22/8  │ $25/1    │   │
│  │ Green    │ $20/5  │ $20/5  │ $22/5  │ $25/0    │   │
│  └─────────────────────────────────────────────────┘   │
│  [Generate All Combinations]                            │
└─────────────────────────────────────────────────────────┘
```

**Color Editor Modal (TemplUI Dialog):**
- Select existing color OR create new
- Upload multiple images (drag & drop)
- Reorder images (drag to reorder)
- Set primary image (shown in selector)
- Preview how it appears to customers

**SKU Bulk Generator:**
- Shows matrix: Colors (rows) × Sizes (columns)
- Checkboxes for which combos to create
- Set base stock quantity for all
- Set size-based price adjustments
  - Small/Medium: +$0
  - Large: +$2
  - X-Large: +$5
- Creates all selected SKUs at once

---

## Stripe Integration

### Approach: Dynamic price_data

**Why:** Maximum flexibility, no pre-created Stripe Products needed, can generate dynamically.

### Implementation

```go
// When customer adds variant to cart and proceeds to checkout:

func (s *StripeService) CreateCheckoutSession(
    productID string,
    skuID string,
    quantity int,
) (*stripe.CheckoutSession, error) {
    // Load product and SKU from database
    product := queries.GetProduct(ctx, productID)
    sku := queries.GetProductSKU(ctx, skuID)
    attributes := queries.GetSKUAttributes(ctx, skuID)  // [{color: red}, {size: large}]

    // Get color's primary image
    colorImage := queries.GetColorPrimaryImage(ctx, attributes.ColorID)

    // Format variant name: "T-Rex - Red, Large"
    variantName := formatVariantName(product.Name, attributes)

    // Calculate effective price
    effectivePrice := product.PriceCents + sku.PriceAdjustmentCents

    // Create Stripe session
    params := &stripe.CheckoutSessionParams{
        LineItems: []*stripe.CheckoutSessionLineItemParams{
            {
                PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
                    Currency: stripe.String("usd"),
                    UnitAmount: stripe.Int64(effectivePrice),
                    ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
                        Name: stripe.String(variantName),
                        Description: stripe.String(product.Description),
                        Images: []*string{
                            stripe.String(fmt.Sprintf("https://example.com/public/images/products/variants/%s", colorImage)),
                        },
                        Metadata: map[string]string{
                            "product_id": product.ID,
                            "sku_id": sku.ID,
                            "sku": sku.SKU,
                            "color": attributes.Color,
                            "size": attributes.Size,
                        },
                    },
                },
                Quantity: stripe.Int64(int64(quantity)),
            },
        },
        Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
        SuccessURL: stripe.String("https://example.com/success?session_id={CHECKOUT_SESSION_ID}"),
        CancelURL: stripe.String("https://example.com/cancel"),
        Metadata: map[string]string{
            "product_id": product.ID,
            "sku_id": sku.ID,
        },
    }

    return session.New(params)
}
```

### Webhook Handling

When payment succeeds:
1. Extract `sku_id` from metadata
2. Deduct inventory from that specific SKU
3. Store order with full variant details (color, size)

---

## Implementation Phases

### Phase 1: Database Foundation
- [ ] Create migration file
- [ ] Add variant_attributes table
- [ ] Add variant_attribute_values table
- [ ] Add variant_attribute_images table
- [ ] Rename product_variants → product_skus
- [ ] Add product_sku_attributes junction table
- [ ] Add indexes
- [ ] Seed color and size attributes
- [ ] Test migration up/down

### Phase 2: Backend - SQLC Queries
- [ ] Create `storage/queries/variants.sql`
- [ ] Query: Get product colors with primary images
- [ ] Query: Get color images (all, ordered)
- [ ] Query: Get available sizes for product+color
- [ ] Query: Get SKU by product+color+size
- [ ] Query: Create SKU
- [ ] Query: Update SKU (price, stock)
- [ ] Query: Delete SKU
- [ ] Query: Get all SKUs for product (admin view)
- [ ] Run `go generate ./...` to generate Go code

### Phase 3: SKU Utilities
- [ ] Create `internal/utils/sku.go`
- [ ] Function: GenerateSKU(baseSkU, color, size) → "TREX-RED-LG"
- [ ] Function: ValidateSKU(sku) → unique check
- [ ] Function: ParseSKU(sku) → extract color/size codes

### Phase 4: Admin UI - Variant Management
- [ ] Update `views/admin/product_detail.templ`:
  - [ ] Add "Has Variants" toggle
  - [ ] Add Colors section (table + Add Color button)
  - [ ] Add SKU matrix section (grid view)
  - [ ] Add "Generate All Combinations" button
- [ ] Create `views/admin/modals/color_editor.templ` (TemplUI Dialog)
  - [ ] Color selector/creator
  - [ ] Multi-image uploader
  - [ ] Image reordering (drag & drop)
  - [ ] Primary image selector
- [ ] Create `views/admin/modals/sku_bulk_generator.templ`
  - [ ] Color×Size matrix with checkboxes
  - [ ] Price adjustment inputs
  - [ ] Stock quantity inputs
  - [ ] Generate button
- [ ] Create `internal/handlers/admin_variants.go`:
  - [ ] POST /admin/products/:id/colors (add color + images)
  - [ ] DELETE /admin/products/:id/colors/:color_id
  - [ ] POST /admin/products/:id/skus (create SKU)
  - [ ] POST /admin/products/:id/skus/bulk (bulk generate)
  - [ ] PUT /admin/products/:id/skus/:sku_id (update)
  - [ ] DELETE /admin/products/:id/skus/:sku_id

### Phase 5: Customer UI - Variant Selection
- [ ] Update `views/shop/product_detail.templ`:
  - [ ] Add Alpine.js productVariants() component
  - [ ] Add color selector (image grid)
  - [ ] Add size selector (button group)
  - [ ] Add real-time price display
  - [ ] Add stock status display
  - [ ] Update main image gallery on color change
  - [ ] Disable Add to Cart until variant selected
- [ ] Update `internal/handlers/shop.go`:
  - [ ] Load all SKUs with attributes for product
  - [ ] Load color images
  - [ ] Pass as JSON to Alpine.js
- [ ] Update cart to use SKU ID instead of product ID

### Phase 6: Stripe Integration
- [ ] Update `service/stripe.go`:
  - [ ] Modify CreateCheckoutSession to accept SKU ID
  - [ ] Load SKU attributes (color, size)
  - [ ] Use price_data with variant info
  - [ ] Include color image in Stripe
  - [ ] Add metadata (sku_id, color, size)
- [ ] Update webhook handler:
  - [ ] Extract SKU ID from metadata
  - [ ] Deduct inventory from correct SKU
  - [ ] Store variant details in order

### Phase 7: Data Migration
- [ ] Create `scripts/migrate-variants/main.go`:
  - [ ] Identify products with existing variants
  - [ ] Parse variant names to extract color/size
  - [ ] Create color attribute values
  - [ ] Link product images to colors
  - [ ] Create SKUs with proper attributes
  - [ ] Verify migration (dry run first)

### Phase 8: Testing
- [ ] Test variant selection flow (customer)
- [ ] Test color changes update gallery
- [ ] Test size changes update price
- [ ] Test out-of-stock variants disabled
- [ ] Test Add to Cart with variants
- [ ] Test cart displays variant info
- [ ] Test Stripe checkout with variants
- [ ] Test webhook inventory deduction
- [ ] Test admin: create product with variants
- [ ] Test admin: upload color images
- [ ] Test admin: bulk generate SKUs
- [ ] Test admin: edit individual SKU

### Phase 9: Documentation & Polish
- [ ] Update CLAUDE.md with variant patterns
- [ ] Add validation: prevent duplicate SKU combos
- [ ] Add error handling: clear out-of-stock messages
- [ ] Add accessibility: keyboard navigation, ARIA labels
- [ ] Add loading states during variant operations
- [ ] Optimize queries (check N+1 issues)

---

## Key Files to Create/Modify

### Database
- `storage/migrations/XXX_add_flexible_product_variants.sql` (new)
- `storage/queries/variants.sql` (new)

### Backend
- `internal/utils/sku.go` (new)
- `internal/handlers/admin_variants.go` (new)
- `service/stripe.go` (modify)
- `internal/handlers/shop.go` (modify)
- `internal/handlers/cart.go` (modify)

### Admin UI
- `views/admin/product_detail.templ` (modify - add variants section)
- `views/admin/modals/color_editor.templ` (new)
- `views/admin/modals/sku_bulk_generator.templ` (new)
- `views/admin/settings/variant_attributes.templ` (new)

### Customer UI
- `views/shop/product_detail.templ` (modify - add variant selectors)
- `views/cart/cart.templ` (modify - show variant info)

### Scripts
- `scripts/migrate-variants/main.go` (new - one-time data migration)

---

## Best Practices & Gotchas

### Database
- ✅ Store only filenames in image_url, not paths
- ✅ Use price adjustments, not absolute prices on SKUs
- ✅ Each SKU must have unique combination of attributes
- ❌ Don't store color/size as concatenated string - use junction table

### Images
- ✅ Images belong to colors (attribute values), not SKUs
- ✅ Multiple sizes with same color share images
- ✅ Store files in `/public/images/products/variants/`
- ✅ Construct paths in view layer

### SKU Generation
- ✅ Format: `{BASE}-{COLOR}-{SIZE}` (e.g., "TREX-RED-LG")
- ✅ Use 2-3 char codes for variants (RED, BLU, SM, LG)
- ✅ Validate uniqueness before creating
- ❌ Avoid confusing characters (0 vs O, 1 vs I)

### UI/UX
- ✅ Show product images for color selection (not swatches)
- ✅ Use buttons for sizes (not dropdown)
- ✅ Disable unavailable combinations
- ✅ Force user to select variants (no defaults)
- ✅ Update price immediately when variant changes
- ❌ Don't use color swatches - use actual product photos

### Stripe
- ✅ Use price_data for dynamic pricing
- ✅ Include rich metadata (sku_id, color, size)
- ✅ Use color's primary image in Stripe
- ❌ Don't pre-create Stripe Products for every SKU

### Testing
- ✅ Test migration up AND down
- ✅ Test out-of-stock edge cases
- ✅ Test products with no variants (simple products)
- ✅ Verify inventory deduction hits correct SKU

---

## Future Enhancements (Not in Phase 1)

### Additional Variant Types
- Material (PLA, PETG, Resin, Premium)
- Finish (Matte, Glossy, Textured)
- Custom options (engraving, personalization)

### Advanced Features
- Variant-specific descriptions (e.g., "Large size includes stand")
- Bulk pricing (10+ units = 20% off)
- Pre-orders for out-of-stock variants
- Variant comparison table
- Quick view / hover preview
- Variant-specific shipping rules

### Admin Improvements
- Import variants from CSV
- Clone product with all variants
- Bulk edit (change all prices by 10%)
- Low stock alerts per variant
- Variant sales analytics

---

## Success Criteria

### MVP Success (Phase 1 Complete)
- [ ] Customer can select color (via images) and size
- [ ] Price updates based on selected variant
- [ ] Out of stock variants are disabled
- [ ] Add to cart works with selected variant
- [ ] Stripe checkout includes variant details
- [ ] Inventory deducted from correct SKU
- [ ] Admin can create product with variants
- [ ] Admin can upload color images
- [ ] Admin can bulk generate SKUs

### Production Ready
- [ ] All automated tests pass
- [ ] Manual testing complete (all edge cases)
- [ ] Existing products migrated successfully
- [ ] Documentation updated
- [ ] Performance optimized (no N+1 queries)
- [ ] Error handling comprehensive
- [ ] Accessibility verified (keyboard nav, screen readers)

---

## Questions & Decisions Log

**Q: Should we use color swatches or product images for color selection?**
**A:** Product images (like Amazon). Customers want to see the actual product in that color.

**Q: Do sizes need separate images?**
**A:** No, only colors have images. All sizes of the same color share the same images.

**Q: Where should variant images be stored in the database schema?**
**A:** Link images to `variant_attribute_values` (the color itself), not to individual SKUs. This way Red-Small and Red-Large share the same images.

**Q: Should we support materials (PLA, PETG) in Phase 1?**
**A:** No, only PLA for now. Add materials as a future variant type when needed for custom prints.

**Q: How should SKUs be generated?**
**A:** Auto-generate from base SKU + color code + size code (e.g., "TREX-RED-LG"). Allow manual override in admin.

**Q: How to handle Stripe with variants?**
**A:** Use `price_data` to create products dynamically. Include variant metadata and color image.

**Q: Schema approach: simple columns vs flexible attributes?**
**A:** Flexible attributes (three-tier system) for extensibility. Allows adding new variant types without schema changes.

**Q: How to handle products without variants?**
**A:** Add `has_variants` boolean flag. Simple products bypass variant selection, use base price directly.

---

**Last Updated:** 2025-11-20
**Document Owner:** Product Development Team
**Review Frequency:** After each phase completion
