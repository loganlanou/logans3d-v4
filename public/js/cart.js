// Validate cart session on page load - clears cart if checkout was completed
async function validateCartSession() {
    try {
        const response = await fetch('/api/cart/validate', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            }
        });

        if (response.ok) {
            const data = await response.json();
            if (data.should_clear) {
                // Clear localStorage cart
                localStorage.removeItem('stripe_cart');

                // Update cart count display
                const cartCount = document.getElementById('cart-count');
                if (cartCount) {
                    cartCount.textContent = '0';
                    cartCount.classList.add('hidden');
                }
            }
        }
    } catch (error) {
        console.error('Error validating cart session:', error);
        // Silently fail - don't disrupt user experience
    }
}

// Run validation when page loads
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', validateCartSession);
} else {
    validateCartSession();
}

// Buy Now functionality - goes directly to Stripe checkout
async function buyNow(productId, productName, productPrice, quantity = 1, productSkuId = '') {
    try {
        const response = await fetch('/checkout/create-session-single', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                productId: productId,
                productSkuId: productSkuId,
                quantity: parseInt(quantity)
            })
        });

        if (!response.ok) {
            throw new Error('Failed to create checkout session');
        }

        const data = await response.json();
        
        // Redirect to Stripe checkout
        if (data.url) {
            window.location.href = data.url;
        } else {
            throw new Error('No checkout URL received');
        }

    } catch (error) {
        console.error('Error creating checkout session:', error);
        showToast('Failed to start checkout', 'error');
    }
}

// Cart functionality
async function addToCart(productId, quantity = 1, productName = '', productSkuId = '') {
    try {
        const response = await fetch('/api/cart/add', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                productId: productId,
                productSkuId: productSkuId,
                quantity: parseInt(quantity)
            })
        });

        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || 'Failed to add item to cart');
        }

        const data = await response.json();
        const displayName = productName || 'item';
        showToast(`Added ${displayName} to cart!`, 'success');
        
        // Update cart count if element exists
        await updateCartCount();
        
    } catch (error) {
        console.error('Error adding to cart:', error);
        showToast(error.message, 'error');
    }
}

async function removeFromCart(cartItemId) {
    try {
        const response = await fetch(`/api/cart/item/${cartItemId}`, {
            method: 'DELETE'
        });

        if (!response.ok) {
            throw new Error('Failed to remove item from cart');
        }

        showToast('Item removed from cart', 'success');
        
        // Refresh cart if on cart page
        if (window.location.pathname === '/cart' && window.refreshCart) {
            window.refreshCart();
        }
        
        // Refresh modal if it's open (check if modal is visible)
        const cartModal = document.getElementById('cart-modal');
        if (cartModal && getComputedStyle(cartModal).display !== 'none') {
            setTimeout(async () => {
                const modalResponse = await fetch('/api/cart');
                if (modalResponse.ok) {
                    const cart = await modalResponse.json();
                    renderCartModal(cart);
                }
            }, 100);
        }
        
        await updateCartCount();
        
    } catch (error) {
        console.error('Error removing from cart:', error);
        showToast('Failed to remove item', 'error');
    }
}

async function updateCartQuantity(cartItemId, quantity) {
    if (quantity <= 0) {
        removeFromCart(cartItemId);
        return;
    }

    try {
        const response = await fetch(`/api/cart/item/${cartItemId}`, {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                quantity: parseInt(quantity)
            })
        });

        if (!response.ok) {
            throw new Error('Failed to update cart item');
        }

        showToast('Cart updated', 'success');

        // Refresh cart if on cart page
        if (window.location.pathname === '/cart' && window.refreshCart) {
            window.refreshCart();
        }

        // Refresh modal if it's open
        const cartModal = document.getElementById('modal-cart-items');
        if (cartModal && cartModal.innerHTML.trim() !== '') {
            setTimeout(async () => {
                const modalResponse = await fetch('/api/cart');
                if (modalResponse.ok) {
                    const cart = await modalResponse.json();
                    renderCartModal(cart);
                }
            }, 100);
        }

        await updateCartCount();
        
    } catch (error) {
        console.error('Error updating cart:', error);
        showToast('Failed to update cart', 'error');
    }
}

async function updateCartCount() {
    try {
        const response = await fetch('/api/cart');
        if (response.ok) {
            const cart = await response.json();
            const totalItems = cart.items ? cart.items.reduce((sum, item) => sum + item.quantity, 0) : 0;
            
            // Update cart count badge if it exists
            const cartCountElement = document.querySelector('#cart-count');
            if (cartCountElement) {
                cartCountElement.textContent = totalItems;
                if (totalItems > 0) {
                    cartCountElement.style.display = 'flex';
                    cartCountElement.classList.add('show');
                } else {
                    cartCountElement.style.display = 'none';
                    cartCountElement.classList.remove('show');
                }
            }
        }
    } catch (error) {
        console.error('Error updating cart count:', error);
    }
}

async function proceedToCheckout() {
    try {
        // Check if shipping is selected
        if (!window.shippingManager || !window.shippingManager.selectedShippingOption) {
            showToast('Please select a shipping method before checkout', 'error');
            // Scroll to shipping section
            const shippingSection = document.getElementById('cart-shipping');
            if (shippingSection) {
                shippingSection.scrollIntoView({ behavior: 'smooth', block: 'center' });
            }
            return;
        }

        const response = await fetch('/checkout/create-session-cart', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            }
        });

        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || 'Failed to create checkout session');
        }

        const data = await response.json();

        // Redirect to Stripe checkout
        if (data.url) {
            window.location.href = data.url;
        } else {
            throw new Error('No checkout URL received');
        }

    } catch (error) {
        console.error('Error creating checkout session:', error);
        showToast(error.message, 'error');
    }
}

// Toast notification system
function showToast(message, type = 'info') {
    // Remove existing toast if any
    const existingToast = document.getElementById('toast');
    if (existingToast) {
        existingToast.remove();
    }

    // Create toast element
    const toast = document.createElement('div');
    toast.id = 'toast';
    toast.className = `fixed top-24 right-4 z-50 p-4 rounded-lg shadow-lg transform transition-transform duration-300 translate-x-full ${
        type === 'success' ? 'bg-green-600 text-white' :
        type === 'error' ? 'bg-red-600 text-white' :
        'bg-blue-600 text-white'
    }`;
    toast.textContent = message;

    document.body.appendChild(toast);

    // Animate in
    setTimeout(() => {
        toast.classList.remove('translate-x-full');
        toast.classList.add('translate-x-0');
    }, 100);

    // Auto remove after 3 seconds
    setTimeout(() => {
        toast.classList.remove('translate-x-0');
        toast.classList.add('translate-x-full');
        setTimeout(() => {
            toast.remove();
        }, 300);
    }, 3000);
}

// Initialize event listeners when DOM is loaded
document.addEventListener('DOMContentLoaded', async function() {
    // Wait for Clerk authentication to be ready before checking cart
    // This prevents race condition where cart API is called before auth token exists
    if (window.Clerk) {
        try {
            await window.Clerk.load();
        } catch (error) {
            console.debug('Clerk load waited, proceeding with cart update');
        }
    }

    // Update cart count on page load (now with auth ready)
    updateCartCount();

    // Initialize interactive cart button
    initializeCartHoverEffects();
    
    // Buy Now buttons - direct to Stripe checkout
    document.addEventListener('click', function(e) {
        if (e.target.classList.contains('buy-now-btn')) {
            e.preventDefault();
            const productId = e.target.dataset.productId;
            const productName = e.target.dataset.productName;
            const productPrice = e.target.dataset.productPrice;
            const productSkuId = e.target.dataset.productSkuId || '';

            // Check for quantity from dropdown first, then fallback to data attribute
            let quantity = e.target.dataset.quantity || '1';
            const quantitySelect = document.getElementById('product-quantity');
            if (quantitySelect) {
                quantity = quantitySelect.value || '1';
            }

            if (productId && productName && productPrice) {
                buyNow(productId, productName, productPrice, parseInt(quantity), productSkuId);
            }
        }
        
        // Add to Cart buttons - now functional
        if (e.target.classList.contains('add-to-cart-btn')) {
            e.preventDefault();
            const productId = e.target.dataset.productId;
            const productName = e.target.dataset.productName || '';
            const productSkuId = e.target.dataset.productSkuId || '';

            // Check for quantity from dropdown first, then fallback to data attribute
            let quantity = e.target.dataset.quantity || '1';
            const quantitySelect = document.getElementById('product-quantity');
            if (quantitySelect) {
                quantity = quantitySelect.value || '1';
            }

            if (productId) {
                addToCart(productId, parseInt(quantity), productName, productSkuId);
            }
        }
        
        // Cart item remove buttons - improved SVG handling
        let removeButton = null;
        
        // First try standard approach
        if (e.target.closest) {
            removeButton = e.target.closest('.cart-remove-btn');
        }
        
        // Fallback for SVG elements that might not work with closest()
        if (!removeButton) {
            let current = e.target;
            while (current && current !== document) {
                if (current.classList && current.classList.contains('cart-remove-btn')) {
                    removeButton = current;
                    break;
                }
                current = current.parentElement || current.parentNode;
            }
        }
        
        if (removeButton) {
            e.preventDefault();
            const cartItemId = removeButton.dataset.cartItemId;
            if (cartItemId) {
                console.log('Calling removeFromCart with ID:', cartItemId);
                removeFromCart(cartItemId);
            }
        }
        
        // Cart quantity update buttons
        const updateButton = e.target.closest('.cart-update-btn');
        if (updateButton) {
            e.preventDefault();
            const cartItemId = updateButton.dataset.cartItemId;
            const quantity = updateButton.dataset.quantity;
            
            if (cartItemId && quantity) {
                updateCartQuantity(cartItemId, parseInt(quantity));
            }
        }
        
        // Proceed to checkout button
        if (e.target.classList.contains('proceed-checkout-btn')) {
            e.preventDefault();
            proceedToCheckout();
        }

        // DISABLED: Cart preview modal button - now links directly to /cart page
        // if (e.target.classList.contains('cart-preview-btn') || e.target.closest('.cart-preview-btn')) {
        //     e.preventDefault();
        //     openCartModal();
        // }

        // DISABLED: Modal checkout button - modal no longer used
        // if (e.target.classList.contains('modal-checkout-btn')) {
        //     e.preventDefault();
        //     proceedToCheckout();
        // }
    });
});

// DISABLED: Cart modal functions - Cart now uses dedicated /cart page instead of popup modal
// Keeping these commented out for reference in case they're needed in the future

// async function openCartModal() {
//     try {
//         // Fetch current cart data
//         const response = await fetch('/api/cart');
//         if (response.ok) {
//             const cart = await response.json();
//             renderCartModal(cart);
//
//             // Dispatch event to open modal
//             window.dispatchEvent(new CustomEvent('cart-modal-open'));
//         }
//     } catch (error) {
//         console.error('Error opening cart modal:', error);
//     }
// }

// function renderCartModal(cart) {
//     const items = cart.items || [];
//     const modalItems = document.getElementById('modal-cart-items');
//     const modalEmpty = document.getElementById('modal-empty-cart');
//     const modalFooter = document.getElementById('modal-cart-footer');
//     const modalShippingSection = document.getElementById('modal-shipping-section');
//     const modalTotal = document.getElementById('modal-cart-total');
//     const modalSubtotal = document.getElementById('modal-cart-subtotal');

//     if (items.length === 0) {
//         modalItems.innerHTML = '';
//         modalEmpty.classList.remove('hidden');
//         modalFooter.classList.add('hidden');
//         modalShippingSection.classList.add('hidden');
//         return;
//     }

//     modalEmpty.classList.add('hidden');
//     modalFooter.classList.remove('hidden');
//     modalShippingSection.classList.remove('hidden');
//
//     // Render cart items
//     modalItems.innerHTML = items.map(item => {
//         // Fix image URL - check if it already has the full path
//         let imageSrc = '';
//         if (item.image_url) {
//             if (item.image_url.startsWith('/public/')) {
//                 imageSrc = item.image_url;
//             } else if (item.image_url.startsWith('/images/')) {
//                 imageSrc = '/public' + item.image_url;
//             } else {
//                 imageSrc = '/public/images/products/' + item.image_url;
//             }
//         }

//         const imageHtml = imageSrc ?
//             `<img src="${imageSrc}" alt="${item.name}" class="w-16 h-16 rounded-lg object-cover bg-slate-700/50">` :
//             `<div class="w-16 h-16 rounded-lg bg-slate-700/50 flex items-center justify-center">
//                 <svg class="w-8 h-8 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
//                     <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"></path>
//                 </svg>
//             </div>`;
//
//         return `<div class="flex items-center space-x-4 p-4 bg-slate-700/50 rounded-xl mb-3">
//             ${imageHtml}
//             <div class="flex-1">
//                 <h4 class="font-semibold text-white">${item.name}</h4>
//                 <p class="text-emerald-400">$${(item.price_cents / 100).toFixed(2)}</p>
//             </div>
//             <div class="flex items-center space-x-2">
//                 <div class="flex items-center space-x-2">
//                     <span class="text-slate-300 text-sm">Qty:</span>
//                     <button class="cart-update-btn w-6 h-6 rounded-full bg-slate-600/50 hover:bg-slate-500/50 text-white flex items-center justify-center text-sm transition-colors duration-200" data-cart-item-id="${item.id}" data-quantity="${item.quantity - 1}">−</button>
//                     <span class="text-white font-semibold min-w-[1.5rem] text-center text-sm">${item.quantity}</span>
//                     <button class="cart-update-btn w-6 h-6 rounded-full bg-slate-600/50 hover:bg-slate-500/50 text-white flex items-center justify-center text-sm transition-colors duration-200" data-cart-item-id="${item.id}" data-quantity="${item.quantity + 1}">+</button>
//                 </div>
//                 <button class="cart-remove-btn text-red-400 hover:text-red-300 p-1" data-cart-item-id="${item.id}">
//                     <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
//                         <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path>
//                     </svg>
//                 </button>
//             </div>
//         </div>`;
//     }).join('');
//
//     // Update subtotal and initialize shipping
//     const subtotal = cart.totalCents || 0;
//     modalSubtotal.textContent = '$' + (subtotal / 100).toFixed(2);
//     modalTotal.textContent = '$' + (subtotal / 100).toFixed(2); // Initial total = subtotal

//     // Initialize shipping options
//     if (window.shippingManager) {
//         window.shippingManager.updateShippingUI('address-required');
//     }
//
//     // Add event listeners to modal remove buttons after rendering
//     const modalRemoveButtons = document.querySelectorAll('#modal-cart-items .cart-remove-btn');
//     modalRemoveButtons.forEach(button => {
//         // Remove any existing listeners to avoid duplicates
//         button.removeEventListener('click', handleModalRemove);
//         button.addEventListener('click', handleModalRemove);
//     });

//     // Add event listeners to modal quantity update buttons
//     const modalUpdateButtons = document.querySelectorAll('#modal-cart-items .cart-update-btn');
//     modalUpdateButtons.forEach(button => {
//         // Remove any existing listeners to avoid duplicates
//         button.removeEventListener('click', handleModalQuantityUpdate);
//         button.addEventListener('click', handleModalQuantityUpdate);
//     });

//     // Add event listener to modal checkout button
//     const modalCheckoutButton = document.querySelector('.modal-checkout-btn');
//     if (modalCheckoutButton) {
//         modalCheckoutButton.removeEventListener('click', handleModalCheckout);
//         modalCheckoutButton.addEventListener('click', handleModalCheckout);
//     }
// }

// // Handler specifically for modal remove buttons
// function handleModalRemove(e) {
//     e.preventDefault();
//     e.stopPropagation();
//
//     const button = e.currentTarget;
//     const cartItemId = button.dataset.cartItemId;
//
//     console.log('Modal remove button clicked, ID:', cartItemId);
//
//     if (cartItemId) {
//         removeFromCart(cartItemId);
//     }
// }

// // Handler specifically for modal quantity update buttons
// function handleModalQuantityUpdate(e) {
//     e.preventDefault();
//     e.stopPropagation();

//     const button = e.currentTarget;
//     const cartItemId = button.dataset.cartItemId;
//     const quantity = parseInt(button.dataset.quantity);

//     console.log('Modal quantity update button clicked, ID:', cartItemId, 'New quantity:', quantity);

//     if (cartItemId && !isNaN(quantity)) {
//         updateCartQuantity(cartItemId, quantity);
//     }
// }

// // Handler specifically for modal checkout button
// function handleModalCheckout(e) {
//     e.preventDefault();
//     e.stopPropagation();

//     console.log('Modal checkout button clicked');
//     proceedToCheckout();
// }

// Interactive Cart Button Effects
function initializeCartHoverEffects() {
    const cartButton = document.querySelector('.chatgpt-cart-btn, .modern-cart-btn, .custom-cart-btn');
    const cartContainer = document.querySelector('.cart-logo-container');
    
    if (!cartButton || !cartContainer) return;
    
    // Mouse tracking for 3D tilt effect
    cartButton.addEventListener('mousemove', function(e) {
        const rect = cartButton.getBoundingClientRect();
        const centerX = rect.left + rect.width / 2;
        const centerY = rect.top + rect.height / 2;
        
        const mouseX = e.clientX - centerX;
        const mouseY = e.clientY - centerY;
        
        // Calculate rotation angles (limited range for subtle effect)
        const rotateX = (mouseY / rect.height) * -5; // Reduced for cleaner effect
        const rotateY = (mouseX / rect.width) * 5;   // Reduced for cleaner effect
        
        // Apply transform with CSS custom properties
        cartContainer.style.setProperty('--tilt-x', `${rotateY}deg`);
        cartContainer.style.setProperty('--tilt-y', `${rotateX}deg`);
        cartContainer.classList.add('cart-tilt-both');
    });
    
    // Reset on mouse leave
    cartButton.addEventListener('mouseleave', function() {
        cartContainer.style.removeProperty('--tilt-x');
        cartContainer.style.removeProperty('--tilt-y');
        cartContainer.classList.remove('cart-tilt-both');
    });
    
    // Pulse effect when items are added to cart
    const originalAddToCart = window.addToCart;
    if (originalAddToCart) {
        window.addToCart = async function(...args) {
            const result = await originalAddToCart.apply(this, args);
            
            // Add pulse animation class
            cartButton.classList.add('cart-added');
            
            // Remove class after animation
            setTimeout(() => {
                cartButton.classList.remove('cart-added');
            }, 500);
            
            return result;
        };
    }
}

// Enhanced cart count update with animation
function updateCartCountWithAnimation(newCount) {
    const cartCountElement = document.querySelector('#cart-count');
    const cartButton = document.querySelector('.chatgpt-cart-btn, .modern-cart-btn, .custom-cart-btn');

    if (cartCountElement && cartButton) {
        const oldCount = parseInt(cartCountElement.textContent) || 0;

        if (newCount > oldCount) {
            // Trigger pulse animation for new items
            cartButton.classList.add('cart-added');
            setTimeout(() => {
                cartButton.classList.remove('cart-added');
            }, 500);
        }

        cartCountElement.textContent = newCount;
        if (newCount > 0) {
            cartCountElement.style.display = 'flex';
            cartCountElement.classList.add('show');
        } else {
            cartCountElement.style.display = 'none';
            cartCountElement.classList.remove('show');
        }
    }
}

// Fallback: Also update cart count when Clerk finishes loading
// This handles the case where Clerk loads after DOMContentLoaded
window.addEventListener('clerk:loaded', () => {
    console.debug('Clerk loaded event fired, updating cart count');
    updateCartCount();
});

// Additional fallback: Update cart count when Clerk session becomes available
if (window.Clerk) {
    window.Clerk.addListener((event) => {
        if (event.session) {
            console.debug('Clerk session available, updating cart count');
            updateCartCount();
        }
    });
}

// Global event listener for cart updates from any source
// This allows components (like "Buy Again" buttons) to trigger cart count refresh
window.addEventListener('cart-updated', () => {
    console.debug('cart-updated event received, refreshing cart count');
    updateCartCount();
});
