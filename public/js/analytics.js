// analytics.js - GA4 E-commerce & Lead Tracking Utilities for Logan's 3D Creations
// GA4 Measurement ID: G-0DMM8W9JY7

const Analytics = {
    // ========================================
    // E-commerce Events
    // ========================================

    /**
     * Track when a user views a product detail page
     * @param {Object} product - Product data
     * @param {string} product.id - Product ID
     * @param {string} product.name - Product name
     * @param {string} product.category - Product category
     * @param {number} product.price - Product price in dollars
     */
    viewItem: function(product) {
        if (typeof gtag === 'undefined') return;

        gtag('event', 'view_item', {
            currency: 'USD',
            value: product.price,
            items: [{
                item_id: product.id,
                item_name: product.name,
                item_category: product.category,
                price: product.price,
                quantity: 1
            }]
        });
    },

    /**
     * Track when a user adds a product to cart
     * @param {Object} product - Product data
     * @param {string} product.id - Product ID
     * @param {string} product.name - Product name
     * @param {string} product.category - Product category (optional)
     * @param {number} product.price - Product price in dollars
     * @param {number} quantity - Quantity added
     */
    addToCart: function(product, quantity = 1) {
        if (typeof gtag === 'undefined') return;

        gtag('event', 'add_to_cart', {
            currency: 'USD',
            value: product.price * quantity,
            items: [{
                item_id: product.id,
                item_name: product.name,
                item_category: product.category || '',
                price: product.price,
                quantity: quantity
            }]
        });
    },

    /**
     * Track when a user removes a product from cart
     * @param {Object} product - Product data
     * @param {string} product.id - Product ID
     * @param {string} product.name - Product name
     * @param {string} product.category - Product category (optional)
     * @param {number} product.price - Product price in dollars
     * @param {number} quantity - Quantity removed
     */
    removeFromCart: function(product, quantity = 1) {
        if (typeof gtag === 'undefined') return;

        gtag('event', 'remove_from_cart', {
            currency: 'USD',
            value: product.price * quantity,
            items: [{
                item_id: product.id,
                item_name: product.name,
                item_category: product.category || '',
                price: product.price,
                quantity: quantity
            }]
        });
    },

    /**
     * Track when a user views their cart
     * @param {Object} cart - Cart data
     * @param {number} cart.total - Cart total in dollars
     * @param {Array} cart.items - Array of cart items
     */
    viewCart: function(cart) {
        if (typeof gtag === 'undefined') return;

        gtag('event', 'view_cart', {
            currency: 'USD',
            value: cart.total,
            items: cart.items.map(item => ({
                item_id: item.id,
                item_name: item.name,
                item_category: item.category || '',
                price: item.price,
                quantity: item.quantity
            }))
        });
    },

    /**
     * Track when a user begins checkout
     * @param {Object} cart - Cart data
     * @param {number} cart.total - Cart total in dollars
     * @param {Array} cart.items - Array of cart items
     */
    beginCheckout: function(cart) {
        if (typeof gtag === 'undefined') return;

        gtag('event', 'begin_checkout', {
            currency: 'USD',
            value: cart.total,
            items: cart.items.map(item => ({
                item_id: item.id,
                item_name: item.name,
                item_category: item.category || '',
                price: item.price,
                quantity: item.quantity
            }))
        });
    },

    /**
     * Track when a user adds shipping info
     * @param {Object} cart - Cart data
     * @param {string} shippingTier - Shipping tier name
     */
    addShippingInfo: function(cart, shippingTier) {
        if (typeof gtag === 'undefined') return;

        gtag('event', 'add_shipping_info', {
            currency: 'USD',
            value: cart.total,
            shipping_tier: shippingTier,
            items: cart.items.map(item => ({
                item_id: item.id,
                item_name: item.name,
                item_category: item.category || '',
                price: item.price,
                quantity: item.quantity
            }))
        });
    },

    /**
     * Track when a user adds payment info
     * @param {Object} cart - Cart data
     * @param {string} paymentType - Payment type (e.g., 'Credit Card', 'PayPal')
     */
    addPaymentInfo: function(cart, paymentType) {
        if (typeof gtag === 'undefined') return;

        gtag('event', 'add_payment_info', {
            currency: 'USD',
            value: cart.total,
            payment_type: paymentType,
            items: cart.items.map(item => ({
                item_id: item.id,
                item_name: item.name,
                item_category: item.category || '',
                price: item.price,
                quantity: item.quantity
            }))
        });
    },

    /**
     * Track a completed purchase (CRITICAL - only fire once per transaction)
     * @param {Object} order - Order data
     * @param {string} order.id - Unique order/transaction ID
     * @param {number} order.total - Order total in dollars
     * @param {number} order.tax - Tax amount in dollars
     * @param {number} order.shipping - Shipping amount in dollars
     * @param {Array} order.items - Array of order items
     */
    purchase: function(order) {
        if (typeof gtag === 'undefined') return;

        // Prevent duplicate purchase tracking
        const purchaseKey = 'ga4_purchase_' + order.id;
        if (sessionStorage.getItem(purchaseKey)) {
            console.log('GA4: Purchase already tracked for order', order.id);
            return;
        }

        gtag('event', 'purchase', {
            transaction_id: order.id,
            value: order.total,
            currency: 'USD',
            tax: order.tax || 0,
            shipping: order.shipping || 0,
            items: order.items.map(item => ({
                item_id: item.id,
                item_name: item.name,
                item_category: item.category || '',
                price: item.price,
                quantity: item.quantity
            }))
        });

        // Mark as tracked in session storage
        sessionStorage.setItem(purchaseKey, 'true');
        console.log('GA4: Purchase tracked for order', order.id);
    },

    // ========================================
    // Lead Generation Events
    // ========================================

    /**
     * Track a lead generation event
     * @param {string} source - Lead source (e.g., 'contact_form', 'custom_order')
     * @param {number} value - Estimated value of the lead in dollars
     * @param {Object} extraData - Additional data to include
     */
    generateLead: function(source, value = 50, extraData = {}) {
        if (typeof gtag === 'undefined') return;

        gtag('event', 'generate_lead', {
            currency: 'USD',
            value: value,
            lead_source: source,
            ...extraData
        });
    },

    /**
     * Track when a user starts interacting with a form
     * @param {string} formName - Name of the form
     */
    formStart: function(formName) {
        if (typeof gtag === 'undefined') return;

        gtag('event', 'form_start', {
            form_name: formName
        });
    },

    /**
     * Track when a user submits a form
     * @param {string} formName - Name of the form
     * @param {boolean} success - Whether submission was successful
     */
    formSubmit: function(formName, success = true) {
        if (typeof gtag === 'undefined') return;

        gtag('event', 'form_submit', {
            form_name: formName,
            success: success
        });
    },

    /**
     * Track when a lead is qualified by admin
     * @param {string} leadId - Lead/contact ID
     * @param {string} source - Lead source (e.g., 'contact_form', 'custom_order')
     * @param {number} value - Estimated value of the qualified lead in dollars
     * @param {Object} extraData - Additional data (status, priority, etc.)
     */
    qualifyLead: function(leadId, source, value = 100, extraData = {}) {
        if (typeof gtag === 'undefined') return;

        gtag('event', 'qualify_lead', {
            currency: 'USD',
            value: value,
            lead_id: leadId,
            lead_source: source,
            ...extraData
        });
    },

    /**
     * Track when a lead converts to a customer
     * @param {string} leadId - Lead/contact ID
     * @param {string} orderId - Order ID that represents the conversion
     * @param {number} value - Order value in dollars
     * @param {string} source - Original lead source
     */
    closeConvertLead: function(leadId, orderId, value, source = 'contact_form') {
        if (typeof gtag === 'undefined') return;

        gtag('event', 'close_convert_lead', {
            currency: 'USD',
            value: value,
            lead_id: leadId,
            transaction_id: orderId,
            lead_source: source
        });
    },

    // ========================================
    // Custom Order Funnel Events
    // ========================================

    /**
     * Track custom order form step progression
     * @param {number} stepNumber - Step number (1-5)
     * @param {Object} stepData - Additional step data
     */
    customOrderStep: function(stepNumber, stepData = {}) {
        if (typeof gtag === 'undefined') return;

        const stepNames = {
            1: 'project_type',
            2: 'contact_info',
            3: 'model_details',
            4: 'customization',
            5: 'review_submit'
        };

        gtag('event', 'custom_order_step_' + stepNumber, {
            step_number: stepNumber,
            step_name: stepNames[stepNumber] || 'unknown',
            ...stepData
        });
    },

    /**
     * Track when a user clicks "Need a Custom Version?" on a product page
     * @param {string} productName - Source product name
     * @param {string} productCategory - Source product category
     */
    customVersionInterest: function(productName, productCategory) {
        if (typeof gtag === 'undefined') return;

        gtag('event', 'custom_version_interest', {
            source_product: productName,
            source_category: productCategory
        });
    }
};

// Make globally available
window.Analytics = Analytics;
