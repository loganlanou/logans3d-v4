import { test, expect } from '@playwright/test';

test.describe('Admin Dashboard - Orders, Quotes, and Events', () => {
  // Test the admin dashboard navigation and tab functionality
  test('should have working navigation tabs in admin dashboard', async ({ page }) => {
    // Navigate to admin dashboard
    await page.goto('/admin');

    // Check if the page loads correctly
    await expect(page).toHaveTitle(/Dashboard.*Logan's 3D Creations/);
    
    // Verify the navigation exists and has all expected tabs
    const nav = page.locator('nav');
    await expect(nav.getByRole('link', { name: 'Dashboard' })).toBeVisible();
    await expect(nav.getByRole('link', { name: 'Products' })).toBeVisible();
    await expect(nav.getByRole('link', { name: 'Orders' })).toBeVisible();
    await expect(nav.getByRole('link', { name: 'Quotes' })).toBeVisible();
    await expect(nav.getByRole('link', { name: 'Events' })).toBeVisible();
    await expect(nav.getByRole('link', { name: 'Developer' })).toBeVisible();

    // Check dashboard main content
    await expect(page.getByText('Dashboard')).toBeVisible();
    await expect(page.getByText('Total Products')).toBeVisible();
  });

  test('should navigate to Orders page and display orders interface', async ({ page }) => {
    // Navigate to orders page directly
    await page.goto('/admin/orders');

    // Check if the orders page loads correctly
    await expect(page).toHaveTitle(/Orders.*Logan's 3D Creations/);
    await expect(page.getByText('Orders Management')).toBeVisible();

    // Verify stats cards are present
    await expect(page.getByText('Total Orders')).toBeVisible();
    await expect(page.getByText('Pending Orders')).toBeVisible();
    await expect(page.getByText('Processing')).toBeVisible();
    await expect(page.getByText('Shipped')).toBeVisible();

    // Verify filter buttons are present
    await expect(page.getByRole('link', { name: 'All Orders' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Pending' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Processing' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Shipped' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Delivered' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Cancelled' })).toBeVisible();

    // Verify orders table headers
    await expect(page.getByText('Order ID')).toBeVisible();
    await expect(page.getByText('Customer')).toBeVisible();
    await expect(page.getByText('Total')).toBeVisible();
    await expect(page.getByText('Status')).toBeVisible();
    await expect(page.getByText('Payment')).toBeVisible();
    await expect(page.getByText('Fulfillment')).toBeVisible();
    await expect(page.getByText('Date')).toBeVisible();
    await expect(page.getByText('Actions')).toBeVisible();
  });

  test('should navigate to Quotes page and display quotes interface', async ({ page }) => {
    // Navigate to quotes page directly
    await page.goto('/admin/quotes');

    // Check if the quotes page loads correctly
    await expect(page).toHaveTitle(/Quote Requests.*Logan's 3D Creations/);
    await expect(page.getByText('Quote Requests')).toBeVisible();

    // Verify stats cards are present
    await expect(page.getByText('Total Quotes')).toBeVisible();
    await expect(page.getByText('Pending Review')).toBeVisible();
    await expect(page.getByText('Quoted')).toBeVisible();
    await expect(page.getByText('Approved')).toBeVisible();

    // Verify filter buttons are present
    await expect(page.getByRole('link', { name: 'All Quotes' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Pending' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Reviewing' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Quoted' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Approved' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Rejected' })).toBeVisible();

    // Verify quotes table headers
    await expect(page.getByText('Quote ID')).toBeVisible();
    await expect(page.getByText('Customer')).toBeVisible();
    await expect(page.getByText('Project')).toBeVisible();
    await expect(page.getByText('Status')).toBeVisible();
    await expect(page.getByText('Quoted Price')).toBeVisible();
    await expect(page.getByText('Deadline')).toBeVisible();
    await expect(page.getByText('Created')).toBeVisible();
    await expect(page.getByText('Actions')).toBeVisible();
  });

  test('should navigate to Events page and display events interface', async ({ page }) => {
    // Navigate to events page directly
    await page.goto('/admin/events');

    // Check if the events page loads correctly
    await expect(page).toHaveTitle(/Events.*Logan's 3D Creations/);
    await expect(page.getByText('Events Management')).toBeVisible();

    // Verify stats cards are present
    await expect(page.getByText('Total Events')).toBeVisible();
    await expect(page.getByText('Upcoming Events')).toBeVisible();
    await expect(page.getByText('Active Events')).toBeVisible();
    await expect(page.getByText('Past Events')).toBeVisible();

    // Verify Add Event button
    await expect(page.getByRole('link', { name: '+ Add Event' })).toBeVisible();

    // Verify filter buttons are present
    await expect(page.getByRole('link', { name: 'All Events' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Upcoming' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Active' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Past' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Inactive' })).toBeVisible();

    // Verify events table headers
    await expect(page.getByText('Title')).toBeVisible();
    await expect(page.getByText('Location')).toBeVisible();
    await expect(page.getByText('Start Date')).toBeVisible();
    await expect(page.getByText('End Date')).toBeVisible();
    await expect(page.getByText('Status')).toBeVisible();
    await expect(page.getByText('Actions')).toBeVisible();
  });

  test('should open Add Event form and display all required fields', async ({ page }) => {
    // Navigate to add event form
    await page.goto('/admin/events/new');

    // Check if the form page loads correctly
    await expect(page).toHaveTitle(/Event Form.*Logan's 3D Creations/);
    await expect(page.getByText('Add New Event')).toBeVisible();

    // Verify back link
    await expect(page.getByRole('link', { name: '← Back to Events' })).toBeVisible();

    // Verify form fields are present
    await expect(page.getByLabel('Title')).toBeVisible();
    await expect(page.getByLabel('Description')).toBeVisible();
    await expect(page.getByLabel('Location')).toBeVisible();
    await expect(page.getByLabel('Event URL')).toBeVisible();
    await expect(page.getByLabel('Address')).toBeVisible();
    await expect(page.getByLabel('Start Date & Time')).toBeVisible();
    await expect(page.getByLabel('End Date & Time')).toBeVisible();
    await expect(page.getByLabel('Active Event')).toBeVisible();

    // Verify form buttons
    await expect(page.getByRole('button', { name: 'Create Event' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Cancel' })).toBeVisible();
  });

  test('should navigate between admin tabs correctly', async ({ page }) => {
    // Start at admin dashboard
    await page.goto('/admin');
    await expect(page.getByText('Dashboard')).toBeVisible();

    // Navigate to Orders
    await page.getByRole('link', { name: 'Orders' }).click();
    await expect(page).toHaveURL('/admin/orders');
    await expect(page.getByText('Orders Management')).toBeVisible();

    // Navigate to Quotes
    await page.getByRole('link', { name: 'Quotes' }).click();
    await expect(page).toHaveURL('/admin/quotes');
    await expect(page.getByText('Quote Requests')).toBeVisible();

    // Navigate to Events
    await page.getByRole('link', { name: 'Events' }).click();
    await expect(page).toHaveURL('/admin/events');
    await expect(page.getByText('Events Management')).toBeVisible();

    // Navigate back to Dashboard
    await page.getByRole('link', { name: 'Dashboard' }).click();
    await expect(page).toHaveURL('/admin');
    await expect(page.getByText('Dashboard')).toBeVisible();
  });

  test('should have responsive design and proper CSS styling', async ({ page }) => {
    // Test desktop view
    await page.setViewportSize({ width: 1280, height: 720 });
    await page.goto('/admin');
    
    // Check that navigation is horizontal on desktop
    const nav = page.locator('nav');
    await expect(nav).toBeVisible();
    
    // Navigate to orders and check responsive layout
    await page.goto('/admin/orders');
    
    // Check that stats cards are in grid layout
    const statsGrid = page.locator('.admin-stats-grid');
    await expect(statsGrid).toBeVisible();

    // Test tablet view
    await page.setViewportSize({ width: 768, height: 1024 });
    await page.goto('/admin/events/new');
    
    // Check that form is still usable on tablet
    await expect(page.getByLabel('Title')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Create Event' })).toBeVisible();

    // Test mobile view
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/admin/quotes');
    
    // Check that content is accessible on mobile
    await expect(page.getByText('Quote Requests')).toBeVisible();
  });

  test('should verify all admin interface CSS classes are working', async ({ page }) => {
    await page.goto('/admin/orders');
    
    // Verify admin CSS classes are applied
    const adminCard = page.locator('.admin-card').first();
    await expect(adminCard).toBeVisible();

    const adminTable = page.locator('.admin-table');
    await expect(adminTable).toBeVisible();

    const adminButton = page.locator('.admin-btn').first();
    await expect(adminButton).toBeVisible();

    // Check that text is readable (not transparent/invisible)
    const cardTitle = page.locator('.admin-card-title').first();
    await expect(cardTitle).toBeVisible();
    
    // Verify buttons have proper styling and are clickable
    const filterButtons = page.locator('.admin-btn-sm');
    const buttonCount = await filterButtons.count();
    expect(buttonCount).toBeGreaterThan(0);
    
    // Test button hover state by checking if button is interactive
    const firstFilterButton = filterButtons.first();
    await expect(firstFilterButton).toBeVisible();
  });

  test('should handle empty data states gracefully', async ({ page }) => {
    // Test empty orders page
    await page.goto('/admin/orders');
    
    // Since there's no data, should show "No orders found" message or empty table
    const ordersTable = page.locator('.admin-table tbody');
    await expect(ordersTable).toBeVisible();

    // Test empty quotes page
    await page.goto('/admin/quotes');
    const quotesTable = page.locator('.admin-table tbody');
    await expect(quotesTable).toBeVisible();

    // Test empty events page
    await page.goto('/admin/events');
    const eventsTable = page.locator('.admin-table tbody');
    await expect(eventsTable).toBeVisible();
  });

  test('should have proper accessibility features', async ({ page }) => {
    await page.goto('/admin/events/new');
    
    // Check for proper form labels
    const titleField = page.getByLabel('Title');
    await expect(titleField).toBeVisible();
    await expect(titleField).toHaveAttribute('required');

    const descriptionField = page.getByLabel('Description');
    await expect(descriptionField).toBeVisible();

    // Check for proper button roles
    const createButton = page.getByRole('button', { name: 'Create Event' });
    await expect(createButton).toBeVisible();

    const cancelLink = page.getByRole('link', { name: 'Cancel' });
    await expect(cancelLink).toBeVisible();

    // Check that required fields are marked
    const requiredFields = page.locator('input[required]');
    const requiredCount = await requiredFields.count();
    expect(requiredCount).toBeGreaterThan(0);
  });

  // Test form validation
  test('should validate required fields in event form', async ({ page }) => {
    await page.goto('/admin/events/new');
    
    // Try to submit form without required fields
    await page.getByRole('button', { name: 'Create Event' }).click();
    
    // Browser should prevent form submission and show validation messages
    // The title field should be focused or show validation
    const titleField = page.getByLabel('Title');
    await expect(titleField).toBeFocused();
  });
});