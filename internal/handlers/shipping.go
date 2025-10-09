package handlers

import (
	"database/sql"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/internal/shipping"
	"github.com/loganlanou/logans3d-v4/storage/db"
)

type ShippingHandler struct {
	queries         *db.Queries
	shippingService *shipping.ShippingService
}

func NewShippingHandler(queries *db.Queries, shippingService *shipping.ShippingService) *ShippingHandler {
	return &ShippingHandler{
		queries:         queries,
		shippingService: shippingService,
	}
}

type GetShippingRatesRequest struct {
	ShipTo shipping.Address `json:"ship_to"`
}

type GetShippingRatesResponse struct {
	Options       []shipping.ShippingOption `json:"options"`
	DefaultOption *shipping.ShippingOption  `json:"default_option,omitempty"`
	Error         string                    `json:"error,omitempty"`
}

func (h *ShippingHandler) GetShippingRates(c echo.Context) error {
	var req GetShippingRatesRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if req.ShipTo.PostalCode == "" || req.ShipTo.CountryCode == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Shipping address is required")
	}

	// Get session ID from cookie
	sessionID, err := h.getSessionID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get session")
	}

	counts, err := h.getCartItemCounts(c, sessionID, "")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get cart items")
	}

	shippingReq := &shipping.ShippingQuoteRequest{
		ItemCounts: *counts,
		ShipTo:     req.ShipTo,
	}

	quote, err := h.shippingService.GetShippingQuote(shippingReq)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get shipping rates")
	}

	response := GetShippingRatesResponse{
		Options:       quote.Options,
		DefaultOption: quote.DefaultOption,
		Error:         quote.Error,
	}

	return c.JSON(http.StatusOK, response)
}

// getSessionID extracts session ID from cookie
func (h *ShippingHandler) getSessionID(c echo.Context) (string, error) {
	cookie, err := c.Cookie("session_id")
	if err != nil || cookie.Value == "" {
		return "", echo.NewHTTPError(http.StatusBadRequest, "No session found")
	}
	return cookie.Value, nil
}

func (h *ShippingHandler) getCartItemCounts(c echo.Context, sessionID, userID string) (*shipping.ItemCounts, error) {
	counts, err := h.queries.CountCartItemsByShippingCategory(c.Request().Context(), db.CountCartItemsByShippingCategoryParams{
		SessionID: sql.NullString{String: sessionID, Valid: sessionID != ""},
		UserID:    sql.NullString{String: userID, Valid: userID != ""},
	})
	if err != nil {
		return nil, err
	}

	var small, medium, large, xl int
	if counts.SmallItems.Valid {
		small = int(counts.SmallItems.Float64)
	}
	if counts.MediumItems.Valid {
		medium = int(counts.MediumItems.Float64)
	}
	if counts.LargeItems.Valid {
		large = int(counts.LargeItems.Float64)
	}
	if counts.XlargeItems.Valid {
		xl = int(counts.XlargeItems.Float64)
	}

	return &shipping.ItemCounts{
		Small:  small,
		Medium: medium,
		Large:  large,
		XL:     xl,
	}, nil
}

type SaveShippingSelectionRequest struct {
	OrderID string `json:"order_id"`
	RateID  string `json:"rate_id"`
}

func (h *ShippingHandler) SaveShippingSelection(c echo.Context) error {
	var req SaveShippingSelectionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if req.OrderID == "" || req.RateID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Order ID and Rate ID are required")
	}

	// This would typically be called during the checkout process
	// For now, we'll just return success - the actual implementation
	// would store the selected rate ID for later label creation

	return c.JSON(http.StatusOK, map[string]string{
		"status": "success",
		"order_id": req.OrderID,
		"rate_id": req.RateID,
	})
}

type CreateLabelRequest struct {
	OrderID string `json:"order_id"`
	RateID  string `json:"rate_id"`
}

func (h *ShippingHandler) CreateLabel(c echo.Context) error {
	var req CreateLabelRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	label, err := h.shippingService.CreateLabel(req.RateID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create shipping label")
	}

	// TODO: Save label to database
	// This would involve creating a shipping_labels record

	return c.JSON(http.StatusOK, map[string]interface{}{
		"label_id": label.LabelID,
		"tracking_number": label.TrackingNumber,
		"status": label.Status,
		"pdf_url": label.LabelDownload.Hrefs.PDF,
	})
}

type VoidLabelRequest struct {
	LabelID string `json:"label_id"`
}

func (h *ShippingHandler) VoidLabel(c echo.Context) error {
	var req VoidLabelRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if err := h.shippingService.VoidLabel(req.LabelID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to void shipping label")
	}

	// TODO: Update label status in database

	return c.JSON(http.StatusOK, map[string]string{
		"status": "voided",
		"label_id": req.LabelID,
	})
}

func (h *ShippingHandler) DownloadLabel(c echo.Context) error {
	labelID := c.Param("labelId")
	if labelID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Label ID is required")
	}

	// TODO: Get label from database first
	// For now, we'll return an error since we need the label details
	return echo.NewHTTPError(http.StatusNotImplemented, "Label download not yet implemented")
}

func (h *ShippingHandler) ValidateAddress(c echo.Context) error {
	var addr shipping.Address
	if err := c.Bind(&addr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid address data")
	}

	if err := h.shippingService.ValidateAddress(addr); err != nil {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"valid": false,
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"valid": true,
	})
}