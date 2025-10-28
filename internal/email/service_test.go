package email

import (
	"context"
	"database/sql"
	"testing"

	"github.com/loganlanou/logans3d-v4/storage"
	"github.com/loganlanou/logans3d-v4/storage/db"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCheckEmailPreference_NoRecord_Promotional tests Bug #2 fix
// When no email preferences record exists, promotional emails should default to FALSE
// This is correct behavior - users must explicitly opt in
func TestCheckEmailPreference_NoRecord_Promotional(t *testing.T) {
	_, queries, cleanup, err := storage.NewTestDB()
	require.NoError(t, err)
	defer cleanup()

	service := NewService(queries)
	ctx := context.Background()

	// No email preferences exist for this email
	canSend, err := service.CheckEmailPreference(ctx, "nonexistent@example.com", "promotional")

	assert.NoError(t, err)
	assert.False(t, canSend, "Promotional emails should default to FALSE when no record exists")
}

// TestCheckEmailPreference_NoRecord_AbandonedCart tests default for abandoned cart emails
func TestCheckEmailPreference_NoRecord_AbandonedCart(t *testing.T) {
	_, queries, cleanup, err := storage.NewTestDB()
	require.NoError(t, err)
	defer cleanup()

	service := NewService(queries)
	ctx := context.Background()

	canSend, err := service.CheckEmailPreference(ctx, "nonexistent@example.com", "abandoned_cart")

	assert.NoError(t, err)
	assert.True(t, canSend, "Abandoned cart emails should default to TRUE")
}

// TestCheckEmailPreference_NoRecord_Transactional tests transactional emails
func TestCheckEmailPreference_NoRecord_Transactional(t *testing.T) {
	_, queries, cleanup, err := storage.NewTestDB()
	require.NoError(t, err)
	defer cleanup()

	service := NewService(queries)
	ctx := context.Background()

	canSend, err := service.CheckEmailPreference(ctx, "nonexistent@example.com", "transactional")

	assert.NoError(t, err)
	assert.True(t, canSend, "Transactional emails should always be TRUE")
}

// TestCheckEmailPreference_OptedIn tests when user has opted in
func TestCheckEmailPreference_OptedIn(t *testing.T) {
	_, queries, cleanup, err := storage.NewTestDB()
	require.NoError(t, err)
	defer cleanup()

	service := NewService(queries)
	ctx := context.Background()

	// Create email preference with promotional opted in
	email := "opted-in@example.com"
	_, err = queries.GetOrCreateEmailPreferences(ctx, db.GetOrCreateEmailPreferencesParams{
		ID:               ulid.Make().String(),
		UserID:           sql.NullString{},
		Email:            email,
		UnsubscribeToken: sql.NullString{String: ulid.Make().String(), Valid: true},
	})
	require.NoError(t, err)

	// Update to opt in to promotional
	prefs, err := queries.GetEmailPreferencesByEmail(ctx, email)
	require.NoError(t, err)

	err = queries.UpdateEmailPreferences(ctx, db.UpdateEmailPreferencesParams{
		ID:             prefs.ID,
		Transactional:  sql.NullInt64{Int64: 1, Valid: true},
		Promotional:    sql.NullInt64{Int64: 1, Valid: true},
		AbandonedCart:  sql.NullInt64{Int64: 1, Valid: true},
		Newsletter:     sql.NullInt64{Int64: 0, Valid: true},
		ProductUpdates: sql.NullInt64{Int64: 0, Valid: true},
	})
	require.NoError(t, err)

	canSend, err := service.CheckEmailPreference(ctx, email, "promotional")

	assert.NoError(t, err)
	assert.True(t, canSend)
}

// TestCheckEmailPreference_OptedOut tests when user has opted out
func TestCheckEmailPreference_OptedOut(t *testing.T) {
	_, queries, cleanup, err := storage.NewTestDB()
	require.NoError(t, err)
	defer cleanup()

	service := NewService(queries)
	ctx := context.Background()

	// Create email preference with promotional opted out
	email := "opted-out@example.com"
	_, err = queries.GetOrCreateEmailPreferences(ctx, db.GetOrCreateEmailPreferencesParams{
		ID:               ulid.Make().String(),
		UserID:           sql.NullString{},
		Email:            email,
		UnsubscribeToken: sql.NullString{String: ulid.Make().String(), Valid: true},
	})
	require.NoError(t, err)

	// Ensure promotional is 0
	prefs, err := queries.GetEmailPreferencesByEmail(ctx, email)
	require.NoError(t, err)

	err = queries.UpdateEmailPreferences(ctx, db.UpdateEmailPreferencesParams{
		ID:             prefs.ID,
		Transactional:  sql.NullInt64{Int64: 1, Valid: true},
		Promotional:    sql.NullInt64{Int64: 0, Valid: true}, // Opted out
		AbandonedCart:  sql.NullInt64{Int64: 1, Valid: true},
		Newsletter:     sql.NullInt64{Int64: 0, Valid: true},
		ProductUpdates: sql.NullInt64{Int64: 0, Valid: true},
	})
	require.NoError(t, err)

	canSend, err := service.CheckEmailPreference(ctx, email, "promotional")

	assert.NoError(t, err)
	assert.False(t, canSend, "Should respect opt-out preference")
}

// TestGetOrCreateEmailPreferences_Creates tests creating new preferences
func TestGetOrCreateEmailPreferences_Creates(t *testing.T) {
	_, queries, cleanup, err := storage.NewTestDB()
	require.NoError(t, err)
	defer cleanup()

	service := NewService(queries)
	ctx := context.Background()

	email := "new@example.com"
	prefs, err := service.GetOrCreateEmailPreferences(ctx, email, nil)

	assert.NoError(t, err)
	assert.NotNil(t, prefs)
	assert.Equal(t, email, prefs.Email)
	assert.True(t, prefs.UnsubscribeToken.Valid, "Should have unsubscribe token")
}

// TestGetOrCreateEmailPreferences_Exists tests getting existing preferences
func TestGetOrCreateEmailPreferences_Exists(t *testing.T) {
	_, queries, cleanup, err := storage.NewTestDB()
	require.NoError(t, err)
	defer cleanup()

	service := NewService(queries)
	ctx := context.Background()

	email := "existing@example.com"

	// Create initial preferences
	prefs1, err := service.GetOrCreateEmailPreferences(ctx, email, nil)
	require.NoError(t, err)

	// Get them again - should return existing
	prefs2, err := service.GetOrCreateEmailPreferences(ctx, email, nil)
	require.NoError(t, err)

	assert.Equal(t, prefs1.ID, prefs2.ID, "Should return same preference record")
	assert.Equal(t, prefs1.UnsubscribeToken.String, prefs2.UnsubscribeToken.String)
}

// TestLogEmailSend_Success tests email send logging
func TestLogEmailSend_Success(t *testing.T) {
	_, queries, cleanup, err := storage.NewTestDB()
	require.NoError(t, err)
	defer cleanup()

	service := NewService(queries)
	ctx := context.Background()

	email := "test@example.com"
	emailType := "promotional"
	subject := "Test Email"
	templateName := "welcome"
	trackingToken := "test-token-123"

	err = service.LogEmailSend(ctx, email, emailType, subject, templateName, trackingToken, nil)

	assert.NoError(t, err)

	// Verify it was logged (would need to add a GetEmailHistory query to fully test)
}
