package handlers

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/views/auth"
)

// AuthHandler handles authentication routes
type AuthHandler struct {
	publishableKey string
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		publishableKey: os.Getenv("CLERK_PUBLISHABLE_KEY"),
	}
}

// HandleLogin renders the Clerk sign-in component page
func (h *AuthHandler) HandleLogin(c echo.Context) error {
	// Get redirect URL from query params (where to go after login)
	redirectURL := c.QueryParam("redirect_url")
	if redirectURL == "" {
		redirectURL = "/"
	}

	slog.Info("Rendering sign-in page", "redirect_url", redirectURL)

	// Render the sign-in template with Clerk JS SDK
	return auth.SignIn(h.publishableKey, redirectURL).Render(c.Request().Context(), c.Response().Writer)
}

// HandleSignUp renders the Clerk sign-up component page
func (h *AuthHandler) HandleSignUp(c echo.Context) error {
	slog.Info("Rendering sign-up page")

	// Render the sign-up template with Clerk JS SDK
	return auth.SignUp(h.publishableKey).Render(c.Request().Context(), c.Response().Writer)
}

// HandleLogout logs out the user using Clerk JS SDK
func (h *AuthHandler) HandleLogout(c echo.Context) error {
	slog.Debug("=== LOGOUT: Starting logout process ===", "path", c.Request().URL.Path)

	// Log all current cookies before clearing
	slog.Debug("=== LOGOUT: Current cookies before clearing ===")
	for _, cookie := range c.Request().Cookies() {
		slog.Debug("=== LOGOUT: Cookie ===", "name", cookie.Name, "value_prefix", cookie.Value[:min(len(cookie.Value), 20)])
	}

	// Clear all Clerk cookies server-side FIRST
	clearClerkCookies(c)

	slog.Debug("=== LOGOUT: Rendering sign-out page ===")
	// Render the sign-out template which will handle Clerk signOut() via JS SDK
	return auth.SignOut(h.publishableKey).Render(c.Request().Context(), c.Response().Writer)
}

// clearClerkCookies clears all Clerk-related cookies
func clearClerkCookies(c echo.Context) {
	cookiesToClear := []string{"__session", "__clerk_db_jwt", "__client_uat", "__client"}

	slog.Debug("=== LOGOUT: Clearing cookies server-side ===")
	for _, name := range cookiesToClear {
		c.SetCookie(&http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   false,
			SameSite: http.SameSiteLaxMode,
		})
		slog.Debug("=== LOGOUT: Cleared cookie ===", "name", name)
	}
}
