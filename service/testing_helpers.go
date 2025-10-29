package service

import (
	"context"
	"database/sql"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/internal/auth"
	"github.com/loganlanou/logans3d-v4/internal/email"
	"github.com/loganlanou/logans3d-v4/internal/handlers"
	"github.com/loganlanou/logans3d-v4/storage"
	"github.com/loganlanou/logans3d-v4/storage/db"
	"github.com/oklog/ulid/v2"
)

// setupTestService creates a service instance with an in-memory database for testing
func setupTestService(t *testing.T) *Service {
	t.Helper()

	// Create test database
	_, queries, cleanup, err := storage.NewTestDB()
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	t.Cleanup(cleanup)

	// Create storage (DB field is private, so we can't set it directly)
	// We'll use the queries which is what we need for testing
	store := &storage.Storage{
		Queries: queries,
	}

	// Initialize email service for tests
	emailService := email.NewService(queries)

	// Create service with minimal config
	svc := &Service{
		storage:         store,
		emailService:    emailService,
		paymentHandler:  handlers.NewPaymentHandler(queries, emailService),
		authHandler:     handlers.NewAuthHandler(),
		shippingService: nil, // Not needed for route testing
		shippingHandler: nil, // Not needed for route testing
		config: &Config{
			Environment: "test",
			Port:        "8080",
		},
	}

	return svc
}

// setupTestEcho creates an Echo instance with routes registered
func setupTestEcho(t *testing.T) (*echo.Echo, *Service) {
	t.Helper()

	e := echo.New()
	// Disable Echo's default error handler for cleaner test output
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		// Just set status code, don't write response
		if he, ok := err.(*echo.HTTPError); ok {
			c.Response().WriteHeader(he.Code)
		} else {
			c.Response().WriteHeader(500)
		}
	}

	svc := setupTestService(t)
	svc.RegisterRoutes(e)

	return e, svc
}

// createTestUser creates a test user in the database
// Note: For route testing, we don't need to actually test admin permissions,
// just that routes exist. Admin middleware testing should be done separately.
func createTestUser(t *testing.T, queries *db.Queries, email string) *db.User {
	t.Helper()

	userID := ulid.Make().String()
	user, err := queries.CreateUser(context.Background(), db.CreateUserParams{
		ID:        userID,
		Email:     email,
		FirstName: sql.NullString{String: "Test", Valid: true},
		LastName:  sql.NullString{String: "User", Valid: true},
		FullName:  "Test User",
	})
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	return &user
}

// setAuthUser sets an authenticated user in the Echo context (simulates auth middleware)
func setAuthUser(c echo.Context, user *db.User) {
	c.Set(auth.DBUserKey, user)
	c.Set(auth.IsAuthenticatedKey, true)
}
