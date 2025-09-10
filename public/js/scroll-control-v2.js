// Reduce scroll speed by half
(function() {
  // Intercept wheel events to reduce scroll speed
  let scrollTimeout;
  
  document.addEventListener('wheel', function(e) {
    // Prevent default scroll
    e.preventDefault();
    
    // Calculate scroll amount (150% of original for faster scrolling)
    const scrollAmount = e.deltaY * 1.5;
    
    // Apply the increased scroll
    window.scrollBy({
      top: scrollAmount,
      behavior: 'auto' // Use auto for immediate response
    });
  }, { passive: false });
  
  // Smooth scrolling for anchor links
  document.addEventListener('DOMContentLoaded', function() {
    const links = document.querySelectorAll('a[href^="#"]');
    links.forEach(link => {
      link.addEventListener('click', function(e) {
        const targetId = this.getAttribute('href').substring(1);
        const targetElement = document.getElementById(targetId);
        
        if (targetElement) {
          e.preventDefault();
          targetElement.scrollIntoView({
            behavior: 'smooth',
            block: 'start'
          });
        }
      });
    });
  });
  
  console.log('🚀 FAST SCROLL MODE ACTIVE - 150% speed - Cache bust v2');
})();