package auth

import (
	"database/sql"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/internal/middleware"
	"github.com/loganlanou/logans3d-v4/storage/db"
)

// GetAuthContext returns authentication context for templates
// This provides ALL auth data needed by templates in a single call
func GetAuthContext(c echo.Context) *Context {
	isAuth, _ := c.Get(middleware.IsAuthenticatedKey).(bool)

	if !isAuth {
		return &Context{
			IsAuthenticated: false,
			User:            nil,
		}
	}

	// Get database user
	dbUser, ok := c.Get(middleware.DBUserKey).(*db.User)
	if !ok || dbUser == nil {
		return &Context{
			IsAuthenticated: false,
			User:            nil,
		}
	}

	// Convert to template-friendly format
	userData := &UserData{
		ID:        dbUser.ID,
		Email:     dbUser.Email,
		FirstName: stringValue(dbUser.FirstName),
		LastName:  stringValue(dbUser.LastName),
		FullName:  dbUser.FullName,
		ImageURL:  stringValue(dbUser.ProfileImageUrl),
		Username:  stringValue(dbUser.Username),
		HasImage:  stringValue(dbUser.ProfileImageUrl) != "",
	}

	return &Context{
		IsAuthenticated: true,
		User:            userData,
	}
}

// GetClerkUser retrieves the full Clerk user object from context
func GetClerkUser(c echo.Context) (*clerk.User, bool) {
	clerkUser, ok := c.Get(middleware.ClerkUserKey).(*clerk.User)
	return clerkUser, ok && clerkUser != nil
}

// GetDBUser retrieves the database user from context
func GetDBUser(c echo.Context) (*db.User, bool) {
	dbUser, ok := c.Get(middleware.DBUserKey).(*db.User)
	return dbUser, ok && dbUser != nil
}

// GetClerkClaims retrieves the Clerk session claims from context
func GetClerkClaims(c echo.Context) (*clerk.SessionClaims, bool) {
	claims, ok := c.Get(middleware.ClerkClaimsKey).(*clerk.SessionClaims)
	return claims, ok && claims != nil
}

// IsAuthenticated checks if the current request is authenticated
func IsAuthenticated(c echo.Context) bool {
	isAuth, _ := c.Get(middleware.IsAuthenticatedKey).(bool)
	return isAuth
}

// GetUserID gets the user ID from the database user (preferred) or Clerk claims
func GetUserID(c echo.Context) (string, bool) {
	// Try database user first
	if dbUser, ok := GetDBUser(c); ok {
		return dbUser.ID, true
	}

	// Fallback to Clerk claims
	if claims, ok := GetClerkClaims(c); ok {
		return claims.Subject, true
	}

	return "", false
}

// GetClerkID gets the Clerk user ID
func GetClerkID(c echo.Context) (string, bool) {
	if claims, ok := GetClerkClaims(c); ok {
		return claims.Subject, true
	}
	return "", false
}

// RequireAuth is a helper that checks auth and returns error if not authenticated
// Use this in handlers that need auth
func RequireAuth(c echo.Context) error {
	if !IsAuthenticated(c) {
		return echo.NewHTTPError(401, "Authentication required")
	}
	return nil
}

// stringValue safely converts a sql.NullString to string
func stringValue(s sql.NullString) string {
	if s.Valid {
		return s.String
	}
	return ""
}
