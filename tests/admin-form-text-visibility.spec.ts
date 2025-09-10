import { test, expect } from '@playwright/test';

test.describe('Admin Form Text Visibility', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to a product edit form to test text input visibility
    await page.goto('/admin/product/edit?id=d06f35f9-c1e6-44c2-9a23-28d32c78fad3');
  });

  test('should have white text in all text input fields', async ({ page }) => {
    // Test Product Name input
    const nameInput = page.locator('input[name="name"]');
    await expect(nameInput).toBeVisible();
    await expect(nameInput).toHaveCSS('color', 'rgb(255, 255, 255)'); // white text
    await expect(nameInput).toHaveValue('Shark');
    
    // Test Description textarea
    const descriptionTextarea = page.locator('textarea[name="description"]');
    await expect(descriptionTextarea).toBeVisible();
    await expect(descriptionTextarea).toHaveCSS('color', 'rgb(255, 255, 255)'); // white text
    await expect(descriptionTextarea).toHaveValue('Realistic shark model with fin details');
    
    // Test Short Description input
    const shortDescInput = page.locator('input[name="short_description"]');
    await expect(shortDescInput).toBeVisible();
    await expect(shortDescInput).toHaveCSS('color', 'rgb(255, 255, 255)'); // white text
    
    // Test SKU input
    const skuInput = page.locator('input[name="sku"]');
    await expect(skuInput).toBeVisible();
    await expect(skuInput).toHaveCSS('color', 'rgb(255, 255, 255)'); // white text
    
    // Test Price input
    const priceInput = page.locator('input[name="price"]');
    await expect(priceInput).toBeVisible();
    await expect(priceInput).toHaveCSS('color', 'rgb(255, 255, 255)'); // white text
    await expect(priceInput).toHaveValue('5.00');
    
    // Test Stock Quantity input
    const stockInput = page.locator('input[name="stock_quantity"]');
    await expect(stockInput).toBeVisible();
    await expect(stockInput).toHaveCSS('color', 'rgb(255, 255, 255)'); // white text
    await expect(stockInput).toHaveValue('18');
  });

  test('should have proper dark background for input fields', async ({ page }) => {
    // All inputs should have dark slate backgrounds
    const inputs = page.locator('input[type="text"], input[type="number"], textarea');
    const count = await inputs.count();
    
    for (let i = 0; i < count; i++) {
      const input = inputs.nth(i);
      await expect(input).toBeVisible();
      
      // Check for dark background (slate-900/50 should be very dark)
      const backgroundColor = await input.evaluate(el => {
        return window.getComputedStyle(el).backgroundColor;
      });
      
      // Should be a dark color (RGB values should be low)
      expect(backgroundColor).toMatch(/rgba?\(\s*\d+,\s*\d+,\s*\d+/);
      
      // Verify it's not white or light colored
      expect(backgroundColor).not.toBe('rgb(255, 255, 255)');
      expect(backgroundColor).not.toBe('rgba(255, 255, 255, 1)');
    }
  });

  test('should be able to edit text in all fields with visible text', async ({ page }) => {
    // Test typing in Product Name field
    const nameInput = page.locator('input[name="name"]');
    await nameInput.clear();
    await nameInput.fill('Test Product Name');
    await expect(nameInput).toHaveValue('Test Product Name');
    
    // Test typing in Description field
    const descriptionTextarea = page.locator('textarea[name="description"]');
    await descriptionTextarea.clear();
    await descriptionTextarea.fill('Test product description that should be clearly visible');
    await expect(descriptionTextarea).toHaveValue('Test product description that should be clearly visible');
    
    // Test typing in Short Description field
    const shortDescInput = page.locator('input[name="short_description"]');
    await shortDescInput.fill('Short test description');
    await expect(shortDescInput).toHaveValue('Short test description');
    
    // Test typing in SKU field
    const skuInput = page.locator('input[name="sku"]');
    await skuInput.fill('TEST-SKU-001');
    await expect(skuInput).toHaveValue('TEST-SKU-001');
    
    // Test typing in Price field
    const priceInput = page.locator('input[name="price"]');
    await priceInput.clear();
    await priceInput.fill('15.99');
    await expect(priceInput).toHaveValue('15.99');
    
    // Test typing in Stock Quantity field
    const stockInput = page.locator('input[name="stock_quantity"]');
    await stockInput.clear();
    await stockInput.fill('25');
    await expect(stockInput).toHaveValue('25');
  });

  test('should have proper admin layout with correct CSS loaded', async ({ page }) => {
    // Verify the page is using admin layout (should have admin-root class)
    const body = page.locator('body');
    await expect(body).toHaveClass(/admin-root/);
    
    // Verify admin header is present
    await expect(page.locator('header')).toBeVisible();
    await expect(page.getByText('Admin Dashboard')).toBeVisible();
    
    // Verify admin navigation is present
    await expect(page.locator('nav')).toBeVisible();
    await expect(page.getByText('Dashboard')).toBeVisible();
    await expect(page.getByText('Products')).toBeVisible();
    
    // Verify the page title indicates it's using admin layout
    await expect(page).toHaveTitle(/Admin/);
  });

  test('should work consistently across different form states', async ({ page }) => {
    // Test the "Add New Product" form as well
    await page.goto('/admin/product/new');
    
    // All text inputs should still have white text
    const nameInput = page.locator('input[name="name"]');
    await expect(nameInput).toBeVisible();
    await expect(nameInput).toHaveCSS('color', 'rgb(255, 255, 255)');
    
    const descriptionTextarea = page.locator('textarea[name="description"]');
    await expect(descriptionTextarea).toBeVisible();
    await expect(descriptionTextarea).toHaveCSS('color', 'rgb(255, 255, 255)');
    
    // Test typing in new product form
    await nameInput.fill('New Product Test');
    await expect(nameInput).toHaveValue('New Product Test');
    
    await descriptionTextarea.fill('New product description test');
    await expect(descriptionTextarea).toHaveValue('New product description test');
  });

  test('should maintain text visibility on form interaction', async ({ page }) => {
    // Test focus states
    const nameInput = page.locator('input[name="name"]');
    await nameInput.focus();
    await expect(nameInput).toBeFocused();
    await expect(nameInput).toHaveCSS('color', 'rgb(255, 255, 255)');
    
    // Test hover states (if applicable)
    await nameInput.hover();
    await expect(nameInput).toHaveCSS('color', 'rgb(255, 255, 255)');
    
    // Test after typing
    await nameInput.fill('Testing text visibility');
    await expect(nameInput).toHaveCSS('color', 'rgb(255, 255, 255)');
    await expect(nameInput).toHaveValue('Testing text visibility');
  });
});