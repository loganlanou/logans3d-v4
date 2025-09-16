#!/bin/bash

# Fix product images to only store filenames in the database
# This script updates all image_url values to only contain the filename

set -e

DB_PATH="./data/database.db"

echo "🔧 Fixing product image URLs to only store filenames..."

# Create backup
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_PATH="./data/database.db.backup_${TIMESTAMP}"
cp "$DB_PATH" "$BACKUP_PATH"
echo "✅ Created backup at $BACKUP_PATH"

# Update all image URLs to only contain the filename
sqlite3 "$DB_PATH" << 'EOF'
-- Remove any path prefixes and keep only the filename
UPDATE product_images
SET image_url = CASE
    -- Handle URLs with /public/images/products/ prefix
    WHEN image_url LIKE '/public/images/products/%'
        THEN substr(image_url, length('/public/images/products/') + 1)
    -- Handle URLs with /images/products/ prefix
    WHEN image_url LIKE '/images/products/%'
        THEN substr(image_url, length('/images/products/') + 1)
    -- Handle URLs with /public/uploads/products/ prefix
    WHEN image_url LIKE '/public/uploads/products/%'
        THEN substr(image_url, length('/public/uploads/products/') + 1)
    -- Handle URLs with /uploads/products/ prefix
    WHEN image_url LIKE '/uploads/products/%'
        THEN substr(image_url, length('/uploads/products/') + 1)
    -- Already just a filename or unknown format, keep as is
    ELSE image_url
END;

-- Show the results
SELECT 'Updated image URLs:' as message;
SELECT COUNT(*) || ' total image records' FROM product_images;
SELECT '---' as separator;
SELECT 'Sample of updated URLs:' as message;
SELECT image_url FROM product_images LIMIT 10;
EOF

echo "✅ Database updated successfully!"
echo ""
echo "Next steps:"
echo "1. Review the handlers to ensure they build paths correctly"
echo "2. Test locally"
echo "3. Deploy to staging"