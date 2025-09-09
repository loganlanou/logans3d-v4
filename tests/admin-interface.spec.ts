import { test, expect } from '@playwright/test';

test.describe('Admin Interface Text Readability', () => {
  
  test('admin dashboard has readable white text and status symbols', async ({ page }) => {
    // Navigate to admin dashboard
    await page.goto('http://localhost:8000/admin');
    
    // Check page loads
    await expect(page).toHaveTitle(/Admin Dashboard/);
    
    // Check header text is visible and white
    const header = page.locator('h1');
    await expect(header).toBeVisible();
    await expect(header).toContainText('Admin Dashboard');
    
    // Check stats cards have readable white text
    const statsCards = page.locator('.bg-slate-800\\/50');
    await expect(statsCards.first()).toBeVisible();
    
    // Check white text in stats
    const whiteText = page.locator('.text-white');
    const whiteTextCount = await whiteText.count();
    expect(whiteTextCount).toBeGreaterThan(0);
    
    // Check table headers are visible with slate-400 color
    const tableHeaders = page.locator('thead th.text-slate-400');
    await expect(tableHeaders.first()).toBeVisible();
    
    // Check if products table has content and status symbols are visible
    const productRows = page.locator('tbody tr');
    const rowCount = await productRows.count();
    
    if (rowCount > 0) {
      // Check first product row has readable text
      const firstRow = productRows.first();
      await expect(firstRow).toBeVisible();
      
      // Check product name is white and visible
      const productName = firstRow.locator('td').first().locator('.text-white');
      await expect(productName).toBeVisible();
      
      // Check status badges are visible and have proper contrast colors
      const statusCell = firstRow.locator('td').nth(4);
      await expect(statusCell).toBeVisible();
      
      // Check for status badges with proper bright colors (updated to 300 variants)
      const activeStatus = statusCell.locator('.text-emerald-300, .text-emerald-400');
      const inactiveStatus = statusCell.locator('.text-red-300, .text-red-400');
      const featuredStatus = statusCell.locator('.text-amber-300, .text-amber-400');
      
      // At least one status should be visible
      const hasActiveStatus = await activeStatus.count() > 0;
      const hasInactiveStatus = await inactiveStatus.count() > 0;
      const hasFeaturedStatus = await featuredStatus.count() > 0;
      
      expect(hasActiveStatus || hasInactiveStatus).toBeTruthy();
      
      // If status badges exist, verify they're visible
      if (hasActiveStatus) {
        await expect(activeStatus.first()).toBeVisible();
      }
      if (hasInactiveStatus) {
        await expect(inactiveStatus.first()).toBeVisible();
      }
      if (hasFeaturedStatus) {
        await expect(featuredStatus.first()).toBeVisible();
      }
    }
  });

  test('product edit form has white editable text fields', async ({ page }) => {
    // First go to admin to see if there are products
    await page.goto('http://localhost:8000/admin');
    
    // Check if there are any products to edit
    const editLinks = page.locator('a:has-text("Edit")');
    const editLinkCount = await editLinks.count();
    
    if (editLinkCount > 0) {
      // Click the first edit link
      await editLinks.first().click();
      
      // Wait for edit form to load
      await expect(page.locator('h1')).toContainText('Edit Product');
      
      // Check all input fields have white text
      const textInputs = page.locator('input[type="text"], input[type="number"], textarea, select');
      const inputCount = await textInputs.count();
      
      for (let i = 0; i < Math.min(inputCount, 10); i++) { // Limit to first 10 inputs
        const input = textInputs.nth(i);
        await expect(input).toBeVisible();
        
        // Check if input has white text class
        const inputClass = await input.getAttribute('class');
        expect(inputClass).toContain('text-white');
      }
      
      // Check status checkboxes and labels are visible with good contrast
      const activeCheckbox = page.locator('input[name="is_active"]');
      const featuredCheckbox = page.locator('input[name="is_featured"]');
      
      if (await activeCheckbox.count() > 0) {
        await expect(activeCheckbox).toBeVisible();
        
        // Check associated label has readable color (accept white or bright colors)
        const activeLabel = page.locator('span:has-text("Active")');
        await expect(activeLabel).toBeVisible();
        const labelClass = await activeLabel.getAttribute('class');
        // Accept various readable text colors (white or slate variants)
        const hasGoodContrast = labelClass?.includes('text-white') || 
                              labelClass?.includes('text-slate-200') || 
                              labelClass?.includes('text-slate-100') ||
                              labelClass?.includes('text-slate-300') ||
                              labelClass?.includes('text-gray-100');
        expect(hasGoodContrast).toBeTruthy();
      }
      
      if (await featuredCheckbox.count() > 0) {
        await expect(featuredCheckbox).toBeVisible();
        
        const featuredLabel = page.locator('span:has-text("Featured")');
        await expect(featuredLabel).toBeVisible();
        const labelClass = await featuredLabel.getAttribute('class');
        // Accept various readable text colors (white or slate variants)
        const hasGoodContrast = labelClass?.includes('text-white') || 
                              labelClass?.includes('text-slate-200') || 
                              labelClass?.includes('text-slate-100') ||
                              labelClass?.includes('text-slate-300') ||
                              labelClass?.includes('text-gray-100');
        expect(hasGoodContrast).toBeTruthy();
      }
      
    } else {
      // If no products exist, test the new product form
      await page.goto('http://localhost:8000/admin/product/new');
      
      // Check new product form has white text inputs
      await expect(page.locator('h1')).toContainText('Add New Product');
      
      // Check input fields have white text
      const textInputs = page.locator('input[type="text"], input[type="number"], textarea, select');
      const inputCount = await textInputs.count();
      
      for (let i = 0; i < Math.min(inputCount, 10); i++) {
        const input = textInputs.nth(i);
        await expect(input).toBeVisible();
        
        const inputClass = await input.getAttribute('class');
        expect(inputClass).toContain('text-white');
      }
    }
  });

  test('admin interface has sufficient color contrast', async ({ page }) => {
    await page.goto('http://localhost:8000/admin');
    
    // Check background is dark (slate-900) for good contrast
    const mainDiv = page.locator('.from-slate-900');
    await expect(mainDiv).toBeVisible();
    
    // Check header text is white for maximum contrast
    const header = page.locator('h1.text-white');
    await expect(header).toBeVisible();
    
    // Verify form labels have maximum contrast (white text)
    const formLabels = page.locator('.text-white');
    const labelCount = await formLabels.count();
    expect(labelCount).toBeGreaterThan(0);
    
    // Check that there are bright text elements (including improved contrast colors)
    const brightText = page.locator('.text-white, .text-emerald-300, .text-emerald-400, .text-amber-300, .text-amber-400, .text-red-300, .text-red-400');
    const brightCount = await brightText.count();
    expect(brightCount).toBeGreaterThan(0);
  });

  test('status symbols are clearly visible in product list', async ({ page }) => {
    await page.goto('http://localhost:8000/admin');
    
    // Check if there are products with status symbols (updated to include 300 variants)
    const statusBadges = page.locator('.text-emerald-300, .text-emerald-400, .text-red-300, .text-red-400, .text-amber-300, .text-amber-400');
    const badgeCount = await statusBadges.count();
    
    if (badgeCount > 0) {
      // Verify status badges are visible and have good contrast
      for (let i = 0; i < Math.min(badgeCount, 5); i++) { // Check first 5 badges
        const badge = statusBadges.nth(i);
        await expect(badge).toBeVisible();
        
        // Check badge has readable background
        const badgeElement = await badge.innerHTML();
        expect(badgeElement).toBeTruthy();
      }
      
      // Check for specific status colors (including both 300 and 400 variants)
      const emeraldBadges = page.locator('.text-emerald-300, .text-emerald-400');
      const redBadges = page.locator('.text-red-300, .text-red-400');
      const amberBadges = page.locator('.text-amber-300, .text-amber-400');
      
      const emeraldCount = await emeraldBadges.count();
      const redCount = await redBadges.count();
      const amberCount = await amberBadges.count();
      
      // Should have at least one type of status
      expect(emeraldCount + redCount + amberCount).toBeGreaterThan(0);
    } else {
      // If no products exist, that's also valid - just log it
      console.log('No products found in admin dashboard for status symbol testing');
    }
  });
});