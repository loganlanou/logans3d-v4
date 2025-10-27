package handlers

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/storage"
	"github.com/loganlanou/logans3d-v4/storage/db"
	"github.com/loganlanou/logans3d-v4/views/admin"
)

type CartHandler struct {
	storage *storage.Storage
}

func NewCartHandler(storage *storage.Storage) *CartHandler {
	return &CartHandler{
		storage: storage,
	}
}

// HandleCartsList shows all shopping carts with filters
func (h *CartHandler) HandleCartsList(c echo.Context) error {
	ctx := c.Request().Context()

	// Get query parameters
	statusFilter := c.QueryParam("status")
	customerTypeFilter := c.QueryParam("customer_type")
	searchQuery := c.QueryParam("search")

	// Get cart metrics
	metricsRow, err := h.storage.Queries.GetCartMetrics(ctx)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch metrics: "+err.Error())
	}

	metrics := admin.CartMetrics{
		TotalCarts:      metricsRow.TotalCarts,
		GuestCarts:      metricsRow.GuestCarts,
		RegisteredCarts: metricsRow.RegisteredCarts,
		TotalValueCents: toInt64(metricsRow.TotalValueCents),
		AvgValueCents:   toInt64(metricsRow.AvgCartValueCents),
		AbandonedCount:  metricsRow.AbandonedCount,
		ActiveCount:     metricsRow.ActiveCount,
	}

	// Get carts list
	var carts []admin.CartListItem

	if searchQuery != "" {
		// Search mode
		results, err := h.storage.Queries.SearchCarts(ctx, sql.NullString{String: searchQuery, Valid: true})
		if err != nil {
			return c.String(http.StatusInternalServerError, "Failed to search carts: "+err.Error())
		}

		carts = make([]admin.CartListItem, 0, len(results))
		for _, row := range results {
			status := determineCartStatus(row.LastActivity)

			customerEmail := ""
			if row.CustomerEmail.Valid {
				customerEmail = row.CustomerEmail.String
			}

			customerName := ""
			if row.CustomerName.Valid {
				customerName = row.CustomerName.String
			}

			carts = append(carts, admin.CartListItem{
				SessionID:       row.SessionID,
				UserID:          row.UserID,
				CustomerEmail:   customerEmail,
				CustomerName:    customerName,
				ItemCount:       row.ItemCount,
				CartValueCents:  toInt64(row.CartValueCents),
				CreatedAt:       parseTime(row.CreatedAt),
				LastActivity:    parseTime(row.LastActivity),
				Status:          status,
			})
		}
	} else {
		// Normal listing with filters
		results, err := h.storage.Queries.GetAllCartsWithDetails(ctx, db.GetAllCartsWithDetailsParams{
			Status:       sql.NullString{String: statusFilter, Valid: statusFilter != ""},
			CustomerType: sql.NullString{String: customerTypeFilter, Valid: customerTypeFilter != ""},
			Search:       sql.NullString{String: searchQuery, Valid: searchQuery != ""},
			PageSize:     100,
			Offset:       0,
		})
		if err != nil {
			return c.String(http.StatusInternalServerError, "Failed to fetch carts: "+err.Error())
		}

		carts = make([]admin.CartListItem, 0, len(results))
		for _, row := range results {
			customerEmail := ""
			if row.CustomerEmail.Valid {
				customerEmail = row.CustomerEmail.String
			}

			customerName := ""
			if row.CustomerName.Valid {
				customerName = row.CustomerName.String
			}

			customerAvatar := ""
			if row.CustomerAvatar.Valid {
				customerAvatar = row.CustomerAvatar.String
			}

			carts = append(carts, admin.CartListItem{
				SessionID:       row.SessionID,
				UserID:          row.UserID,
				CustomerEmail:   customerEmail,
				CustomerName:    customerName,
				CustomerAvatar:  customerAvatar,
				ItemCount:       row.ItemCount,
				CartValueCents:  toInt64(row.CartValueCents),
				CreatedAt:       parseTime(row.CreatedAt),
				LastActivity:    parseTime(row.LastActivity),
				Status:          row.Status,
			})
		}
	}

	return Render(c, admin.CartsList(
		c,
		metrics,
		carts,
		statusFilter,
		customerTypeFilter,
		searchQuery,
	))
}

// HandleCartDetail shows detailed information about a specific cart
func (h *CartHandler) HandleCartDetail(c echo.Context) error {
	ctx := c.Request().Context()
	cartID := c.Param("id")

	// Parse cart ID (format: "session-xxx" or "user-xxx")
	var sessionID, userID string
	if strings.HasPrefix(cartID, "session-") {
		sessionID = strings.TrimPrefix(cartID, "session-")
	} else if strings.HasPrefix(cartID, "user-") {
		userID = strings.TrimPrefix(cartID, "user-")
	} else {
		return c.String(http.StatusBadRequest, "Invalid cart ID format")
	}

	// Get cart details
	var cartDetail admin.CartDetail

	if sessionID != "" {
		row, err := h.storage.Queries.GetCartDetailsBySession(ctx, sql.NullString{String: sessionID, Valid: true})
		if err != nil {
			if err == sql.ErrNoRows {
				return c.String(http.StatusNotFound, "Cart not found")
			}
			return c.String(http.StatusInternalServerError, "Failed to fetch cart: "+err.Error())
		}

		status := determineCartStatus(row.LastActivity)

		cartDetail = admin.CartDetail{
			SessionID:      row.SessionID.String,
			ItemCount:      row.ItemCount,
			CartValueCents: toInt64(row.CartValueCents),
			CreatedAt:      parseTime(row.CreatedAt),
			LastActivity:   parseTime(row.LastActivity),
			Status:         status,
		}
	} else {
		row, err := h.storage.Queries.GetCartDetailsByUser(ctx, sql.NullString{String: userID, Valid: true})
		if err != nil {
			if err == sql.ErrNoRows {
				return c.String(http.StatusNotFound, "Cart not found")
			}
			return c.String(http.StatusInternalServerError, "Failed to fetch cart: "+err.Error())
		}

		status := determineCartStatus(row.LastActivity)

		customerEmail := ""
		if row.CustomerEmail.Valid {
			customerEmail = row.CustomerEmail.String
		}

		customerName := ""
		if row.CustomerName.Valid {
			customerName = row.CustomerName.String
		}

		customerAvatar := ""
		if row.CustomerAvatar.Valid {
			customerAvatar = row.CustomerAvatar.String
		}

		cartDetail = admin.CartDetail{
			UserID:         row.UserID.String,
			CustomerEmail:  customerEmail,
			CustomerName:   customerName,
			CustomerAvatar: customerAvatar,
			ItemCount:      row.ItemCount,
			CartValueCents: toInt64(row.CartValueCents),
			CreatedAt:      parseTime(row.CreatedAt),
			LastActivity:   parseTime(row.LastActivity),
			Status:         status,
		}
	}

	// Get cart items
	itemsRows, err := h.storage.Queries.GetCartItemsWithDetails(ctx, db.GetCartItemsWithDetailsParams{
		SessionID: sql.NullString{String: sessionID, Valid: sessionID != ""},
		UserID:    sql.NullString{String: userID, Valid: userID != ""},
	})
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch cart items: "+err.Error())
	}

	items := make([]admin.CartItem, 0, len(itemsRows))
	for _, row := range itemsRows {
		variantSKU := ""
		if row.VariantSku.Valid {
			variantSKU = row.VariantSku.String
		}

		variantName := ""
		if row.VariantName.Valid {
			variantName = row.VariantName.String
		}

		productImage := ""
		if row.ProductImage.Valid {
			productImage = row.ProductImage.String
		}

		items = append(items, admin.CartItem{
			ID:             row.ID,
			ProductID:      row.ProductID,
			ProductName:    row.ProductName,
			ProductImage:   productImage,
			VariantSKU:     variantSKU,
			VariantName:    variantName,
			Quantity:       row.Quantity,
			PriceCents:     row.PriceCents,
			LineTotalCents: toInt64(row.LineTotalCents),
			CreatedAt:      parseNullTime(row.CreatedAt),
			UpdatedAt:      parseNullTime(row.UpdatedAt),
		})
	}

	return Render(c, admin.CartDetailPage(
		c,
		cartDetail,
		items,
	))
}

// Helper function to convert interface{} to int64
func toInt64(v interface{}) int64 {
	if v == nil {
		return 0
	}
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
		return 0
	case sql.NullInt64:
		if val.Valid {
			return val.Int64
		}
		return 0
	default:
		return 0
	}
}

// Helper function to parse time from interface{}
func parseTime(v interface{}) time.Time {
	if v == nil {
		return time.Now()
	}

	switch val := v.(type) {
	case time.Time:
		return val
	case string:
		// Try parsing as SQLite datetime string
		if parsed, err := time.Parse("2006-01-02 15:04:05", val); err == nil {
			return parsed
		}
		return time.Now()
	default:
		return time.Now()
	}
}

// Helper function to parse sql.NullTime to time.Time
func parseNullTime(nt sql.NullTime) time.Time {
	if nt.Valid {
		return nt.Time
	}
	return time.Now()
}

// Helper function to determine cart status based on last activity
func determineCartStatus(lastActivity interface{}) string {
	t := parseTime(lastActivity)
	duration := time.Since(t)

	if duration >= 30*time.Minute {
		return "abandoned"
	} else if duration >= 25*time.Minute {
		return "at_risk"
	} else if duration >= 15*time.Minute {
		return "idle"
	}
	return "active"
}
