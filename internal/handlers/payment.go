package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	stripego "github.com/stripe/stripe-go/v80"
	"github.com/stripe/stripe-go/v80/webhook"
	"github.com/loganlanou/logans3d-v4/internal/stripe"
)

type PaymentHandler struct {
	stripeService *stripe.StripeService
}

func NewPaymentHandler() *PaymentHandler {
	return &PaymentHandler{
		stripeService: stripe.NewStripeService(),
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

	event, err := webhook.ConstructEvent(payload, signatureHeader, endpointSecret)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid signature")
	}

	switch event.Type {
	case "payment_intent.succeeded":
		var paymentIntent stripego.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Error parsing webhook JSON")
		}
		// Handle successful payment
		
	case "payment_intent.payment_failed":
		var paymentIntent stripego.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Error parsing webhook JSON")
		}
		// Handle failed payment
		
	default:
		// Unexpected event type
	}

	return c.NoContent(http.StatusOK)
}