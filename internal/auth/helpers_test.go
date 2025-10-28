package auth

import (
	"database/sql"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/storage/db"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
)

func TestGetDBUser_Found(t *testing.T) {
	e := echo.New()
	c := e.NewContext(nil, nil)

	// Create test user
	testUser := &db.User{
		ID:        ulid.Make().String(),
		Email:     "test@example.com",
		FirstName: sql.NullString{String: "Test", Valid: true},
		LastName:  sql.NullString{String: "User", Valid: true},
	}

	// Set user in context using correct key
	c.Set(DBUserKey, testUser)

	// Get user
	user, ok := GetDBUser(c)

	assert.True(t, ok, "Should find user in context")
	assert.NotNil(t, user)
	assert.Equal(t, testUser.ID, user.ID)
	assert.Equal(t, testUser.Email, user.Email)
}

func TestGetDBUser_NotFound(t *testing.T) {
	e := echo.New()
	c := e.NewContext(nil, nil)

	// Don't set anything in context

	user, ok := GetDBUser(c)

	assert.False(t, ok, "Should not find user in context")
	assert.Nil(t, user)
}

func TestGetDBUser_WrongKey(t *testing.T) {
	e := echo.New()
	c := e.NewContext(nil, nil)

	// Set user with wrong key (this was Bug #1)
	testUser := &db.User{
		ID:    ulid.Make().String(),
		Email: "test@example.com",
	}
	c.Set("user", testUser) // Wrong key!

	user, ok := GetDBUser(c)

	// Should return false because it's looking for DBUserKey, not "user"
	assert.False(t, ok, "Should not find user with wrong key")
	assert.Nil(t, user)
}

func TestGetDBUser_WrongType(t *testing.T) {
	e := echo.New()
	c := e.NewContext(nil, nil)

	// Set wrong type in context
	c.Set(DBUserKey, "not a user")

	user, ok := GetDBUser(c)

	assert.False(t, ok, "Should not cast wrong type")
	assert.Nil(t, user)
}

func TestIsAuthenticated_True(t *testing.T) {
	e := echo.New()
	c := e.NewContext(nil, nil)

	// IsAuthenticated checks the IsAuthenticatedKey, not DBUserKey
	c.Set(IsAuthenticatedKey, true)

	assert.True(t, IsAuthenticated(c))
}

func TestIsAuthenticated_False(t *testing.T) {
	e := echo.New()
	c := e.NewContext(nil, nil)

	// No user in context
	assert.False(t, IsAuthenticated(c))
}

func TestIsAdmin_True(t *testing.T) {
	e := echo.New()
	c := e.NewContext(nil, nil)

	testUser := &db.User{
		ID:      ulid.Make().String(),
		Email:   "admin@example.com",
		IsAdmin: true,
	}
	c.Set(DBUserKey, testUser)

	assert.True(t, IsAdmin(c))
}

func TestIsAdmin_False_NotAdmin(t *testing.T) {
	e := echo.New()
	c := e.NewContext(nil, nil)

	testUser := &db.User{
		ID:      ulid.Make().String(),
		Email:   "user@example.com",
		IsAdmin: false,
	}
	c.Set(DBUserKey, testUser)

	assert.False(t, IsAdmin(c))
}

func TestIsAdmin_False_NoUser(t *testing.T) {
	e := echo.New()
	c := e.NewContext(nil, nil)

	// No user in context
	assert.False(t, IsAdmin(c))
}

// Test that DBUserKey constant has the expected value
// This test would have caught Bug #1 where code was using hardcoded "user" instead of DBUserKey
func TestDBUserKey_Constant(t *testing.T) {
	assert.Equal(t, "db_user", DBUserKey, "DBUserKey constant should be 'db_user'")
}
