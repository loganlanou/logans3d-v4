package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/clerk/clerk-sdk-go/v2"
	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
	"github.com/labstack/echo/v4"
)

// ContextKey is the key type for storing values in context
type ContextKey string

const (
	// SessionClaimsKey is the context key for session claims
	SessionClaimsKey ContextKey = "clerk_session_claims"
	// UserIDKey is the context key for user ID
	UserIDKey ContextKey = "user_id"
)

// ClerkAuth is middleware that optionally authenticates requests using Clerk
// It uses Clerk's native HTTP middleware with automatic JWK caching
func ClerkAuth() echo.MiddlewareFunc {
	// Use Clerk's WithHeaderAuthorization middleware which handles JWK caching
	clerkMiddleware := clerkhttp.WithHeaderAuthorization()

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Request().URL.Path

			// Check if token exists before applying middleware
			token := getSessionToken(c.Request())
			fmt.Printf("[CLERK] %s - Token found: %v (len=%d)\n", path, token != "", len(token))

			if token == "" {
				// No token, continue without auth
				fmt.Printf("[CLERK] %s - No token, skipping auth\n", path)
				return next(c)
			}

			// Create a wrapped handler that extracts claims from Clerk middleware
			done := make(chan bool)
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Extract session claims from the request context (set by Clerk middleware)
				claims, ok := clerk.SessionClaimsFromContext(r.Context())
				fmt.Printf("[CLERK] %s - Claims extracted: ok=%v, claims=%v\n", path, ok, claims != nil)
				if ok && claims != nil {
					fmt.Printf("[CLERK] %s - User ID from claims: %s\n", path, claims.Subject)
					// Add to Echo context
					c.Set(string(SessionClaimsKey), claims)
					if claims.Subject != "" {
						c.Set(string(UserIDKey), claims.Subject)
					}
				} else {
					fmt.Printf("[CLERK] %s - Failed to extract claims from Clerk context\n", path)
				}
				close(done)
			})

			// Apply Clerk middleware
			clerkMiddleware(handler).ServeHTTP(c.Response(), c.Request())
			<-done

			return next(c)
		}
	}
}

// RequireAuth is middleware that requires authentication
// Redirects to home page if no valid session is found (prevents redirect loops)
func RequireAuth() echo.MiddlewareFunc {
	// Use Clerk's RequireHeaderAuthorization middleware
	clerkMiddleware := clerkhttp.RequireHeaderAuthorization()

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Check if token exists
			token := getSessionToken(c.Request())
			if token == "" {
				// Clear any existing session cookie to prevent loops
				c.SetCookie(&http.Cookie{
					Name:     "__session",
					Value:    "",
					Path:     "/",
					MaxAge:   -1,
					HttpOnly: true,
					Secure:   false, // Set to true in production with HTTPS
					SameSite: http.SameSiteLaxMode,
				})
				// Redirect to home page instead of login to prevent loops
				return c.Redirect(http.StatusFound, "/")
			}

			// Create a wrapped handler
			done := make(chan error)
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Extract session claims from the request context
				claims, ok := clerk.SessionClaimsFromContext(r.Context())
				if !ok || claims == nil {
					done <- echo.NewHTTPError(http.StatusUnauthorized, "Invalid session token")
					return
				}

				// Add to Echo context
				c.Set(string(SessionClaimsKey), claims)
				if claims.Subject != "" {
					c.Set(string(UserIDKey), claims.Subject)
				}
				close(done)
			})

			// Apply Clerk middleware
			clerkMiddleware(handler).ServeHTTP(c.Response(), c.Request())

			if err := <-done; err != nil {
				// Clear session cookie on validation error
				c.SetCookie(&http.Cookie{
					Name:     "__session",
					Value:    "",
					Path:     "/",
					MaxAge:   -1,
					HttpOnly: true,
					Secure:   false, // Set to true in production with HTTPS
					SameSite: http.SameSiteLaxMode,
				})
				// Redirect to home page instead of returning error
				return c.Redirect(http.StatusFound, "/")
			}

			return next(c)
		}
	}
}

// GetSessionClaims extracts session claims from the Echo context
func GetSessionClaims(c echo.Context) (*clerk.SessionClaims, bool) {
	claims, ok := c.Get(string(SessionClaimsKey)).(*clerk.SessionClaims)
	return claims, ok
}

// GetUserID extracts the user ID from the Echo context
func GetUserID(c echo.Context) (string, bool) {
	userID, ok := c.Get(string(UserIDKey)).(string)
	return userID, ok
}

// IsAuthenticated checks if the request is authenticated
func IsAuthenticated(c echo.Context) bool {
	_, ok := GetUserID(c)
	return ok
}

// getSessionToken extracts the session token from either the __session cookie or Authorization header
func getSessionToken(r *http.Request) string {
	// First, try to get from __session cookie
	cookie, err := r.Cookie("__session")
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// Fallback to Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		// Remove "Bearer " prefix if present
		if strings.HasPrefix(authHeader, "Bearer ") {
			return strings.TrimPrefix(authHeader, "Bearer ")
		}
		return authHeader
	}

	return ""
}
