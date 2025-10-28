package email

// customerOrderContentTemplate is the content section for customer order emails
const customerOrderContentTemplate = `
<div style="text-align: center; margin-bottom: 30px;">
    <h1 style="color: #E85D5D; margin: 0; font-size: 28px;">Order Confirmation</h1>
    <p style="font-size: 18px; color: #666; margin: 10px 0;">Thank you for your order, {{.CustomerName}}!</p>
</div>

<div style="background-color: #f9f9f9; padding: 20px; border-radius: 8px; border-left: 4px solid #E85D5D; margin-bottom: 25px;">
    <p style="margin: 5px 0;"><strong style="color: #555;">Order Number:</strong> #{{.OrderID}}</p>
    <p style="margin: 5px 0;"><strong style="color: #555;">Order Date:</strong> {{.OrderDate}}</p>
    <p style="margin: 5px 0;"><strong style="color: #555;">Customer Email:</strong> {{.CustomerEmail}}</p>
</div>

<h2 style="color: #555; font-size: 20px; margin-top: 30px;">Order Items</h2>
<table style="width: 100%; border-collapse: collapse; margin: 20px 0;">
    <thead>
        <tr style="background-color: #E85D5D;">
            <th style="color: white; padding: 12px; text-align: left; font-weight: 600;">Product</th>
            <th style="color: white; padding: 12px; text-align: center; font-weight: 600;">Quantity</th>
            <th style="color: white; padding: 12px; text-align: right; font-weight: 600;">Price</th>
            <th style="color: white; padding: 12px; text-align: right; font-weight: 600;">Total</th>
        </tr>
    </thead>
    <tbody>
        {{range .Items}}
        <tr style="border-bottom: 1px solid #ddd;">
            <td style="padding: 12px;">
                <div style="display: flex; align-items: center; gap: 12px;">
                    {{if .ProductImage}}
                    <img src="https://www.logans3dcreations.com/public/images/products/{{.ProductImage}}" alt="{{.ProductName}}" style="width: 60px; height: 60px; object-fit: cover; border-radius: 4px; border: 1px solid #ddd;">
                    {{end}}
                    <span>{{.ProductName}}</span>
                </div>
            </td>
            <td style="padding: 12px; text-align: center;">{{.Quantity}}</td>
            <td style="padding: 12px; text-align: right;">{{FormatCents .PriceCents}}</td>
            <td style="padding: 12px; text-align: right;">{{FormatCents .TotalCents}}</td>
        </tr>
        {{end}}
    </tbody>
</table>

<div style="margin-top: 20px; padding-top: 20px; border-top: 2px solid #ddd;">
    <div style="display: flex; justify-content: space-between; padding: 8px 0;">
        <span>Subtotal:</span>
        <span>{{FormatCents .SubtotalCents}}</span>
    </div>
    <div style="display: flex; justify-content: space-between; padding: 8px 0;">
        <span>Tax:</span>
        <span>{{FormatCents .TaxCents}}</span>
    </div>
    <div style="display: flex; justify-content: space-between; padding: 8px 0;">
        <span>Shipping:</span>
        <span>{{FormatCents .ShippingCents}}</span>
    </div>
    <div style="display: flex; justify-content: space-between; padding: 15px 0 0 0; margin-top: 10px; border-top: 2px solid #E85D5D; font-size: 18px; font-weight: bold; color: #E85D5D;">
        <span>Total:</span>
        <span>{{FormatCents .TotalCents}}</span>
    </div>
</div>

<div style="background-color: #f9f9f9; padding: 20px; border-radius: 8px; margin-top: 25px;">
    <h3 style="margin-top: 0; color: #555; font-size: 16px;">Shipping Address</h3>
    <p style="margin: 5px 0;">{{.ShippingAddress.Name}}<br>
    {{.ShippingAddress.Line1}}<br>
    {{if .ShippingAddress.Line2}}{{.ShippingAddress.Line2}}<br>{{end}}
    {{.ShippingAddress.City}}, {{.ShippingAddress.State}} {{.ShippingAddress.PostalCode}}<br>
    {{.ShippingAddress.Country}}</p>
</div>

<div style="text-align: center; margin: 30px 0;">
    <p style="color: #555; margin-bottom: 15px;">Track your order status and view details:</p>
    <a href="https://www.logans3dcreations.com/account/orders/{{.OrderID}}" style="display: inline-block; padding: 14px 35px; background-color: #E85D5D; color: white; text-decoration: none; border-radius: 5px; font-weight: 600; font-size: 16px; margin-bottom: 10px;">View Order Status</a>
    <br>
    <a href="https://www.logans3dcreations.com" style="display: inline-block; padding: 10px 25px; background-color: transparent; color: #E85D5D; text-decoration: none; border: 2px solid #E85D5D; border-radius: 5px; font-weight: 600; margin-top: 10px;">Continue Shopping</a>
</div>

<div style="text-align: center; margin-top: 30px; padding-top: 20px; border-top: 1px solid #ddd; color: #777; font-size: 14px;">
    <p>If you have any questions about your order, please contact us at<br>
    <a href="mailto:prints@logans3dcreations.com" style="color: #E85D5D; text-decoration: none;">prints@logans3dcreations.com</a></p>
    <p style="font-size: 12px; margin-top: 20px; color: #999;">Payment Intent ID: {{.PaymentIntentID}}</p>
</div>
`

// adminOrderContentTemplate is the content section for admin order notification emails
const adminOrderContentTemplate = `
<div style="text-align: center; margin-bottom: 30px;">
    <span style="display: inline-block; background-color: #FF9800; color: white; padding: 5px 15px; border-radius: 20px; font-weight: 600; font-size: 14px; margin-bottom: 10px;">NEW ORDER</span>
    <h1 style="color: #FF9800; margin: 10px 0; font-size: 28px;">Order Received</h1>
    <p style="font-size: 16px; color: #666; margin: 10px 0;">Action required - Process this order</p>
</div>

<div style="background-color: #fff3e0; padding: 20px; border-left: 4px solid #FF9800; margin-bottom: 25px;">
    <p style="margin: 8px 0;"><strong style="color: #e65100; min-width: 150px; display: inline-block;">Order Number:</strong> #{{.OrderID}}</p>
    <p style="margin: 8px 0;"><strong style="color: #e65100; min-width: 150px; display: inline-block;">Order Date:</strong> {{.OrderDate}}</p>
    <p style="margin: 8px 0;"><strong style="color: #e65100; min-width: 150px; display: inline-block;">Payment Intent:</strong> {{.PaymentIntentID}}</p>
    <p style="margin: 8px 0;"><strong style="color: #e65100; min-width: 150px; display: inline-block;">Order Total:</strong> {{FormatCents .TotalCents}}</p>
</div>

<div style="background-color: #e3f2fd; padding: 20px; border-radius: 8px; border-left: 4px solid #2196F3; margin: 20px 0;">
    <h3 style="margin-top: 0; color: #1976D2; font-size: 16px;">Customer Information</h3>
    <p style="margin: 5px 0;"><strong>Name:</strong> {{.CustomerName}}</p>
    <p style="margin: 5px 0;"><strong>Email:</strong> <a href="mailto:{{.CustomerEmail}}" style="color: #2196F3; text-decoration: none;">{{.CustomerEmail}}</a></p>
</div>

<h2 style="color: #555; font-size: 20px; margin-top: 30px;">Order Items</h2>
<table style="width: 100%; border-collapse: collapse; margin: 20px 0;">
    <thead>
        <tr style="background-color: #FF9800;">
            <th style="color: white; padding: 12px; text-align: left; font-weight: 600;">Product</th>
            <th style="color: white; padding: 12px; text-align: center; font-weight: 600;">Quantity</th>
            <th style="color: white; padding: 12px; text-align: right; font-weight: 600;">Unit Price</th>
            <th style="color: white; padding: 12px; text-align: right; font-weight: 600;">Total</th>
        </tr>
    </thead>
    <tbody>
        {{range .Items}}
        <tr style="border-bottom: 1px solid #ddd;">
            <td style="padding: 12px;">
                <div style="display: flex; align-items: center; gap: 12px;">
                    {{if .ProductImage}}
                    <img src="https://www.logans3dcreations.com/public/images/products/{{.ProductImage}}" alt="{{.ProductName}}" style="width: 60px; height: 60px; object-fit: cover; border-radius: 4px; border: 1px solid #ddd;">
                    {{end}}
                    <strong>{{.ProductName}}</strong>
                </div>
            </td>
            <td style="padding: 12px; text-align: center;">{{.Quantity}}</td>
            <td style="padding: 12px; text-align: right;">{{FormatCents .PriceCents}}</td>
            <td style="padding: 12px; text-align: right;">{{FormatCents .TotalCents}}</td>
        </tr>
        {{end}}
    </tbody>
</table>

<div style="background-color: #f9f9f9; padding: 20px; border-radius: 8px; margin-top: 20px;">
    <div style="display: flex; justify-content: space-between; padding: 8px 0; font-size: 16px;">
        <span>Subtotal:</span>
        <span>{{FormatCents .SubtotalCents}}</span>
    </div>
    <div style="display: flex; justify-content: space-between; padding: 8px 0; font-size: 16px;">
        <span>Tax:</span>
        <span>{{FormatCents .TaxCents}}</span>
    </div>
    <div style="display: flex; justify-content: space-between; padding: 8px 0; font-size: 16px;">
        <span>Shipping:</span>
        <span>{{FormatCents .ShippingCents}}</span>
    </div>
    <div style="display: flex; justify-content: space-between; padding: 15px 0 0 0; margin-top: 10px; border-top: 2px solid #FF9800; font-size: 20px; font-weight: bold; color: #FF9800;">
        <span>TOTAL:</span>
        <span>{{FormatCents .TotalCents}}</span>
    </div>
</div>

<h2 style="color: #555; font-size: 20px; margin-top: 30px;">Shipping & Billing Details</h2>
<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 20px; margin: 25px 0;">
    <div style="padding: 15px; background-color: #f9f9f9; border-radius: 8px; border: 1px solid #ddd;">
        <h3 style="margin-top: 0; color: #e65100; font-size: 16px; border-bottom: 2px solid #FF9800; padding-bottom: 8px;">Ship To</h3>
        <p style="margin: 5px 0;"><strong>{{.ShippingAddress.Name}}</strong><br>
        {{.ShippingAddress.Line1}}<br>
        {{if .ShippingAddress.Line2}}{{.ShippingAddress.Line2}}<br>{{end}}
        {{.ShippingAddress.City}}, {{.ShippingAddress.State}} {{.ShippingAddress.PostalCode}}<br>
        {{.ShippingAddress.Country}}</p>
    </div>

    <div style="padding: 15px; background-color: #f9f9f9; border-radius: 8px; border: 1px solid #ddd;">
        <h3 style="margin-top: 0; color: #e65100; font-size: 16px; border-bottom: 2px solid #FF9800; padding-bottom: 8px;">Bill To</h3>
        <p style="margin: 5px 0;"><strong>{{.BillingAddress.Name}}</strong><br>
        {{.BillingAddress.Line1}}<br>
        {{if .BillingAddress.Line2}}{{.BillingAddress.Line2}}<br>{{end}}
        {{.BillingAddress.City}}, {{.BillingAddress.State}} {{.BillingAddress.PostalCode}}<br>
        {{.BillingAddress.Country}}</p>
    </div>
</div>

<div style="text-align: center; margin: 30px 0;">
    <a href="https://www.logans3dcreations.com/admin/orders/{{.OrderID}}" style="display: inline-block; padding: 12px 30px; background-color: #FF9800; color: white; text-decoration: none; border-radius: 5px; font-weight: 600;">View Order</a>
</div>

<div style="margin-top: 30px; padding: 20px; background-color: #f9f9f9; border-radius: 8px;">
    <p style="margin: 0 0 10px 0; font-weight: bold;">Next Steps:</p>
    <ol style="margin: 10px 0; padding-left: 20px;">
        <li>Verify payment was received in Stripe dashboard</li>
        <li>Prepare items for shipment</li>
        <li>Create shipping label through admin panel</li>
        <li>Update order status when shipped</li>
    </ol>
    <p style="margin-top: 20px; font-size: 12px; color: #999;">
        This is an automated notification from Logan's 3D Creations order system.
    </p>
</div>
`

// abandonedCartRecovery1HrTemplate is for the first recovery email (1 hour after abandonment)
const abandonedCartRecovery1HrTemplate = `
<div style="text-align: center; margin-bottom: 30px;">
    <h1 style="color: #E85D5D; margin: 0; font-size: 28px;">You Left Something Behind!</h1>
    <p style="font-size: 18px; color: #666; margin: 10px 0;">We saved your cart for you</p>
</div>

<div style="background-color: #f9f9f9; padding: 20px; border-radius: 8px; border-left: 4px solid #E85D5D; margin-bottom: 25px; text-align: center;">
    <p style="font-size: 16px; margin: 5px 0;">Hi {{.CustomerName}},</p>
    <p style="font-size: 16px; margin: 15px 0;">We noticed you left {{.ItemCount}} item{{if ne .ItemCount 1}}s{{end}} in your cart. Don't worry - we've saved everything for you!</p>
</div>

<h2 style="color: #555; font-size: 20px; margin-top: 30px;">Your Cart</h2>
<table style="width: 100%; border-collapse: collapse; margin: 20px 0;">
    <thead>
        <tr style="background-color: #E85D5D;">
            <th style="color: white; padding: 12px; text-align: left; font-weight: 600;">Product</th>
            <th style="color: white; padding: 12px; text-align: center; font-weight: 600;">Quantity</th>
            <th style="color: white; padding: 12px; text-align: right; font-weight: 600;">Price</th>
        </tr>
    </thead>
    <tbody>
        {{range .Items}}
        <tr style="border-bottom: 1px solid #ddd;">
            <td style="padding: 12px;">
                <div style="display: flex; align-items: center; gap: 12px;">
                    {{if .ProductImage}}
                    <img src="https://www.logans3dcreations.com/public/images/products/{{.ProductImage}}" alt="{{.ProductName}}" style="width: 60px; height: 60px; object-fit: cover; border-radius: 4px; border: 1px solid #ddd;">
                    {{end}}
                    <span>{{.ProductName}}</span>
                </div>
            </td>
            <td style="padding: 12px; text-align: center;">{{.Quantity}}</td>
            <td style="padding: 12px; text-align: right;">{{FormatCents .UnitPrice}}</td>
        </tr>
        {{end}}
    </tbody>
</table>

<div style="margin-top: 20px; padding-top: 20px; border-top: 2px solid #ddd;">
    <div style="display: flex; justify-content: space-between; padding: 15px 0 0 0; font-size: 20px; font-weight: bold; color: #E85D5D;">
        <span>Cart Total:</span>
        <span>{{FormatCents .CartValue}}</span>
    </div>
</div>

<div style="text-align: center; margin: 40px 0;">
    <a href="https://www.logans3dcreations.com/cart?token={{.TrackingToken}}" style="display: inline-block; padding: 16px 40px; background-color: #E85D5D; color: white; text-decoration: none; border-radius: 5px; font-weight: 600; font-size: 18px;">Complete Your Order</a>
</div>

<div style="text-align: center; margin-top: 30px; padding-top: 20px; border-top: 1px solid #ddd; color: #777; font-size: 14px;">
    <p>Have questions? We're here to help!<br>
    <a href="mailto:prints@logans3dcreations.com" style="color: #E85D5D; text-decoration: none;">prints@logans3dcreations.com</a></p>
</div>
`

// abandonedCartRecovery24HrTemplate is for the second recovery email (24 hours after abandonment)
const abandonedCartRecovery24HrTemplate = `
<div style="text-align: center; margin-bottom: 30px;">
    <h1 style="color: #E85D5D; margin: 0; font-size: 28px;">Still Interested?</h1>
    <p style="font-size: 18px; color: #666; margin: 10px 0;">Your cart is waiting for you</p>
</div>

<div style="background-color: #f9f9f9; padding: 20px; border-radius: 8px; border-left: 4px solid #E85D5D; margin-bottom: 25px; text-align: center;">
    <p style="font-size: 16px; margin: 5px 0;">Hi {{.CustomerName}},</p>
    <p style="font-size: 16px; margin: 15px 0;">We're holding these items for you. Complete your order today!</p>
</div>

<h2 style="color: #555; font-size: 20px; margin-top: 30px;">Items in Your Cart</h2>
<table style="width: 100%; border-collapse: collapse; margin: 20px 0;">
    <thead>
        <tr style="background-color: #E85D5D;">
            <th style="color: white; padding: 12px; text-align: left; font-weight: 600;">Product</th>
            <th style="color: white; padding: 12px; text-align: center; font-weight: 600;">Quantity</th>
            <th style="color: white; padding: 12px; text-align: right; font-weight: 600;">Price</th>
        </tr>
    </thead>
    <tbody>
        {{range .Items}}
        <tr style="border-bottom: 1px solid #ddd;">
            <td style="padding: 12px;">
                <div style="display: flex; align-items: center; gap: 12px;">
                    {{if .ProductImage}}
                    <img src="https://www.logans3dcreations.com/public/images/products/{{.ProductImage}}" alt="{{.ProductName}}" style="width: 60px; height: 60px; object-fit: cover; border-radius: 4px; border: 1px solid #ddd;">
                    {{end}}
                    <span>{{.ProductName}}</span>
                </div>
            </td>
            <td style="padding: 12px; text-align: center;">{{.Quantity}}</td>
            <td style="padding: 12px; text-align: right;">{{FormatCents .UnitPrice}}</td>
        </tr>
        {{end}}
    </tbody>
</table>

<div style="margin-top: 20px; padding-top: 20px; border-top: 2px solid #ddd;">
    <div style="display: flex; justify-content: space-between; padding: 15px 0 0 0; font-size: 20px; font-weight: bold; color: #E85D5D;">
        <span>Cart Total:</span>
        <span>{{FormatCents .CartValue}}</span>
    </div>
</div>

{{if .PromoCode}}
<div style="background-color: #E8F5E9; padding: 25px; border-radius: 8px; margin: 30px 0; text-align: center; border: 2px solid #4CAF50;">
    <div style="font-size: 18px; font-weight: 600; color: #2E7D32; margin-bottom: 12px;">🎉 Special Offer Just For You!</div>
    <div style="font-size: 16px; color: #555; margin-bottom: 15px;">Save 5% on your first order</div>
    <div style="background-color: white; padding: 12px 20px; border-radius: 6px; display: inline-block; border: 2px dashed #4CAF50;">
        <div style="font-size: 13px; color: #666; text-transform: uppercase; letter-spacing: 1px; margin-bottom: 4px;">Your Code</div>
        <div style="font-size: 24px; font-weight: 700; color: #2E7D32; letter-spacing: 2px; font-family: 'Courier New', monospace;">{{.PromoCode}}</div>
    </div>
    <div style="font-size: 13px; color: #666; margin-top: 12px;">✓ Auto-applied at checkout | Expires {{.PromoExpires}}</div>
</div>
{{else}}
<div style="background-color: #fff9e6; padding: 20px; border-radius: 8px; margin: 30px 0; text-align: center; border: 2px dashed #FFA000;">
    <p style="font-size: 16px; margin: 0; color: #555;">💡 <strong>Need help deciding?</strong> Contact us with any questions!</p>
</div>
{{end}}

<div style="text-align: center; margin: 40px 0;">
    <a href="https://www.logans3dcreations.com/cart?token={{.TrackingToken}}{{if .PromoCode}}&promo={{.PromoCode}}{{end}}" style="display: inline-block; padding: 16px 40px; background-color: #E85D5D; color: white; text-decoration: none; border-radius: 5px; font-weight: 600; font-size: 18px;">{{if .PromoCode}}Claim Your 5% Discount{{else}}Return to Cart{{end}}</a>
</div>

<div style="text-align: center; margin-top: 30px; padding-top: 20px; border-top: 1px solid #ddd; color: #777; font-size: 14px;">
    <p>Questions? We're here to help!<br>
    <a href="mailto:prints@logans3dcreations.com" style="color: #E85D5D; text-decoration: none;">prints@logans3dcreations.com</a></p>
</div>
`

// abandonedCartRecovery72HrTemplate is for the final recovery email (72 hours after abandonment)
const abandonedCartRecovery72HrTemplate = `
<div style="text-align: center; margin-bottom: 30px;">
    <h1 style="color: #E85D5D; margin: 0; font-size: 28px;">Last Chance!</h1>
    <p style="font-size: 18px; color: #666; margin: 10px 0;">Your cart expires soon</p>
</div>

<div style="background-color: #fff3e0; padding: 20px; border-radius: 8px; border-left: 4px solid #FF6B6B; margin-bottom: 25px; text-align: center;">
    <p style="font-size: 16px; margin: 5px 0;">Hi {{.CustomerName}},</p>
    <p style="font-size: 16px; margin: 15px 0;"><strong>This is your final reminder!</strong> We're holding {{.ItemCount}} item{{if ne .ItemCount 1}}s{{end}} for you, but we can only save your cart for a limited time.</p>
</div>

<h2 style="color: #555; font-size: 20px; margin-top: 30px;">Last Chance Items</h2>
<table style="width: 100%; border-collapse: collapse; margin: 20px 0;">
    <thead>
        <tr style="background-color: #E85D5D;">
            <th style="color: white; padding: 12px; text-align: left; font-weight: 600;">Product</th>
            <th style="color: white; padding: 12px; text-align: center; font-weight: 600;">Quantity</th>
            <th style="color: white; padding: 12px; text-align: right; font-weight: 600;">Price</th>
        </tr>
    </thead>
    <tbody>
        {{range .Items}}
        <tr style="border-bottom: 1px solid #ddd;">
            <td style="padding: 12px;">
                <div style="display: flex; align-items: center; gap: 12px;">
                    {{if .ProductImage}}
                    <img src="https://www.logans3dcreations.com/public/images/products/{{.ProductImage}}" alt="{{.ProductName}}" style="width: 60px; height: 60px; object-fit: cover; border-radius: 4px; border: 1px solid #ddd;">
                    {{end}}
                    <span>{{.ProductName}}</span>
                </div>
            </td>
            <td style="padding: 12px; text-align: center;">{{.Quantity}}</td>
            <td style="padding: 12px; text-align: right;">{{FormatCents .UnitPrice}}</td>
        </tr>
        {{end}}
    </tbody>
</table>

<div style="margin-top: 20px; padding-top: 20px; border-top: 2px solid #ddd;">
    <div style="display: flex; justify-content: space-between; padding: 15px 0 0 0; font-size: 20px; font-weight: bold; color: #E85D5D;">
        <span>Cart Total:</span>
        <span>{{FormatCents .CartValue}}</span>
    </div>
</div>

{{if .PromoCode}}
<div style="background-color: #FFF3E0; padding: 30px; border-radius: 8px; margin: 30px 0; text-align: center; border: 3px solid #FF6B6B; box-shadow: 0 4px 12px rgba(255, 107, 107, 0.2);">
    <div style="font-size: 20px; font-weight: 700; color: #C62828; margin-bottom: 10px;">⏰ FINAL OFFER - Don't Miss Out!</div>
    <div style="font-size: 17px; color: #555; margin-bottom: 18px; font-weight: 600;">Save 5% Before Your Cart Expires</div>
    <div style="background-color: white; padding: 15px 24px; border-radius: 8px; display: inline-block; border: 3px dashed #FF6B6B; box-shadow: 0 2px 8px rgba(0,0,0,0.1);">
        <div style="font-size: 14px; color: #666; text-transform: uppercase; letter-spacing: 1px; margin-bottom: 6px;">Exclusive Code</div>
        <div style="font-size: 28px; font-weight: 700; color: #C62828; letter-spacing: 2px; font-family: 'Courier New', monospace;">{{.PromoCode}}</div>
    </div>
    <div style="font-size: 14px; color: #C62828; margin-top: 15px; font-weight: 600;">⚡ Auto-applied | Expires {{.PromoExpires}} ⚡</div>
</div>
{{end}}

<div style="text-align: center; margin: 40px 0;">
    <a href="https://www.logans3dcreations.com/cart?token={{.TrackingToken}}{{if .PromoCode}}&promo={{.PromoCode}}{{end}}" style="display: inline-block; padding: 18px 45px; background-color: #E85D5D; color: white; text-decoration: none; border-radius: 5px; font-weight: 700; font-size: 20px; box-shadow: 0 4px 6px rgba(0,0,0,0.1);">{{if .PromoCode}}Claim 5% Off Now{{else}}Complete Order Now{{end}}</a>
    <p style="font-size: 14px; color: #999; margin-top: 15px;">⏰ {{if .PromoCode}}Discount and cart expire{{else}}Cart expires{{end}} in 24 hours</p>
</div>

<div style="text-align: center; margin-top: 30px; padding-top: 20px; border-top: 1px solid #ddd; color: #777; font-size: 14px;">
    <p>Need assistance? Contact us anytime!<br>
    <a href="mailto:prints@logans3dcreations.com" style="color: #E85D5D; text-decoration: none;">prints@logans3dcreations.com</a></p>
</div>
`

// contactRequestContentTemplate is the content section for contact request notification emails
const contactRequestContentTemplate = `
<div style="text-align: center; margin-bottom: 30px;">
    <span style="display: inline-block; background-color: #3B82F6; color: white; padding: 5px 15px; border-radius: 20px; font-weight: 600; font-size: 14px; margin-bottom: 10px;">NEW CONTACT REQUEST</span>
    <h1 style="color: #3B82F6; margin: 10px 0; font-size: 28px;">Contact Form Submission</h1>
    <p style="font-size: 16px; color: #666; margin: 10px 0;">New message from your website</p>
</div>

<div style="background-color: #EFF6FF; padding: 20px; border-left: 4px solid #3B82F6; margin-bottom: 25px;">
    <p style="margin: 8px 0;"><strong style="color: #1E40AF; min-width: 150px; display: inline-block;">Request ID:</strong> #{{.ID}}</p>
    <p style="margin: 8px 0;"><strong style="color: #1E40AF; min-width: 150px; display: inline-block;">Submitted:</strong> {{.SubmittedAt}}</p>
    <p style="margin: 8px 0;"><strong style="color: #1E40AF; min-width: 150px; display: inline-block;">Subject:</strong> {{.Subject}}</p>
</div>

<div style="background-color: #F0FDF4; padding: 20px; border-radius: 8px; border-left: 4px solid #10B981; margin: 20px 0;">
    <h3 style="margin-top: 0; color: #065F46; font-size: 16px;">Contact Information</h3>
    <p style="margin: 5px 0;"><strong>Name:</strong> {{.FirstName}} {{.LastName}}</p>
    {{if .Email}}
    <p style="margin: 5px 0;"><strong>Email:</strong> <a href="mailto:{{.Email}}" style="color: #10B981; text-decoration: none;">{{.Email}}</a></p>
    {{end}}
    {{if .Phone}}
    <p style="margin: 5px 0;"><strong>Phone:</strong> <a href="tel:{{.Phone}}" style="color: #10B981; text-decoration: none;">{{.Phone}}</a></p>
    {{end}}
    {{if .NewsletterSubscribe}}
    <p style="margin: 5px 0;"><strong>Newsletter:</strong> <span style="color: #10B981;">✓ Subscribed</span></p>
    {{end}}
</div>

<div style="background-color: #F9FAFB; padding: 20px; border-radius: 8px; margin: 20px 0;">
    <h3 style="margin-top: 0; color: #374151; font-size: 16px;">Message</h3>
    <p style="margin: 5px 0; white-space: pre-wrap; line-height: 1.6;">{{.Message}}</p>
</div>
`



// welcomeCouponContentTemplate is the content section for welcome coupon emails
const welcomeCouponContentTemplate = `
<div style="text-align: center; margin-bottom: 30px;">
    <div style="display: inline-block; background-color: #10b981; color: white; padding: 8px 20px; border-radius: 25px; font-weight: 700; font-size: 16px; margin-bottom: 20px;">🎉 WELCOME GIFT</div>
    <h1 style="color: #E85D5D; margin: 10px 0; font-size: 32px;">Welcome to Logans 3D Creations!</h1>
    <p style="font-size: 18px; color: #666; margin: 15px 0;">Thanks for signing up! Here's a special gift just for you.</p>
</div>

<div style="background: linear-gradient(135deg, #E85D5D 0%, #ff6b6b 100%); padding: 40px 30px; border-radius: 12px; text-align: center; margin: 30px 0; box-shadow: 0 10px 25px rgba(232, 93, 93, 0.3);">
    <p style="color: rgba(255,255,255,0.95); font-size: 18px; margin: 0 0 15px 0; font-weight: 600;">Your Exclusive Discount Code</p>
    <div style="background-color: white; padding: 20px 30px; border-radius: 8px; margin: 20px auto; max-width: 300px; box-shadow: 0 4px 12px rgba(0,0,0,0.1);">
        <p style="color: #E85D5D; font-size: 36px; font-weight: 800; letter-spacing: 2px; margin: 0; font-family: 'Courier New', monospace;">{{.PromoCode}}</p>
    </div>
    <p style="color: white; font-size: 24px; font-weight: 700; margin: 15px 0;">{{.DiscountText}} Your First Order</p>
    <p style="color: rgba(255,255,255,0.9); font-size: 14px; margin: 10px 0;">Valid until {{.ExpiresAt}}</p>
</div>

<div style="margin: 35px 0;">
    <h2 style="color: #555; font-size: 22px; margin-bottom: 20px; text-align: center;">Why Choose Logan's 3D Creations?</h2>
    <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; margin: 25px 0;">
        <div style="text-align: center; padding: 20px;">
            <div style="font-size: 40px; margin-bottom: 10px;">🎨</div>
            <h3 style="color: #E85D5D; font-size: 16px; margin: 10px 0;">Premium Quality</h3>
            <p style="color: #666; font-size: 14px; margin: 5px 0;">High-quality 3D prints with attention to detail</p>
        </div>
        <div style="text-align: center; padding: 20px;">
            <div style="font-size: 40px; margin-bottom: 10px;">⚡</div>
            <h3 style="color: #E85D5D; font-size: 16px; margin: 10px 0;">Fast Shipping</h3>
            <p style="color: #666; font-size: 14px; margin: 5px 0;">Quick turnaround and reliable delivery</p>
        </div>
        <div style="text-align: center; padding: 20px;">
            <div style="font-size: 40px; margin-bottom: 10px;">💪</div>
            <h3 style="color: #E85D5D; font-size: 16px; margin: 10px 0;">Durable Prints</h3>
            <p style="color: #666; font-size: 14px; margin: 5px 0;">Long-lasting materials and expert craftsmanship</p>
        </div>
    </div>
</div>

<div style="text-align: center; margin: 40px 0;">
    <a href="https://www.logans3dcreations.com/shop" style="display: inline-block; padding: 18px 45px; background-color: #E85D5D; color: white; text-decoration: none; border-radius: 8px; font-weight: 700; font-size: 18px; box-shadow: 0 4px 12px rgba(232, 93, 93, 0.3); transition: all 0.3s;">Start Shopping Now →</a>
    <p style="font-size: 14px; color: #999; margin-top: 15px;">Use code <strong style="color: #E85D5D;">{{.PromoCode}}</strong> at checkout</p>
</div>

<div style="background-color: #f9f9f9; padding: 25px; border-radius: 8px; margin: 30px 0; border-left: 4px solid #10b981;">
    <h3 style="color: #555; font-size: 18px; margin: 0 0 15px 0;">📝 How to Redeem:</h3>
    <ol style="color: #666; margin: 0; padding-left: 20px; line-height: 1.8;">
        <li>Browse our shop and add items to your cart</li>
        <li>At checkout, enter code <strong>{{.PromoCode}}</strong></li>
        <li>Your discount will be applied automatically</li>
        <li>Complete your order and enjoy!</li>
    </ol>
</div>

<div style="text-align: center; margin-top: 35px; padding-top: 25px; border-top: 1px solid #ddd; color: #777; font-size: 14px;">
    <p style="margin: 5px 0;">Questions? We're here to help!</p>
    <p style="margin: 5px 0;">
        <a href="mailto:prints@logans3dcreations.com" style="color: #E85D5D; text-decoration: none; font-weight: 600;">prints@logans3dcreations.com</a>
    </p>
</div>
`
