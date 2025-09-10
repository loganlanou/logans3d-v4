import { test, expect } from '@playwright/test';

test.describe('Category Management', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/admin');
  });

  test('should display "Add Category" button in admin dashboard', async ({ page }) => {
    // Check that the Add Category button is visible and correctly positioned
    await expect(page.locator('a[href="/admin/category/new"]')).toBeVisible();
    await expect(page.locator('a[href="/admin/category/new"]')).toContainText('+ Add Category');
    
    // Verify it's next to the Add Product button
    const buttons = page.locator('.flex.space-x-3 a');
    await expect(buttons).toHaveCount(2);
    await expect(buttons.first()).toContainText('+ Add Category');
    await expect(buttons.last()).toContainText('+ Add Product');
  });

  test('should display categories table with correct headers', async ({ page }) => {
    // Check that the categories table is present
    const categoriesSection = page.locator('.admin-card').nth(1); // Second card after products
    await expect(categoriesSection.locator('.admin-card-title')).toContainText('Categories');
    
    // Check table headers
    const headers = categoriesSection.locator('th');
    await expect(headers.nth(0)).toContainText('Name');
    await expect(headers.nth(1)).toContainText('Description');
    await expect(headers.nth(2)).toContainText('Parent');
    await expect(headers.nth(3)).toContainText('Display Order');
    await expect(headers.nth(4)).toContainText('Products');
    await expect(headers.nth(5)).toContainText('Actions');
  });

  test('should navigate to category form when clicking "Add Category"', async ({ page }) => {
    await page.click('a[href="/admin/category/new"]');
    
    // Should navigate to category form
    await expect(page).toHaveURL('/admin/category/new');
    await expect(page.locator('h1')).toContainText('Add New Category');
  });

  test('should display category form with correct fields', async ({ page }) => {
    await page.goto('/admin/category/new');
    
    // Check form fields are present
    await expect(page.locator('input[name="name"]')).toBeVisible();
    await expect(page.locator('textarea[name="description"]')).toBeVisible();
    await expect(page.locator('select[name="parent_id"]')).toBeVisible();
    await expect(page.locator('input[name="display_order"]')).toBeVisible();
    
    // Check form labels
    await expect(page.getByText('Category Name')).toBeVisible();
    await expect(page.getByText('Description')).toBeVisible();
    await expect(page.getByText('Parent Category')).toBeVisible();
    await expect(page.getByText('Display Order')).toBeVisible();
    
    // Check buttons
    await expect(page.locator('button[type="submit"]')).toContainText('Create Category');
    await expect(page.locator('a[href="/admin"]')).toContainText('Cancel');
  });

  test('should create a new category successfully', async ({ page }) => {
    await page.goto('/admin/category/new');
    
    // Fill out the form
    await page.fill('input[name="name"]', 'Test Category');
    await page.fill('textarea[name="description"]', 'This is a test category');
    await page.fill('input[name="display_order"]', '1');
    
    // Submit the form
    await page.click('button[type="submit"]');
    
    // Should redirect to admin dashboard
    await expect(page).toHaveURL('/admin');
    
    // Should see the new category in the categories table
    const categoryRow = page.locator('tbody tr').filter({ hasText: 'Test Category' });
    await expect(categoryRow).toBeVisible();
    await expect(categoryRow).toContainText('This is a test category');
    await expect(categoryRow).toContainText('Root Category');
    await expect(categoryRow).toContainText('1');
  });

  test('should edit an existing category', async ({ page }) => {
    // First create a category (assuming one exists or create it)
    await page.goto('/admin/category/new');
    await page.fill('input[name="name"]', 'Editable Category');
    await page.fill('textarea[name="description"]', 'Original description');
    await page.click('button[type="submit"]');
    
    // Now edit it
    const editButton = page.locator('tbody tr').filter({ hasText: 'Editable Category' }).locator('a:has-text("Edit")');
    await editButton.click();
    
    // Should navigate to edit form
    await expect(page).toHaveURL(/\/admin\/category\/edit\?id=/);
    await expect(page.locator('h1')).toContainText('Edit Category');
    
    // Form should be pre-filled
    await expect(page.locator('input[name="name"]')).toHaveValue('Editable Category');
    await expect(page.locator('textarea[name="description"]')).toHaveValue('Original description');
    
    // Update the category
    await page.fill('input[name="name"]', 'Updated Category');
    await page.fill('textarea[name="description"]', 'Updated description');
    await page.click('button[type="submit"]');
    
    // Should see updated category in dashboard
    await expect(page).toHaveURL('/admin');
    const updatedRow = page.locator('tbody tr').filter({ hasText: 'Updated Category' });
    await expect(updatedRow).toBeVisible();
    await expect(updatedRow).toContainText('Updated description');
  });

  test('should show category count in stats', async ({ page }) => {
    // Check that the categories stat card shows the correct count
    const categoryStatCard = page.locator('.admin-stat-card').nth(1); // Second stat card
    await expect(categoryStatCard.locator('.admin-stat-label')).toContainText('Categories');
    
    // The number should be a valid integer
    const categoryCount = await categoryStatCard.locator('.admin-stat-number').textContent();
    expect(parseInt(categoryCount!)).toBeGreaterThanOrEqual(0);
  });

  test('should show product count per category', async ({ page }) => {
    // Find any category row and check it has a product count
    const categoryRows = page.locator('tbody tr');
    const rowCount = await categoryRows.count();
    
    if (rowCount > 0) {
      const firstRow = categoryRows.first();
      // Product count column should show a number (5th column, 0-indexed)
      const productCountCell = firstRow.locator('td').nth(4);
      const productCount = await productCountCell.textContent();
      expect(parseInt(productCount!)).toBeGreaterThanOrEqual(0);
    }
  });

  test('should handle category deletion with confirmation', async ({ page }) => {
    // First create a category to delete
    await page.goto('/admin/category/new');
    await page.fill('input[name="name"]', 'Delete Me Category');
    await page.fill('textarea[name="description"]', 'This will be deleted');
    await page.click('button[type="submit"]');
    
    // Find the delete button for this category
    const categoryRow = page.locator('tbody tr').filter({ hasText: 'Delete Me Category' });
    const deleteButton = categoryRow.locator('button:has-text("Delete")');
    
    // Set up dialog handler
    page.on('dialog', async dialog => {
      expect(dialog.message()).toContain('Are you sure');
      await dialog.accept();
    });
    
    await deleteButton.click();
    
    // Should redirect back to admin dashboard
    await expect(page).toHaveURL('/admin');
    
    // Category should be removed from the table
    await expect(page.locator('tbody tr').filter({ hasText: 'Delete Me Category' })).not.toBeVisible();
  });
});