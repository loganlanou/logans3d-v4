package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/internal/auth"
	"github.com/loganlanou/logans3d-v4/storage"
	"github.com/loganlanou/logans3d-v4/storage/db"
	"github.com/oklog/ulid/v2"
)

// TestContext creates a new Echo context for testing
func NewTestContext(method, path string, body interface{}) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()

	var req *http.Request
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, bytes.NewReader(jsonBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath(path)

	return c, rec
}

// SetTestUser sets a user in the Echo context for authenticated tests
func SetTestUser(c echo.Context, user *db.User) {
	c.Set(auth.DBUserKey, user)
}

// CreateTestUser creates a test user in the database
func CreateTestUser(queries *db.Queries) (*db.User, error) {
	return CreateTestUserWithEmail(queries, "test@example.com")
}

// CreateTestUserWithEmail creates a test user with a specific email
func CreateTestUserWithEmail(queries *db.Queries, email string) (*db.User, error) {
	userID := ulid.Make().String()

	user, err := queries.CreateUser(context.Background(), db.CreateUserParams{
		ID:        userID,
		Email:     email,
		FirstName: sql.NullString{String: "Test", Valid: true},
		LastName:  sql.NullString{String: "User", Valid: true},
	})

	return &user, err
}

// NewTestDB creates a test database with migrations applied
func NewTestDB() (*sql.DB, *db.Queries, func()) {
	database, queries, cleanup, err := storage.NewTestDB()
	if err != nil {
		panic("failed to create test database: " + err.Error())
	}
	return database, queries, cleanup
}

// AssertJSONResponse checks if the response is valid JSON and returns the parsed body
func AssertJSONResponse(rec *httptest.ResponseRecorder) (map[string]interface{}, error) {
	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		return nil, err
	}
	return body, nil
}
