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
			slog.Error("error handling checkout completed", "error", err, "session_id", session.ID)
			// Return error to Stripe so they retry the webhook
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process checkout")
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

// HandleCheckoutCompleted is a public wrapper for order creation that can be called from webhooks or success pages
func (h *PaymentHandler) HandleCheckoutCompleted(c echo.Context, session *stripego.CheckoutSession) error {
	return h.handleCheckoutCompleted(c, session)
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

	// Get user ID and session ID from metadata if present
	userID := ""
	sessionID := ""
	if session.Metadata != nil {
		slog.Info("stripe metadata received", "metadata", session.Metadata)
		if uid, ok := session.Metadata["user_id"]; ok && uid != "" {
			userID = uid
			slog.Info("order linked to user", "user_id", uid, "order_id", orderID)
		} else {
			slog.Warn("user_id not found in stripe metadata", "order_id", orderID)
		}
		if sid, ok := session.Metadata["session_id"]; ok && sid != "" {
			sessionID = sid
			slog.Info("order linked to session", "session_id", sid, "order_id", orderID)
		}
	} else {
		slog.Warn("stripe session has no metadata", "order_id", orderID)
	}

	// Try to get EasyPost shipment ID and shipping costs from session shipping selection
	easypostShipmentID := sql.NullString{}
	var sessionShippingSelection db.SessionShippingSelection
	shippingCents := int64(0)
	hasShippingSelection := false

	if sessionID != "" {
		shippingSelection, err := h.queries.GetSessionShippingSelection(ctx, sessionID)
		if err == nil && shippingSelection.ShipmentID != "" {
			sessionShippingSelection = shippingSelection
			hasShippingSelection = true
			easypostShipmentID = sql.NullString{String: shippingSelection.ShipmentID, Valid: true}
			shippingCents = shippingSelection.PriceCents
			slog.Info("order linked to EasyPost shipment",
				"shipment_id", shippingSelection.ShipmentID,
				"order_id", orderID,
				"carrier", shippingSelection.CarrierName,
				"service", shippingSelection.ServiceName,
				"shipping_cents", shippingCents)
		} else if err != nil && err != sql.ErrNoRows {
			slog.Warn("failed to get session shipping selection", "error", err, "session_id", sessionID)
		}
	}

	// Get billing and shipping addresses
	billingAddress := session.CustomerDetails.Address
	shippingAddress := session.ShippingDetails.Address
	if shippingAddress == nil {
		shippingAddress = billingAddress // Fallback to billing if no shipping address
	}

	// Calculate amounts (Stripe amounts are in cents)
	totalCents := session.AmountTotal
	taxCents := int64(0)
	if session.TotalDetails != nil && session.TotalDetails.AmountTax != 0 {
		taxCents = session.TotalDetails.AmountTax
	}

	// Calculate subtotal excluding shipping (since we track shipping separately)
	subtotalCents := totalCents - taxCents - shippingCents

	// Check if order already exists (idempotency)
	existing, existErr := h.queries.GetOrderByStripeSessionID(ctx, sql.NullString{String: session.ID, Valid: true})
	if existErr == nil {
		slog.Info("order already exists for this checkout session", "order_id", existing.ID, "session_id", session.ID)
		return nil // Order already created, skip
	} else if existErr != sql.ErrNoRows {
		return fmt.Errorf("failed to check existing order: %w", existErr)
	}

	// Create order
	_, createErr := h.queries.CreateOrder(ctx, db.CreateOrderParams{
		ID:                   orderID,
		UserID:               userID, // Set from metadata if user was authenticated
		CustomerEmail:        customerEmail,
		CustomerName:         customerName,
		CustomerPhone:        sql.NullString{String: session.CustomerDetails.Phone, Valid: session.CustomerDetails.Phone != ""},
		ShippingAddressLine1: shippingAddress.Line1,
		ShippingAddressLine2: sql.NullString{String: shippingAddress.Line2, Valid: shippingAddress.Line2 != ""},
		ShippingCity:         shippingAddress.City,
		ShippingState:        shippingAddress.State,
		ShippingPostalCode:   shippingAddress.PostalCode,
		ShippingCountry:      shippingAddress.Country,
		SubtotalCents:        subtotalCents,
		TaxCents:             taxCents,
		ShippingCents:        shippingCents,
		TotalCents:           totalCents,
		StripePaymentIntentID:   sql.NullString{String: session.PaymentIntent.ID, Valid: true},
		StripeCustomerID:        sql.NullString{String: session.Customer.ID, Valid: true},
		StripeCheckoutSessionID: sql.NullString{String: session.ID, Valid: true},
		EasypostShipmentID:      easypostShipmentID,
		Status:                  sql.NullString{String: "received", Valid: true},
		Notes:                   sql.NullString{},
	})
	if createErr != nil {
		return fmt.Errorf("failed to create order: %w", createErr)
	}

	slog.Info("order created successfully", "order_id", orderID)

	// Create order_shipping_selection record if we have shipping data
	if hasShippingSelection {
		_, shippingSelErr := h.queries.CreateOrderShippingSelection(ctx, db.CreateOrderShippingSelectionParams{
			ID:                          uuid.New().String(),
			OrderID:                     orderID,
			CandidateBoxSku:             sessionShippingSelection.BoxSku,
			RateID:                      sessionShippingSelection.RateID,
			CarrierID:                   sessionShippingSelection.CarrierName,
			ServiceCode:                 sessionShippingSelection.ServiceName,
			ServiceName:                 sessionShippingSelection.ServiceName,
			QuotedShippingAmountCents:   sessionShippingSelection.ShippingAmountCents,
			QuotedBoxCostCents:          sessionShippingSelection.BoxCostCents,
			QuotedHandlingCostCents:     sessionShippingSelection.HandlingCostCents,
			QuotedTotalCents:            sessionShippingSelection.PriceCents,
			DeliveryDays:                sessionShippingSelection.DeliveryDays,
			EstimatedDeliveryDate:       sessionShippingSelection.EstimatedDate,
			PackingSolutionJson:         sql.NullString{String: "{}", Valid: true}, // Could parse from shipping_address_json if needed
			ShipmentID:                  sql.NullString{String: sessionShippingSelection.ShipmentID, Valid: true},
		})
		if shippingSelErr != nil {
			slog.Error("failed to create order shipping selection", "error", shippingSelErr, "order_id", orderID)
			return fmt.Errorf("failed to create order shipping selection: %w", shippingSelErr)
		}
		slog.Info("order shipping selection created", "order_id", orderID)
	}

	// Clear cart for this session/user after successful order creation
	if sessionID != "" || userID != "" {
		if err := h.queries.ClearCart(ctx, db.ClearCartParams{
			SessionID: sql.NullString{String: sessionID, Valid: sessionID != ""},
			UserID:    sql.NullString{String: userID, Valid: userID != ""},
		}); err != nil {
			slog.Error("failed to clear cart after order creation", "error", err, "session_id", sessionID, "user_id", userID)
		} else {
			slog.Info("cart cleared after successful checkout", "session_id", sessionID, "user_id", userID, "order_id", orderID)
		}
	}

	// Get line items from session (need to expand)
	orderItems := []email.OrderItem{}
	if session.LineItems != nil {
		for _, item := range session.LineItems.Data {
			// Skip shipping line items
			if item.Price != nil && item.Price.Product != nil && item.Price.Product.Metadata != nil {
				productID, hasProductID := item.Price.Product.Metadata["product_id"]

				// Only create order item if it's a real product (not shipping)
				if hasProductID && productID != "" {
					// Calculate item total (excluding tax - Stripe's AmountTotal includes tax)
					itemTotal := item.Price.UnitAmount * item.Quantity

					orderItems = append(orderItems, email.OrderItem{
						ProductName:  item.Description,
						Quantity:     item.Quantity,
						PriceCents:   item.Price.UnitAmount,
						TotalCents:   itemTotal,
					})

					// Create order item in database - CRITICAL: Must succeed or order is corrupt
					_, itemErr := h.queries.CreateOrderItem(ctx, db.CreateOrderItemParams{
						ID:               uuid.New().String(),
						OrderID:          orderID,
						ProductID:        productID,
						ProductVariantID: sql.NullString{},
						Quantity:         item.Quantity,
						UnitPriceCents:   item.Price.UnitAmount,
						TotalPriceCents:  itemTotal,
						ProductName:      item.Description,
						ProductSku:       sql.NullString{},
					})
					if itemErr != nil {
						slog.Error("failed to create order item", "error", itemErr, "product_id", productID, "order_id", orderID)
						return fmt.Errorf("failed to create order item for product %s: %w", productID, itemErr)
					}
				}
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