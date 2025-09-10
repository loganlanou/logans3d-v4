import { test, expect } from '@playwright/test';

test.describe('Developer Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to the developer dashboard
    await page.goto('/dev');
  });

  test('should load developer dashboard with correct title and branding', async ({ page }) => {
    // Check page title
    await expect(page).toHaveTitle(/Developer Dashboard.*Logan's 3D Creations/);
    
    // Check main header (specifically the developer dashboard header)
    await expect(page.locator('.dev-header h1')).toContainText('Developer Dashboard');
    
    // Check that favicon images are loaded
    const headerImage = page.locator('header img[alt="Logan\'s 3D"]');
    await expect(headerImage).toBeVisible();
    await expect(headerImage).toHaveAttribute('src', '/public/images/favicon.png');
    
    // Check application status card has correct branding
    await expect(page.locator('h3')).toContainText('Logan\'s 3D Creations v4');
    
    // Check status card image
    const statusCardImage = page.locator('.dev-card img[alt="Logan\'s 3D"]');
    await expect(statusCardImage).toBeVisible();
    await expect(statusCardImage).toHaveAttribute('src', '/public/images/favicon.png');
  });

  test('should have functional navigation tabs', async ({ page }) => {
    // Check all navigation links are present
    const navLinks = [
      'Overview',
      'System Info', 
      'Database',
      'Memory',
      'Logs',
      'Config'
    ];

    for (const linkText of navLinks) {
      await expect(page.locator('.dev-nav-link', { hasText: linkText })).toBeVisible();
    }

    // Test tab navigation
    await page.click('.dev-nav-link[href="#system"]');
    await expect(page.locator('#system')).toBeVisible();
    
    await page.click('.dev-nav-link[href="#database"]');
    await expect(page.locator('#database')).toBeVisible();
    
    await page.click('.dev-nav-link[href="#memory"]');
    await expect(page.locator('#memory')).toBeVisible();
    
    await page.click('.dev-nav-link[href="#logs"]');
    await expect(page.locator('#logs')).toBeVisible();
    
    await page.click('.dev-nav-link[href="#config"]');
    await expect(page.locator('#config')).toBeVisible();
    
    // Return to overview
    await page.click('.dev-nav-link[href="#overview"]');
    await expect(page.locator('#overview')).toBeVisible();
  });

  test('should display system information correctly', async ({ page }) => {
    // Navigate to system info tab
    await page.click('.dev-nav-link[href="#system"]');
    
    // Check system information table
    await expect(page.locator('table.dev-table')).toBeVisible();
    await expect(page.locator('td')).toContainText('Logan\'s 3D Creations v4');
    await expect(page.locator('td')).toContainText('4.0.0'); // Version
    
    // Check for runtime information
    const tableRows = page.locator('table.dev-table tr');
    await expect(tableRows).toContainText('Application Name');
    await expect(tableRows).toContainText('Version');
    await expect(tableRows).toContainText('Environment');
    await expect(tableRows).toContainText('Start Time');
    await expect(tableRows).toContainText('Port');
    await expect(tableRows).toContainText('Database Path');
  });

  test('should display memory statistics', async ({ page }) => {
    // Navigate to memory tab
    await page.click('.dev-nav-link[href="#memory"]');
    
    // Check memory statistics table
    await expect(page.locator('#memory-stats table.dev-table')).toBeVisible();
    
    const memoryRows = page.locator('#memory-stats table.dev-table tr');
    await expect(memoryRows).toContainText('Allocated Memory');
    await expect(memoryRows).toContainText('System Memory'); 
    await expect(memoryRows).toContainText('Total Allocations');
    await expect(memoryRows).toContainText('GC Cycles');
    await expect(memoryRows).toContainText('Active Goroutines');
    
    // Test memory stats update button
    await page.click('button:has-text("Update Stats")');
    // Wait a moment for the update
    await page.waitForTimeout(1000);
  });

  test('should display database information', async ({ page }) => {
    // Navigate to database tab
    await page.click('.dev-nav-link[href="#database"]');
    
    // Check database information
    await expect(page.locator('#database-stats')).toBeVisible();
    await expect(page.locator('.dev-alert')).toContainText('Database Location');
    
    // Check database table
    const dbTable = page.locator('#database-stats table.dev-table');
    await expect(dbTable).toBeVisible();
    
    const dbRows = page.locator('#database-stats table tbody tr');
    await expect(dbRows.first()).toContainText('Products');
    await expect(dbRows.nth(1)).toContainText('Categories');
    await expect(dbRows.first()).toContainText('Active');
    
    // Test database stats refresh
    await page.click('button:has-text("Refresh Stats")');
    await page.waitForTimeout(1000);
  });

  test('should display configuration correctly', async ({ page }) => {
    // Navigate to config tab
    await page.click('.dev-nav-link[href="#config"]');
    
    // Check configuration display
    await expect(page.locator('#config-display')).toBeVisible();
    
    const configText = await page.locator('#config-display').textContent();
    expect(configText).toContain('DB_PATH=');
    expect(configText).toContain('PORT=');
    expect(configText).toContain('ENVIRONMENT=');
  });

  test('should have functional control buttons', async ({ page }) => {
    // Test refresh button in overview section
    await expect(page.locator('#overview button:has-text("Refresh")')).toBeVisible();
    await page.click('#overview button:has-text("Refresh")');
    
    // Test Force GC button
    await expect(page.locator('button:has-text("Force GC")')).toBeVisible();
    
    // Test memory update button in memory section
    await page.click('.dev-nav-link[href="#memory"]');
    await expect(page.locator('button:has-text("Update Stats")')).toBeVisible();
    
    // Test database refresh button in database section
    await page.click('.dev-nav-link[href="#database"]');
    await expect(page.locator('button:has-text("Refresh Stats")')).toBeVisible();
    
    // Test log controls in logs section
    await page.click('.dev-nav-link[href="#logs"]');
    await expect(page.locator('button:has-text("Refresh")')).toBeVisible();
    await expect(page.locator('button:has-text("Clear")')).toBeVisible();
  });

  test('should test API endpoints functionality', async ({ page }) => {
    // Test Force GC functionality
    const [response] = await Promise.all([
      page.waitForResponse('/dev/gc'),
      page.click('button:has-text("Force GC")')
    ]);
    
    expect(response.status()).toBe(200);
    
    // Test memory stats API
    await page.click('.dev-nav-link[href="#memory"]');
    const [memoryResponse] = await Promise.all([
      page.waitForResponse('/dev/memory'),
      page.click('button:has-text("Update Stats")')
    ]);
    
    expect(memoryResponse.status()).toBe(200);
    
    // Test database stats API
    await page.click('.dev-nav-link[href="#database"]');
    const [dbResponse] = await Promise.all([
      page.waitForResponse('/dev/database'),
      page.click('button:has-text("Refresh Stats")')
    ]);
    
    expect(dbResponse.status()).toBe(200);
  });

  test('should have responsive design elements', async ({ page }) => {
    // Test on mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });
    
    // Check that elements are still visible and functional
    await expect(page.locator('.dev-header')).toBeVisible();
    await expect(page.locator('.dev-nav')).toBeVisible();
    await expect(page.locator('.dev-dashboard')).toBeVisible();
    
    // Test navigation still works on mobile
    await page.click('.dev-nav-link[href="#system"]');
    await expect(page.locator('#system')).toBeVisible();
    
    // Reset to desktop
    await page.setViewportSize({ width: 1280, height: 720 });
  });

  test('should display metrics with proper formatting', async ({ page }) => {
    // Check overview metrics display
    const metrics = page.locator('.dev-metric');
    await expect(metrics).toHaveCount(7); // Uptime, PID, Go Version, OS, Arch, Allocated MB, System MB, etc.
    
    // Check that metric values are displayed
    const metricValues = page.locator('.dev-metric-value');
    await expect(metricValues.first()).toBeVisible();
    
    // Check that metric labels are displayed
    const metricLabels = page.locator('.dev-metric-label');
    await expect(metricLabels.first()).toBeVisible();
  });

  test('should handle error states gracefully', async ({ page }) => {
    // Test what happens when JavaScript functions encounter errors
    await page.evaluate(() => {
      // Temporarily override fetch to simulate network error
      window.originalFetch = window.fetch;
      window.fetch = () => Promise.reject(new Error('Network error'));
    });
    
    // Try to trigger API calls and ensure page doesn't crash
    await page.click('button:has-text("Force GC")');
    await page.waitForTimeout(500);
    
    // Page should still be functional
    await expect(page.locator('.dev-header')).toBeVisible();
    
    // Restore fetch
    await page.evaluate(() => {
      window.fetch = window.originalFetch;
    });
  });
});

test.describe('Admin Dashboard Integration', () => {
  test('should navigate between admin and developer dashboards', async ({ page }) => {
    // Start at admin dashboard
    await page.goto('/admin');
    await expect(page.locator('h1')).toContainText('Admin Dashboard');
    
    // Check that developer link is present in navigation
    await expect(page.locator('a[href="/dev"]')).toBeVisible();
    
    // Navigate to developer dashboard
    await page.click('a[href="/dev"]');
    await expect(page).toHaveURL('/dev');
    await expect(page.locator('h1')).toContainText('Developer Dashboard');
    
    // Check navigation back to admin
    await expect(page.locator('a[href="/admin"]')).toBeVisible();
    await page.click('a[href="/admin"]');
    await expect(page).toHaveURL('/admin');
  });

  test('should have consistent styling between admin and developer sections', async ({ page }) => {
    // Test admin dashboard styling
    await page.goto('/admin');
    
    // Check that admin CSS is loaded
    const adminStyles = await page.locator('link[href*="admin-styles.css"]');
    await expect(adminStyles).toHaveAttribute('href', '/public/css/admin-styles.css');
    
    // Check that developer CSS is also loaded
    const devStyles = await page.locator('link[href*="developer-styles.css"]');
    await expect(devStyles).toHaveAttribute('href', '/public/css/developer-styles.css');
    
    // Navigate to developer dashboard
    await page.goto('/dev');
    
    // Check that both stylesheets are available
    await expect(page.locator('link[href*="admin-styles.css"]')).toHaveAttribute('href', '/public/css/admin-styles.css');
    await expect(page.locator('link[href*="developer-styles.css"]')).toHaveAttribute('href', '/public/css/developer-styles.css');
  });
});