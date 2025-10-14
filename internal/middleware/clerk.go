package middleware

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/clerk/clerk-sdk-go/v2"
	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
	"github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/storage"
	"github.com/loganlanou/logans3d-v4/storage/db"
)

// Context keys for storing auth data
const (
	ClerkUserKey     = "clerk_user"
	ClerkClaimsKey   = "clerk_claims"
	DBUserKey        = "db_user"
	IsAuthenticatedKey = "is_authenticated"
)

// ClerkAuthMiddleware handles Clerk authentication with the following flow:
// 1. Extracts session token from __session cookie
// 2. Validates token using Clerk's JWT verification
// 3. Syncs user data to database
// 4. Populates Echo context with auth data
func ClerkAuthMiddleware(storage *storage.Storage) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Request().URL.Path

			// Extract session token from __session cookie
			token := extractSessionToken(c.Request())

			if token == "" {
				// No token - continue as unauthenticated
				slog.Debug("No Clerk session token found", "path", path)
				c.Set(IsAuthenticatedKey, false)
				return next(c)
			}

			// Add token to Authorization header for Clerk middleware
			c.Request().Header.Set("Authorization", "Bearer "+token)

			// Use Clerk's middleware to verify and extract claims
			done := make(chan error, 1)
			clerkMiddleware := clerkhttp.WithHeaderAuthorization()

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Extract claims from Clerk middleware
				claims, ok := clerk.SessionClaimsFromContext(r.Context())
				if !ok || claims == nil {
					slog.Warn("Failed to extract Clerk claims", "path", path)
					done <- fmt.Errorf("invalid session")
					return
				}

				slog.Info("Clerk session validated", "path", path, "user_id", claims.Subject)

				// Fetch full user data from Clerk API
				clerkUser, err := user.Get(context.Background(), claims.Subject)
				if err != nil {
					slog.Error("Failed to fetch Clerk user", "error", err, "user_id", claims.Subject)
					done <- err
					return
				}

				// Sync user to database
				dbUser, err := syncUserToDatabase(storage, clerkUser)
				if err != nil {
					slog.Error("Failed to sync user to database", "error", err, "clerk_id", clerkUser.ID)
					done <- err
					return
				}

				// Store in Echo context
				c.Set(ClerkClaimsKey, claims)
				c.Set(ClerkUserKey, clerkUser)
				c.Set(DBUserKey, dbUser)
				c.Set(IsAuthenticatedKey, true)

				slog.Info("User authenticated and synced",
					"path", path,
					"clerk_id", clerkUser.ID,
					"db_id", dbUser.ID,
					"email", dbUser.Email)

				done <- nil
			})

			// Apply Clerk middleware
			clerkMiddleware(handler).ServeHTTP(c.Response(), c.Request())

			// Check result
			if err := <-done; err != nil {
				// Authentication failed - continue as unauthenticated
				slog.Warn("Authentication failed", "error", err, "path", path)
				c.Set(IsAuthenticatedKey, false)
				return next(c)
			}

			return next(c)
		}
	}
}

// RequireClerkAuth middleware requires authentication and returns 401 if not authenticated
func RequireClerkAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			isAuth, _ := c.Get(IsAuthenticatedKey).(bool)

			if !isAuth {
				// Redirect to login page
				return c.Redirect(http.StatusFound, "/login")
			}

			return next(c)
		}
	}
}

// extractSessionToken gets the Clerk session token from __session cookie
func extractSessionToken(r *http.Request) string {
	// First try the __session cookie
	cookie, err := r.Cookie("__session")
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// Fallback to Authorization header (for API requests)
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

// syncUserToDatabase upserts the Clerk user data to the local database
func syncUserToDatabase(storage *storage.Storage, clerkUser *clerk.User) (*db.User, error) {
	ctx := context.Background()

	// Extract user data
	email := getFirstEmail(clerkUser)
	firstName := stringValue(clerkUser.FirstName)
	lastName := stringValue(clerkUser.LastName)
	username := stringValue(clerkUser.Username)
	imageURL := stringValue(clerkUser.ImageURL)

	// Build full name
	fullName := buildFullName(firstName, lastName, username, email)

	// Generate or use existing UUID
	userID := uuid.New().String()

	// Check if user exists
	existingUser, err := storage.Queries.GetUserByClerkID(ctx, toNullString(clerkUser.ID))
	if err == nil {
		// User exists, use their ID
		userID = existingUser.ID
	}

	// Upsert user
	dbUser, err := storage.Queries.UpsertUserByClerkID(ctx, db.UpsertUserByClerkIDParams{
		ID:              userID,
		ClerkID:         toNullString(clerkUser.ID),
		Email:           email,
		FirstName:       toNullString(firstName),
		LastName:        toNullString(lastName),
		FullName:        fullName,
		Username:        toNullString(username),
		ProfileImageUrl: toNullString(imageURL),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to upsert user: %w", err)
	}

	return &dbUser, nil
}

// Helper functions

func getFirstEmail(clerkUser *clerk.User) string {
	if len(clerkUser.EmailAddresses) == 0 {
		return ""
	}

	// Try to find primary email
	primaryID := stringValue(clerkUser.PrimaryEmailAddressID)
	for _, email := range clerkUser.EmailAddresses {
		if email.ID == primaryID {
			return email.EmailAddress
		}
	}

	// Fallback to first email
	return clerkUser.EmailAddresses[0].EmailAddress
}

func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func buildFullName(firstName, lastName, username, email string) string {
	if firstName != "" && lastName != "" {
		return firstName + " " + lastName
	}
	if firstName != "" {
		return firstName
	}
	if lastName != "" {
		return lastName
	}
	if username != "" {
		return username
	}
	if email != "" {
		return email
	}
	return "User"
}

func toNullString(s string) sql.NullString {
	return sql.NullString{
		String: s,
		Valid:  s != "",
	}
}
