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
            <td style="padding: 12px;">{{.ProductName}}</td>
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
    <a href="https://www.logans3dcreations.com" style="display: inline-block; padding: 12px 30px; background-color: #E85D5D; color: white; text-decoration: none; border-radius: 5px; font-weight: 600;">Visit Our Store</a>
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
            <td style="padding: 12px;"><strong>{{.ProductName}}</strong></td>
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
    <a href="https://www.logans3dcreations.com/admin/orders" style="display: inline-block; padding: 12px 30px; background-color: #FF9800; color: white; text-decoration: none; border-radius: 5px; font-weight: 600; margin: 0 10px;">View Order</a>
    <a href="https://www.logans3dcreations.com/admin/orders" style="display: inline-block; padding: 12px 30px; background-color: #FF9800; color: white; text-decoration: none; border-radius: 5px; font-weight: 600; margin: 0 10px;">Process Order</a>
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
