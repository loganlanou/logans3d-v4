// Email Capture Popup
(function() {
    'use strict';

    const COOKIE_NAME = 'email_capture_shown';
    const COOKIE_DAYS = 7;
    const SHOW_DELAY = 10000; // 10 seconds
    const SCROLL_THRESHOLD = 0.5; // 50% page scroll
    const EXIT_INTENT_SENSITIVITY = 10; // pixels from top

    let popupShown = false;
    let scrollTriggered = false;
    let timeTriggered = false;

    // Check if popup was already shown
    function getCookie(name) {
        const value = `; ${document.cookie}`;
        const parts = value.split(`; ${name}=`);
        if (parts.length === 2) return parts.pop().split(';').shift();
        return null;
    }

    function setCookie(name, value, days) {
        const expires = new Date();
        expires.setTime(expires.getTime() + (days * 24 * 60 * 60 * 1000));
        document.cookie = `${name}=${value};expires=${expires.toUTCString()};path=/`;
    }

    function shouldShowPopup() {
        // Don't show if already shown in this session
        if (popupShown) return false;

        // Don't show if cookie exists
        if (getCookie(COOKIE_NAME)) return false;

        // Don't show on checkout or cart pages
        if (window.location.pathname.includes('/checkout') ||
            window.location.pathname.includes('/cart')) {
            return false;
        }

        return true;
    }

    function showPopup() {
        if (!shouldShowPopup()) return;

        popupShown = true;
        const popup = document.getElementById('email-capture-popup');
        if (popup) {
            popup.classList.remove('hidden');
            popup.classList.add('flex');
            document.body.style.overflow = 'hidden'; // Prevent scroll
        }
    }

    function hidePopup() {
        const popup = document.getElementById('email-capture-popup');
        if (popup) {
            popup.classList.add('hidden');
            popup.classList.remove('flex');
            document.body.style.overflow = ''; // Restore scroll
        }
        setCookie(COOKIE_NAME, 'true', COOKIE_DAYS);
    }

    function setupEventListeners() {
        // Time-based trigger
        setTimeout(() => {
            if (!timeTriggered && !popupShown) {
                timeTriggered = true;
                showPopup();
            }
        }, SHOW_DELAY);

        // Scroll-based trigger
        window.addEventListener('scroll', () => {
            if (scrollTriggered || popupShown) return;

            const scrollPercent = (window.scrollY / (document.documentElement.scrollHeight - window.innerHeight));
            if (scrollPercent >= SCROLL_THRESHOLD) {
                scrollTriggered = true;
                showPopup();
            }
        });

        // Exit intent trigger
        document.addEventListener('mouseleave', (e) => {
            if (popupShown) return;
            if (e.clientY < EXIT_INTENT_SENSITIVITY) {
                showPopup();
            }
        });

        // Close button
        const closeBtn = document.getElementById('email-popup-close');
        if (closeBtn) {
            closeBtn.addEventListener('click', hidePopup);
        }

        // Click outside to close
        const popup = document.getElementById('email-capture-popup');
        if (popup) {
            popup.addEventListener('click', (e) => {
                if (e.target === popup) {
                    hidePopup();
                }
            });
        }

        // Form submission
        const form = document.getElementById('email-capture-form');
        if (form) {
            form.addEventListener('submit', handleSubmit);
        }
    }

    async function handleSubmit(e) {
        e.preventDefault();

        const form = e.target;
        const email = form.querySelector('input[name="email"]').value;
        const firstName = form.querySelector('input[name="first_name"]')?.value || '';
        const submitBtn = form.querySelector('button[type="submit"]');
        const errorDiv = document.getElementById('email-popup-error');
        const successDiv = document.getElementById('email-popup-success');

        // Disable submit button
        submitBtn.disabled = true;
        submitBtn.textContent = 'Processing...';

        // Hide previous messages
        if (errorDiv) errorDiv.classList.add('hidden');
        if (successDiv) successDiv.classList.add('hidden');

        try {
            const response = await fetch('/api/promotions/capture-email', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    email: email,
                    first_name: firstName,
                    source: 'popup'
                })
            });

            const data = await response.json();

            if (response.ok && data.success) {
                // Show success message with code
                if (successDiv) {
                    successDiv.textContent = `Success! Your code: ${data.code}. Check your email!`;
                    successDiv.classList.remove('hidden');
                }

                // Hide form, show success
                form.classList.add('hidden');

                // Auto-close after 5 seconds
                setTimeout(() => {
                    hidePopup();
                }, 5000);
            } else {
                throw new Error(data.error || 'Failed to process request');
            }
        } catch (error) {
            console.error('Email capture error:', error);
            if (errorDiv) {
                errorDiv.textContent = error.message || 'Something went wrong. Please try again.';
                errorDiv.classList.remove('hidden');
            }
            submitBtn.disabled = false;
            submitBtn.textContent = 'Get My Discount';
        }
    }

    // Initialize when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', setupEventListeners);
    } else {
        setupEventListeners();
    }
})();
