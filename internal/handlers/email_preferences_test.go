package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"testing"

	"github.com/loganlanou/logans3d-v4/storage/db"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEmailPreferences_DatabaseIntegration tests the database integration for email preferences
// This test verifies Bug #1 fix - the GetDBUser integration
func TestEmailPreferences_DatabaseIntegration(t *testing.T) {
	_, queries, cleanup := NewTestDB()
	defer cleanup()

	ctx := context.Background()

	// Create test user
	user, err := CreateTestUser(queries)
	require.NoError(t, err)

	// Verify no preferences exist initially
	_, err = queries.GetEmailPreferencesByEmail(ctx, user.Email)
	assert.Error(t, err) // Should be sql.ErrNoRows

	// Create preferences using GetOrCreateEmailPreferences
	prefs, err := queries.GetOrCreateEmailPreferences(ctx, db.GetOrCreateEmailPreferencesParams{
		ID:               ulid.Make().String(),
		UserID:           sql.NullString{String: user.ID, Valid: true},
		Email:            user.Email,
		UnsubscribeToken: sql.NullString{String: ulid.Make().String(), Valid: true},
	})
	require.NoError(t, err)

	// Verify preferences were created with correct user linkage
	assert.Equal(t, user.Email, prefs.Email)
	assert.Equal(t, user.ID, prefs.UserID.String)
	assert.True(t, prefs.UnsubscribeToken.Valid)
}

// TestHandleGetEmailPreferences_Success tests getting preferences via API
func TestHandleGetEmailPreferences_Success(t *testing.T) {
	_, queries, cleanup := NewTestDB()
	defer cleanup()

	handler := NewEmailPreferencesHandler(queries)

	// Create test user
	user, err := CreateTestUser(queries)
	require.NoError(t, err)

	// Create email preferences
	ctx := context.Background()
	_, err = queries.GetOrCreateEmailPreferences(ctx, db.GetOrCreateEmailPreferencesParams{
		ID:               ulid.Make().String(),
		UserID:           sql.NullString{String: user.ID, Valid: true},
		Email:            user.Email,
		UnsubscribeToken: sql.NullString{String: ulid.Make().String(), Valid: true},
	})
	require.NoError(t, err)

	// Create context with authenticated user
	c, rec := NewTestContext(http.MethodGet, "/api/email-preferences?email="+user.Email, nil)
	SetTestUser(c, user)
	c.QueryParams().Set("email", user.Email)

	// Call handler
	err = handler.HandleGetEmailPreferences(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestHandleUpdateEmailPreferences_Success tests updating preferences
func TestHandleUpdateEmailPreferences_Success(t *testing.T) {
	_, queries, cleanup := NewTestDB()
	defer cleanup()

	handler := NewEmailPreferencesHandler(queries)

	// Create test user
	user, err := CreateTestUser(queries)
	require.NoError(t, err)

	// Create email preferences
	ctx := context.Background()
	prefs, err := queries.GetOrCreateEmailPreferences(ctx, db.GetOrCreateEmailPreferencesParams{
		ID:               ulid.Make().String(),
		UserID:           sql.NullString{String: user.ID, Valid: true},
		Email:            user.Email,
		UnsubscribeToken: sql.NullString{String: ulid.Make().String(), Valid: true},
	})
	require.NoError(t, err)

	// Update request
	updateReq := map[string]interface{}{
		"email":           user.Email,
		"promotional":     true,
		"abandoned_cart":  true,
		"newsletter":      false,
		"product_updates": false,
	}

	// Create context with authenticated user
	c, rec := NewTestContext(http.MethodPut, "/api/email-preferences", updateReq)
	SetTestUser(c, user)

	// Call handler
	err = handler.HandleUpdateEmailPreferences(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify update
	updatedPrefs, err := queries.GetEmailPreferencesByEmail(ctx, user.Email)
	require.NoError(t, err)
	assert.Equal(t, int64(1), updatedPrefs.Promotional.Int64)
	assert.Equal(t, int64(1), updatedPrefs.AbandonedCart.Int64)
	assert.Equal(t, prefs.ID, updatedPrefs.ID) // Same record, just updated
}

// TestHandleUnsubscribe_ValidToken tests unsubscribe with valid token
func TestHandleUnsubscribe_ValidToken(t *testing.T) {
	_, queries, cleanup := NewTestDB()
	defer cleanup()

	handler := NewEmailPreferencesHandler(queries)

	// Create email preferences with token
	ctx := context.Background()
	token := ulid.Make().String()
	prefs, err := queries.GetOrCreateEmailPreferences(ctx, db.GetOrCreateEmailPreferencesParams{
		ID:               ulid.Make().String(),
		UserID:           sql.NullString{},
		Email:            "test@example.com",
		UnsubscribeToken: sql.NullString{String: token, Valid: true},
	})
	require.NoError(t, err)

	// Create unsubscribe request
	c, rec := NewTestContext(http.MethodGet, "/unsubscribe/"+token, nil)
	c.SetParamNames("token")
	c.SetParamValues(token)

	// Call handler
	err = handler.HandleUnsubscribe(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify all marketing emails are disabled
	updatedPrefs, err := queries.GetEmailPreferencesByEmail(ctx, prefs.Email)
	require.NoError(t, err)
	assert.Equal(t, int64(0), updatedPrefs.Promotional.Int64)
	assert.Equal(t, int64(0), updatedPrefs.AbandonedCart.Int64)
	assert.Equal(t, int64(0), updatedPrefs.Newsletter.Int64)
	assert.Equal(t, int64(0), updatedPrefs.ProductUpdates.Int64)
	// Transactional should still be enabled
	assert.Equal(t, int64(1), updatedPrefs.Transactional.Int64)
}

// TestHandleUnsubscribe_InvalidToken tests unsubscribe with invalid token
func TestHandleUnsubscribe_InvalidToken(t *testing.T) {
	_, queries, cleanup := NewTestDB()
	defer cleanup()

	handler := NewEmailPreferencesHandler(queries)

	// Create unsubscribe request with invalid token
	invalidToken := "invalid-token-123"
	c, rec := NewTestContext(http.MethodGet, "/unsubscribe/"+invalidToken, nil)
	c.SetParamNames("token")
	c.SetParamValues(invalidToken)

	// Call handler
	err := handler.HandleUnsubscribe(c)

	// Handler writes HTTP response directly, so no error returned
	assert.NoError(t, err)
	// But status should be 404 Not Found
	assert.Equal(t, http.StatusNotFound, rec.Code)
}
