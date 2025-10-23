package email

// customerOrderTemplate is the HTML email template sent to customers
const customerOrderTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Order Confirmation</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            background-color: white;
            padding: 30px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .header {
            text-align: center;
            border-bottom: 3px solid #4CAF50;
            padding-bottom: 20px;
            margin-bottom: 30px;
        }
        .header h1 {
            color: #4CAF50;
            margin: 0;
            font-size: 28px;
        }
        .order-info {
            background-color: #f9f9f9;
            padding: 15px;
            border-radius: 5px;
            margin-bottom: 25px;
        }
        .order-info p {
            margin: 5px 0;
        }
        .order-info strong {
            color: #555;
        }
        .items-table {
            width: 100%;
            border-collapse: collapse;
            margin: 20px 0;
        }
        .items-table th {
            background-color: #4CAF50;
            color: white;
            padding: 12px;
            text-align: left;
            font-weight: 600;
        }
        .items-table td {
            padding: 12px;
            border-bottom: 1px solid #ddd;
        }
        .items-table tr:last-child td {
            border-bottom: none;
        }
        .totals {
            margin-top: 20px;
            padding-top: 20px;
            border-top: 2px solid #ddd;
        }
        .totals-row {
            display: flex;
            justify-content: space-between;
            padding: 8px 0;
        }
        .totals-row.total {
            font-size: 18px;
            font-weight: bold;
            color: #4CAF50;
            border-top: 2px solid #4CAF50;
            margin-top: 10px;
            padding-top: 15px;
        }
        .address-section {
            margin: 25px 0;
            padding: 15px;
            background-color: #f9f9f9;
            border-radius: 5px;
        }
        .address-section h3 {
            margin-top: 0;
            color: #555;
            font-size: 16px;
        }
        .footer {
            margin-top: 30px;
            padding-top: 20px;
            border-top: 1px solid #ddd;
            text-align: center;
            color: #777;
            font-size: 14px;
        }
        .button {
            display: inline-block;
            padding: 12px 30px;
            background-color: #4CAF50;
            color: white;
            text-decoration: none;
            border-radius: 5px;
            margin: 20px 0;
            font-weight: 600;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Order Confirmation</h1>
            <p>Thank you for your order, {{.CustomerName}}!</p>
        </div>

        <div class="order-info">
            <p><strong>Order Number:</strong> #{{.OrderID}}</p>
            <p><strong>Order Date:</strong> {{.OrderDate}}</p>
            <p><strong>Customer Email:</strong> {{.CustomerEmail}}</p>
        </div>

        <h2 style="color: #555;">Order Items</h2>
        <table class="items-table">
            <thead>
                <tr>
                    <th>Product</th>
                    <th style="text-align: center;">Quantity</th>
                    <th style="text-align: right;">Price</th>
                    <th style="text-align: right;">Total</th>
                </tr>
            </thead>
            <tbody>
                {{range .Items}}
                <tr>
                    <td>{{.ProductName}}</td>
                    <td style="text-align: center;">{{.Quantity}}</td>
                    <td style="text-align: right;">{{FormatCents .PriceCents}}</td>
                    <td style="text-align: right;">{{FormatCents .TotalCents}}</td>
                </tr>
                {{end}}
            </tbody>
        </table>

        <div class="totals">
            <div class="totals-row">
                <span>Subtotal:</span>
                <span>{{FormatCents .SubtotalCents}}</span>
            </div>
            <div class="totals-row">
                <span>Tax:</span>
                <span>{{FormatCents .TaxCents}}</span>
            </div>
            <div class="totals-row">
                <span>Shipping:</span>
                <span>{{FormatCents .ShippingCents}}</span>
            </div>
            <div class="totals-row total">
                <span>Total:</span>
                <span>{{FormatCents .TotalCents}}</span>
            </div>
        </div>

        <div class="address-section">
            <h3>Shipping Address</h3>
            <p>{{.ShippingAddress.Name}}<br>
            {{.ShippingAddress.Line1}}<br>
            {{if .ShippingAddress.Line2}}{{.ShippingAddress.Line2}}<br>{{end}}
            {{.ShippingAddress.City}}, {{.ShippingAddress.State}} {{.ShippingAddress.PostalCode}}<br>
            {{.ShippingAddress.Country}}</p>
        </div>

        <div style="text-align: center;">
            <a href="http://localhost:8000" class="button">Visit Our Store</a>
        </div>

        <div class="footer">
            <p><strong>Logan's 3D Creations</strong></p>
            <p>If you have any questions about your order, please contact us at prints@logans3dcreations.com</p>
            <p style="font-size: 12px; margin-top: 20px;">Payment Intent ID: {{.PaymentIntentID}}</p>
        </div>
    </div>
</body>
</html>
`

// adminOrderTemplate is the HTML email template sent to admins/internal staff
const adminOrderTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>New Order Notification</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 700px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            background-color: white;
            padding: 30px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .header {
            text-align: center;
            border-bottom: 3px solid #FF9800;
            padding-bottom: 20px;
            margin-bottom: 30px;
        }
        .header h1 {
            color: #FF9800;
            margin: 0;
            font-size: 28px;
        }
        .alert-badge {
            display: inline-block;
            background-color: #FF9800;
            color: white;
            padding: 5px 15px;
            border-radius: 20px;
            font-weight: 600;
            font-size: 14px;
            margin-bottom: 10px;
        }
        .order-info {
            background-color: #fff3e0;
            padding: 20px;
            border-left: 4px solid #FF9800;
            margin-bottom: 25px;
        }
        .order-info p {
            margin: 8px 0;
        }
        .order-info strong {
            color: #e65100;
            min-width: 150px;
            display: inline-block;
        }
        .items-table {
            width: 100%;
            border-collapse: collapse;
            margin: 20px 0;
        }
        .items-table th {
            background-color: #FF9800;
            color: white;
            padding: 12px;
            text-align: left;
            font-weight: 600;
        }
        .items-table td {
            padding: 12px;
            border-bottom: 1px solid #ddd;
        }
        .items-table tr:hover {
            background-color: #f9f9f9;
        }
        .totals {
            margin-top: 20px;
            padding: 20px;
            background-color: #f9f9f9;
            border-radius: 5px;
        }
        .totals-row {
            display: flex;
            justify-content: space-between;
            padding: 8px 0;
            font-size: 16px;
        }
        .totals-row.total {
            font-size: 20px;
            font-weight: bold;
            color: #FF9800;
            border-top: 2px solid #FF9800;
            margin-top: 10px;
            padding-top: 15px;
        }
        .address-grid {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 20px;
            margin: 25px 0;
        }
        .address-section {
            padding: 15px;
            background-color: #f9f9f9;
            border-radius: 5px;
            border: 1px solid #ddd;
        }
        .address-section h3 {
            margin-top: 0;
            color: #e65100;
            font-size: 16px;
            border-bottom: 2px solid #FF9800;
            padding-bottom: 8px;
        }
        .action-button {
            display: inline-block;
            padding: 12px 30px;
            background-color: #FF9800;
            color: white;
            text-decoration: none;
            border-radius: 5px;
            margin: 20px 10px;
            font-weight: 600;
        }
        .action-buttons {
            text-align: center;
            margin: 30px 0;
        }
        .footer {
            margin-top: 30px;
            padding-top: 20px;
            border-top: 1px solid #ddd;
            color: #777;
            font-size: 13px;
        }
        .customer-info {
            background-color: #e3f2fd;
            padding: 15px;
            border-radius: 5px;
            margin: 20px 0;
            border-left: 4px solid #2196F3;
        }
        @media only screen and (max-width: 600px) {
            .address-grid {
                grid-template-columns: 1fr;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <span class="alert-badge">NEW ORDER</span>
            <h1>Order Received</h1>
            <p>Action required - Process this order</p>
        </div>

        <div class="order-info">
            <p><strong>Order Number:</strong> #{{.OrderID}}</p>
            <p><strong>Order Date:</strong> {{.OrderDate}}</p>
            <p><strong>Payment Intent:</strong> {{.PaymentIntentID}}</p>
            <p><strong>Order Total:</strong> {{FormatCents .TotalCents}}</p>
        </div>

        <div class="customer-info">
            <h3 style="margin-top: 0; color: #1976D2;">Customer Information</h3>
            <p><strong>Name:</strong> {{.CustomerName}}</p>
            <p><strong>Email:</strong> <a href="mailto:{{.CustomerEmail}}">{{.CustomerEmail}}</a></p>
        </div>

        <h2 style="color: #555;">Order Items</h2>
        <table class="items-table">
            <thead>
                <tr>
                    <th>Product</th>
                    <th style="text-align: center;">Quantity</th>
                    <th style="text-align: right;">Unit Price</th>
                    <th style="text-align: right;">Total</th>
                </tr>
            </thead>
            <tbody>
                {{range .Items}}
                <tr>
                    <td><strong>{{.ProductName}}</strong></td>
                    <td style="text-align: center;">{{.Quantity}}</td>
                    <td style="text-align: right;">{{FormatCents .PriceCents}}</td>
                    <td style="text-align: right;">{{FormatCents .TotalCents}}</td>
                </tr>
                {{end}}
            </tbody>
        </table>

        <div class="totals">
            <div class="totals-row">
                <span>Subtotal:</span>
                <span>{{FormatCents .SubtotalCents}}</span>
            </div>
            <div class="totals-row">
                <span>Tax:</span>
                <span>{{FormatCents .TaxCents}}</span>
            </div>
            <div class="totals-row">
                <span>Shipping:</span>
                <span>{{FormatCents .ShippingCents}}</span>
            </div>
            <div class="totals-row total">
                <span>TOTAL:</span>
                <span>{{FormatCents .TotalCents}}</span>
            </div>
        </div>

        <h2 style="color: #555; margin-top: 30px;">Shipping & Billing Details</h2>
        <div class="address-grid">
            <div class="address-section">
                <h3>Ship To</h3>
                <p><strong>{{.ShippingAddress.Name}}</strong><br>
                {{.ShippingAddress.Line1}}<br>
                {{if .ShippingAddress.Line2}}{{.ShippingAddress.Line2}}<br>{{end}}
                {{.ShippingAddress.City}}, {{.ShippingAddress.State}} {{.ShippingAddress.PostalCode}}<br>
                {{.ShippingAddress.Country}}</p>
            </div>

            <div class="address-section">
                <h3>Bill To</h3>
                <p><strong>{{.BillingAddress.Name}}</strong><br>
                {{.BillingAddress.Line1}}<br>
                {{if .BillingAddress.Line2}}{{.BillingAddress.Line2}}<br>{{end}}
                {{.BillingAddress.City}}, {{.BillingAddress.State}} {{.BillingAddress.PostalCode}}<br>
                {{.BillingAddress.Country}}</p>
            </div>
        </div>

        <div class="action-buttons">
            <a href="http://localhost:8000/admin/orders" class="action-button">View Order</a>
            <a href="http://localhost:8000/admin/orders" class="action-button">Process Order</a>
        </div>

        <div class="footer">
            <p><strong>Next Steps:</strong></p>
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
    </div>
</body>
</html>
`
