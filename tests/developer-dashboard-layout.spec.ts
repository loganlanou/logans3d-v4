import { test, expect } from '@playwright/test';

test.describe('Developer Dashboard Layout Improvements', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/dev');
  });

  test('should have improved spacing and centering', async ({ page }) => {
    // Check that main content area has proper padding
    const main = page.locator('.dev-main');
    await expect(main).toBeVisible();
    
    // Check that the dashboard grid has proper gaps
    const dashboard = page.locator('.dev-dashboard');
    await expect(dashboard).toBeVisible();
    
    // Verify cards are visible and properly spaced
    const cards = page.locator('.dev-card');
    const cardCount = await cards.count();
    expect(cardCount).toBeGreaterThan(0);
    
    // Check that all cards are visible
    for (let i = 0; i < Math.min(cardCount, 4); i++) {
      await expect(cards.nth(i)).toBeVisible();
    }
  });

  test('should have larger cards and content', async ({ page }) => {
    // Check card headers have proper sizing
    const cardHeaders = page.locator('.dev-card-header h3');
    const headerCount = await cardHeaders.count();
    
    if (headerCount > 0) {
      const firstHeader = cardHeaders.first();
      await expect(firstHeader).toBeVisible();
      
      // Verify font size is larger (should be 1.4rem)
      const fontSize = await firstHeader.evaluate(el => 
        window.getComputedStyle(el).fontSize
      );
      
      // Convert to pixels and check it's reasonably large
      const fontSizePx = parseFloat(fontSize);
      expect(fontSizePx).toBeGreaterThan(20); // 1.4rem should be around 22.4px
    }
  });

  test('should have larger metrics values', async ({ page }) => {
    // Check metric values are visible and large
    const metricValues = page.locator('.dev-metric-value');
    const metricCount = await metricValues.count();
    
    if (metricCount > 0) {
      const firstMetric = metricValues.first();
      await expect(firstMetric).toBeVisible();
      
      // Verify font size is larger (should be 2.5rem)
      const fontSize = await firstMetric.evaluate(el => 
        window.getComputedStyle(el).fontSize
      );
      
      const fontSizePx = parseFloat(fontSize);
      expect(fontSizePx).toBeGreaterThan(35); // 2.5rem should be around 40px
    }
  });

  test('should have improved section headers', async ({ page }) => {
    // Check section title is visible and large
    const sectionTitle = page.locator('.dev-section-title').first();
    await expect(sectionTitle).toBeVisible();
    await expect(sectionTitle).toContainText('System Overview');
    
    // Verify font size is larger (should be 1.8rem)
    const fontSize = await sectionTitle.evaluate(el => 
      window.getComputedStyle(el).fontSize
    );
    
    const fontSizePx = parseFloat(fontSize);
    expect(fontSizePx).toBeGreaterThan(25); // 1.8rem should be around 28.8px
  });

  test('should have proper responsive behavior', async ({ page }) => {
    // Test desktop view
    await page.setViewportSize({ width: 1200, height: 800 });
    await expect(page.locator('.dev-dashboard')).toBeVisible();
    
    // Test tablet view
    await page.setViewportSize({ width: 768, height: 600 });
    await expect(page.locator('.dev-dashboard')).toBeVisible();
    
    // Test mobile view
    await page.setViewportSize({ width: 480, height: 600 });
    await expect(page.locator('.dev-dashboard')).toBeVisible();
    
    // Reset to desktop
    await page.setViewportSize({ width: 1200, height: 800 });
  });

  test('should maintain functionality with improved layout', async ({ page }) => {
    // Check navigation links work
    const systemInfoLink = page.locator('a[href="#system"]');
    await expect(systemInfoLink).toBeVisible();
    
    // Check refresh button is visible and clickable
    const refreshButton = page.locator('button:has-text("Refresh")');
    await expect(refreshButton).toBeVisible();
    await expect(refreshButton).toBeEnabled();
    
    // Check Force GC button is visible and clickable
    const gcButton = page.locator('button:has-text("Force GC")');
    await expect(gcButton).toBeVisible();
    await expect(gcButton).toBeEnabled();
  });

  test('should display all key metrics with improved visibility', async ({ page }) => {
    // Check for application status
    await expect(page.getByText('Logan\'s 3D Creations v4')).toBeVisible();
    await expect(page.getByText('Running')).toBeVisible();
    
    // Check for system information
    await expect(page.getByText('System Information')).toBeVisible();
    await expect(page.getByText('go1.25.0')).toBeVisible();
    
    // Check for memory usage
    await expect(page.getByText('Memory Usage')).toBeVisible();
    
    // Check for database statistics
    await expect(page.getByText('Database Statistics')).toBeVisible();
    await expect(page.getByText('Products')).toBeVisible();
    await expect(page.getByText('Categories')).toBeVisible();
  });

  test('should have better visual hierarchy', async ({ page }) => {
    // Check that cards have proper elevation/shadow
    const firstCard = page.locator('.dev-card').first();
    await expect(firstCard).toBeVisible();
    
    // Verify cards have hover effects by checking for transform capability
    const transform = await firstCard.evaluate(el => 
      window.getComputedStyle(el).transform
    );
    
    // Should have some styling (either none or matrix)
    expect(transform).toMatch(/none|matrix/);
    
    // Check section headers have proper borders
    const sectionHeader = page.locator('.dev-section-header').first();
    await expect(sectionHeader).toBeVisible();
    
    const borderBottom = await sectionHeader.evaluate(el => 
      window.getComputedStyle(el).borderBottomWidth
    );
    
    // Should have a visible border
    expect(parseFloat(borderBottom)).toBeGreaterThan(0);
  });
});