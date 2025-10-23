package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	stripego "github.com/stripe/stripe-go/v80"
	"github.com/stripe/stripe-go/v80/webhook"
	"github.com/loganlanou/logans3d-v4/internal/email"
	"github.com/loganlanou/logans3d-v4/internal/stripe"
	"github.com/loganlanou/logans3d-v4/storage/db"
)

type PaymentHandler struct {
	stripeService *stripe.StripeService
	queries       *db.Queries
	emailService  *email.Service
}

func NewPaymentHandler(queries *db.Queries, emailService *email.Service) *PaymentHandler {
	return &PaymentHandler{
		stripeService: stripe.NewStripeService(),
		queries:       queries,
		emailService:  emailService,
	}
}

type CreatePaymentIntentRequest struct {
	Amount     int64  `json:"amount"`
	Currency   string `json:"currency"`
	CustomerID string `json:"customer_id,omitempty"`
}

type CreatePaymentIntentResponse struct {
	ClientSecret string `json:"client_secret"`
	PaymentIntentID string `json:"payment_intent_id"`
}

func (h *PaymentHandler) CreatePaymentIntent(c echo.Context) error {
	var req CreatePaymentIntentRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if req.Currency == "" {
		req.Currency = "usd"
	}

	paymentIntent, err := h.stripeService.CreatePaymentIntent(req.Amount, req.Currency, req.CustomerID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create payment intent")
	}

	response := CreatePaymentIntentResponse{
		ClientSecret:    paymentIntent.ClientSecret,
		PaymentIntentID: paymentIntent.ID,
	}

	return c.JSON(http.StatusOK, response)
}

type CreateCustomerRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

func (h *PaymentHandler) CreateCustomer(c echo.Context) error {
	var req CreateCustomerRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	customer, err := h.stripeService.CreateCustomer(req.Email, req.Name)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create customer")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"customer_id": customer.ID,
		"email":       customer.Email,
		"name":        customer.Name,
	})
}

func (h *PaymentHandler) HandleWebhook(c echo.Context) error {
	payload, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Request body too large")
	}

	endpointSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	signatureHeader := c.Request().Header.Get("Stripe-Signature")

	// Allow webhook processing without signature verification if webhook secret is not configured
	var event stripego.Event
	if endpointSecret != "" {
		event, err = webhook.ConstructEvent(payload, signatureHeader, endpointSecret)
		if err != nil {
			slog.Error("webhook signature verification failed", "error", err)
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid signature")
		}
	} else {
		// For development/testing: parse event without verification
		if err := json.Unmarshal(payload, &event); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Error parsing webhook JSON")
		}
	}

	switch event.Type {
	case "checkout.session.completed":
		var session stripego.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			slog.Error("error parsing checkout session", "error", err)
			return echo.NewHTTPError(http.StatusBadRequest, "Error parsing webhook JSON")
		}

		// Handle successful checkout - create order and send emails
		if err := h.handleCheckoutCompleted(c, &session); err != nil {
			slog.Error("error handling checkout completed", "error", err)
			// Don't return error to Stripe - we'll log it and let them retry
		}

	case "payment_intent.succeeded":
		var paymentIntent stripego.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Error parsing webhook JSON")
		}
		slog.Info("payment intent succeeded", "payment_intent_id", paymentIntent.ID)

	case "payment_intent.payment_failed":
		var paymentIntent stripego.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Error parsing webhook JSON")
		}
		slog.Warn("payment intent failed", "payment_intent_id", paymentIntent.ID)

	default:
		slog.Debug("unhandled webhook event type", "type", event.Type)
	}

	return c.NoContent(http.StatusOK)
}

func (h *PaymentHandler) handleCheckoutCompleted(c echo.Context, session *stripego.CheckoutSession) error {
	ctx := c.Request().Context()

	slog.Info("handling checkout completed",
		"session_id", session.ID,
		"customer_email", session.CustomerDetails.Email,
		"amount_total", session.AmountTotal)

	// Create order in database
	orderID := uuid.New().String()

	// Extract customer and address details
	customerName := session.CustomerDetails.Name
	customerEmail := session.CustomerDetails.Email

	// Get billing and shipping addresses
	billingAddress := session.CustomerDetails.Address
	shippingAddress := session.ShippingDetails.Address
	if shippingAddress == nil {
		shippingAddress = billingAddress // Fallback to billing if no shipping address
	}

	// Calculate amounts (Stripe amounts are in cents)
	totalCents := session.AmountTotal
	subtotalCents := session.AmountSubtotal
	taxCents := int64(0)
	if session.TotalDetails != nil && session.TotalDetails.AmountTax != 0 {
		taxCents = session.TotalDetails.AmountTax
	}
	shippingCents := int64(0)
	if session.TotalDetails != nil && session.TotalDetails.AmountShipping != 0 {
		shippingCents = session.TotalDetails.AmountShipping
	}

	// Create order
	_, err = h.queries.CreateOrder(ctx, db.CreateOrderParams{
		ID:                    orderID,
		UserID:                sql.NullString{}, // Optional - may be guest checkout
		CustomerEmail:         customerEmail,
		CustomerName:          customerName,
		CustomerPhone:         sql.NullString{String: session.CustomerDetails.Phone, Valid: session.CustomerDetails.Phone != ""},
		BillingAddressLine1:   billingAddress.Line1,
		BillingAddressLine2:   sql.NullString{String: billingAddress.Line2, Valid: billingAddress.Line2 != ""},
		BillingCity:           billingAddress.City,
		BillingState:          billingAddress.State,
		BillingPostalCode:     billingAddress.PostalCode,
		BillingCountry:        billingAddress.Country,
		ShippingAddressLine1:  shippingAddress.Line1,
		ShippingAddressLine2:  sql.NullString{String: shippingAddress.Line2, Valid: shippingAddress.Line2 != ""},
		ShippingCity:          shippingAddress.City,
		ShippingState:         shippingAddress.State,
		ShippingPostalCode:    shippingAddress.PostalCode,
		ShippingCountry:       shippingAddress.Country,
		SubtotalCents:         subtotalCents,
		TaxCents:              taxCents,
		ShippingCents:         shippingCents,
		TotalCents:            totalCents,
		StripePaymentIntentID: sql.NullString{String: session.PaymentIntent.ID, Valid: true},
		StripeCustomerID:      sql.NullString{String: session.Customer.ID, Valid: true},
		Status:                sql.NullString{String: "confirmed", Valid: true},
		FulfillmentStatus:     sql.NullString{String: "pending", Valid: true},
		PaymentStatus:         sql.NullString{String: "paid", Valid: true},
		Notes:                 sql.NullString{},
	})
	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	slog.Info("order created successfully", "order_id", orderID)

	// Get line items from session (need to expand)
	orderItems := []email.OrderItem{}
	if session.LineItems != nil {
		for _, item := range session.LineItems.Data {
			orderItems = append(orderItems, email.OrderItem{
				ProductName:  item.Description,
				Quantity:     item.Quantity,
				PriceCents:   item.Price.UnitAmount,
				TotalCents:   item.AmountTotal,
			})

			// Create order item in database
			_, err := h.queries.CreateOrderItem(ctx, db.CreateOrderItemParams{
				ID:          uuid.New().String(),
				OrderID:     orderID,
				ProductID:   sql.NullString{}, // May need to parse from metadata
				ProductName: item.Description,
				Quantity:    item.Quantity,
				PriceCents:  item.Price.UnitAmount,
				TotalCents:  item.AmountTotal,
			})
			if err != nil {
				slog.Error("failed to create order item", "error", err)
			}
		}
	}

	// Prepare email data
	emailData := &email.OrderData{
		OrderID:       orderID,
		CustomerName:  customerName,
		CustomerEmail: customerEmail,
		OrderDate:     time.Now().Format("January 2, 2006 at 3:04 PM"),
		Items:         orderItems,
		SubtotalCents: subtotalCents,
		TaxCents:      taxCents,
		ShippingCents: shippingCents,
		TotalCents:    totalCents,
		ShippingAddress: email.Address{
			Name:       customerName,
			Line1:      shippingAddress.Line1,
			Line2:      shippingAddress.Line2,
			City:       shippingAddress.City,
			State:      shippingAddress.State,
			PostalCode: shippingAddress.PostalCode,
			Country:    shippingAddress.Country,
		},
		BillingAddress: email.Address{
			Name:       customerName,
			Line1:      billingAddress.Line1,
			Line2:      billingAddress.Line2,
			City:       billingAddress.City,
			State:      billingAddress.State,
			PostalCode: billingAddress.PostalCode,
			Country:    billingAddress.Country,
		},
		PaymentIntentID: session.PaymentIntent.ID,
	}

	// Send customer confirmation email
	if err := h.emailService.SendOrderConfirmation(emailData); err != nil {
		slog.Error("failed to send customer confirmation email", "error", err, "order_id", orderID)
		// Don't fail the webhook if email fails
	} else {
		slog.Info("customer confirmation email sent", "order_id", orderID, "email", customerEmail)
	}

	// Send admin notification email
	if err := h.emailService.SendOrderNotificationToAdmin(emailData); err != nil {
		slog.Error("failed to send admin notification email", "error", err, "order_id", orderID)
		// Don't fail the webhook if email fails
	} else {
		slog.Info("admin notification email sent", "order_id", orderID)
	}

	return nil
}