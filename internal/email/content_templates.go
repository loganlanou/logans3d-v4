package email

// customerOrderContentTemplate is the content section for customer order emails
const customerOrderContentTemplate = `
<div style="text-align: center; margin-bottom: 30px;">
    <h1 style="color: #E85D5D; margin: 0; font-size: 28px;">Order Confirmation</h1>
    <p style="font-size: 18px; color: #666; margin: 10px 0;">Thank you for your order, {{.CustomerName}}!</p>
</div>

<table width="100%" cellpadding="0" cellspacing="0" border="0" bgcolor="#f9f9f9" style="background-color: #f9f9f9; margin-bottom: 25px;">
    <tr>
        <td style="padding: 20px; border-left: 4px solid #E85D5D;">
            <p style="margin: 5px 0;"><strong style="color: #555;">Order Number:</strong> #{{.OrderID}}</p>
            <p style="margin: 5px 0;"><strong style="color: #555;">Order Date:</strong> {{.OrderDate}}</p>
            <p style="margin: 5px 0;"><strong style="color: #555;">Customer Email:</strong> {{.CustomerEmail}}</p>
        </td>
    </tr>
</table>

<h2 style="color: #555; font-size: 20px; margin-top: 30px;">Order Items</h2>
<table width="100%" cellpadding="0" cellspacing="0" border="0" style="margin: 20px 0;">
    <thead>
        <tr bgcolor="#E85D5D" style="background-color: #E85D5D;">
            <th style="color: white; padding: 12px; text-align: left; font-weight: 600;">Product</th>
            <th style="color: white; padding: 12px; text-align: center; font-weight: 600;">Quantity</th>
            <th style="color: white; padding: 12px; text-align: right; font-weight: 600;">Price</th>
            <th style="color: white; padding: 12px; text-align: right; font-weight: 600;">Total</th>
        </tr>
    </thead>
    <tbody>
        {{range .Items}}
        <tr style="border-bottom: 1px solid #ddd;">
            <td style="padding: 12px; border-bottom: 1px solid #ddd;">
                <table cellpadding="0" cellspacing="0" border="0">
                    <tr>
                        {{if .ProductImage}}
                        <td style="padding-right: 12px; vertical-align: top;">
                            <img src="https://www.logans3dcreations.com/public/images/products/{{.ProductImage}}" alt="{{.ProductName}}" width="60" height="60" style="display: block; width: 60px; height: 60px; border-radius: 4px; border: 1px solid #ddd;" />
                        </td>
                        {{end}}
                        <td style="vertical-align: top;">{{.ProductName}}</td>
                    </tr>
                </table>
            </td>
            <td style="padding: 12px; text-align: center; border-bottom: 1px solid #ddd;">{{.Quantity}}</td>
            <td style="padding: 12px; text-align: right; border-bottom: 1px solid #ddd;">{{FormatCents .PriceCents}}</td>
            <td style="padding: 12px; text-align: right; border-bottom: 1px solid #ddd;">{{FormatCents .TotalCents}}</td>
        </tr>
        {{end}}
    </tbody>
</table>

<div style="margin-top: 20px; padding-top: 20px; border-top: 2px solid #ddd;">
    <table width="100%" cellpadding="0" cellspacing="0" border="0">
        <tr>
            <td style="padding: 8px 0;">Subtotal:</td>
            <td style="padding: 8px 0; text-align: right;">{{FormatCents .SubtotalCents}}</td>
        </tr>
        <tr>
            <td style="padding: 8px 0;">Tax:</td>
            <td style="padding: 8px 0; text-align: right;">{{FormatCents .TaxCents}}</td>
        </tr>
        <tr>
            <td style="padding: 8px 0;">Shipping:</td>
            <td style="padding: 8px 0; text-align: right;">{{FormatCents .ShippingCents}}</td>
        </tr>
        <tr>
            <td style="padding: 15px 0 0 0; margin-top: 10px; border-top: 2px solid #E85D5D; font-size: 18px; font-weight: bold; color: #E85D5D; padding-top: 15px;">Total:</td>
            <td style="padding: 15px 0 0 0; margin-top: 10px; border-top: 2px solid #E85D5D; font-size: 18px; font-weight: bold; color: #E85D5D; text-align: right; padding-top: 15px;">{{FormatCents .TotalCents}}</td>
        </tr>
    </table>
</div>

<table width="100%" cellpadding="0" cellspacing="0" border="0" bgcolor="#f9f9f9" style="background-color: #f9f9f9; margin-top: 25px;">
    <tr>
        <td style="padding: 20px;">
            <h3 style="margin-top: 0; color: #555; font-size: 16px;">Shipping Address</h3>
            <p style="margin: 5px 0;">{{.ShippingAddress.Name}}<br>
            {{.ShippingAddress.Line1}}<br>
            {{if .ShippingAddress.Line2}}{{.ShippingAddress.Line2}}<br>{{end}}
            {{.ShippingAddress.City}}, {{.ShippingAddress.State}} {{.ShippingAddress.PostalCode}}<br>
            {{.ShippingAddress.Country}}</p>
        </td>
    </tr>
</table>

<div style="text-align: center; margin: 30px 0;">
    <p style="color: #555; margin-bottom: 15px;">Track your order status and view details:</p>
    <table cellpadding="0" cellspacing="0" border="0" align="center">
        <tr>
            <td bgcolor="#E85D5D" style="background-color: #E85D5D; padding: 14px 35px; border-radius: 5px;">
                <a href="https://www.logans3dcreations.com/account/orders/{{.OrderID}}" style="color: white; text-decoration: none; font-weight: 600; font-size: 16px; display: block;">View Order Status</a>
            </td>
        </tr>
    </table>
    <br>
    <table cellpadding="0" cellspacing="0" border="1" align="center" style="border: 2px solid #E85D5D;">
        <tr>
            <td style="padding: 10px 25px;">
                <a href="https://www.logans3dcreations.com" style="color: #E85D5D; text-decoration: none; font-weight: 600;">Continue Shopping</a>
            </td>
        </tr>
    </table>
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
    <table cellpadding="0" cellspacing="0" border="0" align="center">
        <tr>
            <td bgcolor="#FF9800" style="background-color: #FF9800; color: white; padding: 5px 15px; border-radius: 20px; font-weight: 600; font-size: 14px;">NEW ORDER</td>
        </tr>
    </table>
    <h1 style="color: #FF9800; margin: 10px 0; font-size: 28px;">Order Received</h1>
    <p style="font-size: 16px; color: #666; margin: 10px 0;">Action required - Process this order</p>
</div>

<table width="100%" cellpadding="0" cellspacing="0" border="0" bgcolor="#fff3e0" style="background-color: #fff3e0; margin-bottom: 25px;">
    <tr>
        <td style="padding: 20px; border-left: 4px solid #FF9800;">
            <table width="100%" cellpadding="0" cellspacing="0" border="0">
                <tr>
                    <td style="padding: 8px 0;"><strong style="color: #e65100; width: 150px; display: inline-block;">Order Number:</strong> #{{.OrderID}}</td>
                </tr>
                <tr>
                    <td style="padding: 8px 0;"><strong style="color: #e65100; width: 150px; display: inline-block;">Order Date:</strong> {{.OrderDate}}</td>
                </tr>
                <tr>
                    <td style="padding: 8px 0;"><strong style="color: #e65100; width: 150px; display: inline-block;">Payment Intent:</strong> {{.PaymentIntentID}}</td>
                </tr>
                <tr>
                    <td style="padding: 8px 0;"><strong style="color: #e65100; width: 150px; display: inline-block;">Order Total:</strong> {{FormatCents .TotalCents}}</td>
                </tr>
            </table>
        </td>
    </tr>
</table>

<table width="100%" cellpadding="0" cellspacing="0" border="0" bgcolor="#e3f2fd" style="background-color: #e3f2fd; margin: 20px 0; padding: 20px;">
    <tr>
        <td style="padding: 20px;">
            <h3 style="margin-top: 0; color: #1976D2; font-size: 16px;">Customer Information</h3>
            <p style="margin: 5px 0;"><strong>Name:</strong> {{.CustomerName}}</p>
            <p style="margin: 5px 0;"><strong>Email:</strong> <a href="mailto:{{.CustomerEmail}}" style="color: #2196F3; text-decoration: none;">{{.CustomerEmail}}</a></p>
        </td>
    </tr>
</table>

<h2 style="color: #555; font-size: 20px; margin-top: 30px;">Order Items</h2>
<table width="100%" cellpadding="0" cellspacing="0" border="0" style="margin: 20px 0;">
    <thead>
        <tr bgcolor="#FF9800" style="background-color: #FF9800;">
            <th style="color: white; padding: 12px; text-align: left; font-weight: 600;">Product</th>
            <th style="color: white; padding: 12px; text-align: center; font-weight: 600;">Quantity</th>
            <th style="color: white; padding: 12px; text-align: right; font-weight: 600;">Unit Price</th>
            <th style="color: white; padding: 12px; text-align: right; font-weight: 600;">Total</th>
        </tr>
    </thead>
    <tbody>
        {{range .Items}}
        <tr style="border-bottom: 1px solid #ddd;">
            <td style="padding: 12px; border-bottom: 1px solid #ddd;">
                <table cellpadding="0" cellspacing="0" border="0">
                    <tr>
                        {{if .ProductImage}}
                        <td style="padding-right: 12px; vertical-align: top;">
                            <img src="https://www.logans3dcreations.com/public/images/products/{{.ProductImage}}" alt="{{.ProductName}}" width="60" height="60" style="display: block; width: 60px; height: 60px; border-radius: 4px; border: 1px solid #ddd;" />
                        </td>
                        {{end}}
                        <td style="vertical-align: top;"><strong>{{.ProductName}}</strong></td>
                    </tr>
                </table>
            </td>
            <td style="padding: 12px; text-align: center; border-bottom: 1px solid #ddd;">{{.Quantity}}</td>
            <td style="padding: 12px; text-align: right; border-bottom: 1px solid #ddd;">{{FormatCents .PriceCents}}</td>
            <td style="padding: 12px; text-align: right; border-bottom: 1px solid #ddd;">{{FormatCents .TotalCents}}</td>
        </tr>
        {{end}}
    </tbody>
</table>

<table width="100%" cellpadding="0" cellspacing="0" border="0" bgcolor="#f9f9f9" style="background-color: #f9f9f9; margin-top: 20px;">
    <tr>
        <td style="padding: 20px;">
            <table width="100%" cellpadding="0" cellspacing="0" border="0">
                <tr>
                    <td style="padding: 8px 0; font-size: 16px;">Subtotal:</td>
                    <td style="padding: 8px 0; text-align: right; font-size: 16px;">{{FormatCents .SubtotalCents}}</td>
                </tr>
                <tr>
                    <td style="padding: 8px 0; font-size: 16px;">Tax:</td>
                    <td style="padding: 8px 0; text-align: right; font-size: 16px;">{{FormatCents .TaxCents}}</td>
                </tr>
                <tr>
                    <td style="padding: 8px 0; font-size: 16px;">Shipping:</td>
                    <td style="padding: 8px 0; text-align: right; font-size: 16px;">{{FormatCents .ShippingCents}}</td>
                </tr>
                <tr>
                    <td style="padding: 15px 0 0 0; border-top: 2px solid #FF9800; font-size: 20px; font-weight: bold; color: #FF9800;">TOTAL:</td>
                    <td style="padding: 15px 0 0 0; border-top: 2px solid #FF9800; font-size: 20px; font-weight: bold; color: #FF9800; text-align: right;">{{FormatCents .TotalCents}}</td>
                </tr>
            </table>
        </td>
    </tr>
</table>

<h2 style="color: #555; font-size: 20px; margin-top: 30px;">Shipping & Billing Details</h2>
<table width="100%" cellpadding="0" cellspacing="0" border="0" style="margin: 25px 0;">
    <tr>
        <td style="padding-right: 10px; width: 50%;">
            <table width="100%" cellpadding="0" cellspacing="0" border="1" bgcolor="#f9f9f9" style="background-color: #f9f9f9; border: 1px solid #ddd;">
                <tr>
                    <td style="padding: 15px; border-bottom: 2px solid #FF9800;">
                        <h3 style="margin-top: 0; color: #e65100; font-size: 16px; margin-bottom: 8px;">Ship To</h3>
                    </td>
                </tr>
                <tr>
                    <td style="padding: 15px; padding-top: 0;">
                        <p style="margin: 5px 0;"><strong>{{.ShippingAddress.Name}}</strong><br>
                        {{.ShippingAddress.Line1}}<br>
                        {{if .ShippingAddress.Line2}}{{.ShippingAddress.Line2}}<br>{{end}}
                        {{.ShippingAddress.City}}, {{.ShippingAddress.State}} {{.ShippingAddress.PostalCode}}<br>
                        {{.ShippingAddress.Country}}</p>
                    </td>
                </tr>
            </table>
        </td>
        <td style="padding-left: 10px; width: 50%;">
            <table width="100%" cellpadding="0" cellspacing="0" border="1" bgcolor="#f9f9f9" style="background-color: #f9f9f9; border: 1px solid #ddd;">
                <tr>
                    <td style="padding: 15px; border-bottom: 2px solid #FF9800;">
                        <h3 style="margin-top: 0; color: #e65100; font-size: 16px; margin-bottom: 8px;">Bill To</h3>
                    </td>
                </tr>
                <tr>
                    <td style="padding: 15px; padding-top: 0;">
                        <p style="margin: 5px 0;"><strong>{{.BillingAddress.Name}}</strong><br>
                        {{.BillingAddress.Line1}}<br>
                        {{if .BillingAddress.Line2}}{{.BillingAddress.Line2}}<br>{{end}}
                        {{.BillingAddress.City}}, {{.BillingAddress.State}} {{.BillingAddress.PostalCode}}<br>
                        {{.BillingAddress.Country}}</p>
                    </td>
                </tr>
            </table>
        </td>
    </tr>
</table>

<div style="text-align: center; margin: 30px 0;">
    <table cellpadding="0" cellspacing="0" border="0" align="center">
        <tr>
            <td bgcolor="#FF9800" style="background-color: #FF9800; padding: 12px 30px; border-radius: 5px;">
                <a href="https://www.logans3dcreations.com/admin/orders/{{.OrderID}}" style="color: white; text-decoration: none; font-weight: 600; display: block;">View Order</a>
            </td>
        </tr>
    </table>
</div>

<table width="100%" cellpadding="0" cellspacing="0" border="0" bgcolor="#f9f9f9" style="background-color: #f9f9f9; margin-top: 30px;">
    <tr>
        <td style="padding: 20px;">
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
        </td>
    </tr>
</table>
`

// abandonedCartRecovery1HrTemplate is for the first recovery email (1 hour after abandonment)
const abandonedCartRecovery1HrTemplate = `
<div style="text-align: center; margin-bottom: 30px;">
    <h1 style="color: #E85D5D; margin: 0; font-size: 28px;">You Left Something Behind!</h1>
    <p style="font-size: 18px; color: #666; margin: 10px 0;">We saved your cart for you</p>
</div>

<table width="100%" cellpadding="0" cellspacing="0" border="0" bgcolor="#f9f9f9" style="background-color: #f9f9f9; margin-bottom: 25px;">
    <tr>
        <td style="padding: 20px; border-left: 4px solid #E85D5D; text-align: center;">
            <p style="font-size: 16px; margin: 5px 0;">Hi {{.CustomerName}},</p>
            <p style="font-size: 16px; margin: 15px 0;">We noticed you left {{.ItemCount}} item{{if ne .ItemCount 1}}s{{end}} in your cart. Don't worry - we've saved everything for you!</p>
        </td>
    </tr>
</table>

<h2 style="color: #555; font-size: 20px; margin-top: 30px;">Your Cart</h2>
<table width="100%" cellpadding="0" cellspacing="0" border="0" style="margin: 20px 0;">
    <thead>
        <tr bgcolor="#E85D5D" style="background-color: #E85D5D;">
            <th style="color: white; padding: 12px; text-align: left; font-weight: 600;">Product</th>
            <th style="color: white; padding: 12px; text-align: center; font-weight: 600;">Quantity</th>
            <th style="color: white; padding: 12px; text-align: right; font-weight: 600;">Price</th>
        </tr>
    </thead>
    <tbody>
        {{range .Items}}
        <tr style="border-bottom: 1px solid #ddd;">
            <td style="padding: 12px; border-bottom: 1px solid #ddd;">
                <table cellpadding="0" cellspacing="0" border="0">
                    <tr>
                        {{if .ProductImage}}
                        <td style="padding-right: 12px; vertical-align: top;">
                            <img src="https://www.logans3dcreations.com/public/images/products/{{.ProductImage}}" alt="{{.ProductName}}" width="60" height="60" style="display: block; width: 60px; height: 60px; border-radius: 4px; border: 1px solid #ddd;" />
                        </td>
                        {{end}}
                        <td style="vertical-align: top;">{{.ProductName}}</td>
                    </tr>
                </table>
            </td>
            <td style="padding: 12px; text-align: center; border-bottom: 1px solid #ddd;">{{.Quantity}}</td>
            <td style="padding: 12px; text-align: right; border-bottom: 1px solid #ddd;">{{FormatCents .UnitPrice}}</td>
        </tr>
        {{end}}
    </tbody>
</table>

<div style="margin-top: 20px; padding-top: 20px; border-top: 2px solid #ddd;">
    <table width="100%" cellpadding="0" cellspacing="0" border="0">
        <tr>
            <td style="padding: 15px 0 0 0; font-size: 20px; font-weight: bold; color: #E85D5D;">Cart Total:</td>
            <td style="padding: 15px 0 0 0; text-align: right; font-size: 20px; font-weight: bold; color: #E85D5D;">{{FormatCents .CartValue}}</td>
        </tr>
    </table>
</div>

<div style="text-align: center; margin: 40px 0;">
    <table cellpadding="0" cellspacing="0" border="0" align="center">
        <tr>
            <td bgcolor="#E85D5D" style="background-color: #E85D5D; padding: 16px 40px; border-radius: 5px;">
                <a href="https://www.logans3dcreations.com/cart?token={{.TrackingToken}}" style="color: white; text-decoration: none; font-weight: 600; font-size: 18px; display: block;">Complete Your Order</a>
            </td>
        </tr>
    </table>
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

<table width="100%" cellpadding="0" cellspacing="0" border="0" bgcolor="#f9f9f9" style="background-color: #f9f9f9; margin-bottom: 25px;">
    <tr>
        <td style="padding: 20px; border-left: 4px solid #E85D5D; text-align: center;">
            <p style="font-size: 16px; margin: 5px 0;">Hi {{.CustomerName}},</p>
            <p style="font-size: 16px; margin: 15px 0;">We're holding these items for you. Complete your order today!</p>
        </td>
    </tr>
</table>

<h2 style="color: #555; font-size: 20px; margin-top: 30px;">Items in Your Cart</h2>
<table width="100%" cellpadding="0" cellspacing="0" border="0" style="margin: 20px 0;">
    <thead>
        <tr bgcolor="#E85D5D" style="background-color: #E85D5D;">
            <th style="color: white; padding: 12px; text-align: left; font-weight: 600;">Product</th>
            <th style="color: white; padding: 12px; text-align: center; font-weight: 600;">Quantity</th>
            <th style="color: white; padding: 12px; text-align: right; font-weight: 600;">Price</th>
        </tr>
    </thead>
    <tbody>
        {{range .Items}}
        <tr style="border-bottom: 1px solid #ddd;">
            <td style="padding: 12px; border-bottom: 1px solid #ddd;">
                <table cellpadding="0" cellspacing="0" border="0">
                    <tr>
                        {{if .ProductImage}}
                        <td style="padding-right: 12px; vertical-align: top;">
                            <img src="https://www.logans3dcreations.com/public/images/products/{{.ProductImage}}" alt="{{.ProductName}}" width="60" height="60" style="display: block; width: 60px; height: 60px; border-radius: 4px; border: 1px solid #ddd;" />
                        </td>
                        {{end}}
                        <td style="vertical-align: top;">{{.ProductName}}</td>
                    </tr>
                </table>
            </td>
            <td style="padding: 12px; text-align: center; border-bottom: 1px solid #ddd;">{{.Quantity}}</td>
            <td style="padding: 12px; text-align: right; border-bottom: 1px solid #ddd;">{{FormatCents .UnitPrice}}</td>
        </tr>
        {{end}}
    </tbody>
</table>

<div style="margin-top: 20px; padding-top: 20px; border-top: 2px solid #ddd;">
    <table width="100%" cellpadding="0" cellspacing="0" border="0">
        <tr>
            <td style="padding: 15px 0 0 0; font-size: 20px; font-weight: bold; color: #E85D5D;">Cart Total:</td>
            <td style="padding: 15px 0 0 0; text-align: right; font-size: 20px; font-weight: bold; color: #E85D5D;">{{FormatCents .CartValue}}</td>
        </tr>
    </table>
</div>

{{if .PromoCode}}
<table width="100%" cellpadding="0" cellspacing="0" border="0" bgcolor="#E8F5E9" style="background-color: #E8F5E9; margin: 30px 0; border: 2px solid #4CAF50;">
    <tr>
        <td style="padding: 25px; text-align: center;">
            <div style="font-size: 18px; font-weight: 600; color: #2E7D32; margin-bottom: 12px;">🎉 Special Offer Just For You!</div>
            <div style="font-size: 16px; color: #555; margin-bottom: 15px;">Save 5% on your first order</div>
            <table cellpadding="0" cellspacing="0" border="2" align="center" style="border: 2px dashed #4CAF50; background-color: white;">
                <tr>
                    <td style="padding: 12px 20px;">
                        <div style="font-size: 13px; color: #666; text-transform: uppercase; letter-spacing: 1px; margin-bottom: 4px;">Your Code</div>
                        <div style="font-size: 24px; font-weight: 700; color: #2E7D32; letter-spacing: 2px; font-family: 'Courier New', monospace;">{{.PromoCode}}</div>
                    </td>
                </tr>
            </table>
            <div style="font-size: 13px; color: #666; margin-top: 12px;">✓ Auto-applied at checkout | Expires {{.PromoExpires}}</div>
        </td>
    </tr>
</table>
{{else}}
<table width="100%" cellpadding="0" cellspacing="0" border="0" bgcolor="#fff9e6" style="background-color: #fff9e6; margin: 30px 0; border: 2px dashed #FFA000;">
    <tr>
        <td style="padding: 20px; text-align: center;">
            <p style="font-size: 16px; margin: 0; color: #555;">💡 <strong>Need help deciding?</strong> Contact us with any questions!</p>
        </td>
    </tr>
</table>
{{end}}

<div style="text-align: center; margin: 40px 0;">
    <table cellpadding="0" cellspacing="0" border="0" align="center">
        <tr>
            <td bgcolor="#E85D5D" style="background-color: #E85D5D; padding: 16px 40px; border-radius: 5px;">
                <a href="https://www.logans3dcreations.com/cart?token={{.TrackingToken}}{{if .PromoCode}}&promo={{.PromoCode}}{{end}}" style="color: white; text-decoration: none; font-weight: 600; font-size: 18px; display: block;">{{if .PromoCode}}Claim Your 5% Discount{{else}}Return to Cart{{end}}</a>
            </td>
        </tr>
    </table>
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

<table width="100%" cellpadding="0" cellspacing="0" border="0" bgcolor="#fff3e0" style="background-color: #fff3e0; margin-bottom: 25px;">
    <tr>
        <td style="padding: 20px; border-left: 4px solid #FF6B6B; text-align: center;">
            <p style="font-size: 16px; margin: 5px 0;">Hi {{.CustomerName}},</p>
            <p style="font-size: 16px; margin: 15px 0;"><strong>This is your final reminder!</strong> We're holding {{.ItemCount}} item{{if ne .ItemCount 1}}s{{end}} for you, but we can only save your cart for a limited time.</p>
        </td>
    </tr>
</table>

<h2 style="color: #555; font-size: 20px; margin-top: 30px;">Last Chance Items</h2>
<table width="100%" cellpadding="0" cellspacing="0" border="0" style="margin: 20px 0;">
    <thead>
        <tr bgcolor="#E85D5D" style="background-color: #E85D5D;">
            <th style="color: white; padding: 12px; text-align: left; font-weight: 600;">Product</th>
            <th style="color: white; padding: 12px; text-align: center; font-weight: 600;">Quantity</th>
            <th style="color: white; padding: 12px; text-align: right; font-weight: 600;">Price</th>
        </tr>
    </thead>
    <tbody>
        {{range .Items}}
        <tr style="border-bottom: 1px solid #ddd;">
            <td style="padding: 12px; border-bottom: 1px solid #ddd;">
                <table cellpadding="0" cellspacing="0" border="0">
                    <tr>
                        {{if .ProductImage}}
                        <td style="padding-right: 12px; vertical-align: top;">
                            <img src="https://www.logans3dcreations.com/public/images/products/{{.ProductImage}}" alt="{{.ProductName}}" width="60" height="60" style="display: block; width: 60px; height: 60px; border-radius: 4px; border: 1px solid #ddd;" />
                        </td>
                        {{end}}
                        <td style="vertical-align: top;">{{.ProductName}}</td>
                    </tr>
                </table>
            </td>
            <td style="padding: 12px; text-align: center; border-bottom: 1px solid #ddd;">{{.Quantity}}</td>
            <td style="padding: 12px; text-align: right; border-bottom: 1px solid #ddd;">{{FormatCents .UnitPrice}}</td>
        </tr>
        {{end}}
    </tbody>
</table>

<div style="margin-top: 20px; padding-top: 20px; border-top: 2px solid #ddd;">
    <table width="100%" cellpadding="0" cellspacing="0" border="0">
        <tr>
            <td style="padding: 15px 0 0 0; font-size: 20px; font-weight: bold; color: #E85D5D;">Cart Total:</td>
            <td style="padding: 15px 0 0 0; text-align: right; font-size: 20px; font-weight: bold; color: #E85D5D;">{{FormatCents .CartValue}}</td>
        </tr>
    </table>
</div>

{{if .PromoCode}}
<table width="100%" cellpadding="0" cellspacing="0" border="0" bgcolor="#FFF3E0" style="background-color: #FFF3E0; margin: 30px 0; border: 3px solid #FF6B6B;">
    <tr>
        <td style="padding: 30px; text-align: center;">
            <div style="font-size: 20px; font-weight: 700; color: #C62828; margin-bottom: 10px;">⏰ FINAL OFFER - Don't Miss Out!</div>
            <div style="font-size: 17px; color: #555; margin-bottom: 18px; font-weight: 600;">Save 5% Before Your Cart Expires</div>
            <table cellpadding="0" cellspacing="0" border="3" align="center" style="border: 3px dashed #FF6B6B; background-color: white;">
                <tr>
                    <td style="padding: 15px 24px;">
                        <div style="font-size: 14px; color: #666; text-transform: uppercase; letter-spacing: 1px; margin-bottom: 6px;">Exclusive Code</div>
                        <div style="font-size: 28px; font-weight: 700; color: #C62828; letter-spacing: 2px; font-family: 'Courier New', monospace;">{{.PromoCode}}</div>
                    </td>
                </tr>
            </table>
            <div style="font-size: 14px; color: #C62828; margin-top: 15px; font-weight: 600;">⚡ Auto-applied | Expires {{.PromoExpires}} ⚡</div>
        </td>
    </tr>
</table>
{{end}}

<div style="text-align: center; margin: 40px 0;">
    <table cellpadding="0" cellspacing="0" border="0" align="center">
        <tr>
            <td bgcolor="#E85D5D" style="background-color: #E85D5D; padding: 18px 45px; border-radius: 5px;">
                <a href="https://www.logans3dcreations.com/cart?token={{.TrackingToken}}{{if .PromoCode}}&promo={{.PromoCode}}{{end}}" style="color: white; text-decoration: none; font-weight: 700; font-size: 20px; display: block;">{{if .PromoCode}}Claim 5% Off Now{{else}}Complete Order Now{{end}}</a>
            </td>
        </tr>
    </table>
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
    <table cellpadding="0" cellspacing="0" border="0" align="center">
        <tr>
            <td bgcolor="#10b981" style="background-color: #10b981; color: white; padding: 8px 20px; border-radius: 25px; font-weight: 700; font-size: 16px;">🎉 WELCOME GIFT</td>
        </tr>
    </table>
    <h1 style="color: #E85D5D; margin: 10px 0; font-size: 32px;">Welcome to Logans 3D Creations!</h1>
    <p style="font-size: 18px; color: #666; margin: 15px 0;">Thanks for signing up! Here's a special gift just for you.</p>
</div>

<table width="100%" cellpadding="0" cellspacing="0" border="0" bgcolor="#E85D5D" style="background-color: #E85D5D; margin: 30px 0; border-radius: 12px;">
    <tr>
        <td style="padding: 40px 30px; text-align: center;">
            <p style="color: rgba(255,255,255,0.95); font-size: 18px; margin: 0 0 15px 0; font-weight: 600;">Your Exclusive Discount Code</p>
            <table cellpadding="0" cellspacing="0" border="0" align="center" bgcolor="white" style="background-color: white; margin: 20px auto;">
                <tr>
                    <td style="padding: 20px 30px;">
                        <p style="color: #E85D5D; font-size: 36px; font-weight: 800; letter-spacing: 2px; margin: 0; font-family: 'Courier New', monospace;">{{.PromoCode}}</p>
                    </td>
                </tr>
            </table>
            <p style="color: white; font-size: 24px; font-weight: 700; margin: 15px 0;">{{.DiscountText}} Your First Order</p>
            <p style="color: rgba(255,255,255,0.9); font-size: 14px; margin: 10px 0;">Valid until {{.ExpiresAt}}</p>
        </td>
    </tr>
</table>

<div style="margin: 35px 0;">
    <h2 style="color: #555; font-size: 22px; margin-bottom: 20px; text-align: center;">Why Choose Logan's 3D Creations?</h2>
    <table width="100%" cellpadding="0" cellspacing="0" border="0" style="margin: 25px 0;">
        <tr>
            <td style="width: 33%; padding: 20px; text-align: center; vertical-align: top;">
                <div style="font-size: 40px; margin-bottom: 10px;">🎨</div>
                <h3 style="color: #E85D5D; font-size: 16px; margin: 10px 0;">Premium Quality</h3>
                <p style="color: #666; font-size: 14px; margin: 5px 0;">High-quality 3D prints with attention to detail</p>
            </td>
            <td style="width: 33%; padding: 20px; text-align: center; vertical-align: top;">
                <div style="font-size: 40px; margin-bottom: 10px;">⚡</div>
                <h3 style="color: #E85D5D; font-size: 16px; margin: 10px 0;">Fast Shipping</h3>
                <p style="color: #666; font-size: 14px; margin: 5px 0;">Quick turnaround and reliable delivery</p>
            </td>
            <td style="width: 33%; padding: 20px; text-align: center; vertical-align: top;">
                <div style="font-size: 40px; margin-bottom: 10px;">💪</div>
                <h3 style="color: #E85D5D; font-size: 16px; margin: 10px 0;">Durable Prints</h3>
                <p style="color: #666; font-size: 14px; margin: 5px 0;">Long-lasting materials and expert craftsmanship</p>
            </td>
        </tr>
    </table>
</div>

<div style="text-align: center; margin: 40px 0;">
    <table cellpadding="0" cellspacing="0" border="0" align="center">
        <tr>
            <td bgcolor="#E85D5D" style="background-color: #E85D5D; padding: 18px 45px; border-radius: 8px;">
                <a href="https://www.logans3dcreations.com/shop" style="color: white; text-decoration: none; font-weight: 700; font-size: 18px; display: block;">Start Shopping Now →</a>
            </td>
        </tr>
    </table>
    <p style="font-size: 14px; color: #999; margin-top: 15px;">Use code <strong style="color: #E85D5D;">{{.PromoCode}}</strong> at checkout</p>
</div>

<table width="100%" cellpadding="0" cellspacing="0" border="0" bgcolor="#f9f9f9" style="background-color: #f9f9f9; margin: 30px 0;">
    <tr>
        <td style="padding: 25px; border-left: 4px solid #10b981;">
            <h3 style="color: #555; font-size: 18px; margin: 0 0 15px 0;">📝 How to Redeem:</h3>
            <ol style="color: #666; margin: 0; padding-left: 20px; line-height: 1.8;">
                <li>Browse our shop and add items to your cart</li>
                <li>At checkout, enter code <strong>{{.PromoCode}}</strong></li>
                <li>Your discount will be applied automatically</li>
                <li>Complete your order and enjoy!</li>
            </ol>
        </td>
    </tr>
</table>

<div style="text-align: center; margin-top: 35px; padding-top: 25px; border-top: 1px solid #ddd; color: #777; font-size: 14px;">
    <p style="margin: 5px 0;">Questions? We're here to help!</p>
    <p style="margin: 5px 0;">
        <a href="mailto:prints@logans3dcreations.com" style="color: #E85D5D; text-decoration: none; font-weight: 600;">prints@logans3dcreations.com</a>
    </p>
</div>
`
