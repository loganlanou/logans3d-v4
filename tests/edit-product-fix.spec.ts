import { test, expect } from '@playwright/test';

test.describe('Product Edit Fix Verification', () => {
  test('edit product route is accessible', async ({ page }) => {
    // Navigate to admin dashboard
    await page.goto('/admin');
    
    // Check that the page loads successfully
    await expect(page.locator('h1').first()).toContainText('Admin Dashboard');
    
    // Navigate directly to the edit route to verify it's working (with non-existent ID)
    await page.goto('/admin/product/edit?id=test-id');
    
    // Since product doesn't exist, it should show "Add New Product" but still work (no 404)
    await expect(page.locator('h1')).toContainText('Add New Product');
    
    // Verify the form is rendered
    await expect(page.locator('form')).toBeVisible();
    
    // Check for form fields
    await expect(page.locator('input[name="name"]')).toBeVisible();
    await expect(page.locator('input[name="price"]')).toBeVisible();
  });

  test('new product route is accessible', async ({ page }) => {
    // Navigate directly to the new product route
    await page.goto('/admin/product/new');
    
    // Check that we get the new product form page
    await expect(page.locator('h1')).toContainText('Add New Product');
    
    // Verify the form is rendered
    await expect(page.locator('form')).toBeVisible();
    
    // Check for form fields
    await expect(page.locator('input[name="name"]')).toBeVisible();
    await expect(page.locator('input[name="price"]')).toBeVisible();
  });

  test('admin dashboard loads without errors', async ({ page }) => {
    // Navigate to admin dashboard
    await page.goto('/admin');
    
    // Check basic elements are present
    await expect(page.locator('h1').first()).toContainText('Admin Dashboard');
    await expect(page.locator('.admin-stats-grid')).toBeVisible();
    await expect(page.locator('.admin-table')).toBeVisible();
    
    // Check for the Add Product button
    await expect(page.locator('a[href="/admin/product/new"]')).toBeVisible();
  });
});