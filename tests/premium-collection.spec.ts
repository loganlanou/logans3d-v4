import { test, expect } from '@playwright/test';

test.describe('Premium Collection Functionality', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/admin');
  });

  test('should display premium collection checkbox in product form', async ({ page }) => {
    await page.goto('/admin/product/new');
    
    // Check that the premium collection checkbox is present
    const premiumCheckbox = page.locator('input[name="is_premium_collection"]');
    await expect(premiumCheckbox).toBeVisible();
    await expect(premiumCheckbox).toHaveAttribute('type', 'checkbox');
    
    // Check the label
    await expect(page.getByText('Premium Collection')).toBeVisible();
    await expect(page.getByText('Featured as part of premium collection showcase')).toBeVisible();
  });

  test('should create product with premium collection enabled', async ({ page }) => {
    await page.goto('/admin/product/new');
    
    // Fill out the form
    await page.fill('input[name="name"]', 'Premium Test Product');
    await page.fill('textarea[name="description"]', 'This is a premium collection product');
    await page.fill('input[name="price"]', '99.99');
    await page.fill('input[name="stock_quantity"]', '10');
    
    // Enable premium collection
    await page.check('input[name="is_premium_collection"]');
    
    // Submit the form
    await page.click('button[type="submit"]');
    
    // Should redirect to admin dashboard
    await expect(page).toHaveURL('/admin');
    
    // Should see the new product marked as premium collection
    const productRow = page.locator('tbody tr').filter({ hasText: 'Premium Test Product' });
    await expect(productRow).toBeVisible();
    await expect(productRow.locator('.admin-status-featured')).toBeVisible();
    await expect(productRow.locator('.admin-status-featured')).toContainText('Premium Collection');
  });

  test('should edit product and toggle premium collection status', async ({ page }) => {
    // First create a regular product
    await page.goto('/admin/product/new');
    await page.fill('input[name="name"]', 'Toggle Premium Product');
    await page.fill('textarea[name="description"]', 'Test product for toggling');
    await page.fill('input[name="price"]', '49.99');
    await page.fill('input[name="stock_quantity"]', '5');
    await page.click('button[type="submit"]');
    
    // Now edit it to make it premium
    const editButton = page.locator('tbody tr').filter({ hasText: 'Toggle Premium Product' }).locator('a:has-text("Edit")');
    await editButton.click();
    
    // Should navigate to edit form
    await expect(page).toHaveURL(/\/admin\/product\/edit\?id=/);
    await expect(page.locator('h1')).toContainText('Edit Product');
    
    // Premium collection checkbox should be unchecked
    const premiumCheckbox = page.locator('input[name="is_premium_collection"]');
    await expect(premiumCheckbox).not.toBeChecked();
    
    // Enable premium collection
    await premiumCheckbox.check();
    await page.click('button[type="submit"]');
    
    // Should see updated product with premium collection status
    await expect(page).toHaveURL('/admin');
    const updatedRow = page.locator('tbody tr').filter({ hasText: 'Toggle Premium Product' });
    await expect(updatedRow.locator('.admin-status-featured')).toBeVisible();
    await expect(updatedRow.locator('.admin-status-featured')).toContainText('Premium Collection');
  });

  test('should display "Premium Collection" instead of "Featured" in product status', async ({ page }) => {
    // Check if any products are marked as premium collection
    const premiumStatus = page.locator('.admin-status-featured');
    const premiumCount = await premiumStatus.count();
    
    if (premiumCount > 0) {
      // Verify all featured statuses show "Premium Collection" text
      for (let i = 0; i < Math.min(premiumCount, 3); i++) {
        await expect(premiumStatus.nth(i)).toContainText('Premium Collection');
        await expect(premiumStatus.nth(i)).not.toContainText('Featured');
      }
    }
  });

  test('should preserve premium collection status when editing other fields', async ({ page }) => {
    // Create a premium product first
    await page.goto('/admin/product/new');
    await page.fill('input[name="name"]', 'Preserve Premium Product');
    await page.fill('textarea[name="description"]', 'Original description');
    await page.fill('input[name="price"]', '199.99');
    await page.check('input[name="is_premium_collection"]');
    await page.click('button[type="submit"]');
    
    // Edit the product but only change non-premium fields
    const editButton = page.locator('tbody tr').filter({ hasText: 'Preserve Premium Product' }).locator('a:has-text("Edit")');
    await editButton.click();
    
    // Verify premium collection is checked
    await expect(page.locator('input[name="is_premium_collection"]')).toBeChecked();
    
    // Change only the description
    await page.fill('textarea[name="description"]', 'Updated description but still premium');
    await page.click('button[type="submit"]');
    
    // Should still be marked as premium collection
    const updatedRow = page.locator('tbody tr').filter({ hasText: 'Preserve Premium Product' });
    await expect(updatedRow.locator('.admin-status-featured')).toBeVisible();
    await expect(updatedRow.locator('.admin-status-featured')).toContainText('Premium Collection');
  });

  test('should show premium collection checkbox in edit form with correct state', async ({ page }) => {
    // Create a premium product
    await page.goto('/admin/product/new');
    await page.fill('input[name="name"]', 'Premium Edit Test');
    await page.fill('input[name="price"]', '299.99');
    await page.check('input[name="is_premium_collection"]');
    await page.click('button[type="submit"]');
    
    // Edit the product
    const editButton = page.locator('tbody tr').filter({ hasText: 'Premium Edit Test' }).locator('a:has-text("Edit")');
    await editButton.click();
    
    // Premium collection checkbox should be checked in edit form
    await expect(page.locator('input[name="is_premium_collection"]')).toBeChecked();
    
    // Uncheck it
    await page.uncheck('input[name="is_premium_collection"]');
    await page.click('button[type="submit"]');
    
    // Should no longer show premium collection status
    const updatedRow = page.locator('tbody tr').filter({ hasText: 'Premium Edit Test' });
    await expect(updatedRow.locator('.admin-status-featured')).not.toBeVisible();
  });

  test('should handle premium collection with categories correctly', async ({ page }) => {
    // First create a category
    await page.goto('/admin/category/new');
    await page.fill('input[name="name"]', 'Premium Category');
    await page.fill('textarea[name="description"]', 'For premium products');
    await page.click('button[type="submit"]');
    
    // Create a premium product with this category
    await page.goto('/admin/product/new');
    await page.fill('input[name="name"]', 'Premium Category Product');
    await page.fill('input[name="price"]', '399.99');
    
    // Select the category
    await page.selectOption('select[name="category_id"]', { label: 'Premium Category' });
    
    // Enable premium collection
    await page.check('input[name="is_premium_collection"]');
    await page.click('button[type="submit"]');
    
    // Verify the product shows both category and premium status
    const productRow = page.locator('tbody tr').filter({ hasText: 'Premium Category Product' });
    await expect(productRow).toContainText('Premium Category');
    await expect(productRow.locator('.admin-status-featured')).toContainText('Premium Collection');
  });

  test('should maintain form functionality with premium collection field', async ({ page }) => {
    await page.goto('/admin/product/new');
    
    // Fill all required fields including premium collection
    await page.fill('input[name="name"]', 'Complete Form Test');
    await page.fill('textarea[name="description"]', 'Complete product with all fields');
    await page.fill('input[name="short_description"]', 'Short desc');
    await page.fill('input[name="price"]', '149.99');
    await page.fill('input[name="sku"]', 'COMP-001');
    await page.fill('input[name="stock_quantity"]', '25');
    await page.check('input[name="is_premium_collection"]');
    
    // Submit form
    await page.click('button[type="submit"]');
    
    // Verify all data was saved correctly
    const productRow = page.locator('tbody tr').filter({ hasText: 'Complete Form Test' });
    await expect(productRow).toBeVisible();
    await expect(productRow).toContainText('COMP-001');
    await expect(productRow).toContainText('$149.99');
    await expect(productRow).toContainText('25');
    await expect(productRow.locator('.admin-status-featured')).toContainText('Premium Collection');
  });
});