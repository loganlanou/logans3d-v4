package handlers

import (
	"context"
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
	checkoutsession "github.com/stripe/stripe-go/v80/checkout/session"
	promotioncode "github.com/stripe/stripe-go/v80/promotioncode"
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

	// Debug logging for session data
	slog.Debug("session data check",
		"session_id", session.ID,
		"has_line_items", session.LineItems != nil,
		"has_total_details", session.TotalDetails != nil)

	if session.LineItems != nil {
		slog.Debug("line items present", "count", len(session.LineItems.Data))
	}

	if session.TotalDetails != nil {
		slog.Debug("total details present",
			"amount_discount", session.TotalDetails.AmountDiscount,
			"has_breakdown", session.TotalDetails.Breakdown != nil)
	}

	// If breakdown is missing and there's a discount, re-fetch session with expanded details
	if session.TotalDetails != nil && session.TotalDetails.AmountDiscount > 0 && session.TotalDetails.Breakdown == nil {
		slog.Debug("re-fetching session to get discount breakdown", "session_id", session.ID)
		stripego.Key = os.Getenv("STRIPE_SECRET_KEY")
		params := &stripego.CheckoutSessionParams{}
		params.AddExpand("total_details.breakdown")
		params.AddExpand("line_items")
		params.AddExpand("line_items.data.price.product")
		expandedSession, err := checkoutsession.Get(session.ID, params)
		if err != nil {
			slog.Warn("failed to re-fetch session for breakdown", "error", err, "session_id", session.ID)
			// Continue with original session - we'll still have the discount amount
		} else {
			session = expandedSession
			slog.Debug("successfully retrieved discount breakdown",
				"session_id", session.ID,
				"has_breakdown", session.TotalDetails != nil && session.TotalDetails.Breakdown != nil)
		}
	}

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

	// Get discount amount from Stripe
	discountCents := int64(0)
	promotionCode := sql.NullString{}
	promotionCodeID := sql.NullString{}

	if session.TotalDetails != nil && session.TotalDetails.AmountDiscount != 0 {
		discountCents = session.TotalDetails.AmountDiscount
		slog.Info("discount applied to order", "discount_cents", discountCents, "order_id", orderID)

		// Try to get the promotion code from the session
		// The session object may have discount information in the discounts array
		if session.TotalDetails.Breakdown != nil && len(session.TotalDetails.Breakdown.Discounts) > 0 {
			// Get the first discount's promotion code
			firstDiscount := session.TotalDetails.Breakdown.Discounts[0]
			slog.Debug("discount breakdown structure",
				"has_discount", firstDiscount.Discount != nil,
				"has_promo_code", firstDiscount.Discount != nil && firstDiscount.Discount.PromotionCode != nil,
				"discount_id", firstDiscount.Discount.ID,
				"order_id", orderID)

			if firstDiscount.Discount != nil && firstDiscount.Discount.PromotionCode != nil {
				promoCodeObj := firstDiscount.Discount.PromotionCode
				slog.Debug("promotion code object details",
					"code", promoCodeObj.Code,
					"id", promoCodeObj.ID,
					"active", promoCodeObj.Active,
					"order_id", orderID)

				code := promoCodeObj.Code

				// Stripe expansion depth limit means we only get the ID, not the full object
				// If code is empty but we have an ID, retrieve the full promotion code
				if code == "" && promoCodeObj.ID != "" {
					slog.Debug("promotion code is empty, retrieving full object by ID",
						"promo_code_id", promoCodeObj.ID,
						"order_id", orderID)

					stripego.Key = os.Getenv("STRIPE_SECRET_KEY")
					fullPromoCode, err := promotioncode.Get(promoCodeObj.ID, nil)
					if err != nil {
						slog.Error("failed to retrieve promotion code details",
							"error", err,
							"promo_code_id", promoCodeObj.ID,
							"order_id", orderID)
					} else {
						code = fullPromoCode.Code
						slog.Debug("retrieved full promotion code",
							"code", code,
							"promo_code_id", fullPromoCode.ID,
							"order_id", orderID)
					}
				}

				// Only proceed if we successfully got a code
				if code != "" {
					promotionCode = sql.NullString{String: code, Valid: true}
					slog.Debug("promotion code found", "code", code, "order_id", orderID)
				} else {
					slog.Warn("promotion code ID present but code string is empty",
						"promo_code_id", promoCodeObj.ID,
						"order_id", orderID)
				}

				// Only try to create/link the code if we have a non-empty code string
				if code != "" {
					// Try to find this code in our promotion_codes table
					promoRecord, err := h.queries.GetPromotionCodeByCode(ctx, code)
					if err == nil {
						// Code exists in our database
						promotionCodeID = sql.NullString{String: promoRecord.ID, Valid: true}
						slog.Debug("promotion code matched to database", "promotion_code_id", promoRecord.ID, "order_id", orderID)
					} else if err == sql.ErrNoRows {
						// Code doesn't exist in our database - create it
						slog.Debug("promotion code not found in database, creating external promotion code", "code", code, "order_id", orderID)
						promoRecord, createErr := h.createExternalPromotionCode(ctx, code, firstDiscount)
						if createErr != nil {
							slog.Error("failed to create external promotion code", "error", createErr, "code", code, "order_id", orderID)
							// Don't fail the order - just log and continue with NULL promotion_code_id
						} else {
							promotionCodeID = sql.NullString{String: promoRecord.ID, Valid: true}
							slog.Debug("created and linked external promotion code", "promotion_code_id", promoRecord.ID, "code", code, "order_id", orderID)
						}
					} else {
						// Some other error occurred during lookup
						slog.Warn("failed to lookup promotion code in database", "error", err, "code", code)
					}
				}
			}
		}
	}

	// Calculate subtotal excluding shipping (since we track shipping separately)
	// This is the discounted subtotal
	subtotalCents := totalCents - taxCents - shippingCents

	// Calculate original subtotal (before discount)
	originalSubtotalCents := subtotalCents + discountCents

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
		ID:                      orderID,
		UserID:                  userID, // Set from metadata if user was authenticated
		CustomerEmail:           customerEmail,
		CustomerName:            customerName,
		CustomerPhone:           sql.NullString{String: session.CustomerDetails.Phone, Valid: session.CustomerDetails.Phone != ""},
		ShippingAddressLine1:    shippingAddress.Line1,
		ShippingAddressLine2:    sql.NullString{String: shippingAddress.Line2, Valid: shippingAddress.Line2 != ""},
		ShippingCity:            shippingAddress.City,
		ShippingState:           shippingAddress.State,
		ShippingPostalCode:      shippingAddress.PostalCode,
		ShippingCountry:         shippingAddress.Country,
		SubtotalCents:           subtotalCents,
		TaxCents:                taxCents,
		ShippingCents:           shippingCents,
		TotalCents:              totalCents,
		OriginalSubtotalCents:   sql.NullInt64{Int64: originalSubtotalCents, Valid: true},
		DiscountCents:           sql.NullInt64{Int64: discountCents, Valid: discountCents > 0},
		PromotionCode:           promotionCode,
		PromotionCodeID:         promotionCodeID,
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
	} else {
		slog.Warn("session.LineItems is nil - no order items will be created", "session_id", session.ID)
	}

	slog.Debug("order items processed", "order_id", orderID, "item_count", len(orderItems))

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

// getOrCreateCampaignForDiscount gets or creates a campaign for an external Stripe discount
func (h *PaymentHandler) getOrCreateCampaignForDiscount(ctx context.Context, discount *stripego.CheckoutSessionTotalDetailsBreakdownDiscount) (*db.PromotionCampaign, error) {
	if discount == nil || discount.Discount == nil || discount.Discount.Coupon == nil {
		return nil, fmt.Errorf("invalid discount structure")
	}

	coupon := discount.Discount.Coupon

	// Determine discount type and value
	var discountType string
	var discountValue int64
	var campaignName string

	if coupon.PercentOff > 0 {
		discountType = "percentage"
		discountValue = int64(coupon.PercentOff)
		campaignName = fmt.Sprintf("External Stripe - %.0f%% Off", coupon.PercentOff)
	} else if coupon.AmountOff > 0 {
		discountType = "amount"
		discountValue = coupon.AmountOff
		campaignName = fmt.Sprintf("External Stripe - $%.2f Off", float64(coupon.AmountOff)/100)
	} else {
		discountType = "percentage"
		discountValue = 0
		campaignName = "External Stripe - Variable Discount"
	}

	// Try to find existing campaign by name
	campaign, err := h.queries.GetPromotionCampaignByName(ctx, campaignName)
	if err == nil {
		slog.Debug("found existing campaign for external discount", "campaign_id", campaign.ID, "campaign_name", campaignName)
		return &campaign, nil
	}

	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to lookup campaign: %w", err)
	}

	// Campaign doesn't exist - create it
	slog.Debug("creating new campaign for external discount", "campaign_name", campaignName, "discount_type", discountType, "discount_value", discountValue)

	campaign, err = h.queries.CreatePromotionCampaign(ctx, db.CreatePromotionCampaignParams{
		ID:                uuid.New().String(),
		Name:              campaignName,
		Description:       sql.NullString{String: "Promotion codes created externally in Stripe", Valid: true},
		DiscountType:      discountType,
		DiscountValue:     discountValue,
		StripePromotionID: sql.NullString{String: coupon.ID, Valid: true},
		StartDate:         time.Now(),
		EndDate:           sql.NullTime{},
		MaxUses:           sql.NullInt64{},
		Active:            sql.NullInt64{Int64: 1, Valid: true},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create campaign: %w", err)
	}

	slog.Debug("created new campaign for external discount", "campaign_id", campaign.ID, "campaign_name", campaignName)
	return &campaign, nil
}

// createExternalPromotionCode creates a promotion code record for an external Stripe promotion code
func (h *PaymentHandler) createExternalPromotionCode(ctx context.Context, code string, discount *stripego.CheckoutSessionTotalDetailsBreakdownDiscount) (*db.PromotionCode, error) {
	// Get or create the appropriate campaign for this discount
	campaign, err := h.getOrCreateCampaignForDiscount(ctx, discount)
	if err != nil {
		return nil, fmt.Errorf("failed to get/create campaign: %w", err)
	}

	// Extract Stripe promotion code ID if available
	stripePromoCodeID := ""
	if discount.Discount != nil && discount.Discount.PromotionCode != nil {
		stripePromoCodeID = discount.Discount.PromotionCode.ID
	}

	slog.Info("creating external promotion code", "code", code, "campaign_id", campaign.ID, "stripe_promo_code_id", stripePromoCodeID)

	// Create the promotion code
	promoCode, err := h.queries.CreatePromotionCode(ctx, db.CreatePromotionCodeParams{
		ID:                    uuid.New().String(),
		CampaignID:            campaign.ID,
		Code:                  code,
		StripePromotionCodeID: sql.NullString{String: stripePromoCodeID, Valid: stripePromoCodeID != ""},
		Email:                 sql.NullString{},
		UserID:                sql.NullString{},
		MaxUses:               sql.NullInt64{},
		ExpiresAt:             sql.NullTime{},
	})

	if err != nil {
		// Check if this is a unique constraint violation (race condition)
		if err.Error() == "UNIQUE constraint failed: promotion_codes.code" {
			slog.Info("promotion code already exists (race condition), fetching existing record", "code", code)
			// Another request created this code - fetch it
			existingCode, fetchErr := h.queries.GetPromotionCodeByCode(ctx, code)
			if fetchErr != nil {
				return nil, fmt.Errorf("failed to fetch existing promotion code after race condition: %w", fetchErr)
			}
			return &existingCode, nil
		}
		return nil, fmt.Errorf("failed to create promotion code: %w", err)
	}

	slog.Info("created external promotion code", "code", code, "promotion_code_id", promoCode.ID, "campaign_id", campaign.ID)
	return &promoCode, nil
}