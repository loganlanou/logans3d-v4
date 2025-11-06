// Shipping functionality for rate display and selection
class ShippingManager {
    constructor() {
        this.selectedShippingOption = null;
        this.shippingRates = [];
        this.shippingAddress = {};
        this.isLoadingRates = false;
    }

    // Get shipping rates from backend
    async getShippingRates(shippingAddress) {
        this.isLoadingRates = true;
        this.updateShippingUI('loading');

        try {
            const response = await fetch('/api/shipping/rates', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    ship_to: {
                        name: shippingAddress.name || '',
                        address_line1: shippingAddress.address_line1 || '',
                        city_locality: shippingAddress.city_locality || '',
                        state_province: shippingAddress.state_province || '',
                        postal_code: shippingAddress.postal_code || '',
                        country_code: shippingAddress.country_code || 'US'
                    }
                })
            });

            if (!response.ok) {
                const errorData = await response.json();
                throw new Error(errorData.error || 'Failed to get shipping rates');
            }

            const data = await response.json();
            this.shippingRates = data.options || [];
            this.shippingAddress = shippingAddress;

            this.updateShippingUI('rates');
            return this.shippingRates;

        } catch (error) {
            console.error('Error getting shipping rates:', error);
            this.updateShippingUI('error', error.message);
            throw error;
        } finally {
            this.isLoadingRates = false;
        }
    }

    // Select a shipping option
    selectShippingOption(optionId) {
        const option = this.shippingRates.find(rate => rate.rate_id === optionId);
        if (option) {
            this.selectedShippingOption = option;
            this.updateSelectedShippingUI();
            this.saveShippingSelection(option);
        }
    }

    // Save shipping selection to backend
    async saveShippingSelection(option) {
        try {
            console.log('Saving shipping selection:', option);
            const response = await fetch('/api/shipping/selection', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    rate_id: option.rate_id,
                    shipment_id: option.shipment_id,
                    carrier_name: option.carrier_name,
                    service_name: option.service_name,
                    price_cents: Math.round(option.total_cost * 100),
                    shipping_amount_cents: Math.round(option.price * 100),
                    box_cost_cents: Math.round(option.box_cost * 100),
                    handling_cost_cents: Math.round(option.handling_cost * 100),
                    box_sku: option.box_sku || 'UNKNOWN',
                    delivery_days: option.delivery_days || 0,
                    estimated_date: option.estimated_date || '',
                    shipping_address: this.shippingAddress
                })
            });

            console.log('Shipping selection response:', response.status, response.statusText);

            if (!response.ok) {
                const errorText = await response.text();
                console.error('Shipping selection failed:', response.status, errorText);
                throw new Error(`Failed to save shipping selection: ${response.status} ${errorText}`);
            }

            const result = await response.json();
            console.log('Shipping selection saved successfully:', result);

            // Enable checkout button after successful save
            this.enableCheckoutButton();

            return result;
        } catch (error) {
            console.error('Error saving shipping selection:', error);
            throw error;
        }
    }

    // Update shipping UI based on state
    updateShippingUI(state, message = '') {
        const shippingContainer = document.getElementById('shipping-options-container');
        if (!shippingContainer) return;

        switch (state) {
            case 'loading':
                shippingContainer.innerHTML = this.getLoadingHTML();
                break;
            case 'rates':
                shippingContainer.innerHTML = this.getShippingRatesHTML();
                this.attachShippingEventListeners();
                // Auto-select first rate after rendering
                if (this.shippingRates && this.shippingRates.length > 0) {
                    setTimeout(() => {
                        console.log('Auto-selecting first shipping option:', this.shippingRates[0].rate_id);
                        this.selectShippingOption(this.shippingRates[0].rate_id);
                    }, 100);
                }
                break;
            case 'error':
                shippingContainer.innerHTML = this.getErrorHTML(message);
                break;
            case 'address-required':
                shippingContainer.innerHTML = this.getAddressRequiredHTML();
                break;
        }
    }

    // Update selected shipping option UI
    updateSelectedShippingUI() {
        // Update cart total with shipping
        this.updateCartTotalWithShipping();

        // Update shipping option selection styling
        const radioButtons = document.querySelectorAll('input[name="shipping-option"]');
        radioButtons.forEach(radio => {
            const container = radio.closest('.shipping-option');
            if (container) {
                if (radio.value === this.selectedShippingOption.rate_id) {
                    container.classList.add('selected');
                    radio.checked = true;
                } else {
                    container.classList.remove('selected');
                    radio.checked = false;
                }
            }
        });
    }

    // Update cart total to include shipping cost
    updateCartTotalWithShipping() {
        const cartTotalElement = document.getElementById('cart-total') || document.getElementById('modal-cart-total');
        const subtotalElement = document.getElementById('cart-subtotal') || document.getElementById('modal-cart-subtotal');
        const shippingCostElement = document.getElementById('shipping-cost') || document.getElementById('modal-shipping-cost');

        if (!cartTotalElement) return;

        // Get current cart subtotal (without shipping)
        fetch('/api/cart')
            .then(response => response.json())
            .then(cart => {
                const subtotal = cart.totalCents || 0;

                // Handle both fresh rates (total_cost) and saved selections (price_cents)
                let shippingCost = 0;
                if (this.selectedShippingOption) {
                    if (this.selectedShippingOption.price_cents !== undefined) {
                        // Saved selection from database - already in cents
                        shippingCost = this.selectedShippingOption.price_cents;
                    } else if (this.selectedShippingOption.total_cost !== undefined) {
                        // Fresh rate - need to convert from dollars to cents
                        shippingCost = Math.round(parseFloat(this.selectedShippingOption.total_cost) * 100);
                    }
                }

                const total = subtotal + shippingCost;

                console.log('Cart total calculation:', {
                    subtotal,
                    shippingCost,
                    total,
                    selectedOption: this.selectedShippingOption
                });

                // Update display elements
                if (subtotalElement) {
                    subtotalElement.textContent = '$' + (subtotal / 100).toFixed(2);
                }
                if (shippingCostElement) {
                    shippingCostElement.textContent = this.selectedShippingOption ?
                        '$' + (shippingCost / 100).toFixed(2) : 'TBD';
                }
                cartTotalElement.textContent = '$' + (total / 100).toFixed(2);

                // Update checkout button text with new total
                this.updateCheckoutButtonText();
            })
            .catch(error => {
                console.error('Error updating cart total:', error);
            });
    }

    // Generate HTML templates
    getLoadingHTML() {
        return `
            <div class="shipping-loading text-center py-8">
                <div class="flex justify-center items-center mb-4">
                    <div class="animate-spin rounded-full h-12 w-12 border-4 border-slate-600 border-t-blue-500"></div>
                </div>
                <p class="text-lg font-semibold text-white">Getting shipping rates...</p>
                <p class="text-sm text-slate-400 mt-2">Please wait...</p>
            </div>
        `;
    }

    getShippingRatesHTML() {
        if (!this.shippingRates || this.shippingRates.length === 0) {
            return this.getErrorHTML('No shipping options available for this address');
        }

        return `
            <div class="shipping-rates">
                <div class="flex items-center justify-between mb-4">
                    <h3 class="text-lg font-semibold text-white">Select Shipping Method</h3>
                    <button
                        onclick="window.shippingManager.showAddressForm()"
                        class="text-sm text-blue-400 hover:text-blue-300 transition-colors"
                    >
                        Change ZIP Code
                    </button>
                </div>
                <div class="space-y-3 max-h-96 overflow-y-auto pr-2 shipping-rates-scroll">
                    ${this.shippingRates.map(rate => `
                        <div class="shipping-option bg-slate-700/30 rounded-xl p-4 border border-slate-600/50 hover:border-blue-500/50 transition-all duration-200 cursor-pointer" data-rate-id="${rate.rate_id}">
                            <label class="flex items-center justify-between cursor-pointer">
                                <div class="flex items-center space-x-3">
                                    <input type="radio" name="shipping-option" value="${rate.rate_id}" class="text-blue-500 focus:ring-blue-500">
                                    <div>
                                        <div class="font-semibold text-white">
                                            ${rate.carrier_name} ${rate.service_name}
                                        </div>
                                        <div class="text-sm text-slate-300">
                                            ${rate.delivery_days ? `${rate.delivery_days} business days` : 'Standard delivery'}
                                            ${rate.estimated_date ? ` • Arrives by ${new Date(rate.estimated_date).toLocaleDateString()}` : ''}
                                        </div>
                                    </div>
                                </div>
                                <div class="text-lg font-semibold text-emerald-400">
                                    $${rate.total_cost.toFixed(2)}
                                </div>
                            </label>
                        </div>
                    `).join('')}
                </div>
            </div>
        `;
    }

    getErrorHTML(message) {
        return `
            <div class="shipping-error text-center py-8">
                <div class="text-red-400 mb-4">
                    <svg class="w-12 h-12 mx-auto" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z"></path>
                    </svg>
                </div>
                <p class="text-slate-300">${message}</p>
                <button onclick="window.shippingManager.showAddressForm()" class="mt-4 bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors">
                    Enter Different Address
                </button>
            </div>
        `;
    }

    getAddressRequiredHTML() {
        return `
            <div class="shipping-address-required text-center py-8">
                <div class="text-slate-300 mb-4">
                    <svg class="w-12 h-12 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z"></path>
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z"></path>
                    </svg>
                </div>
                <p class="text-slate-300 mb-6">Enter your ZIP code to see shipping options</p>
                <div class="max-w-sm mx-auto">
                    <div class="flex gap-3">
                        <input
                            type="text"
                            id="shipping-zip-input"
                            placeholder="ZIP Code"
                            class="flex-1 bg-slate-700/50 border border-slate-600 rounded-lg px-4 py-3 text-white placeholder-slate-400 focus:border-blue-500 focus:outline-none focus:text-white"
                            maxlength="10"
                            style="color: white !important;"
                        >
                        <button
                            onclick="window.shippingManager.getQuickRates()"
                            class="bg-blue-600 text-white px-6 py-3 rounded-lg hover:bg-blue-700 transition-colors whitespace-nowrap"
                        >
                            Get Rates
                        </button>
                    </div>
                </div>
            </div>
        `;
    }

    // Get quick rates with ZIP code
    async getQuickRates() {
        const zipInput = document.getElementById('shipping-zip-input');
        if (!zipInput) return;

        const zipCode = zipInput.value.trim();
        if (!zipCode) {
            showToast('Please enter a ZIP code', 'error');
            return;
        }

        this.updateShippingUI('loading');

        try {
            const rates = await this.getEstimatedRates(zipCode);
            if (rates.length > 0) {
                this.shippingRates = rates;
                this.shippingAddress = {
                    postal_code: zipCode,
                    country_code: 'US'
                };
                this.updateShippingUI('rates');
            } else {
                this.updateShippingUI('error', 'No shipping options available for this ZIP code');
            }
        } catch (error) {
            this.updateShippingUI('error', 'Failed to get shipping rates');
        }
    }

    // Show address form
    showAddressForm() {
        // Reset shipping state and show the ZIP code input form again
        this.shippingRates = [];
        this.selectedShippingOption = null;
        this.shippingAddress = {};
        this.updateShippingUI('address-required');
    }

    // Attach event listeners to shipping options
    attachShippingEventListeners() {
        const shippingOptions = document.querySelectorAll('.shipping-option');
        shippingOptions.forEach(option => {
            option.addEventListener('click', (e) => {
                const rateId = option.dataset.rateId;
                if (rateId) {
                    this.selectShippingOption(rateId);
                }
            });
        });

        const radioButtons = document.querySelectorAll('input[name="shipping-option"]');
        radioButtons.forEach(radio => {
            radio.addEventListener('change', (e) => {
                if (e.target.checked) {
                    this.selectShippingOption(e.target.value);
                }
            });
        });
    }

    // Get estimated rates without full address (for quick preview)
    async getEstimatedRates(zipCode) {
        try {
            const response = await fetch('/api/shipping/rates', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    ship_to: {
                        name: 'Customer',
                        address_line1: '123 Main St',
                        city_locality: 'Anytown',
                        state_province: 'CA',
                        postal_code: zipCode,
                        country_code: 'US'
                    }
                })
            });

            if (response.ok) {
                const data = await response.json();
                return data.options || [];
            }
        } catch (error) {
            console.error('Error getting estimated rates:', error);
        }
        return [];
    }

    // Load saved shipping selection from database
    async loadSavedShipping() {
        try {
            const response = await fetch('/api/shipping/selection');
            if (!response.ok) {
                // No saved shipping
                this.disableCheckoutButton();
                return null;
            }

            const data = await response.json();

            if (!data.selection) {
                // Pre-fill address if available
                if (data.shipping_address && data.shipping_address.postal_code) {
                    this.shippingAddress = data.shipping_address;
                    // Need to wait for UI to render before pre-filling
                    setTimeout(() => {
                        this.prefillZipCode(data.shipping_address.postal_code);
                    }, 100);
                }
                this.disableCheckoutButton();
                return null;
            }

            if (!data.selection.is_valid) {
                // Cart changed - show message and pre-fill ZIP
                this.showCartChangedMessage();
                if (data.shipping_address && data.shipping_address.postal_code) {
                    this.shippingAddress = data.shipping_address;
                    // Need to wait for UI to render before pre-filling
                    setTimeout(() => {
                        this.prefillZipCode(data.shipping_address.postal_code);
                    }, 100);
                }
                this.disableCheckoutButton();
                return null;
            }

            // Valid shipping selection exists
            this.selectedShippingOption = data.selection;
            this.updateCartTotalWithShipping();
            this.enableCheckoutButton();
            this.showSelectedShippingInfo();

            return data.selection;
        } catch (error) {
            console.error('Error loading saved shipping:', error);
            this.disableCheckoutButton();
            return null;
        }
    }

    prefillZipCode(zipCode) {
        const zipInput = document.getElementById('shipping-zip-input');
        if (zipInput) {
            zipInput.value = zipCode;
        }
    }

    showCartChangedMessage() {
        const container = document.getElementById('shipping-options-container');
        if (container) {
            const messageDiv = document.createElement('div');
            messageDiv.className = 'bg-yellow-500/20 border border-yellow-500/50 rounded-xl p-4 mb-4';
            messageDiv.innerHTML = `
                <div class="flex items-start">
                    <svg class="w-5 h-5 text-yellow-400 mt-0.5 mr-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z"></path>
                    </svg>
                    <div>
                        <p class="text-yellow-300 font-semibold">Cart has changed</p>
                        <p class="text-yellow-200 text-sm mt-1">Please select shipping again to continue.</p>
                    </div>
                </div>
            `;
            container.insertBefore(messageDiv, container.firstChild);
        }
    }

    showSelectedShippingInfo() {
        const container = document.getElementById('shipping-options-container');
        if (container && this.selectedShippingOption) {
            container.innerHTML = `
                <div class="bg-green-500/20 border border-green-500/50 rounded-xl p-6">
                    <div class="flex items-start justify-between">
                        <div class="flex items-start">
                            <svg class="w-6 h-6 text-green-400 mt-1 mr-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
                            </svg>
                            <div>
                                <p class="text-green-300 font-semibold text-lg">Shipping Selected</p>
                                <p class="text-white font-medium mt-2">${this.selectedShippingOption.carrier_name} ${this.selectedShippingOption.service_name}</p>
                                <p class="text-green-200 text-sm mt-1">
                                    ${this.selectedShippingOption.delivery_days} business days •
                                    $${(this.selectedShippingOption.price_cents / 100).toFixed(2)}
                                </p>
                            </div>
                        </div>
                        <button
                            onclick="window.shippingManager.changeShipping()"
                            class="text-blue-400 hover:text-blue-300 text-sm font-medium"
                        >
                            Change
                        </button>
                    </div>
                </div>
            `;

            // Update step indicators when showing saved shipping
            this.updateStepIndicators('shipping-selected');
        }
    }

    changeShipping() {
        this.selectedShippingOption = null;
        this.disableCheckoutButton();
        this.updateShippingUI('address-required');
    }

    enableCheckoutButton() {
        console.log('Enabling checkout button');
        const checkoutBtn = document.getElementById('proceed-checkout-btn');
        const checkoutBtnText = document.getElementById('checkout-btn-text');

        if (checkoutBtn) {
            console.log('Checkout button found, enabling...');
            checkoutBtn.disabled = false;

            // Remove disabled styling
            checkoutBtn.classList.remove('bg-slate-600', 'cursor-not-allowed', 'opacity-50');

            // Add enabled styling
            checkoutBtn.classList.add('bg-gradient-to-r', 'from-blue-600', 'to-emerald-600', 'hover:from-blue-700', 'hover:to-emerald-700', 'shadow-lg', 'hover:shadow-xl', 'hover:shadow-emerald-500/25', 'transform', 'hover:-translate-y-1');

            checkoutBtn.title = '';

            // Update button text with total
            this.updateCheckoutButtonText();

            console.log('Checkout button enabled successfully');
        } else {
            console.error('Checkout button not found!');
        }

        // Update step indicators
        this.updateStepIndicators('shipping-selected');
    }

    disableCheckoutButton() {
        const checkoutBtn = document.getElementById('proceed-checkout-btn');
        const checkoutBtnText = document.getElementById('checkout-btn-text');

        if (checkoutBtn) {
            checkoutBtn.disabled = true;

            // Remove enabled styling
            checkoutBtn.classList.remove('bg-gradient-to-r', 'from-blue-600', 'to-emerald-600', 'hover:from-blue-700', 'hover:to-emerald-700', 'shadow-lg', 'hover:shadow-xl', 'hover:shadow-emerald-500/25', 'transform', 'hover:-translate-y-1');

            // Add disabled styling
            checkoutBtn.classList.add('bg-slate-600', 'cursor-not-allowed', 'opacity-50');

            checkoutBtn.title = 'Please select shipping to continue';

            if (checkoutBtnText) {
                checkoutBtnText.textContent = 'Select Shipping to Continue';
            }
        }

        // Update step indicators
        this.updateStepIndicators('shipping-not-selected');
    }

    updateCheckoutButtonText() {
        const checkoutBtnText = document.getElementById('checkout-btn-text');
        const cartTotalElement = document.getElementById('cart-total');

        if (checkoutBtnText && cartTotalElement) {
            const totalText = cartTotalElement.textContent;
            checkoutBtnText.textContent = `Checkout - ${totalText}`;
        }
    }

    updateStepIndicators(state) {
        const step2Indicator = document.getElementById('step-2-indicator');
        const step2Label = document.getElementById('step-2-label');
        const step3Indicator = document.getElementById('step-3-indicator');
        const step3Label = document.getElementById('step-3-label');

        if (!step2Indicator) return;

        if (state === 'shipping-selected') {
            // Mark step 2 as complete
            step2Indicator.innerHTML = '<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path></svg>';
            step2Indicator.classList.remove('bg-slate-700/50', 'border-slate-600', 'text-slate-400');
            step2Indicator.classList.add('bg-green-500/20', 'border-green-500', 'text-green-400');
            if (step2Label) {
                step2Label.classList.remove('text-slate-400');
                step2Label.classList.add('text-green-400');
            }

            // Activate step 3
            step3Indicator.classList.remove('bg-slate-700/50', 'border-slate-600', 'text-slate-400');
            step3Indicator.classList.add('bg-blue-500/20', 'border-blue-500', 'text-blue-400');
            if (step3Label) {
                step3Label.classList.remove('text-slate-400');
                step3Label.classList.add('text-blue-400');
            }
        } else {
            // Reset step 2 to pending
            step2Indicator.textContent = '2';
            step2Indicator.classList.remove('bg-green-500/20', 'border-green-500', 'text-green-400', 'bg-blue-500/20', 'border-blue-500', 'text-blue-400');
            step2Indicator.classList.add('bg-slate-700/50', 'border-slate-600', 'text-slate-400');
            if (step2Label) {
                step2Label.classList.remove('text-green-400', 'text-blue-400');
                step2Label.classList.add('text-slate-400');
            }

            // Reset step 3 to pending
            step3Indicator.textContent = '3';
            step3Indicator.classList.remove('bg-green-500/20', 'border-green-500', 'text-green-400', 'bg-blue-500/20', 'border-blue-500', 'text-blue-400');
            step3Indicator.classList.add('bg-slate-700/50', 'border-slate-600', 'text-slate-400');
            if (step3Label) {
                step3Label.classList.remove('text-green-400', 'text-blue-400');
                step3Label.classList.add('text-slate-400');
            }
        }
    }
}

// Initialize shipping manager
window.shippingManager = new ShippingManager();

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', function() {
    // Check if we're on a page that needs shipping
    if (document.getElementById('shipping-options-container')) {
        window.shippingManager.updateShippingUI('address-required');
    }

    // Add global event listener for Enter key on ZIP input (using keydown for better compatibility)
    document.addEventListener('keydown', function(e) {
        if (e.target.id === 'shipping-zip-input' && e.key === 'Enter') {
            e.preventDefault();
            e.stopPropagation();
            window.shippingManager.getQuickRates();
        }
    });
});