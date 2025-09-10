const { chromium } = require('@playwright/test');

(async () => {
  const browser = await chromium.launch({ headless: false });
  const context = await browser.newContext();
  const page = await context.newPage();
  
  console.log('Opening website...');
  await page.goto('http://localhost:8000');
  
  console.log('Waiting for page to load...');
  await page.waitForLoadState('networkidle');
  
  // Get initial scroll position
  const initialScroll = await page.evaluate(() => window.scrollY);
  console.log('Initial scroll position:', initialScroll);
  
  // Test wheel scrolling
  console.log('\nTesting wheel scroll behavior...');
  
  // Scroll down with wheel
  await page.mouse.wheel(0, 100);
  await page.waitForTimeout(100);
  
  const afterWheelScroll = await page.evaluate(() => window.scrollY);
  const wheelScrollDistance = afterWheelScroll - initialScroll;
  console.log('After wheel scroll (deltaY=100):', afterWheelScroll);
  console.log('Actual scroll distance:', wheelScrollDistance);
  console.log('Scroll multiplier applied:', (wheelScrollDistance / 100).toFixed(2));
  
  // Test multiple wheel scrolls
  console.log('\nTesting multiple rapid scrolls...');
  const beforeRapidScroll = await page.evaluate(() => window.scrollY);
  
  for (let i = 0; i < 5; i++) {
    await page.mouse.wheel(0, 100);
    await page.waitForTimeout(50);
  }
  
  const afterRapidScroll = await page.evaluate(() => window.scrollY);
  const rapidScrollDistance = afterRapidScroll - beforeRapidScroll;
  console.log('After 5 rapid scrolls (5 x deltaY=100):', afterRapidScroll);
  console.log('Total scroll distance:', rapidScrollDistance);
  console.log('Average per scroll:', (rapidScrollDistance / 5).toFixed(2));
  
  // Compare to standard behavior
  console.log('\n=== Scroll Behavior Analysis ===');
  if (wheelScrollDistance === 100) {
    console.log('✅ Scroll speed is NORMAL (1:1 ratio with input)');
  } else if (wheelScrollDistance < 100) {
    console.log(`⚠️ Scroll speed is REDUCED (${(wheelScrollDistance / 100 * 100).toFixed(0)}% of normal)`);
  } else {
    console.log(`⚠️ Scroll speed is INCREASED (${(wheelScrollDistance / 100 * 100).toFixed(0)}% of normal)`);
  }
  
  // Test smooth scrolling to anchor
  console.log('\nTesting anchor link smooth scroll...');
  const hasAnchorLinks = await page.evaluate(() => {
    const links = document.querySelectorAll('a[href^="#"]');
    return links.length > 0;
  });
  
  if (hasAnchorLinks) {
    console.log('Found anchor links, testing smooth scroll behavior...');
    // Click first anchor link if exists
    await page.click('a[href^="#"]').catch(() => {
      console.log('No clickable anchor links found');
    });
  } else {
    console.log('No anchor links found on page');
  }
  
  // Check console for scroll control message
  const consoleMessages = [];
  page.on('console', msg => consoleMessages.push(msg.text()));
  await page.reload();
  await page.waitForTimeout(1000);
  
  const scrollControlMessage = consoleMessages.find(msg => 
    msg.includes('Scroll speed control activated')
  );
  
  if (scrollControlMessage) {
    console.log('\n' + scrollControlMessage);
  }
  
  console.log('\n=== Test Complete ===');
  console.log('Press Ctrl+C to close the browser');
  
  // Keep browser open for manual testing
  await page.waitForTimeout(300000);
  await browser.close();
})();