// Cart rendering functionality for cart page
document.addEventListener('DOMContentLoaded', function() {
    async function fetchAndRenderCart() {
        try {
            const response = await fetch('/api/cart');
            if (!response.ok) {
                throw new Error('Failed to fetch cart');
            }
            const cart = await response.json();
            renderCart(cart);

            // Load saved shipping selection after cart renders
            if (cart.items && cart.items.length > 0 && window.shippingManager) {
                await window.shippingManager.loadSavedShipping();
            }
        } catch (error) {
            console.error('Error fetching cart:', error);
            renderCart({ items: [] });
        }
    }
    
    function renderCart(cart) {
        const items = cart.items || [];
        const emptyCart = document.getElementById('empty-cart');
        const cartItemsSection = document.getElementById('cart-items-section');
        const cartItems = document.getElementById('cart-items');
        const cartSummary = document.getElementById('cart-summary');
        const cartShipping = document.getElementById('cart-shipping');
        const cartTotal = document.getElementById('cart-total');
        const cartSubtotal = document.getElementById('cart-subtotal');
        const checkoutSteps = document.getElementById('checkout-steps');

        if (items.length === 0) {
            emptyCart.classList.remove('hidden');
            if (cartItemsSection) cartItemsSection.classList.add('hidden');
            cartSummary.classList.add('hidden');
            cartShipping.classList.add('hidden');
            if (checkoutSteps) checkoutSteps.classList.add('hidden');
            return;
        }

        emptyCart.classList.add('hidden');
        if (cartItemsSection) cartItemsSection.classList.remove('hidden');
        cartSummary.classList.remove('hidden');
        cartShipping.classList.remove('hidden');
        if (checkoutSteps) checkoutSteps.classList.remove('hidden');

        // Initialize checkout button in disabled state
        initializeCheckoutButton();
        
        // Render cart items using string concatenation
        cartItems.innerHTML = items.map(item => {
            // Fix image URL - check if it already has the full path
            let imageSrc = '';
            if (item.image_url) {
                if (item.image_url.startsWith('/public/')) {
                    imageSrc = item.image_url;
                } else if (item.image_url.startsWith('/images/')) {
                    imageSrc = '/public' + item.image_url;
                } else {
                    imageSrc = '/public/images/products/' + item.image_url;
                }
            }

            const imageHtml = imageSrc ?
                '<img src="' + imageSrc + '" alt="' + item.name + '" class="w-24 h-24 rounded-xl object-cover bg-slate-700/50">' :
                '<div class="w-24 h-24 rounded-xl bg-slate-700/50 flex items-center justify-center">' +
                    '<svg class="w-12 h-12 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">' +
                        '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"></path>' +
                    '</svg>' +
                '</div>';
                
            return '<div class="bg-gradient-to-br from-slate-800/50 to-slate-900/50 rounded-2xl border border-slate-700/50 backdrop-blur-sm shadow-xl p-6">' +
                '<div class="flex flex-col md:flex-row gap-6">' +
                    '<div class="flex-shrink-0">' + imageHtml + '</div>' +
                    '<div class="flex-1 min-w-0">' +
                        '<h3 class="text-xl font-bold text-white mb-2">' + item.name + '</h3>' +
                        '<p class="text-lg font-semibold text-emerald-400 mb-4">$' + (item.price_cents / 100).toFixed(2) + '</p>' +
                        '<div class="flex items-center justify-between">' +
                            '<div class="flex items-center space-x-3">' +
                                '<label class="text-white font-medium">Qty:</label>' +
                                '<button class="cart-update-btn w-8 h-8 rounded-full bg-slate-600/50 hover:bg-slate-500/50 text-white flex items-center justify-center transition-colors duration-200" data-cart-item-id="' + item.id + '" data-quantity="' + (item.quantity - 1) + '">−</button>' +
                                '<span class="text-white font-semibold min-w-[2rem] text-center">' + item.quantity + '</span>' +
                                '<button class="cart-update-btn w-8 h-8 rounded-full bg-slate-600/50 hover:bg-slate-500/50 text-white flex items-center justify-center transition-colors duration-200" data-cart-item-id="' + item.id + '" data-quantity="' + (item.quantity + 1) + '">+</button>' +
                            '</div>' +
                            '<button class="cart-remove-btn text-red-400 hover:text-red-300 font-semibold transition-colors duration-200" data-cart-item-id="' + item.id + '">' +
                                '<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">' +
                                    '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path>' +
                                '</svg>' +
                            '</button>' +
                        '</div>' +
                    '</div>' +
                '</div>' +
            '</div>';
        }).join('');
        
        // Update subtotal and total - API returns totalCents (camelCase), not total_cents (snake_case)
        const subtotal = cart.totalCents || 0;
        cartSubtotal.textContent = '$' + (subtotal / 100).toFixed(2);
        cartTotal.textContent = '$' + (subtotal / 100).toFixed(2); // Initial total = subtotal

        // Initialize shipping options
        if (window.shippingManager) {
            window.shippingManager.updateShippingUI('address-required');
        }
    }
    
    // Initialize checkout button in disabled state
    function initializeCheckoutButton() {
        const checkoutBtn = document.getElementById('proceed-checkout-btn');
        const checkoutBtnText = document.getElementById('checkout-btn-text');

        if (checkoutBtn) {
            checkoutBtn.disabled = true;
            checkoutBtn.classList.remove('bg-gradient-to-r', 'from-blue-600', 'to-emerald-600', 'hover:from-blue-700', 'hover:to-emerald-700', 'shadow-lg', 'hover:shadow-xl', 'hover:shadow-emerald-500/25', 'transform', 'hover:-translate-y-1');
            checkoutBtn.classList.add('bg-slate-600', 'cursor-not-allowed', 'opacity-50');

            if (checkoutBtnText) {
                checkoutBtnText.textContent = 'Select Shipping to Continue';
            }
        }
    }

    // Make initializeCheckoutButton globally available
    window.initializeCheckoutButton = initializeCheckoutButton;

    // Initial render
    fetchAndRenderCart();

    // Make fetchAndRenderCart globally available for cart updates
    window.refreshCart = fetchAndRenderCart;
});