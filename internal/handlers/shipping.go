package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/internal/auth"
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

// Cart snapshot structures
type CartSnapshotItem struct {
	ProductID string `json:"product_id"`
	Quantity  int64  `json:"quantity"`
}

type CartSnapshot struct {
	Items      []CartSnapshotItem `json:"items"`
	TotalCents int64              `json:"total_cents"`
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

	// Check if user is authenticated and prefer user_id for cart lookups
	var userID string
	if user, ok := auth.GetDBUser(c); ok {
		userID = user.ID
	}

	counts, err := h.getCartItemCounts(c, sessionID, userID)
	if err != nil {
		// Check if it's a validation error (starts with known prefix) or a server error
		errMsg := err.Error()
		if strings.HasPrefix(errMsg, "shipping category missing") {
			return echo.NewHTTPError(http.StatusBadRequest, errMsg)
		}
		// Server-side error - log details but return generic message
		slog.Error("failed to get cart item counts for shipping", "error", err, "session_id", sessionID)
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to calculate shipping rates")
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

	if counts.UnknownItems.Valid && counts.UnknownItems.Float64 > 0 {
		return nil, fmt.Errorf("shipping category missing for %d item(s); please configure size chart or SKU overrides", int(counts.UnknownItems.Float64))
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

	toFloat := func(v interface{}) float64 {
		switch val := v.(type) {
		case sql.NullFloat64:
			if val.Valid {
				return val.Float64
			}
		case float64:
			return val
		case int64:
			return float64(val)
		case int:
			return float64(val)
		}
		return 0
	}

	// Get default weights/dimensions from database-driven shipping config
	defaultWeights := h.shippingService.GetDefaultItemWeights()
	defaultDims := h.shippingService.GetDefaultDimensions()

	// Safety fallback if config is missing data (shouldn't happen with validation, but be defensive)
	if len(defaultWeights) == 0 {
		defaultWeights = map[string]float64{"small": 3.0, "medium": 7.05, "large": 15.0, "xlarge": 35.3}
	}
	if len(defaultDims) == 0 {
		defaultDims = map[string]shipping.DimensionGuard{
			"small": {L: 4, W: 4, H: 4}, "medium": {L: 8, W: 5, H: 5},
			"large": {L: 20, W: 10, H: 6}, "xlarge": {L: 24, W: 12, H: 10},
		}
	}

	// Helper to use DB value or fallback to default
	weightOrDefault := func(dbVal interface{}, category string, itemCount int) float64 {
		if itemCount == 0 {
			return 0
		}
		v := toFloat(dbVal)
		if v > 0 {
			return v
		}
		// Use default weight per item * count
		return defaultWeights[category] * float64(itemCount)
	}

	dimsOrDefault := func(l, w, h interface{}, category string) shipping.DimensionGuard {
		dims := shipping.DimensionGuard{
			L: toFloat(l),
			W: toFloat(w),
			H: toFloat(h),
		}
		// If any dimension is missing, use defaults for that category
		def := defaultDims[category]
		if dims.L <= 0 {
			dims.L = def.L
		}
		if dims.W <= 0 {
			dims.W = def.W
		}
		if dims.H <= 0 {
			dims.H = def.H
		}
		return dims
	}

	// Log if we're using fallbacks (helpful for debugging, not an error)
	if small > 0 && toFloat(counts.SmallMissingShipping) > 0 {
		slog.Debug("using default shipping data for small items", "missing_count", int(toFloat(counts.SmallMissingShipping)))
	}
	if medium > 0 && toFloat(counts.MediumMissingShipping) > 0 {
		slog.Debug("using default shipping data for medium items", "missing_count", int(toFloat(counts.MediumMissingShipping)))
	}
	if large > 0 && toFloat(counts.LargeMissingShipping) > 0 {
		slog.Debug("using default shipping data for large items", "missing_count", int(toFloat(counts.LargeMissingShipping)))
	}
	if xl > 0 && toFloat(counts.XlargeMissingShipping) > 0 {
		slog.Debug("using default shipping data for xlarge items", "missing_count", int(toFloat(counts.XlargeMissingShipping)))
	}

	itemCounts := &shipping.ItemCounts{
		Small:          small,
		Medium:         medium,
		Large:          large,
		XL:             xl,
		SmallWeightOz:  weightOrDefault(counts.SmallWeightOz, "small", small),
		MediumWeightOz: weightOrDefault(counts.MediumWeightOz, "medium", medium),
		LargeWeightOz:  weightOrDefault(counts.LargeWeightOz, "large", large),
		XLWeightOz:     weightOrDefault(counts.XlargeWeightOz, "xlarge", xl),
		SmallMaxDims:   dimsOrDefault(counts.SmallMaxLengthIn, counts.SmallMaxWidthIn, counts.SmallMaxHeightIn, "small"),
		MediumMaxDims:  dimsOrDefault(counts.MediumMaxLengthIn, counts.MediumMaxWidthIn, counts.MediumMaxHeightIn, "medium"),
		LargeMaxDims:   dimsOrDefault(counts.LargeMaxLengthIn, counts.LargeMaxWidthIn, counts.LargeMaxHeightIn, "large"),
		XLMaxDims:      dimsOrDefault(counts.XlargeMaxLengthIn, counts.XlargeMaxWidthIn, counts.XlargeMaxHeightIn, "xlarge"),
	}

	return itemCounts, nil
}

// generateCartSnapshot creates a snapshot of current cart state for validation
func (h *ShippingHandler) generateCartSnapshot(c echo.Context, sessionID string) (*CartSnapshot, error) {
	ctx := c.Request().Context()

	cartItems, err := h.queries.GetCartBySession(ctx, sql.NullString{String: sessionID, Valid: true})
	if err != nil {
		return nil, err
	}

	snapshot := &CartSnapshot{
		Items: make([]CartSnapshotItem, 0, len(cartItems)),
	}

	for _, item := range cartItems {
		snapshot.Items = append(snapshot.Items, CartSnapshotItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		})
		snapshot.TotalCents += toInt64Value(item.PriceCents) * item.Quantity
	}

	return snapshot, nil
}

// compareSnapshots checks if two cart snapshots are identical
func (h *ShippingHandler) compareSnapshots(current, stored *CartSnapshot) bool {
	if current.TotalCents != stored.TotalCents {
		return false
	}
	if len(current.Items) != len(stored.Items) {
		return false
	}

	// Create maps for comparison
	currentMap := make(map[string]int64)
	for _, item := range current.Items {
		currentMap[item.ProductID] = item.Quantity
	}

	for _, item := range stored.Items {
		if qty, exists := currentMap[item.ProductID]; !exists || qty != item.Quantity {
			return false
		}
	}

	return true
}

// SaveShippingSelectionRequest - Updated to include all shipping details with breakdown
type SaveShippingSelectionRequest struct {
	RateID              string                 `json:"rate_id"`
	ShipmentID          string                 `json:"shipment_id"`
	CarrierName         string                 `json:"carrier_name"`
	ServiceName         string                 `json:"service_name"`
	PriceCents          int64                  `json:"price_cents"`           // Total price (for backwards compatibility)
	ShippingAmountCents int64                  `json:"shipping_amount_cents"` // Carrier shipping rate only
	BoxCostCents        int64                  `json:"box_cost_cents"`        // Box/packaging cost
	HandlingCostCents   int64                  `json:"handling_cost_cents"`   // Handling cost
	BoxSKU              string                 `json:"box_sku"`               // Box SKU used
	DeliveryDays        int64                  `json:"delivery_days"`
	EstimatedDate       string                 `json:"estimated_date"`
	ShippingAddress     map[string]interface{} `json:"shipping_address"`
}

func (h *ShippingHandler) SaveShippingSelection(c echo.Context) error {
	ctx := c.Request().Context()

	var req SaveShippingSelectionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	// Validate required fields
	if req.RateID == "" || req.ShipmentID == "" || req.CarrierName == "" || req.ServiceName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Missing required shipping fields")
	}

	// Get session ID
	sessionID, err := h.getSessionID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get session")
	}

	// Generate current cart snapshot
	cartSnapshot, err := h.generateCartSnapshot(c, sessionID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate cart snapshot")
	}

	cartSnapshotJSON, err := json.Marshal(cartSnapshot)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to serialize cart snapshot")
	}

	shippingAddressJSON, err := json.Marshal(req.ShippingAddress)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to serialize shipping address")
	}

	// Check if session already has a shipping selection
	_, err = h.queries.GetSessionShippingSelection(ctx, sessionID)
	if err == sql.ErrNoRows {
		// Create new record
		_, err = h.queries.CreateSessionShippingSelection(ctx, db.CreateSessionShippingSelectionParams{
			ID:                  uuid.New().String(),
			SessionID:           sessionID,
			RateID:              req.RateID,
			ShipmentID:          req.ShipmentID,
			CarrierName:         req.CarrierName,
			ServiceName:         req.ServiceName,
			PriceCents:          req.PriceCents,
			ShippingAmountCents: req.ShippingAmountCents,
			BoxCostCents:        req.BoxCostCents,
			HandlingCostCents:   req.HandlingCostCents,
			BoxSku:              req.BoxSKU,
			DeliveryDays:        sql.NullInt64{Int64: req.DeliveryDays, Valid: true},
			EstimatedDate:       sql.NullString{String: req.EstimatedDate, Valid: req.EstimatedDate != ""},
			CartSnapshotJson:    string(cartSnapshotJSON),
			ShippingAddressJson: string(shippingAddressJSON),
			IsValid:             sql.NullBool{Bool: true, Valid: true},
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to save shipping selection")
		}
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to check existing shipping selection")
	} else {
		// Update existing record
		_, err = h.queries.UpdateSessionShippingSelection(ctx, db.UpdateSessionShippingSelectionParams{
			SessionID:           sessionID,
			RateID:              req.RateID,
			ShipmentID:          req.ShipmentID,
			CarrierName:         req.CarrierName,
			ServiceName:         req.ServiceName,
			PriceCents:          req.PriceCents,
			ShippingAmountCents: req.ShippingAmountCents,
			BoxCostCents:        req.BoxCostCents,
			HandlingCostCents:   req.HandlingCostCents,
			BoxSku:              req.BoxSKU,
			DeliveryDays:        sql.NullInt64{Int64: req.DeliveryDays, Valid: true},
			EstimatedDate:       sql.NullString{String: req.EstimatedDate, Valid: req.EstimatedDate != ""},
			CartSnapshotJson:    string(cartSnapshotJSON),
			ShippingAddressJson: string(shippingAddressJSON),
			IsValid:             sql.NullBool{Bool: true, Valid: true},
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update shipping selection")
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "success",
		"rate_id": req.RateID,
	})
}

// GetShippingSelection retrieves saved shipping selection and validates it against current cart
type GetShippingSelectionResponse struct {
	Selection       *ShippingSelectionData `json:"selection"`
	ShippingAddress map[string]interface{} `json:"shipping_address"`
}

type ShippingSelectionData struct {
	RateID        string `json:"rate_id"`
	ShipmentID    string `json:"shipment_id"`
	CarrierName   string `json:"carrier_name"`
	ServiceName   string `json:"service_name"`
	PriceCents    int64  `json:"price_cents"`
	DeliveryDays  int64  `json:"delivery_days"`
	EstimatedDate string `json:"estimated_date"`
	IsValid       bool   `json:"is_valid"`
}

func (h *ShippingHandler) GetShippingSelection(c echo.Context) error {
	ctx := c.Request().Context()

	// Get session ID
	sessionID, err := h.getSessionID(c)
	if err != nil {
		// No session, return empty response
		return c.JSON(http.StatusOK, GetShippingSelectionResponse{})
	}

	// Get saved shipping selection
	selection, err := h.queries.GetSessionShippingSelection(ctx, sessionID)
	if err == sql.ErrNoRows {
		// No saved selection
		return c.JSON(http.StatusOK, GetShippingSelectionResponse{})
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get shipping selection")
	}

	// Parse shipping address for pre-fill
	var shippingAddress map[string]interface{}
	if err := json.Unmarshal([]byte(selection.ShippingAddressJson), &shippingAddress); err != nil {
		shippingAddress = nil
	}

	// Generate current cart snapshot
	currentSnapshot, err := h.generateCartSnapshot(c, sessionID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate cart snapshot")
	}

	// Parse stored cart snapshot
	var storedSnapshot CartSnapshot
	if err := json.Unmarshal([]byte(selection.CartSnapshotJson), &storedSnapshot); err != nil {
		// Invalid snapshot, invalidate selection
		if err := h.queries.InvalidateSessionShipping(ctx, sessionID); err != nil {
			slog.Error("failed to invalidate shipping selection", "error", err, "session_id", sessionID)
		}
		return c.JSON(http.StatusOK, GetShippingSelectionResponse{
			ShippingAddress: shippingAddress,
		})
	}

	// Compare snapshots
	selectionIsValid := selection.IsValid.Valid && selection.IsValid.Bool
	isValid := h.compareSnapshots(currentSnapshot, &storedSnapshot) && selectionIsValid

	// If cart changed, invalidate the selection in database
	if !isValid && selectionIsValid {
		if err := h.queries.InvalidateSessionShipping(ctx, sessionID); err != nil {
			slog.Error("failed to invalidate shipping selection", "error", err, "session_id", sessionID)
		}
	}

	deliveryDays := int64(0)
	if selection.DeliveryDays.Valid {
		deliveryDays = selection.DeliveryDays.Int64
	}

	estimatedDate := ""
	if selection.EstimatedDate.Valid {
		estimatedDate = selection.EstimatedDate.String
	}

	response := GetShippingSelectionResponse{
		Selection: &ShippingSelectionData{
			RateID:        selection.RateID,
			ShipmentID:    selection.ShipmentID,
			CarrierName:   selection.CarrierName,
			ServiceName:   selection.ServiceName,
			PriceCents:    selection.PriceCents,
			DeliveryDays:  deliveryDays,
			EstimatedDate: estimatedDate,
			IsValid:       isValid,
		},
		ShippingAddress: shippingAddress,
	}

	return c.JSON(http.StatusOK, response)
}

// InvalidateShipping invalidates the shipping selection for a session
func (h *ShippingHandler) InvalidateShipping(ctx echo.Context, sessionID string) error {
	return h.queries.InvalidateSessionShipping(ctx.Request().Context(), sessionID)
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
		"label_id":        label.LabelID,
		"tracking_number": label.TrackingNumber,
		"status":          label.Status,
		"pdf_url":         label.LabelDownload.Hrefs.PDF,
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
		"status":   "voided",
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

func toInt64Value(v interface{}) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case float64:
		return int64(val)
	case int:
		return int64(val)
	case sql.NullFloat64:
		if val.Valid {
			return int64(val.Float64)
		}
	case sql.NullInt64:
		if val.Valid {
			return val.Int64
		}
	}
	return 0
}
