import { test, expect } from '@playwright/test';

test.describe('Admin Dashboard Images', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/admin');
  });

  test('should display Image column in products table', async ({ page }) => {
    // Check that the Image column header is present
    await expect(page.locator('th').first()).toContainText('Image');
    
    // Verify the table headers are in correct order
    const headers = page.locator('th');
    await expect(headers.nth(0)).toContainText('Image');
    await expect(headers.nth(1)).toContainText('Name');
    await expect(headers.nth(2)).toContainText('SKU');
    await expect(headers.nth(3)).toContainText('Price');
    await expect(headers.nth(4)).toContainText('Stock');
    await expect(headers.nth(5)).toContainText('Status');
    await expect(headers.nth(6)).toContainText('Actions');
  });

  test('should display product images in the table rows', async ({ page }) => {
    // Wait for the table to load
    await expect(page.locator('table.admin-table')).toBeVisible();
    
    // Check that first few rows have image cells
    const firstRowImageCell = page.locator('tbody tr').first().locator('td').first();
    await expect(firstRowImageCell).toBeVisible();
    
    // Check if the first row has either an image or a "No Image" placeholder
    const hasImage = await firstRowImageCell.locator('img').count() > 0;
    const hasPlaceholder = await firstRowImageCell.locator('div:has-text("No Image")').count() > 0;
    
    expect(hasImage || hasPlaceholder).toBe(true);
  });

  test('should show images with proper attributes when available', async ({ page }) => {
    // Find a row that has an actual image (not placeholder)
    const imageElements = page.locator('tbody td img');
    const imageCount = await imageElements.count();
    
    if (imageCount > 0) {
      const firstImage = imageElements.first();
      
      // Check image has proper attributes
      await expect(firstImage).toHaveAttribute('alt');
      await expect(firstImage).toHaveAttribute('src');
      await expect(firstImage).toHaveClass(/w-12/); // Check for width class
      await expect(firstImage).toHaveClass(/h-12/); // Check for height class
      await expect(firstImage).toHaveClass(/rounded-lg/); // Check for rounded corners
      
      // Verify alt text is meaningful (should contain product name)
      const altText = await firstImage.getAttribute('alt');
      expect(altText).toBeTruthy();
      expect(altText!.length).toBeGreaterThan(0);
    }
  });

  test('should show "No Image" placeholder for products without images', async ({ page }) => {
    // Look for placeholder elements
    const placeholders = page.locator('tbody td div:has-text("No Image")');
    const placeholderCount = await placeholders.count();
    
    // There should be some placeholders (assuming not all products have images)
    if (placeholderCount > 0) {
      const firstPlaceholder = placeholders.first();
      
      // Check placeholder styling
      await expect(firstPlaceholder).toHaveClass(/w-12/);
      await expect(firstPlaceholder).toHaveClass(/h-12/);
      await expect(firstPlaceholder).toHaveClass(/bg-gray-200/);
      await expect(firstPlaceholder).toHaveClass(/rounded-lg/);
      
      // Check placeholder text
      await expect(firstPlaceholder.locator('span')).toContainText('No Image');
    }
  });

  test('should maintain functionality of edit and delete buttons', async ({ page }) => {
    // Wait for the table to load
    await expect(page.locator('table.admin-table')).toBeVisible();
    
    // Check that edit and delete buttons are still present and functional
    const firstRow = page.locator('tbody tr').first();
    const editButton = firstRow.locator('a:has-text("Edit")');
    const deleteButton = firstRow.locator('button:has-text("Delete")');
    
    await expect(editButton).toBeVisible();
    await expect(deleteButton).toBeVisible();
    
    // Check edit button has proper href
    await expect(editButton).toHaveAttribute('href', /\/admin\/product\/edit\?id=/);
    
    // Check delete button is in a form with proper action
    const deleteForm = deleteButton.locator('..');
    await expect(deleteForm).toHaveAttribute('action', /\/admin\/product\/.+\/delete/);
  });

  test('should display correct product information alongside images', async ({ page }) => {
    // Get the first product row
    const firstRow = page.locator('tbody tr').first();
    
    // Verify all cells are present in correct order
    const cells = firstRow.locator('td');
    
    // Image cell (1st)
    await expect(cells.nth(0)).toBeVisible();
    
    // Name cell (2nd) - should have product name and category
    const nameCell = cells.nth(1);
    await expect(nameCell).toBeVisible();
    await expect(nameCell.locator('.admin-text-primary')).toBeVisible();
    
    // SKU cell (3rd)
    await expect(cells.nth(2)).toBeVisible();
    
    // Price cell (4th) - should start with $
    const priceCell = cells.nth(3);
    await expect(priceCell).toBeVisible();
    const priceText = await priceCell.textContent();
    expect(priceText).toMatch(/^\$/);
    
    // Stock cell (5th)
    await expect(cells.nth(4)).toBeVisible();
    
    // Status cell (6th)
    await expect(cells.nth(5)).toBeVisible();
    
    // Actions cell (7th)
    await expect(cells.nth(6)).toBeVisible();
  });

  test('should handle image loading errors gracefully', async ({ page }) => {
    // Check for any broken images
    const images = page.locator('tbody td img');
    const imageCount = await images.count();
    
    if (imageCount > 0) {
      // Check each image loads without errors
      for (let i = 0; i < Math.min(imageCount, 5); i++) { // Test first 5 images
        const img = images.nth(i);
        await expect(img).toBeVisible();
        
        // Image should have valid src attribute
        const src = await img.getAttribute('src');
        expect(src).toBeTruthy();
        expect(src!.length).toBeGreaterThan(0);
      }
    }
  });

  test('should maintain responsive design with image column', async ({ page }) => {
    // Test on different viewport sizes
    await page.setViewportSize({ width: 1200, height: 800 });
    await expect(page.locator('table.admin-table')).toBeVisible();
    
    // Test on smaller viewport
    await page.setViewportSize({ width: 768, height: 600 });
    await expect(page.locator('table.admin-table')).toBeVisible();
    
    // The table should still be scrollable horizontally if needed
    await expect(page.locator('.overflow-x-auto')).toBeVisible();
  });

  test('should show correct stats with products that have images', async ({ page }) => {
    // Check that the stats cards still work correctly
    const totalProductsCard = page.locator('.admin-stat-card').first();
    await expect(totalProductsCard.locator('.admin-stat-number')).toBeVisible();
    await expect(totalProductsCard.locator('.admin-stat-label')).toContainText('Total Products');
    
    // Verify the count matches the number of rows in the table
    const statsNumber = await totalProductsCard.locator('.admin-stat-number').textContent();
    const tableRows = await page.locator('tbody tr').count();
    
    expect(parseInt(statsNumber!)).toBe(tableRows);
  });
});