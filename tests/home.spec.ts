import { test, expect } from '@playwright/test';

test.describe('Home Page', () => {
  test('should load the home page successfully', async ({ page }) => {
    await page.goto('/');
    
    // Check that the page loads and has the correct title
    await expect(page).toHaveTitle(/Logan's 3D Creations/);
    
    // Check for the main heading
    await expect(page.getByRole('heading', { name: 'Bring Your Ideas to Life' })).toBeVisible();
    
    // Check for the hero subtitle
    await expect(page.getByText('Professional 3D printing services')).toBeVisible();
    
    // Check that navigation is present
    await expect(page.getByRole('navigation')).toBeVisible();
    
    // Check for primary CTA buttons
    await expect(page.getByRole('link', { name: 'Explore Products' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Get Custom Quote' })).toBeVisible();
  });

  test('should have responsive navigation', async ({ page }) => {
    await page.goto('/');
    
    // Desktop navigation should be visible on large screens
    await page.setViewportSize({ width: 1200, height: 800 });
    await expect(page.locator('.hidden.md\\:block')).toBeVisible();
    
    // Mobile menu button should be visible on small screens
    await page.setViewportSize({ width: 375, height: 667 });
    await expect(page.locator('.md\\:hidden')).toBeVisible();
  });

  test('should display feature cards', async ({ page }) => {
    await page.goto('/');
    
    // Check for the three main feature cards
    await expect(page.getByText('Precision Quality')).toBeVisible();
    await expect(page.getByText('Custom Design')).toBeVisible();
    await expect(page.getByText('Educational Focus')).toBeVisible();
    
    // Check that feature cards have descriptions
    await expect(page.getByText('State-of-the-art 3D printing technology')).toBeVisible();
    await expect(page.getByText('From concept to creation')).toBeVisible();
    await expect(page.getByText('Interactive workshops and maker education')).toBeVisible();
  });

  test('should have working navigation links', async ({ page }) => {
    await page.goto('/');
    
    // Test navigation to different sections (even if they don't exist yet, links should be present)
    await expect(page.getByRole('link', { name: 'Shop' })).toHaveAttribute('href', '/shop');
    await expect(page.getByRole('link', { name: 'Custom Orders' })).toHaveAttribute('href', '/custom');
    await expect(page.getByRole('link', { name: 'Events' })).toHaveAttribute('href', '/events');
    await expect(page.getByRole('link', { name: 'Portfolio' })).toHaveAttribute('href', '/portfolio');
    await expect(page.getByRole('link', { name: 'About' })).toHaveAttribute('href', '/about');
    await expect(page.getByRole('link', { name: 'Contact' })).toHaveAttribute('href', '/contact');
  });

  test('should have proper footer', async ({ page }) => {
    await page.goto('/');
    
    // Scroll to footer
    await page.locator('footer').scrollIntoViewIfNeeded();
    
    // Check for footer content
    await expect(page.locator('footer')).toBeVisible();
    await expect(page.getByText('Logan\'s 3D Creations')).toBeVisible();
    await expect(page.getByText('Custom 3D printing solutions')).toBeVisible();
    
    // Check for current year in copyright
    const currentYear = new Date().getFullYear();
    await expect(page.getByText(`© ${currentYear} Logan's 3D Creations`)).toBeVisible();
  });

  test('should be accessible', async ({ page }) => {
    await page.goto('/');
    
    // Check for proper heading hierarchy
    const h1 = page.locator('h1');
    await expect(h1).toHaveCount(1);
    await expect(h1).toContainText('Bring Your Ideas to Life');
    
    // Check that images have alt text
    const images = page.locator('img');
    const imageCount = await images.count();
    for (let i = 0; i < imageCount; i++) {
      await expect(images.nth(i)).toHaveAttribute('alt');
    }
    
    // Check for proper link text (no "click here" or "read more")
    const links = page.locator('a');
    const linkCount = await links.count();
    for (let i = 0; i < linkCount; i++) {
      const linkText = await links.nth(i).textContent();
      expect(linkText?.toLowerCase()).not.toMatch(/click here|read more/);
    }
  });
});