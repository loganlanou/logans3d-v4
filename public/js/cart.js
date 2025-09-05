// Cart functionality
async function addToCart(productId, quantity = 1) {
    try {
        const response = await fetch('/cart/add', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                product_id: productId,
                quantity: quantity
            })
        });

        if (!response.ok) {
            throw new Error('Failed to add item to cart');
        }

        const data = await response.json();
        
        // Show success message
        showToast(data.message || 'Item added to cart!', 'success');
        
        // Update cart count in UI if element exists
        const cartCountElement = document.getElementById('cart-count');
        if (cartCountElement && data.cart_count !== undefined) {
            cartCountElement.textContent = data.cart_count;
            cartCountElement.classList.remove('hidden');
        }

    } catch (error) {
        console.error('Error adding to cart:', error);
        showToast('Failed to add item to cart', 'error');
    }
}

async function updateQuantity(itemId, quantity) {
    if (quantity < 1) return;

    try {
        const response = await fetch('/cart/update', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                item_id: itemId,
                quantity: quantity
            })
        });

        if (!response.ok) {
            throw new Error('Failed to update cart');
        }

        // Reload the page to update the cart
        window.location.reload();

    } catch (error) {
        console.error('Error updating cart:', error);
        showToast('Failed to update cart', 'error');
    }
}

async function removeFromCart(itemId) {
    try {
        const response = await fetch('/cart/remove', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                item_id: itemId
            })
        });

        if (!response.ok) {
            throw new Error('Failed to remove item');
        }

        // Reload the page to update the cart
        window.location.reload();

    } catch (error) {
        console.error('Error removing from cart:', error);
        showToast('Failed to remove item', 'error');
    }
}

function proceedToCheckout() {
    window.location.href = '/checkout';
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
    toast.className = `fixed top-4 right-4 z-50 p-4 rounded-lg shadow-lg transform transition-transform duration-300 translate-x-full ${
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
document.addEventListener('DOMContentLoaded', function() {
    // Add to cart buttons
    document.addEventListener('click', function(e) {
        if (e.target.classList.contains('add-to-cart-btn')) {
            e.preventDefault();
            const productId = e.target.dataset.productId;
            if (productId) {
                addToCart(productId);
            }
        }
        
        // Cart update quantity buttons
        if (e.target.classList.contains('cart-update-btn')) {
            e.preventDefault();
            const itemId = e.target.dataset.itemId;
            const quantity = parseInt(e.target.dataset.quantity);
            if (itemId && quantity > 0) {
                updateQuantity(itemId, quantity);
            }
        }
        
        // Cart remove buttons
        if (e.target.classList.contains('cart-remove-btn')) {
            e.preventDefault();
            const itemId = e.target.dataset.itemId;
            if (itemId) {
                removeFromCart(itemId);
            }
        }
    });
});