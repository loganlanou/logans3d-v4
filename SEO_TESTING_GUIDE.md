# SEO & Social Sharing Testing Guide

## Implementation Complete! ✅

Your Logan's 3D Creations website now has complete SEO and social sharing functionality.

## What Was Implemented

### 1. Database Schema ✅
- Added SEO fields to `products` table:
  - `seo_title` - Custom title override
  - `seo_description` - Custom description override
  - `seo_keywords` - Custom keywords (comma-separated)
  - `og_image_url` - Custom Open Graph image
- Created `site_config` table for global settings
- All migrations tested and applied

### 2. PageMeta System ✅
- `views/layout/meta.go` - Comprehensive metadata management
- Helper chain: `NewPageMeta().FromProduct().WithProductImage().WithCategories()`
- Intelligent fallbacks (uses custom SEO fields if set, otherwise defaults to product data)
- All public pages updated to use PageMeta

###3. Meta Tags ✅
Every page now includes:
- **HTML Meta**: title, description, keywords, canonical URL
- **Open Graph**: type, url, title, description, image, site_name
- **Twitter Cards**: card type, title, description, image
- **Schema.org JSON-LD**: Product and Organization structured data

### 4. Share Button Component ✅
- Multi-platform sharing: Facebook, Twitter/X, Pinterest, WhatsApp, Email, Reddit
- Copy Link feature with clipboard API + fallback
- Toast notification on successful copy
- TemplUI Dialog component for beautiful UI
- Added to all product pages

### 5. Default OG Images ✅
Created at `public/images/social/`:
- `default-og.jpg` (1200x630px)
- `shop-og.jpg` (1200x630px)
- `logo-square.png` (400x400px)
- `README.md` with creation guidelines

## Testing Your Implementation

### Local Testing (Quick Check)

1. **View Source** - Open any product page and view source:
   ```bash
   curl -s http://localhost:8007/shop/product/triceratops | grep "og:"
   ```

2. **Check JSON-LD**:
   ```bash
   curl -s http://localhost:8007/shop/product/triceratops | grep -A 20 'application/ld+json'
   ```

3. **Verify Share Button** - Visit product page and click "Share" button

### Social Media Validators (Production Testing)

**IMPORTANT**: These validators require your site to be accessible on the public internet. You must deploy to production first.

#### 1. Facebook Sharing Debugger
**URL**: https://developers.facebook.com/tools/debug/

**Steps**:
1. Enter your product URL: `https://www.logans3dcreations.com/shop/product/triceratops`
2. Click "Debug"
3. Review the preview

**What to Check**:
- ✅ Title appears correctly
- ✅ Description is present and accurate
- ✅ Image loads (1200x630px recommended)
- ✅ URL is correct

**Common Issues**:
- Cache: Click "Scrape Again" to refresh
- Image not loading: Check image URL is absolute and accessible
- Missing data: Verify meta tags in page source

#### 2. Twitter Card Validator
**URL**: https://cards-dev.twitter.com/validator

**Steps**:
1. Enter your product URL
2. Click "Preview card"

**What to Check**:
- ✅ Card type: `summary_large_image`
- ✅ Image displays properly
- ✅ Title and description visible
- ✅ Card renders nicely

#### 3. LinkedIn Post Inspector
**URL**: https://www.linkedin.com/post-inspector/

**Steps**:
1. Enter your URL
2. Click "Inspect"

**What to Check**:
- ✅ Title, description, image all present
- ✅ Preview looks professional

#### 4. Google Rich Results Test
**URL**: https://search.google.com/test/rich-results

**Steps**:
1. Enter your product URL
2. Click "Test URL"
3. Wait for analysis

**What to Check**:
- ✅ Product schema detected
- ✅ No errors in structured data
- ✅ Price, availability, image all present
- ✅ Organization schema valid

#### 5. OpenGraph.xyz (All-in-One)
**URL**: https://www.opengraph.xyz/

**Steps**:
1. Enter your URL
2. View preview for multiple platforms simultaneously

**What to Check**:
- ✅ Facebook preview
- ✅ Twitter preview
- ✅ LinkedIn preview
- ✅ All look consistent

### Manual Social Sharing Test

**Real-World Test** (Most Important!):

1. **Facebook**:
   - Create a post with your product URL
   - Check the preview before posting
   - Post and verify how it looks in feed

2. **Twitter/X**:
   - Tweet your product URL
   - Verify card appears
   - Check image quality

3. **Pinterest**:
   - Pin directly from product page
   - Verify image and description

## Expected Output Examples

### Open Graph Tags (Product Page)
```html
<meta property="og:type" content="product">
<meta property="og:url" content="https://www.logans3dcreations.com/shop/product/triceratops">
<meta property="og:title" content="Triceratops">
<meta property="og:description" content="Classic three-horned dinosaur with protective frill">
<meta property="og:site_name" content="Logan's 3D Creations">
<meta property="og:image" content="https://www.logans3dcreations.com/public/images/products/triceratops.jpeg">
<meta property="og:image:width" content="1200">
<meta property="og:image:height" content="1200">
```

### Twitter Card Tags
```html
<meta name="twitter:card" content="summary_large_image">
<meta name="twitter:title" content="Triceratops">
<meta name="twitter:description" content="Classic three-horned dinosaur with protective frill">
<meta name="twitter:image" content="https://www.logans3dcreations.com/public/images/products/triceratops.jpeg">
```

### Product Schema.org JSON-LD
```json
{
  "@context": "https://schema.org/",
  "@type": "Product",
  "name": "Triceratops",
  "description": "Classic three-horned dinosaur with protective frill",
  "image": "https://www.logans3dcreations.com/public/images/products/triceratops.jpeg",
  "brand": {
    "@type": "Brand",
    "name": "Logan's 3D Creations"
  },
  "offers": {
    "@type": "Offer",
    "url": "https://www.logans3dcreations.com/shop/product/triceratops",
    "priceCurrency": "USD",
    "price": "10.00",
    "availability": "https://schema.org/InStock"
  },
  "category": "Dinosaurs"
}
```

## Troubleshooting

### Images Not Showing in Social Previews

**Problem**: Facebook/Twitter shows no image or wrong image

**Solutions**:
1. Verify image URL is absolute (starts with `https://`)
2. Check image is accessible publicly (no authentication required)
3. Ensure image meets size requirements (1200x630px recommended)
4. Clear cache in validator tools ("Scrape Again" in Facebook)
5. Wait 24-48 hours for some platforms to refresh

### Wrong Information Displayed

**Problem**: Old title/description showing

**Solutions**:
1. Check page source - is new data there?
2. Clear validator cache
3. Check database for custom SEO fields overriding defaults
4. Verify `PageMeta` is being built correctly in handler

### Schema.org Errors

**Problem**: Google Rich Results shows errors

**Solutions**:
1. Verify JSON-LD is valid JSON (use JSONLint.com)
2. Check all required fields are present (name, offers, etc.)
3. Ensure URLs are absolute
4. Verify price format is correct (string with 2 decimals)

## Adding Custom SEO for Products

### Via Database (Recommended for Future)

When you have an admin interface:

```sql
UPDATE products
SET
  seo_title = 'Amazing Triceratops Model - Best 3D Printed Dinosaur',
  seo_description = 'Our bestselling Triceratops model features incredible detail...',
  seo_keywords = 'triceratops,dinosaur model,3D printed dinosaur,collectible',
  og_image_url = '/public/images/products/triceratops-social.jpg'
WHERE slug = 'triceratops';
```

### Smart Fallbacks

If custom fields are empty, the system automatically uses:
- `seo_title` → `product.name`
- `seo_description` → `product.description` or `product.short_description`
- `seo_keywords` → Auto-generated from product name
- `og_image_url` → Primary product image

## Social Media Account Setup

### Add Your Social Handles

Update `site_config` table:

```sql
-- Add Twitter handle
UPDATE site_config SET value = '@Logans3D' WHERE key = 'twitter_handle';

-- Add Facebook Page ID
UPDATE site_config SET value = 'YourPageID' WHERE key = 'facebook_page_id';

-- Add Facebook App ID (optional, for Facebook Insights)
UPDATE site_config SET value = 'YourAppID' WHERE key = 'facebook_app_id';
```

This will add:
- `<meta name="twitter:site" content="@Logans3D">`
- `<meta property="fb:pages" content="YourPageID">`
- Organization schema `sameAs` links

## Performance Tips

### Image Optimization

For best results, optimize your OG images:
```bash
# Using ImageMagick
magick input.jpg -quality 85 -resize 1200x630 output.jpg

# Or use online tools:
# - TinyPNG (tinypng.com)
# - Squoosh (squoosh.app)
# - ImageOptim (Mac)
```

### CDN Considerations

If using a CDN:
1. Ensure social media bots can access images
2. Allow `Facebookexternalbot` and `Twitterbot` user agents
3. Disable authentication for public images
4. Set proper cache headers

## Next Steps

### 1. Deploy to Production
Your SEO implementation is complete! Deploy to make it live:
```bash
# Example deployment
git add .
git commit -m "Add complete SEO and social sharing"
git push origin main
```

### 2. Test with Validators
Once deployed, test all URLs with the validators listed above.

### 3. Monitor Results
- Check Google Search Console for rich results
- Monitor social media engagement
- Track click-through rates from social platforms

### 4. Iterate
- Test different OG images to see what performs best
- Refine product descriptions for SEO
- Add custom SEO fields for your best-selling products

## Support Resources

- [Open Graph Protocol](https://ogp.me/)
- [Twitter Cards Documentation](https://developer.twitter.com/en/docs/twitter-for-websites/cards/overview/abouts-cards)
- [Schema.org Product](https://schema.org/Product)
- [Google Search Central](https://developers.google.com/search/docs/appearance/structured-data/product)

## Files Modified/Created

### Database
- `storage/migrations/20251105211827_add_seo_fields_to_products.sql`
- `storage/migrations/20251105211855_create_site_config_table.sql`
- `storage/queries/products.sql` (updated)
- `storage/queries/site_config.sql` (new)

### Backend
- `views/layout/meta.go` (new - 350+ lines)
- `views/layout/schema.templ` (new)
- `service/service.go` (updated all public handlers)

### Frontend
- `views/layout/base.templ` (updated with complete meta tags)
- `views/components/share_button.templ` (new)
- `views/shop/product.templ` (updated with share button)
- All view templates updated to use `PageMeta`

### Assets
- `public/images/social/default-og.jpg` (1200x630)
- `public/images/social/shop-og.jpg` (1200x630)
- `public/images/social/logo-square.png` (400x400)
- `public/images/social/README.md`

---

**Implementation Status**: ✅ 100% Complete

**Ready for Production**: Yes

**Last Updated**: 2025-11-05
