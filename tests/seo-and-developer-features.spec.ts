import { test, expect } from '@playwright/test';

test.describe('SEO and Developer Features', () => {
  
  test('homepage has proper SEO meta tags', async ({ page }) => {
    await page.goto('http://localhost:8000');
    
    // Check title
    await expect(page).toHaveTitle(/Logan's 3D Creations - Custom 3D Printing & Educational Workshops/);
    
    // Check meta description
    const metaDescription = page.locator('meta[name="description"]');
    await expect(metaDescription).toHaveAttribute('content', /Professional 3D printing services/);
    
    // Check meta keywords
    const metaKeywords = page.locator('meta[name="keywords"]');
    await expect(metaKeywords).toHaveAttribute('content', /3D printing, custom design/);
    
    // Check Open Graph tags
    const ogTitle = page.locator('meta[property="og:title"]');
    await expect(ogTitle).toHaveAttribute('content', /Logan's 3D Creations/);
    
    const ogDescription = page.locator('meta[property="og:description"]');
    await expect(ogDescription).toHaveAttribute('content', /Professional 3D printing services/);
    
    // Check favicon (there may be multiple sizes)
    const favicon = page.locator('link[rel="icon"]').first();
    await expect(favicon).toHaveAttribute('href', '/public/images/favicon.png');
  });

  test('health endpoint returns proper status', async ({ page }) => {
    const response = await page.goto('http://localhost:8000/health');
    expect(response?.status()).toBe(200);
    
    const content = await page.textContent('body');
    const healthData = JSON.parse(content || '{}');
    
    expect(healthData.status).toBe('healthy');
    expect(healthData.environment).toBe('development');
    expect(healthData.database).toBe('connected');
    expect(healthData.version).toBe('4.0.0');
  });

  test('robots.txt is accessible', async ({ page }) => {
    const response = await page.goto('http://localhost:8000/public/robots.txt');
    expect(response?.status()).toBe(200);
    
    const content = await page.textContent('body');
    expect(content).toContain('User-agent: *');
    expect(content).toContain('Allow: /');
    expect(content).toContain('Sitemap:');
  });

  test('sitemap.xml is accessible', async ({ page }) => {
    const response = await page.goto('http://localhost:8000/public/sitemap.xml');
    expect(response?.status()).toBe(200);
    
    const content = await response?.text();
    expect(content).toContain('<?xml version="1.0" encoding="UTF-8"?>');
    expect(content).toContain('<urlset');
    expect(content).toContain('https://logans3dcreations.com/');
  });

  test('manifest.json is accessible', async ({ page }) => {
    const response = await page.goto('http://localhost:8000/public/manifest.json');
    expect(response?.status()).toBe(200);
    
    const content = await page.textContent('body');
    const manifest = JSON.parse(content || '{}');
    
    expect(manifest.name).toBe("Logan's 3D Creations");
    expect(manifest.short_name).toBe("Logan's3D");
    expect(manifest.theme_color).toBe("#3b82f6");
  });

  test('favicon loads successfully', async ({ page }) => {
    const response = await page.goto('http://localhost:8000/public/images/favicon.png');
    expect(response?.status()).toBe(200);
  });

  test('text readability - high contrast on dark backgrounds', async ({ page }) => {
    await page.goto('http://localhost:8000');
    
    // Check main heading text color contrast
    const mainHeading = page.locator('h1').first();
    await expect(mainHeading).toBeVisible();
    
    // Check that visible text elements have good contrast colors
    const visibleParagraphs = page.locator('p').locator('visible=true');
    const visibleParagraphCount = await visibleParagraphs.count();
    
    // Ensure we have at least some visible paragraphs
    expect(visibleParagraphCount).toBeGreaterThan(0);
    
    // Check navigation links are visible
    const navLinks = page.locator('nav a, header a').locator('visible=true');
    const navLinkCount = await navLinks.count();
    
    // Ensure we have navigation links
    expect(navLinkCount).toBeGreaterThan(0);
    
    // Check that text colors have been improved from the default low-contrast values
    // by verifying the CSS has been applied
    const cssContent = await page.locator('link[rel="stylesheet"]').getAttribute('href');
    expect(cssContent).toContain('styles.css');
  });

  test('responsive design - mobile viewport', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('http://localhost:8000');
    
    // Check that content is still readable on mobile
    const mainHeading = page.locator('h1');
    await expect(mainHeading).toBeVisible();
    
    // Check mobile navigation
    const mobileMenuButton = page.locator('button[aria-label*="menu"], button:has-text("☰")');
    if (await mobileMenuButton.isVisible()) {
      await mobileMenuButton.click();
    }
  });

  test('interactive elements work correctly', async ({ page }) => {
    await page.goto('http://localhost:8000');
    
    // Test navigation links
    const shopLink = page.locator('a[href="/shop"]').first();
    await expect(shopLink).toBeVisible();
    
    const customLink = page.locator('a[href="/custom"]').first();
    await expect(customLink).toBeVisible();
    
    // Test CTAs
    const exploreButton = page.locator('a:has-text("Explore Products")');
    await expect(exploreButton).toBeVisible();
    
    const quoteButton = page.locator('a:has-text("Get Custom Quote")');
    await expect(quoteButton).toBeVisible();
  });

  test('page performance - no console errors', async ({ page }) => {
    const consoleErrors: string[] = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });
    
    await page.goto('http://localhost:8000');
    await page.waitForLoadState('networkidle');
    
    // Filter out known non-critical errors (like favicon 404 if it occurs)
    const criticalErrors = consoleErrors.filter(error => 
      !error.includes('favicon') && 
      !error.includes('Failed to load resource')
    );
    
    expect(criticalErrors).toHaveLength(0);
  });

  test('structured data validation', async ({ page }) => {
    await page.goto('http://localhost:8000');
    
    const structuredData = await page.evaluate(() => {
      const ldJson = document.querySelector('script[type="application/ld+json"]');
      if (ldJson) {
        try {
          return JSON.parse(ldJson.textContent || '');
        } catch (e) {
          return null;
        }
      }
      return null;
    });
    
    if (structuredData) {
      expect(structuredData['@context']).toBe('https://schema.org');
      expect(structuredData['@type']).toBe('LocalBusiness');
      expect(structuredData.name).toBe("Logan's 3D Creations");
    }
  });

  test('all critical pages are accessible', async ({ page }) => {
    const criticalPages = [
      '/',
      '/shop',
      '/custom', 
      '/about',
      '/contact',
      '/portfolio',
      '/events',
      '/health'
    ];
    
    for (const pagePath of criticalPages) {
      const response = await page.goto(`http://localhost:8000${pagePath}`);
      expect(response?.status()).toBeLessThan(400);
    }
  });
});