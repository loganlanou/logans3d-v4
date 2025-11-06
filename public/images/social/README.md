# Social Media Images

This directory contains images optimized for social media sharing (Open Graph, Twitter Cards, etc.).

## Required Images

### 1. Default Open Graph Image (default-og.jpg)
- **Dimensions**: 1200x630 pixels (exact)
- **Format**: JPG or PNG
- **Purpose**: Default image when no product-specific image is available
- **Content Suggestions**:
  - Logan's 3D Creations logo/branding
  - Showcase of popular products
  - Generic 3D printing imagery
  - Company name prominently displayed

### 2. Shop Open Graph Image (shop-og.jpg)
- **Dimensions**: 1200x630 pixels
- **Purpose**: Default for shop/category pages
- **Content**: Collection of products or shop branding

### 3. Square Logo (logo-square.png)
- **Dimensions**: 400x400 pixels minimum (square)
- **Purpose**: Organization schema, profile images
- **Content**: Clean logo on transparent or solid background

## Current Status

**Temporary Setup**: Using existing product images as defaults until custom OG images are created.

## Design Guidelines

### Best Practices for OG Images:
- Keep important content in the center (safe zone)
- Use high contrast text (if including text)
- Avoid small details (won't be readable at thumbnail size)
- Test on multiple platforms (Facebook, Twitter, LinkedIn)
- File size: Keep under 1MB for fast loading
- Consider mobile display (many users see these on phones)

### Brand Colors (Logan's 3D Creations):
- Primary: Slate/Dark backgrounds (#1e293b)
- Accent: Blue (#3b82f6) and Emerald (#10b981)
- Text: White for dark backgrounds

### Typography:
- Bold, readable fonts
- Large enough to read in thumbnails
- High contrast with background

## Tools for Creating OG Images

- **Canva**: Pre-sized templates for social media
- **Figma**: Professional design tool with export options
- **Adobe Express**: Quick social media graphics
- **ImageMagick**: Command-line tool for batch processing
- **Online OG Image Generators**: Various free tools available

## Testing Your Images

After creating images, test them with:
- [Facebook Sharing Debugger](https://developers.facebook.com/tools/debug/)
- [Twitter Card Validator](https://cards-dev.twitter.com/validator)
- [LinkedIn Post Inspector](https://www.linkedin.com/post-inspector/)
- [OpenGraph.xyz](https://www.opengraph.xyz/)

## Deployment

1. Create images with exact dimensions above
2. Save to this directory with the specified filenames
3. Update `site_config` table if using different paths:
   ```sql
   UPDATE site_config SET value = '/public/images/social/default-og.jpg' WHERE key = 'default_og_image';
   ```
4. Clear any CDN/cache if applicable
5. Test with social media validators
