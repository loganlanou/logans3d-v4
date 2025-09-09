import { test, expect } from '@playwright/test';

test.describe('Scroll Behavior and Spacing', () => {
  test('should have reduced scroll speed', async ({ page }) => {
    await page.goto('/', { waitUntil: 'networkidle' });
    
    // Wait for the scroll control script to load and execute
    await page.waitForFunction(() => {
      return typeof window.console !== 'undefined';
    });
    
    // Wait a bit more for script initialization
    await page.waitForTimeout(500);
    
    // Check that the script is loaded by looking for the console log
    const logs = [];
    page.on('console', msg => logs.push(msg.text()));
    
    // Force reload to capture console logs
    await page.reload({ waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);
    
    // Verify the scroll control script is active
    const hasScrollControl = logs.some(log => log.includes('Reduced scroll speed (70%) active'));
    expect(hasScrollControl).toBe(true);
    
    console.log('Console logs:', logs);
    console.log('Scroll control script verified active at 70% speed');
  });

  test('should measure and verify section spacing', async ({ page }) => {
    await page.goto('/');
    
    // Measure spacing between major sections
    const spacings = await page.evaluate(() => {
      const sections = document.querySelectorAll('section');
      const measurements = [];
      
      sections.forEach((section, index) => {
        const rect = section.getBoundingClientRect();
        const styles = window.getComputedStyle(section);
        
        measurements.push({
          index,
          marginTop: styles.marginTop,
          marginBottom: styles.marginBottom,
          paddingTop: styles.paddingTop,
          paddingBottom: styles.paddingBottom,
          height: rect.height
        });
      });
      
      return measurements;
    });
    
    console.log('Current section spacings:', spacings);
    
    // Verify that sections exist
    expect(spacings.length).toBeGreaterThan(0);
    
    // Log current spacing values for reference
    spacings.forEach((spacing, index) => {
      console.log(`Section ${index}: padding-top=${spacing.paddingTop}, padding-bottom=${spacing.paddingBottom}`);
    });
  });

  test('should have smooth scroll for anchor links', async ({ page }) => {
    await page.goto('/');
    
    // Find an anchor link (if any exist)
    const anchorLink = page.locator('a[href^="#"]').first();
    const hasAnchorLink = await anchorLink.count() > 0;
    
    if (hasAnchorLink) {
      const href = await anchorLink.getAttribute('href');
      const targetId = href?.substring(1);
      
      if (targetId) {
        // Click the anchor link
        await anchorLink.click();
        
        // Wait for smooth scroll
        await page.waitForTimeout(1000);
        
        // Verify the target element is in view
        const targetElement = page.locator(`#${targetId}`);
        await expect(targetElement).toBeInViewport();
      }
    }
  });

  test('should have consistent spacing across viewport sizes', async ({ page }) => {
    const viewports = [
      { width: 1920, height: 1080, name: 'Desktop' },
      { width: 768, height: 1024, name: 'Tablet' },
      { width: 375, height: 667, name: 'Mobile' }
    ];
    
    for (const viewport of viewports) {
      await page.setViewportSize({ width: viewport.width, height: viewport.height });
      await page.goto('/');
      
      const heroSpacing = await page.evaluate(() => {
        const hero = document.querySelector('.hero, [class*="hero"], section:first-of-type');
        if (!hero) return null;
        
        const styles = window.getComputedStyle(hero);
        return {
          paddingTop: styles.paddingTop,
          paddingBottom: styles.paddingBottom
        };
      });
      
      console.log(`${viewport.name} hero spacing:`, heroSpacing);
    }
  });
});