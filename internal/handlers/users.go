package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/storage"
	"github.com/loganlanou/logans3d-v4/storage/db"
	"github.com/loganlanou/logans3d-v4/views/admin"
)

type UserHandler struct {
	storage *storage.Storage
}

func NewUserHandler(storage *storage.Storage) *UserHandler {
	return &UserHandler{
		storage: storage,
	}
}

// HandleUsersList shows all registered users with search and filters
func (h *UserHandler) HandleUsersList(c echo.Context) error {
	ctx := c.Request().Context()

	// Get query parameters
	searchQuery := c.QueryParam("search")
	dateFrom := c.QueryParam("date_from")
	dateTo := c.QueryParam("date_to")
	sortBy := c.QueryParam("sort")

	// Query users with stats
	users, err := h.storage.Queries.ListUsersWithStats(ctx, db.ListUsersWithStatsParams{
		Search:   sql.NullString{String: searchQuery, Valid: searchQuery != ""},
		DateFrom: sql.NullString{String: dateFrom, Valid: dateFrom != ""},
		DateTo:   sql.NullString{String: dateTo, Valid: dateTo != ""},
	})
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch users: "+err.Error())
	}

	// Convert to display format
	userList := make([]admin.UserListItem, 0, len(users))
	for _, u := range users {
		// Handle LastOrderDate interface{}
		var lastOrderDate time.Time
		if u.LastOrderDate != nil {
			if t, ok := u.LastOrderDate.(time.Time); ok {
				lastOrderDate = t
			} else if str, ok := u.LastOrderDate.(string); ok && str != "" {
				if parsed, err := time.Parse("2006-01-02 15:04:05", str); err == nil {
					lastOrderDate = parsed
				}
			}
		}

		var lastActivity time.Time
		if u.LastSyncedAt.Valid {
			lastActivity = u.LastSyncedAt.Time
		}
		// Use last order date if more recent than last sync
		if !lastOrderDate.IsZero() && lastOrderDate.After(lastActivity) {
			lastActivity = lastOrderDate
		}

		// Handle LifetimeSpendCents interface{}
		var lifetimeSpend int64
		if u.LifetimeSpendCents != nil {
			if spend, ok := u.LifetimeSpendCents.(int64); ok {
				lifetimeSpend = spend
			} else if spend, ok := u.LifetimeSpendCents.(float64); ok {
				lifetimeSpend = int64(spend)
			}
		}

		userList = append(userList, admin.UserListItem{
			ID:                 u.ID,
			Email:              u.Email,
			FullName:           u.FullName,
			FirstName:          nullStringToString(u.FirstName),
			LastName:           nullStringToString(u.LastName),
			Username:           nullStringToString(u.Username),
			ProfileImageUrl:    nullStringToString(u.ProfileImageUrl),
			IsAdmin:            u.IsAdmin,
			CreatedAt:          u.CreatedAt.Time,
			LastActivity:       lastActivity,
			OrderCount:         u.OrderCount,
			LifetimeSpendCents: lifetimeSpend,
		})
	}

	return Render(c, admin.Users(c, userList, searchQuery, dateFrom, dateTo, sortBy))
}

// HandleUserDetail shows comprehensive user information
func (h *UserHandler) HandleUserDetail(c echo.Context) error {
	ctx := c.Request().Context()
	userID := c.Param("id")

	// Get user detail with stats
	userStats, err := h.storage.Queries.GetUserDetailWithStats(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.String(http.StatusNotFound, "User not found")
		}
		return c.String(http.StatusInternalServerError, "Failed to fetch user: "+err.Error())
	}

	// Get user orders
	orders, err := h.storage.Queries.GetUserOrders(ctx, userID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch orders: "+err.Error())
	}

	// Get active carts
	activeCarts, err := h.storage.Queries.GetUserActiveCarts(ctx, sql.NullString{String: userID, Valid: true})
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch active carts: "+err.Error())
	}

	// Get abandoned carts
	abandonedCarts, err := h.storage.Queries.GetUserAbandonedCarts(ctx, sql.NullString{String: userID, Valid: true})
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch abandoned carts: "+err.Error())
	}

	// Get favorites
	favorites, err := h.storage.Queries.GetUserRecentFavorites(ctx, userID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch favorites: "+err.Error())
	}

	// Get collections
	collections, err := h.storage.Queries.GetUserCollectionsList(ctx, userID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch collections: "+err.Error())
	}

	// Handle LifetimeSpendCents interface{}
	var lifetimeSpend int64
	if userStats.LifetimeSpendCents != nil {
		if spend, ok := userStats.LifetimeSpendCents.(int64); ok {
			lifetimeSpend = spend
		} else if spend, ok := userStats.LifetimeSpendCents.(float64); ok {
			lifetimeSpend = int64(spend)
		}
	}

	// Convert to display format
	user := admin.UserDetailData{
		ID:                  userStats.ID,
		Email:               userStats.Email,
		FullName:            userStats.FullName,
		FirstName:           nullStringToString(userStats.FirstName),
		LastName:            nullStringToString(userStats.LastName),
		Username:            nullStringToString(userStats.Username),
		ProfileImageUrl:     nullStringToString(userStats.ProfileImageUrl),
		ClerkID:             nullStringToString(userStats.ClerkID),
		IsAdmin:             userStats.IsAdmin,
		CreatedAt:           userStats.CreatedAt.Time,
		UpdatedAt:           nullTimeToTime(userStats.UpdatedAt),
		LastSyncedAt:        nullTimeToTime(userStats.LastSyncedAt),
		OrderCount:          userStats.OrderCount,
		LifetimeSpendCents:  lifetimeSpend,
		FavoritesCount:      userStats.FavoritesCount,
		CollectionsCount:    userStats.CollectionsCount,
		ActiveCartsCount:    userStats.ActiveCartsCount,
		AbandonedCartsCount: userStats.AbandonedCartsCount,
	}

	// Convert orders to display format
	orderList := make([]admin.UserOrderItem, 0, len(orders))
	for _, o := range orders {
		orderList = append(orderList, admin.UserOrderItem{
			ID:            o.ID,
			CustomerEmail: o.CustomerEmail,
			CustomerName:  o.CustomerName,
			CustomerPhone: nullStringToString(o.CustomerPhone),
			Status:        nullStringToString(o.Status),
			TotalCents:    o.TotalCents,
			CreatedAt:     o.CreatedAt.Time,
		})
	}

	// Convert active carts
	activeCartList := make([]admin.UserCartItem, 0, len(activeCarts))
	for _, cart := range activeCarts {
		// Handle LastActivity interface{}
		var lastActivity time.Time
		if cart.LastActivity != nil {
			if t, ok := cart.LastActivity.(time.Time); ok {
				lastActivity = t
			} else if str, ok := cart.LastActivity.(string); ok && str != "" {
				if parsed, err := time.Parse("2006-01-02 15:04:05", str); err == nil {
					lastActivity = parsed
				}
			}
		}

		// Handle TotalCents sql.NullFloat64
		var totalCents int64
		if cart.TotalCents.Valid {
			totalCents = int64(cart.TotalCents.Float64)
		}

		activeCartList = append(activeCartList, admin.UserCartItem{
			SessionID:    nullStringToString(cart.SessionID),
			ItemCount:    cart.ItemCount,
			TotalCents:   totalCents,
			LastActivity: lastActivity,
		})
	}

	// Convert abandoned carts
	abandonedCartList := make([]admin.UserCartItem, 0, len(abandonedCarts))
	for _, cart := range abandonedCarts {
		// Handle LastActivity interface{}
		var lastActivity time.Time
		if cart.LastActivity != nil {
			if t, ok := cart.LastActivity.(time.Time); ok {
				lastActivity = t
			} else if str, ok := cart.LastActivity.(string); ok && str != "" {
				if parsed, err := time.Parse("2006-01-02 15:04:05", str); err == nil {
					lastActivity = parsed
				}
			}
		}

		// Handle TotalCents sql.NullFloat64
		var totalCents int64
		if cart.TotalCents.Valid {
			totalCents = int64(cart.TotalCents.Float64)
		}

		abandonedCartList = append(abandonedCartList, admin.UserCartItem{
			SessionID:    nullStringToString(cart.SessionID),
			ItemCount:    cart.ItemCount,
			TotalCents:   totalCents,
			LastActivity: lastActivity,
		})
	}

	// Convert favorites
	favoriteList := make([]admin.UserFavoriteItem, 0, len(favorites))
	for _, fav := range favorites {
		favoriteList = append(favoriteList, admin.UserFavoriteItem{
			ProductID:   fav.ID,
			ProductName: fav.Name,
			ProductSlug: fav.Slug,
			PriceCents:  fav.PriceCents,
			ImageUrl:    nullStringToString(fav.ImageUrl),
			FavoritedAt: fav.FavoritedAt.Time,
		})
	}

	// Convert collections
	collectionList := make([]admin.UserCollectionItem, 0, len(collections))
	for _, col := range collections {
		// Handle IsQuoteRequested sql.NullBool
		var isQuoteRequested bool
		if col.IsQuoteRequested.Valid {
			isQuoteRequested = col.IsQuoteRequested.Bool
		}

		collectionList = append(collectionList, admin.UserCollectionItem{
			ID:               col.ID,
			Name:             col.Name,
			Description:      nullStringToString(col.Description),
			IsQuoteRequested: isQuoteRequested,
			CreatedAt:        col.CreatedAt.Time,
		})
	}

	return Render(c, admin.UserDetail(c, user, orderList, activeCartList, abandonedCartList, favoriteList, collectionList))
}

// Helper functions
func nullStringToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func nullTimeToTime(nt sql.NullTime) time.Time {
	if nt.Valid {
		return nt.Time
	}
	return time.Time{}
}
