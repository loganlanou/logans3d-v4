package handlers

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/internal/email"
	"github.com/loganlanou/logans3d-v4/storage/db"
	"github.com/loganlanou/logans3d-v4/views/admin"
)

// toFloat64 safely converts an interface{} value to float64
// SQLite can return numeric values as different types (int64, float64, etc.)
func toFloat64(v interface{}) float64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int64:
		return float64(val)
	case int:
		return float64(val)
	case int32:
		return float64(val)
	default:
		return 0
	}
}

// HandleAbandonedCartsDashboard shows the main abandoned carts analytics dashboard
func (h *AdminHandler) HandleAbandonedCartsDashboard(c echo.Context) error {
	ctx := c.Request().Context()

	// Get query parameters
	statusFilter := c.QueryParam("status")
	highValueFilter := c.QueryParam("high_value")
	searchQuery := c.QueryParam("search")

	// Calculate metrics
	metrics, err := h.getAbandonedCartMetrics(ctx)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch metrics: "+err.Error())
	}

	// Get abandoned carts list and build details
	var cartDetails []admin.AbandonedCartWithDetails

	if searchQuery != "" {
		carts, err := h.storage.Queries.SearchAbandonedCarts(ctx, sql.NullString{String: searchQuery, Valid: true})
		if err != nil {
			return c.String(http.StatusInternalServerError, "Failed to fetch abandoned carts: "+err.Error())
		}
		cartDetails = make([]admin.AbandonedCartWithDetails, 0, len(carts))
		for _, cart := range carts {
			cartDetails = append(cartDetails, admin.AbandonedCartWithDetails{
				Cart:         cart,
				ItemCount:    cart.ItemCount,
				AttemptCount: 0,
				TimeAgo:      formatTimeAgo(cart.AbandonedAt),
			})
		}
	} else if statusFilter != "" {
		carts, err := h.storage.Queries.GetAbandonedCartsByStatus(ctx, sql.NullString{String: statusFilter, Valid: true})
		if err != nil {
			return c.String(http.StatusInternalServerError, "Failed to fetch abandoned carts: "+err.Error())
		}
		cartDetails = make([]admin.AbandonedCartWithDetails, 0, len(carts))
		for _, cart := range carts {
			cartDetails = append(cartDetails, admin.AbandonedCartWithDetails{
				Cart:         cart,
				ItemCount:    cart.ItemCount,
				AttemptCount: 0,
				TimeAgo:      formatTimeAgo(cart.AbandonedAt),
			})
		}
	} else if highValueFilter == "true" {
		carts, err := h.storage.Queries.GetHighValueAbandonedCarts(ctx, db.GetHighValueAbandonedCartsParams{
			MinValueCents: 10000, // $100+
			LimitCount:    50,
		})
		if err != nil {
			return c.String(http.StatusInternalServerError, "Failed to fetch abandoned carts: "+err.Error())
		}
		cartDetails = make([]admin.AbandonedCartWithDetails, 0, len(carts))
		for _, cart := range carts {
			cartDetails = append(cartDetails, admin.AbandonedCartWithDetails{
				Cart:         cart,
				ItemCount:    cart.ItemCount,
				AttemptCount: 0,
				TimeAgo:      formatTimeAgo(cart.AbandonedAt),
			})
		}
	} else {
		recentCarts, err := h.storage.Queries.ListRecentAbandonedCarts(ctx)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Failed to fetch abandoned carts: "+err.Error())
		}
		cartDetails = make([]admin.AbandonedCartWithDetails, 0, len(recentCarts))
		for _, row := range recentCarts {
			// Convert ListRecentAbandonedCartsRow to AbandonedCart
			cart := db.AbandonedCart{
				ID:              row.ID,
				SessionID:       row.SessionID,
				UserID:          row.UserID,
				CustomerEmail:   row.CustomerEmail,
				CustomerName:    row.CustomerName,
				CartValueCents:  row.CartValueCents,
				ItemCount:       row.ItemCount,
				AbandonedAt:     row.AbandonedAt,
				RecoveredAt:     row.RecoveredAt,
				RecoveryMethod:  row.RecoveryMethod,
				Status:          row.Status,
				LastContactedAt: row.LastContactedAt,
				Notes:           row.Notes,
				CreatedAt:       row.CreatedAt,
				UpdatedAt:       row.UpdatedAt,
			}
			cartDetails = append(cartDetails, admin.AbandonedCartWithDetails{
				Cart:         cart,
				ItemCount:    row.ItemCount,
				AttemptCount: row.RecoveryAttemptCount,
				TimeAgo:      formatTimeAgo(row.AbandonedAt),
			})
		}
	}

	// Get trend data (last 7 days)
	trendData, err := h.getTrendChartData(ctx, "-7 days")
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch trend data: "+err.Error())
	}

	// Get top abandoned products
	topProducts, err := h.getTopAbandonedProducts(ctx, 5)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch top products: "+err.Error())
	}

	// Get email recovery stats
	emailStats, err := h.getEmailRecoveryStats(ctx)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch email stats: "+err.Error())
	}

	// Hourly data (not used yet but prepared for future)
	hourlyData := admin.ChartData{
		Labels: []string{},
		Values: []float64{},
	}

	return Render(c, admin.AbandonedCartsDashboard(
		c,
		metrics,
		cartDetails,
		trendData,
		topProducts,
		emailStats,
		hourlyData,
	))
}

// HandleAbandonedCartDetail shows details for a single abandoned cart
func (h *AdminHandler) HandleAbandonedCartDetail(c echo.Context) error {
	ctx := c.Request().Context()
	cartID := c.Param("id")

	cart, err := h.storage.Queries.GetAbandonedCartByID(ctx, cartID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.String(http.StatusNotFound, "Abandoned cart not found")
		}
		return c.String(http.StatusInternalServerError, "Failed to fetch abandoned cart: "+err.Error())
	}

	// Get cart snapshots (products in the cart)
	snapshots, err := h.storage.Queries.GetCartSnapshotsByAbandonedCartID(ctx, cartID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch cart contents: "+err.Error())
	}

	// Get recovery attempts
	attempts, err := h.storage.Queries.GetRecoveryAttemptsByCartID(ctx, cartID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch recovery attempts: "+err.Error())
	}

	// TODO: Create detail page template
	return c.JSON(http.StatusOK, map[string]interface{}{
		"cart":      cart,
		"items":     snapshots,
		"attempts":  attempts,
	})
}

// HandleSendRecoveryEmail manually sends a recovery email for an abandoned cart
func (h *AdminHandler) HandleSendRecoveryEmail(c echo.Context) error {
	ctx := c.Request().Context()
	cartID := c.Param("id")

	// Get the abandoned cart
	cart, err := h.storage.Queries.GetAbandonedCartByID(ctx, cartID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{
			"success": false,
			"error":   "Cart not found",
		})
	}

	// Check if cart has email
	if !cart.CustomerEmail.Valid || cart.CustomerEmail.String == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "No email address for this cart",
		})
	}

	// Create recovery attempt record
	attemptID := uuid.New().String()
	trackingToken := uuid.New().String()

	_, err = h.storage.Queries.CreateRecoveryAttempt(ctx, db.CreateRecoveryAttemptParams{
		ID:              attemptID,
		AbandonedCartID: cartID,
		AttemptType:     "manual",
		SentAt:          time.Now(),
		EmailSubject:    sql.NullString{String: "Complete your order", Valid: true},
		TrackingToken:   sql.NullString{String: trackingToken, Valid: true},
		Status:          sql.NullString{String: "sent", Valid: true},
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   "Failed to create recovery attempt: " + err.Error(),
		})
	}

	// Get cart snapshots to include in email
	snapshots, err := h.storage.Queries.GetCartSnapshotsByAbandonedCartID(ctx, cartID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   "Failed to get cart contents: " + err.Error(),
		})
	}

	// Convert snapshots to email items
	items := make([]email.AbandonedCartItem, 0, len(snapshots))
	for _, snapshot := range snapshots {
		items = append(items, email.AbandonedCartItem{
			ProductName:  snapshot.ProductName,
			ProductImage: snapshot.ProductImageUrl.String,
			Quantity:     snapshot.Quantity,
			UnitPrice:    snapshot.UnitPriceCents,
		})
	}

	// Create email data
	customerName := "Customer"
	if cart.CustomerName.Valid && cart.CustomerName.String != "" {
		customerName = cart.CustomerName.String
	}

	emailData := &email.AbandonedCartData{
		CustomerName:  customerName,
		CustomerEmail: cart.CustomerEmail.String,
		CartValue:     cart.CartValueCents,
		ItemCount:     cart.ItemCount,
		Items:         items,
		TrackingToken: trackingToken,
		AbandonedAt:   cart.AbandonedAt.Format("January 2, 2006 at 3:04 PM"),
	}

	// Send the email
	err = h.emailService.SendAbandonedCartRecoveryEmail(emailData, "manual")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   "Failed to send email: " + err.Error(),
		})
	}

	// Mark the cart as contacted
	err = h.storage.Queries.MarkCartAsContacted(ctx, cartID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   "Failed to mark cart as contacted: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Recovery email sent successfully",
	})
}

// HandleUpdateCartNotes updates admin notes for an abandoned cart
func (h *AdminHandler) HandleUpdateCartNotes(c echo.Context) error {
	ctx := c.Request().Context()
	cartID := c.Param("id")

	var req struct {
		Notes string `json:"notes"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "Invalid request",
		})
	}

	err := h.storage.Queries.UpdateAbandonedCartNotes(ctx, db.UpdateAbandonedCartNotesParams{
		Notes: sql.NullString{String: req.Notes, Valid: true},
		ID:    cartID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   "Failed to update notes: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

// HandleMarkCartRecovered manually marks a cart as recovered
func (h *AdminHandler) HandleMarkCartRecovered(c echo.Context) error {
	ctx := c.Request().Context()
	cartID := c.Param("id")

	err := h.storage.Queries.MarkCartAsRecovered(ctx, db.MarkCartAsRecoveredParams{
		RecoveryMethod: sql.NullString{String: "manual", Valid: true},
		ID:             cartID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   "Failed to mark cart as recovered: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

// HandleExportAbandonedCarts exports abandoned carts data as CSV
func (h *AdminHandler) HandleExportAbandonedCarts(c echo.Context) error {
	ctx := c.Request().Context()

	// Get all recent abandoned carts
	carts, err := h.storage.Queries.ListRecentAbandonedCarts(ctx)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch abandoned carts: "+err.Error())
	}

	// Set headers for CSV download
	c.Response().Header().Set(echo.HeaderContentType, "text/csv")
	c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=\"abandoned-carts-%s.csv\"", time.Now().Format("2006-01-02")))
	c.Response().WriteHeader(http.StatusOK)

	// Create CSV writer
	w := csv.NewWriter(c.Response())
	defer w.Flush()

	// Write header
	w.Write([]string{
		"Cart ID",
		"Customer Name",
		"Customer Email",
		"Cart Value",
		"Item Count",
		"Abandoned At",
		"Status",
		"Recovery Method",
		"Recovered At",
	})

	// Write data
	for _, cart := range carts {
		customerName := ""
		if cart.CustomerName.Valid {
			customerName = cart.CustomerName.String
		}
		customerEmail := ""
		if cart.CustomerEmail.Valid {
			customerEmail = cart.CustomerEmail.String
		}
		recoveryMethod := ""
		if cart.RecoveryMethod.Valid {
			recoveryMethod = cart.RecoveryMethod.String
		}
		recoveredAt := ""
		if cart.RecoveredAt.Valid {
			recoveredAt = cart.RecoveredAt.Time.Format("2006-01-02 15:04:05")
		}

		w.Write([]string{
			cart.ID,
			customerName,
			customerEmail,
			fmt.Sprintf("%.2f", float64(cart.CartValueCents)/100),
			fmt.Sprintf("%d", cart.ItemCount),
			cart.AbandonedAt.Format("2006-01-02 15:04:05"),
			cart.Status.String,
			recoveryMethod,
			recoveredAt,
		})
	}

	return nil
}

// Helper functions

func (h *AdminHandler) getAbandonedCartMetrics(ctx context.Context) (admin.AbandonedCartMetrics, error) {
	// Get total abandoned in last 24h
	recentCarts, err := h.storage.Queries.ListRecentAbandonedCarts(ctx)
	if err != nil {
		return admin.AbandonedCartMetrics{}, err
	}

	// Calculate total value
	totalValue, err := h.storage.Queries.GetTotalAbandonedCartValue(ctx, "-24 hours")
	if err != nil {
		return admin.AbandonedCartMetrics{}, err
	}

	// Calculate recovery rate (last 30 days)
	recoveryData, err := h.storage.Queries.GetRecoveryRateByPeriod(ctx, "-30 days")
	if err != nil {
		return admin.AbandonedCartMetrics{}, err
	}

	// Calculate abandonment rate (last 30 days)
	abandonmentData, err := h.storage.Queries.GetAbandonmentRateByPeriod(ctx, "-30 days")
	if err != nil {
		return admin.AbandonedCartMetrics{}, err
	}

	return admin.AbandonedCartMetrics{
		TotalAbandoned24h:    int64(len(recentCarts)),
		AbandonmentRate:      toFloat64(abandonmentData.AbandonmentRatePercent),
		LostRevenueCents:     int64(totalValue.TotalValueCents.Float64),
		RecoveryRate:         toFloat64(recoveryData.RecoveryRatePercent),
		TotalRecovered:       recoveryData.RecoveredCount,
		RecoveredValueCents:  int64(recoveryData.RecoveredValueCents.Float64),
	}, nil
}

func (h *AdminHandler) getTrendChartData(ctx context.Context, period string) (admin.ChartData, error) {
	trends, err := h.storage.Queries.GetAbandonmentTrendByDay(ctx, period)
	if err != nil {
		return admin.ChartData{}, err
	}

	labels := make([]string, len(trends))
	values := make([]float64, len(trends))

	for i, trend := range trends {
		labels[i] = fmt.Sprintf("%v", trend.Date)
		values[i] = float64(trend.CartCount)
	}

	return admin.ChartData{
		Labels: labels,
		Values: values,
	}, nil
}

func (h *AdminHandler) getTopAbandonedProducts(ctx context.Context, limit int) ([]admin.ProductAbandonmentData, error) {
	products, err := h.storage.Queries.GetTopAbandonedProducts(ctx, db.GetTopAbandonedProductsParams{
		PeriodOffset: "-30 days",
		LimitCount:   int64(limit),
	})
	if err != nil {
		return nil, err
	}

	result := make([]admin.ProductAbandonmentData, len(products))
	for i, p := range products {
		result[i] = admin.ProductAbandonmentData{
			ProductID:   p.ProductID,
			ProductName: p.ProductName,
			Count:       p.AbandonedCount,
			TotalValue:  int64(p.TotalValueCents.Float64),
		}
	}

	return result, nil
}

func (h *AdminHandler) getEmailRecoveryStats(ctx context.Context) ([]admin.RecoveryEmailStats, error) {
	stats, err := h.storage.Queries.GetRecoveryEmailPerformance(ctx, "-30 days")
	if err != nil {
		return nil, err
	}

	result := make([]admin.RecoveryEmailStats, len(stats))
	for i, s := range stats {
		result[i] = admin.RecoveryEmailStats{
			AttemptType:      s.AttemptType,
			TotalSent:        s.TotalSent,
			OpenedCount:      s.OpenedCount,
			ClickedCount:     s.ClickedCount,
			OpenRatePercent:  toFloat64(s.OpenRatePercent),
			ClickRatePercent: toFloat64(s.ClickRatePercent),
		}
	}

	return result, nil
}

// HandleRecoveryEmailTracking tracks when customers click on recovery email links
func (h *AdminHandler) HandleRecoveryEmailTracking(c echo.Context) error {
	ctx := c.Request().Context()
	token := c.QueryParam("token")

	if token == "" {
		// Just redirect to cart page if no token
		return c.Redirect(http.StatusTemporaryRedirect, "/cart")
	}

	// Get the recovery attempt by token
	attempt, err := h.storage.Queries.GetRecoveryAttemptByToken(ctx, sql.NullString{String: token, Valid: true})
	if err != nil {
		if err == sql.ErrNoRows {
			// Token not found, just redirect to cart
			return c.Redirect(http.StatusTemporaryRedirect, "/cart")
		}
		return c.Redirect(http.StatusTemporaryRedirect, "/cart")
	}

	// Mark the attempt as clicked
	err = h.storage.Queries.MarkRecoveryAttemptClicked(ctx, sql.NullString{String: token, Valid: true})
	if err != nil {
		slog.Error("failed to mark recovery attempt as clicked", "error", err, "token", token)
	}

	// Get the abandoned cart
	cart, err := h.storage.Queries.GetAbandonedCartByID(ctx, attempt.AbandonedCartID)
	if err != nil {
		return c.Redirect(http.StatusTemporaryRedirect, "/cart")
	}

	// If cart is still active (not already recovered), mark as recovered via email
	if cart.Status.String == "active" || cart.Status.String == "contacted" {
		err = h.storage.Queries.MarkCartAsRecovered(ctx, db.MarkCartAsRecoveredParams{
			RecoveryMethod: sql.NullString{String: attempt.AttemptType, Valid: true},
			ID:             cart.ID,
		})
		if err != nil {
			slog.Error("failed to mark cart as recovered", "error", err, "cart_id", cart.ID)
		} else {
			slog.Info("cart recovered via email", "cart_id", cart.ID, "attempt_type", attempt.AttemptType)
		}
	}

	// Redirect to cart page
	return c.Redirect(http.StatusTemporaryRedirect, "/cart")
}

func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "Just now"
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}
