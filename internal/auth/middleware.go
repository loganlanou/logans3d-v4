package auth

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/clerk/clerk-sdk-go/v2"
	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
	"github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/storage"
	"github.com/loganlanou/logans3d-v4/storage/db"
)

// Context keys for storing auth data
const (
	DBUserKey          = "db_user"
	IsAuthenticatedKey = "is_authenticated"
)

// ClerkHandshakeMiddleware processes Clerk's handshake to set session cookie for localhost
// This is needed because Clerk's JS SDK tries to set __session with Secure flag which browsers reject on HTTP
func ClerkHandshakeMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Check for __clerk_handshake parameter (Clerk's cookie-setting mechanism)
			handshake := c.QueryParam("__clerk_handshake")
			if handshake != "" {
				slog.Debug("=== HANDSHAKE: Processing Clerk handshake ===")
				// Extract and manually set session cookie without Secure flag for localhost
				if err := processClerkHandshake(c, handshake); err != nil {
					slog.Warn("=== HANDSHAKE: Failed to process handshake ===", "error", err)
				} else {
					// Redirect to the same URL without the handshake parameter to force reload
					// This ensures the server-side nav updates with the new session
					redirectURL := c.Request().URL.Path
					if redirectURL == "" {
						redirectURL = "/"
					}
					slog.Info("=== HANDSHAKE: Redirecting to update nav ===", "url", redirectURL)
					return c.Redirect(http.StatusFound, redirectURL)
				}
			}
			return next(c)
		}
	}
}

// ClerkAuthMiddleware verifies Clerk session tokens and loads user from DB
// This middleware is OPTIONAL - it allows unauthenticated requests through (matches corp's WithAuth)
func ClerkAuthMiddleware(storage *storage.Storage) echo.MiddlewareFunc {
	authMiddleware := clerkhttp.WithHeaderAuthorization(
		clerkhttp.AuthorizationJWTExtractor(extractSessionToken),
	)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// DEBUG: Log all cookies to see what Clerk might have set
			for _, cookie := range c.Request().Cookies() {
				slog.Debug("=== MIDDLEWARE: Cookie available ===", "name", cookie.Name, "value_prefix", cookie.Value[:min(len(cookie.Value), 20)])
			}

			handlerCalled := false

			// Wrap the Echo handler as http.Handler for clerkhttp middleware
			authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true

				// Check if Clerk successfully verified the session (optional - no failure if missing)
				if claims, ok := clerk.SessionClaimsFromContext(r.Context()); ok {
					slog.Debug("=== MIDDLEWARE: Session claims verified ===", "user_id", claims.Subject)

					// Get or create user from Clerk user ID
					dbUser, err := getOrCreateUser(r.Context(), storage, claims.Subject)
					if err != nil {
						slog.Error("=== MIDDLEWARE: Failed to get/create user ===", "error", err)
						c.Set(IsAuthenticatedKey, false)
						next(c)
						return
					}

					slog.Debug("=== MIDDLEWARE: User authenticated ===", "user_id", dbUser.ID, "email", dbUser.Email)

					// Store user in Echo context
					c.Set(DBUserKey, dbUser)
					c.Set(IsAuthenticatedKey, true)
					c.SetRequest(r)
				} else {
					slog.Debug("=== MIDDLEWARE: No valid session claims ===")
					c.Set(IsAuthenticatedKey, false)
				}

				next(c)
			})).ServeHTTP(c.Response().Writer, c.Request())

			// If handler wasn't called, clerkhttp silently rejected the token
			// This can happen after restarts when JWKS cache is cold
			// Allow through as unauthenticated
			if !handlerCalled {
				slog.Warn("=== MIDDLEWARE: Clerk did not call handler - allowing as unauthenticated ===")
				c.Set(IsAuthenticatedKey, false)
				return next(c)
			}

			return nil
		}
	}
}

// processClerkHandshake extracts session JWT from Clerk's handshake and sets it as a cookie
func processClerkHandshake(c echo.Context, handshakeJWT string) error {
	// Decode the handshake JWT to extract cookie instructions
	parts := strings.Split(handshakeJWT, ".")
	if len(parts) != 3 {
		return fmt.Errorf("invalid handshake JWT format")
	}

	// Decode the payload (second part)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("failed to decode handshake payload: %w", err)
	}

	var handshakeData struct {
		Handshake []string `json:"handshake"`
	}
	if err := json.Unmarshal(payload, &handshakeData); err != nil {
		return fmt.Errorf("failed to parse handshake payload: %w", err)
	}

	// Find the __session cookie instruction
	for _, instruction := range handshakeData.Handshake {
		if strings.HasPrefix(instruction, "__session=") {
			// Extract the session token value (everything between = and first ;)
			parts := strings.SplitN(instruction, "=", 2)
			if len(parts) != 2 {
				continue
			}

			valueParts := strings.SplitN(parts[1], ";", 2)
			sessionToken := valueParts[0]

			slog.Debug("=== HANDSHAKE: Extracted session token ===", "token_prefix", sessionToken[:min(len(sessionToken), 20)])

			// Set the __session cookie with Secure: false for localhost
			c.SetCookie(&http.Cookie{
				Name:     "__session",
				Value:    sessionToken,
				Path:     "/",
				HttpOnly: true,
				Secure:   false, // Allow on HTTP for localhost
				SameSite: http.SameSiteLaxMode,
				MaxAge:   31536000, // 1 year (same as Clerk's default)
			})

			slog.Info("=== HANDSHAKE: Set __session cookie for localhost ===")
			return nil
		}
	}

	return fmt.Errorf("no __session cookie found in handshake")
}

// extractSessionToken gets the token from multiple sources (pattern from corp project)
func extractSessionToken(r *http.Request) string {
	// Try Clerk-Session header
	if token := strings.TrimSpace(r.Header.Get("Clerk-Session")); token != "" {
		slog.Debug("=== EXTRACT: Found token in Clerk-Session header ===")
		return token
	}

	// Try Authorization header
	if auth := strings.TrimSpace(r.Header.Get("Authorization")); auth != "" {
		slog.Debug("=== EXTRACT: Found token in Authorization header ===")
		if len(auth) > 7 && strings.EqualFold(auth[:7], "Bearer ") {
			return strings.TrimSpace(auth[7:])
		}
		return auth
	}

	// Try __session cookie (standard - set by Clerk JS SDK)
	if cookie, err := r.Cookie("__session"); err == nil && cookie.Value != "" {
		slog.Debug("=== EXTRACT: Found token in __session cookie ===", "token_prefix", cookie.Value[:min(len(cookie.Value), 20)])
		return cookie.Value
	}

	// Try __client cookie (fallback)
	if cookie, err := r.Cookie("__client"); err == nil && cookie.Value != "" {
		slog.Debug("=== EXTRACT: Found token in __client cookie ===")
		return cookie.Value
	}

	slog.Debug("=== EXTRACT: No session token found in any source ===")
	return ""
}


// getOrCreateUser fetches user from Clerk API and syncs to DB (pattern from corp project)
func getOrCreateUser(ctx context.Context, storage *storage.Storage, clerkUserID string) (*db.User, error) {
	// Try to find user by Clerk ID first (fastest path)
	dbUser, err := storage.Queries.GetUserByClerkID(ctx, sql.NullString{
		String: clerkUserID,
		Valid:  true,
	})
	if err == nil {
		slog.Debug("=== MIDDLEWARE: User found in database ===", "user_id", dbUser.ID)
		return &dbUser, nil
	}

	slog.Debug("=== MIDDLEWARE: User not in database, fetching from Clerk ===", "clerk_id", clerkUserID)

	// User not in DB - fetch full details from Clerk API
	userClient := user.NewClient(&clerk.ClientConfig{})
	clerkUser, err := userClient.Get(ctx, clerkUserID)
	if err != nil {
		return nil, err
	}

	// Sync user to database using Clerk API data
	syncedUser, err := syncUserToDatabase(ctx, storage, clerkUser)
	if err != nil {
		return nil, err
	}

	slog.Debug("=== MIDDLEWARE: User synced successfully ===", "user_id", syncedUser.ID, "clerk_id", clerkUserID)
	return syncedUser, nil
}

// clearAuthCookie clears the invalid session cookie
func clearAuthCookie(c echo.Context) {
	c.SetCookie(&http.Cookie{
		Name:     "__session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})
}

// RequireClerkAuth middleware requires authentication
func RequireClerkAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			isAuth, _ := c.Get(IsAuthenticatedKey).(bool)
			if !isAuth {
				return c.Redirect(http.StatusFound, "/login")
			}
			return next(c)
		}
	}
}


