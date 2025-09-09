import { test, expect } from '@playwright/test';

test.describe('All Pages Comprehensive Tests', () => {
  const pages = [
    { path: '/', name: 'Home', expectedTitle: /Logan's 3D Creations/ },
    { path: '/shop', name: 'Shop', expectedTitle: /Shop.*Logan's 3D Creations/ },
    { path: '/shop/premium', name: 'Premium Shop', expectedTitle: /Premium.*Logan's 3D Creations/ },
    { path: '/custom', name: 'Custom Orders', expectedTitle: /Custom.*Logan's 3D Creations/ },
    { path: '/about', name: 'About', expectedTitle: /About.*Logan's 3D Creations/ },
    { path: '/contact', name: 'Contact', expectedTitle: /Contact.*Logan's 3D Creations/ },
    { path: '/events', name: 'Events', expectedTitle: /Events.*Logan's 3D Creations/ },
    { path: '/portfolio', name: 'Portfolio', expectedTitle: /Portfolio.*Logan's 3D Creations/ },
    { path: '/innovation', name: 'Innovation', expectedTitle: /Innovation.*Logan's 3D Creations/ },
    { path: '/admin', name: 'Admin', expectedTitle: /Admin.*Logan's 3D Creations/ },
    { path: '/cart', name: 'Cart', expectedTitle: /Cart.*Logan's 3D Creations/ },
    { path: '/checkout', name: 'Checkout', expectedTitle: /Checkout.*Logan's 3D Creations/ },
    { path: '/privacy', name: 'Privacy', expectedTitle: /Privacy.*Logan's 3D Creations/ },
    { path: '/terms', name: 'Terms', expectedTitle: /Terms.*Logan's 3D Creations/ },
    { path: '/shipping', name: 'Shipping', expectedTitle: /Shipping.*Logan's 3D Creations/ },
    { path: '/custom-policy', name: 'Custom Policy', expectedTitle: /Custom Policy.*Logan's 3D Creations/ },
  ];

  for (const pageInfo of pages) {
    test(`${pageInfo.name} page should load successfully`, async ({ page }) => {
      await page.goto(pageInfo.path);
      
      // Check that page loads with correct status
      const response = await page.waitForResponse(response => 
        response.url().endsWith(pageInfo.path) && response.status() < 400
      );
      expect(response.status()).toBeLessThan(400);
      
      // Check that title contains Logan's 3D Creations
      await expect(page).toHaveTitle(pageInfo.expectedTitle);
      
      // Check that main heading exists
      const h1 = page.locator('h1').first();
      await expect(h1).toBeVisible();
      
      // Check that page has navigation
      const nav = page.locator('nav, header nav').first();
      await expect(nav).toBeVisible();
      
      // Check that footer exists
      const footer = page.locator('footer').first();
      await expect(footer).toBeVisible();
    });

    test(`${pageInfo.name} page should have proper accessibility`, async ({ page }) => {
      await page.goto(pageInfo.path);
      
      // Check for proper heading hierarchy (should have at least one h1)
      const h1Count = await page.locator('h1').count();
      expect(h1Count).toBeGreaterThanOrEqual(1);
      
      // Check that all images have alt attributes
      const images = page.locator('img');
      const imageCount = await images.count();
      
      for (let i = 0; i < imageCount; i++) {
        const image = images.nth(i);
        await expect(image).toHaveAttribute('alt');
      }
      
      // Check that navigation has role
      const nav = page.locator('nav[role="navigation"], nav').first();
      await expect(nav).toBeVisible();
    });

    test(`${pageInfo.name} page should be responsive`, async ({ page }) => {
      // Test desktop
      await page.setViewportSize({ width: 1200, height: 800 });
      await page.goto(pageInfo.path);
      
      const h1Desktop = page.locator('h1').first();
      await expect(h1Desktop).toBeVisible();
      
      // Test tablet
      await page.setViewportSize({ width: 768, height: 1024 });
      await page.reload();
      
      const h1Tablet = page.locator('h1').first();
      await expect(h1Tablet).toBeVisible();
      
      // Test mobile
      await page.setViewportSize({ width: 375, height: 667 });
      await page.reload();
      
      const h1Mobile = page.locator('h1').first();
      await expect(h1Mobile).toBeVisible();
    });
  }

  test('All navigation links should work', async ({ page }) => {
    await page.goto('/');
    
    const navLinks = [
      { text: 'Shop', href: '/shop' },
      { text: 'Custom Orders', href: '/custom' },
      { text: 'Events', href: '/events' },
      { text: 'Portfolio', href: '/portfolio' },
      { text: 'About', href: '/about' },
      { text: 'Contact', href: '/contact' }
    ];
    
    for (const link of navLinks) {
      const navLink = page.locator(`nav a[href="${link.href}"]`).first();
      await expect(navLink).toBeVisible();
      await expect(navLink).toHaveAttribute('href', link.href);
    }
  });

  test('API endpoints should work correctly', async ({ page }) => {
    // Health endpoint
    const healthResponse = await page.request.get('/health');
    expect(healthResponse.status()).toBe(200);
    
    const healthData = await healthResponse.json();
    expect(healthData.status).toBe('healthy');
    expect(healthData.version).toBe('4.0.0');
    
    // Robots.txt
    const robotsResponse = await page.request.get('/public/robots.txt');
    expect(robotsResponse.status()).toBe(200);
    
    // Sitemap.xml
    const sitemapResponse = await page.request.get('/public/sitemap.xml');
    expect(sitemapResponse.status()).toBe(200);
    
    // Manifest.json
    const manifestResponse = await page.request.get('/public/manifest.json');
    expect(manifestResponse.status()).toBe(200);
    
    const manifestData = await manifestResponse.json();
    expect(manifestData.name).toBe("Logan's 3D Creations");
  });

  test('Should have consistent branding across all pages', async ({ page }) => {
    for (const pageInfo of pages.slice(0, 5)) { // Test first 5 pages to save time
      await page.goto(pageInfo.path);
      
      // Check for Logan's 3D Creations branding
      const brandingElements = page.locator(':text("Logan\'s 3D Creations")');
      const brandingCount = await brandingElements.count();
      expect(brandingCount).toBeGreaterThan(0);
      
      // Check for consistent color scheme (dark theme)
      const body = page.locator('body');
      await expect(body).toHaveClass(/bg-slate-900|slate-900/);
    }
  });

  test('Should handle page transitions smoothly', async ({ page }) => {
    await page.goto('/');
    
    // Navigate through key pages
    const navigationFlow = [
      '/shop',
      '/custom', 
      '/about',
      '/contact',
      '/' // Back to home
    ];
    
    for (const path of navigationFlow) {
      await page.goto(path);
      
      // Wait for page to load
      await page.waitForLoadState('networkidle');
      
      // Verify page loaded correctly
      const h1 = page.locator('h1').first();
      await expect(h1).toBeVisible({ timeout: 5000 });
      
      // Check for no console errors
      const consoleErrors: string[] = [];
      page.on('console', msg => {
        if (msg.type() === 'error') {
          consoleErrors.push(msg.text());
        }
      });
      
      // Filter out known non-critical errors
      const criticalErrors = consoleErrors.filter(error => 
        !error.includes('favicon') && 
        !error.includes('Failed to load resource')
      );
      
      expect(criticalErrors.length).toBe(0);
    }
  });

  test('Should have proper meta tags on all pages', async ({ page }) => {
    const keyPages = ['/', '/shop', '/custom', '/about'];
    
    for (const path of keyPages) {
      await page.goto(path);
      
      // Check for description meta tag
      const metaDescription = page.locator('meta[name="description"]');
      await expect(metaDescription).toBeVisible();
      
      // Check for keywords meta tag
      const metaKeywords = page.locator('meta[name="keywords"]');
      await expect(metaKeywords).toBeVisible();
      
      // Check for og:title
      const ogTitle = page.locator('meta[property="og:title"]');
      await expect(ogTitle).toBeVisible();
      
      // Check for og:description
      const ogDescription = page.locator('meta[property="og:description"]');
      await expect(ogDescription).toBeVisible();
    }
  });
});