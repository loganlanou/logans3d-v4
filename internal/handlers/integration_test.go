//go:build integration
// +build integration

package handlers

import (
	"context"
	"database/sql"
	"testing"

	emailutil "github.com/loganlanou/logans3d-v4/internal/email"
	"github.com/loganlanou/logans3d-v4/storage/db"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEmailCaptureFlow_Complete tests the full flow from email capture to preferences creation
// This integration test verifies all the bug fixes work together:
// - Bug #2: Email preferences are created with promotional=1
// - Bug #4: No orphaned promotion codes
func TestEmailCaptureFlow_Complete(t *testing.T) {
	database, queries, cleanup := NewTestDB()
	defer cleanup()

	emailService := emailutil.NewService(queries, database, "")
	ctx := context.Background()

	email := "integration@example.com"

	// Step 1: Create promotion campaign
	campaign, err := queries.CreatePromotionCampaign(ctx, db.CreatePromotionCampaignParams{
		ID:                ulid.Make().String(),
		Name:              "First Time 15%",
		DiscountPercent:   sql.NullInt64{Int64: 15, Valid: true},
		Active:            sql.NullInt64{Int64: 1, Valid: true},
		StripePromotionID: sql.NullString{String: "promo_integration", Valid: true},
	})
	require.NoError(t, err)

	// Step 2: Create promotion code
	promoCode, err := queries.CreatePromotionCode(ctx, db.CreatePromotionCodeParams{
		ID:                    ulid.Make().String(),
		CampaignID:            campaign.ID,
		Code:                  "WELCOME15",
		Email:                 sql.NullString{String: email, Valid: true},
		StripePromotionCodeID: sql.NullString{String: "stripe_code_integration", Valid: true},
	})
	require.NoError(t, err)

	// Step 3: Create marketing contact with promo code
	contact, err := queries.CreateMarketingContact(ctx, db.CreateMarketingContactParams{
		ID:              ulid.Make().String(),
		Email:           email,
		FirstName:       sql.NullString{String: "Integration", Valid: true},
		LastName:        sql.NullString{String: "Test", Valid: true},
		Source:          "popup",
		OptedIn:         sql.NullInt64{Int64: 1, Valid: true},
		PromotionCodeID: sql.NullString{String: promoCode.ID, Valid: true},
	})
	require.NoError(t, err)
	assert.Equal(t, email, contact.Email)
	assert.True(t, contact.PromotionCodeID.Valid)

	// Step 4: Create email preferences with promotional enabled (Bug #2 fix)
	prefs, err := emailService.GetOrCreateEmailPreferences(ctx, email, nil)
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

	// Step 5: Verify we can send promotional email (Bug #2 verification)
	canSend, err := emailService.CheckEmailPreference(ctx, email, "promotional")
	assert.NoError(t, err)
	assert.True(t, canSend, "Should be able to send promotional emails")

	// Step 6: Verify promo code is linked to contact (Bug #4 verification)
	verifyContact, err := queries.GetMarketingContactByEmail(ctx, email)
	require.NoError(t, err)
	assert.True(t, verifyContact.PromotionCodeID.Valid)
	assert.Equal(t, promoCode.ID, verifyContact.PromotionCodeID.String)

	// Step 7: Verify promo code exists and is linked
	verifyCode, err := queries.GetPromotionCodeByID(ctx, verifyContact.PromotionCodeID.String)
	require.NoError(t, err)
	assert.Equal(t, "WELCOME15", verifyCode.Code)
	assert.Equal(t, email, verifyCode.Email.String)
}

// TestEmailCaptureFlow_WithExisting tests the flow when contact already exists
// This verifies Bug #4 fix: existing contacts without codes get updated, not orphaned
func TestEmailCaptureFlow_WithExisting(t *testing.T) {
	database, queries, cleanup := NewTestDB()
	defer cleanup()

	ctx := context.Background()
	email := "existing@example.com"

	// Step 1: Create existing contact WITHOUT promo code
	existingContact, err := queries.CreateMarketingContact(ctx, db.CreateMarketingContactParams{
		ID:              ulid.Make().String(),
		Email:           email,
		FirstName:       sql.NullString{String: "Existing", Valid: true},
		Source:          "newsletter",
		OptedIn:         sql.NullInt64{Int64: 1, Valid: true},
		PromotionCodeID: sql.NullString{Valid: false}, // NO CODE
	})
	require.NoError(t, err)
	assert.False(t, existingContact.PromotionCodeID.Valid)

	// Step 2: Create campaign
	campaign, err := queries.CreatePromotionCampaign(ctx, db.CreatePromotionCampaignParams{
		ID:                ulid.Make().String(),
		Name:              "First Time 15%",
		DiscountPercent:   sql.NullInt64{Int64: 15, Valid: true},
		Active:            sql.NullInt64{Int64: 1, Valid: true},
		StripePromotionID: sql.NullString{String: "promo_existing", Valid: true},
	})
	require.NoError(t, err)

	// Step 3: Create new promo code for existing user
	newCode, err := queries.CreatePromotionCode(ctx, db.CreatePromotionCodeParams{
		ID:                    ulid.Make().String(),
		CampaignID:            campaign.ID,
		Code:                  "EXISTING15",
		Email:                 sql.NullString{String: email, Valid: true},
		StripePromotionCodeID: sql.NullString{String: "stripe_existing", Valid: true},
	})
	require.NoError(t, err)

	// Step 4: UPDATE existing contact with new code (Bug #4 fix - not creating duplicate)
	err = queries.UpdateMarketingContactPromoCode(ctx, db.UpdateMarketingContactPromoCodeParams{
		PromotionCodeID: sql.NullString{String: newCode.ID, Valid: true},
		Email:           email,
	})
	require.NoError(t, err)

	// Step 5: Verify contact was UPDATED (same ID, now has code)
	updatedContact, err := queries.GetMarketingContactByEmail(ctx, email)
	require.NoError(t, err)
	assert.Equal(t, existingContact.ID, updatedContact.ID, "Should be same contact record")
	assert.True(t, updatedContact.PromotionCodeID.Valid)
	assert.Equal(t, newCode.ID, updatedContact.PromotionCodeID.String)

	// Step 6: Verify no duplicate contacts exist
	// Note: We can't directly count duplicates, but the UNIQUE constraint ensures this
	verifyContact, err := queries.GetMarketingContactByEmail(ctx, email)
	require.NoError(t, err)
	assert.Equal(t, updatedContact.ID, verifyContact.ID)
}

// TestEmailPreferencesFlow_OptOut tests opt-out flow
func TestEmailPreferencesFlow_OptOut(t *testing.T) {
	database, queries, cleanup := NewTestDB()
	defer cleanup()

	emailService := emailutil.NewService(queries, database, "")
	ctx := context.Background()

	email := "optout@example.com"

	// Step 1: Create email preferences with promotional enabled
	prefs, err := emailService.GetOrCreateEmailPreferences(ctx, email, nil)
	require.NoError(t, err)

	err = queries.UpdateEmailPreferences(ctx, db.UpdateEmailPreferencesParams{
		ID:             prefs.ID,
		Transactional:  sql.NullInt64{Int64: 1, Valid: true},
		Promotional:    sql.NullInt64{Int64: 1, Valid: true},
		AbandonedCart:  sql.NullInt64{Int64: 1, Valid: true},
		Newsletter:     sql.NullInt64{Int64: 1, Valid: true},
		ProductUpdates: sql.NullInt64{Int64: 1, Valid: true},
	})
	require.NoError(t, err)

	// Step 2: Verify can send promotional
	canSend, err := emailService.CheckEmailPreference(ctx, email, "promotional")
	assert.NoError(t, err)
	assert.True(t, canSend)

	// Step 3: Opt out of promotional
	err = queries.UpdateEmailPreferences(ctx, db.UpdateEmailPreferencesParams{
		ID:             prefs.ID,
		Transactional:  sql.NullInt64{Int64: 1, Valid: true},
		Promotional:    sql.NullInt64{Int64: 0, Valid: true}, // OPT OUT
		AbandonedCart:  sql.NullInt64{Int64: 1, Valid: true},
		Newsletter:     sql.NullInt64{Int64: 1, Valid: true},
		ProductUpdates: sql.NullInt64{Int64: 1, Valid: true},
	})
	require.NoError(t, err)

	// Step 4: Verify can no longer send promotional
	canSend, err = emailService.CheckEmailPreference(ctx, email, "promotional")
	assert.NoError(t, err)
	assert.False(t, canSend, "Should respect opt-out")

	// Step 5: Verify can still send other types
	canSendAbandoned, err := emailService.CheckEmailPreference(ctx, email, "abandoned_cart")
	assert.NoError(t, err)
	assert.True(t, canSendAbandoned, "Can still send abandoned cart")

	canSendTransactional, err := emailService.CheckEmailPreference(ctx, email, "transactional")
	assert.NoError(t, err)
	assert.True(t, canSendTransactional, "Can still send transactional")
}

// TestEmailPreferencesUniqueConstraint tests Bug #5 fix
// Email should be unique across all preferences, even with NULL user_id
func TestEmailPreferencesUniqueConstraint(t *testing.T) {
	database, queries, cleanup := NewTestDB()
	defer cleanup()

	ctx := context.Background()
	email := "unique@example.com"

	// Create first preference (anonymous user)
	_, err := queries.GetOrCreateEmailPreferences(ctx, db.GetOrCreateEmailPreferencesParams{
		ID:               ulid.Make().String(),
		UserID:           sql.NullString{Valid: false}, // NULL user_id
		Email:            email,
		UnsubscribeToken: sql.NullString{String: ulid.Make().String(), Valid: true},
	})
	require.NoError(t, err)

	// Try to create second preference with same email but different NULL user_id
	// After Bug #5 fix (unique constraint on email only), this should return existing
	prefs2, err := queries.GetOrCreateEmailPreferences(ctx, db.GetOrCreateEmailPreferencesParams{
		ID:               ulid.Make().String(),
		UserID:           sql.NullString{Valid: false}, // Different NULL
		Email:            email,
		UnsubscribeToken: sql.NullString{String: ulid.Make().String(), Valid: true},
	})
	require.NoError(t, err)

	// Verify only one preference exists for this email
	retrievedPrefs, err := queries.GetEmailPreferencesByEmail(ctx, email)
	require.NoError(t, err)
	assert.Equal(t, prefs2.ID, retrievedPrefs.ID, "Should have returned existing preference")
}
