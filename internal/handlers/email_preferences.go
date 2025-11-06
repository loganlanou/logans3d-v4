package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/internal/auth"
	emailutil "github.com/loganlanou/logans3d-v4/internal/email"
	"github.com/loganlanou/logans3d-v4/storage/db"
	"github.com/loganlanou/logans3d-v4/views/account"
	"github.com/loganlanou/logans3d-v4/views/layout"
	"github.com/oklog/ulid/v2"
)

type EmailPreferencesHandler struct {
	queries *db.Queries
}

func NewEmailPreferencesHandler(queries *db.Queries) *EmailPreferencesHandler {
	return &EmailPreferencesHandler{
		queries: queries,
	}
}

// HandleUnsubscribe handles one-click unsubscribe from all marketing emails
func (h *EmailPreferencesHandler) HandleUnsubscribe(c echo.Context) error {
	token := c.Param("token")
	if token == "" {
		return c.String(http.StatusBadRequest, "Invalid unsubscribe link")
	}

	ctx := context.Background()

	// Get preferences by token
	prefs, err := h.queries.GetEmailPreferencesByUnsubscribeToken(ctx, sql.NullString{String: token, Valid: true})
	if err != nil {
		if err == sql.ErrNoRows {
			return c.String(http.StatusNotFound, "Invalid or expired unsubscribe link")
		}
		return c.String(http.StatusInternalServerError, "Failed to process unsubscribe request")
	}

	// Unsubscribe from all marketing emails
	err = h.queries.UnsubscribeFromAllMarketing(ctx, sql.NullString{String: token, Valid: true})
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to update preferences")
	}

	// Return simple success page
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Unsubscribed - Logan's 3D Creations</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            max-width: 600px;
            margin: 100px auto;
            padding: 20px;
            text-align: center;
            color: #333;
        }
        h1 { color: #2563eb; margin-bottom: 20px; }
        p { font-size: 16px; line-height: 1.6; color: #666; }
        .success { background: #10b981; color: white; padding: 12px 24px; border-radius: 8px; display: inline-block; margin: 20px 0; }
        a { color: #2563eb; text-decoration: none; }
        a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <h1>✓ You've been unsubscribed</h1>
    <div class="success">Successfully unsubscribed from all marketing emails</div>
    <p>We've updated your email preferences for: <strong>%s</strong></p>
    <p>You will no longer receive:</p>
    <ul style="text-align: left; display: inline-block; margin: 20px auto;">
        <li>Abandoned cart reminders</li>
        <li>Promotional offers</li>
        <li>Newsletter updates</li>
        <li>Product announcements</li>
    </ul>
    <p style="font-size: 14px; color: #999; margin-top: 40px;">
        Note: You will still receive important transactional emails like order confirmations.
    </p>
    <p><a href="/">Return to Logan's 3D Creations</a></p>
</body>
</html>
`, prefs.Email)

	return c.HTML(http.StatusOK, html)
}

// HandleGetEmailPreferences returns the current user's email preferences (JSON API)
func (h *EmailPreferencesHandler) HandleGetEmailPreferences(c echo.Context) error {
	email := c.QueryParam("email")
	if email == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Email parameter required"})
	}

	ctx := context.Background()
	prefs, err := h.queries.GetEmailPreferencesByEmail(ctx, email)
	if err != nil {
		if err == sql.ErrNoRows {
			// Return default preferences
			return c.JSON(http.StatusOK, map[string]interface{}{
				"email":           email,
				"transactional":   true,
				"abandoned_cart":  true,
				"promotional":     false,
				"newsletter":      false,
				"product_updates": false,
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch preferences"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"email":           prefs.Email,
		"transactional":   prefs.Transactional.Valid && prefs.Transactional.Int64 == 1,
		"abandoned_cart":  prefs.AbandonedCart.Valid && prefs.AbandonedCart.Int64 == 1,
		"promotional":     prefs.Promotional.Valid && prefs.Promotional.Int64 == 1,
		"newsletter":      prefs.Newsletter.Valid && prefs.Newsletter.Int64 == 1,
		"product_updates": prefs.ProductUpdates.Valid && prefs.ProductUpdates.Int64 == 1,
	})
}

// HandleUpdateEmailPreferences updates the user's email preferences (JSON API)
func (h *EmailPreferencesHandler) HandleUpdateEmailPreferences(c echo.Context) error {
	var req struct {
		Email          string `json:"email"`
		AbandonedCart  bool   `json:"abandoned_cart"`
		Promotional    bool   `json:"promotional"`
		Newsletter     bool   `json:"newsletter"`
		ProductUpdates bool   `json:"product_updates"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	if req.Email == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Email required"})
	}

	ctx := context.Background()

	// Get existing preferences
	prefs, err := h.queries.GetEmailPreferencesByEmail(ctx, req.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Preferences not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch preferences"})
	}

	// Update preferences
	err = h.queries.UpdateEmailPreferences(ctx, db.UpdateEmailPreferencesParams{
		Transactional:  sql.NullInt64{Int64: 1, Valid: true}, // Always on
		AbandonedCart:  sql.NullInt64{Int64: boolToInt64(req.AbandonedCart), Valid: true},
		Promotional:    sql.NullInt64{Int64: boolToInt64(req.Promotional), Valid: true},
		Newsletter:     sql.NullInt64{Int64: boolToInt64(req.Newsletter), Valid: true},
		ProductUpdates: sql.NullInt64{Int64: boolToInt64(req.ProductUpdates), Valid: true},
		ID:             prefs.ID,
	})

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update preferences"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Preferences updated successfully"})
}

func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// HandleEmailPreferencesPage renders the user-facing email preferences page
func (h *EmailPreferencesHandler) HandleEmailPreferencesPage(c echo.Context) error {
	// Get user from auth middleware
	dbUser, ok := auth.GetDBUser(c)
	if !ok {
		return c.Redirect(http.StatusFound, "/login")
	}

	email := dbUser.Email
	if email == "" {
		return c.String(http.StatusBadRequest, "Email not found in user profile")
	}

	ctx := context.Background()

	// Get or create email preferences
	prefs, err := h.queries.GetEmailPreferencesByEmail(ctx, email)
	if err != nil {
		if err == sql.ErrNoRows {
			// Create default preferences
			token, _ := emailutil.GenerateUnsubscribeToken()
			prefs, err = h.queries.GetOrCreateEmailPreferences(ctx, db.GetOrCreateEmailPreferencesParams{
				ID:               ulid.Make().String(),
				UserID:           sql.NullString{String: dbUser.ID, Valid: true},
				Email:            email,
				UnsubscribeToken: sql.NullString{String: token, Valid: true},
			})
			if err != nil {
				return c.String(http.StatusInternalServerError, "Failed to create preferences")
			}
		} else {
			return c.String(http.StatusInternalServerError, "Failed to load preferences")
		}
	}

	// Build page metadata
	meta := layout.NewPageMeta(c, h.queries)
	meta.Title = "Email Preferences - Logan's 3D Creations"
	meta.Description = "Manage your email preferences"

	return account.EmailPreferences(c, &prefs, meta).Render(c.Request().Context(), c.Response().Writer)
}
