package auth

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/storage"
	"github.com/loganlanou/logans3d-v4/storage/db"
)

// CustomSessionClaims contains the custom claims we've added to the Clerk session token
type CustomSessionClaims struct {
	UserID              string `json:"userId"`
	ExternalID          string `json:"externalId"`
	FirstName           string `json:"firstName"`
	LastName            string `json:"lastName"`
	FullName            string `json:"fullName"`
	Username            string `json:"username"`
	CreatedAt           string `json:"createdAt"`
	UpdatedAt           string `json:"updatedAt"`
	LastSignInAt        string `json:"lastSignInAt"`
	PrimaryEmailAddress string `json:"primaryEmailAddress"`
	PrimaryPhoneNumber  string `json:"primaryPhoneNumber"`
	PrimaryWeb3Wallet   string `json:"primaryWeb3Wallet"`
	EmailVerified       bool   `json:"emailVerified"`
	PhoneNumberVerified bool   `json:"phoneNumberVerified"`
	ImageURL            string `json:"imageUrl"`
	HasImage            bool   `json:"hasImage"`
	TwoFactorEnabled    bool   `json:"twoFactorEnabled"`
	PublicMetadata      string `json:"publicMetadata"`
	UnsafeMetadata      string `json:"unsafeMetadata"`
}

// GetDBUser retrieves the database user from context
func GetDBUser(c echo.Context) (*db.User, bool) {
	dbUser, ok := c.Get(DBUserKey).(*db.User)
	return dbUser, ok && dbUser != nil
}

// IsAuthenticated checks if the current request is authenticated
func IsAuthenticated(c echo.Context) bool {
	isAuth, _ := c.Get(IsAuthenticatedKey).(bool)
	return isAuth
}

// GetUserID gets the user ID from the database user
func GetUserID(c echo.Context) (string, bool) {
	if dbUser, ok := GetDBUser(c); ok {
		return dbUser.ID, true
	}
	return "", false
}

// GetClerkID gets the Clerk user ID from the database user
func GetClerkID(c echo.Context) (string, bool) {
	if dbUser, ok := GetDBUser(c); ok {
		return nullStringValue(dbUser.ClerkID), true
	}
	return "", false
}

// IsAdmin checks if the current user is an admin
func IsAdmin(c echo.Context) bool {
	if dbUser, ok := GetDBUser(c); ok {
		return dbUser.IsAdmin
	}
	return false
}

// RequireAuth is a helper that checks auth and returns error if not authenticated
// Use this in handlers that need auth
func RequireAuth(c echo.Context) error {
	if !IsAuthenticated(c) {
		return echo.NewHTTPError(401, "Authentication required")
	}
	return nil
}

// stringValue safely dereferences a string pointer
func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// nullStringValue safely converts a sql.NullString to string
func nullStringValue(s sql.NullString) string {
	if s.Valid {
		return s.String
	}
	return ""
}

// toNullString converts a string to sql.NullString
func toNullString(s string) sql.NullString {
	return sql.NullString{
		String: s,
		Valid:  s != "",
	}
}

// getFirstEmail extracts the primary email from a Clerk user
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

// buildFullName constructs a full name from available user data
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

// syncUserFromClaims upserts user data from custom session claims to local database
func syncUserFromClaims(ctx context.Context, storage *storage.Storage, clerkUserID string, claims *CustomSessionClaims) (*db.User, error) {
	// Generate or use existing UUID
	userID := uuid.New().String()

	// Check if user exists
	existingUser, err := storage.Queries.GetUserByClerkID(ctx, toNullString(clerkUserID))
	if err == nil {
		// User exists, use their ID
		userID = existingUser.ID
	}

	// Upsert user using data from custom claims
	dbUser, err := storage.Queries.UpsertUserByClerkID(ctx, db.UpsertUserByClerkIDParams{
		ID:              userID,
		ClerkID:         toNullString(clerkUserID),
		Email:           claims.PrimaryEmailAddress,
		FirstName:       toNullString(claims.FirstName),
		LastName:        toNullString(claims.LastName),
		FullName:        claims.FullName,
		Username:        toNullString(claims.Username),
		ProfileImageUrl: toNullString(claims.ImageURL),
	})

	if err != nil {
		return nil, err
	}

	// Associate any existing email history records with this user
	rowsAffected, err := storage.Queries.AssociateEmailHistoryWithUser(ctx, db.AssociateEmailHistoryWithUserParams{
		UserID:         toNullString(dbUser.ID),
		RecipientEmail: claims.PrimaryEmailAddress,
	})
	if err != nil {
		slog.Error("failed to associate email history with user", "error", err, "user_id", dbUser.ID, "email", claims.PrimaryEmailAddress)
	} else if rowsAffected > 0 {
		slog.Info("associated email history with user", "user_id", dbUser.ID, "email", claims.PrimaryEmailAddress, "rows_affected", rowsAffected)
	}

	return &dbUser, nil
}

// syncUserToDatabase upserts Clerk user data to local database (legacy - uses Clerk API)
func syncUserToDatabase(ctx context.Context, storage *storage.Storage, clerkUser *clerk.User) (*db.User, error) {
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
		return nil, err
	}

	// Associate any existing email history records with this user
	rowsAffected, err := storage.Queries.AssociateEmailHistoryWithUser(ctx, db.AssociateEmailHistoryWithUserParams{
		UserID:         toNullString(dbUser.ID),
		RecipientEmail: email,
	})
	if err != nil {
		slog.Error("failed to associate email history with user", "error", err, "user_id", dbUser.ID, "email", email)
	} else if rowsAffected > 0 {
		slog.Info("associated email history with user", "user_id", dbUser.ID, "email", email, "rows_affected", rowsAffected)
	}

	return &dbUser, nil
}
