package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/labstack/echo/v4"
	"log/slog"
)

// AuthHandler handles authentication routes
type AuthHandler struct {
	clerkDomain string
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		clerkDomain: os.Getenv("CLERK_FRONTEND_API"), // e.g., "clerk.your-domain.com"
	}
}

// HandleLogin redirects to Clerk's hosted sign-in page
func (h *AuthHandler) HandleLogin(c echo.Context) error {
	// Get redirect URL from query params (where to go after login)
	redirectURL := c.QueryParam("redirect_url")
	if redirectURL == "" {
		redirectURL = "/"
	}

	// Build Clerk sign-in URL
	clerkPublishableKey := os.Getenv("CLERK_PUBLISHABLE_KEY")
	signInURL := fmt.Sprintf("https://accounts.clerk.dev/sign-in?redirect_url=%s",
		url.QueryEscape(h.buildCallbackURL(c, redirectURL)))

	slog.Info("Redirecting to Clerk sign-in", "redirect_url", redirectURL)

	// Add publishable key as parameter if using Clerk's hosted pages
	if clerkPublishableKey != "" {
		signInURL += "&publishable_key=" + clerkPublishableKey
	}

	return c.Redirect(http.StatusFound, signInURL)
}

// HandleSignUp redirects to Clerk's hosted sign-up page
func (h *AuthHandler) HandleSignUp(c echo.Context) error {
	// Get redirect URL from query params
	redirectURL := c.QueryParam("redirect_url")
	if redirectURL == "" {
		redirectURL = "/"
	}

	// Build Clerk sign-up URL
	clerkPublishableKey := os.Getenv("CLERK_PUBLISHABLE_KEY")
	signUpURL := fmt.Sprintf("https://accounts.clerk.dev/sign-up?redirect_url=%s",
		url.QueryEscape(h.buildCallbackURL(c, redirectURL)))

	slog.Info("Redirecting to Clerk sign-up", "redirect_url", redirectURL)

	// Add publishable key as parameter
	if clerkPublishableKey != "" {
		signUpURL += "&publishable_key=" + clerkPublishableKey
	}

	return c.Redirect(http.StatusFound, signUpURL)
}

// HandleAuthCallback handles the redirect back from Clerk after authentication
// Clerk will have set the __session cookie with the JWT token
func (h *AuthHandler) HandleAuthCallback(c echo.Context) error {
	// Get the final redirect destination
	redirectURL := c.QueryParam("redirect_url")
	if redirectURL == "" {
		redirectURL = "/"
	}

	// Check if token exists (Clerk sets __session cookie)
	cookie, err := c.Cookie("__session")
	if err != nil || cookie.Value == "" {
		slog.Error("No Clerk session cookie found after callback")
		return c.Redirect(http.StatusFound, "/login?error=no_session")
	}

	slog.Info("Auth callback successful, redirecting", "redirect_url", redirectURL)

	// Redirect to the intended destination
	// The middleware will pick up the cookie and authenticate the user
	return c.Redirect(http.StatusFound, redirectURL)
}

// HandleLogout logs out the user from both Clerk and the application
func (h *AuthHandler) HandleLogout(c echo.Context) error {
	slog.Info("User logging out", "path", c.Request().URL.Path)

	// Clear the __session cookie
	c.SetCookie(&http.Cookie{
		Name:     "__session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	})

	// Build Clerk sign-out URL (this signs out from Clerk's side)
	clerkPublishableKey := os.Getenv("CLERK_PUBLISHABLE_KEY")
	signOutURL := fmt.Sprintf("https://accounts.clerk.dev/sign-out?redirect_url=%s",
		url.QueryEscape(h.buildBaseURL(c)))

	if clerkPublishableKey != "" {
		signOutURL += "&publishable_key=" + clerkPublishableKey
	}

	// Redirect to Clerk sign-out, which will then redirect back to home
	return c.Redirect(http.StatusFound, signOutURL)
}

// HandleAccount redirects to Clerk's user profile management page
func (h *AuthHandler) HandleAccount(c echo.Context) error {
	clerkPublishableKey := os.Getenv("CLERK_PUBLISHABLE_KEY")
	accountURL := fmt.Sprintf("https://accounts.clerk.dev/user?redirect_url=%s",
		url.QueryEscape(h.buildBaseURL(c)))

	if clerkPublishableKey != "" {
		accountURL += "&publishable_key=" + clerkPublishableKey
	}

	return c.Redirect(http.StatusFound, accountURL)
}

// buildCallbackURL constructs the callback URL for OAuth redirects
func (h *AuthHandler) buildCallbackURL(c echo.Context, finalRedirect string) string {
	baseURL := h.buildBaseURL(c)
	return fmt.Sprintf("%s/auth/callback?redirect_url=%s", baseURL, url.QueryEscape(finalRedirect))
}

// buildBaseURL constructs the base URL for the application
func (h *AuthHandler) buildBaseURL(c echo.Context) string {
	scheme := "http"
	if c.Request().TLS != nil {
		scheme = "https"
	}

	// Check for forwarded protocol
	if proto := c.Request().Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}

	return fmt.Sprintf("%s://%s", scheme, c.Request().Host)
}
