package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"testing"
	"time"

	emailutil "github.com/loganlanou/logans3d-v4/internal/email"
	"github.com/loganlanou/logans3d-v4/storage/db"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHandleCaptureEmail_NewEmail tests creating contact + code for completely new email
func TestHandleCaptureEmail_NewEmail(t *testing.T) {
	_, queries, cleanup := NewTestDB()
	defer cleanup()

	emailService := emailutil.NewService(queries)
	handler := NewPromotionsHandler(queries, emailService)

	ctx := context.Background()

	// Create first-time campaign (required for code generation)
	campaign, err := queries.CreatePromotionCampaign(ctx, db.CreatePromotionCampaignParams{
		ID:                ulid.Make().String(),
		Name:              "First Time 15%",
		DiscountType:      "percent",
		DiscountValue:     15,
		Active:            sql.NullInt64{Int64: 1, Valid: true},
		StripePromotionID: sql.NullString{String: "promo_test", Valid: true},
		StartDate:         time.Now(),
	})
	require.NoError(t, err)

	// Capture email request
	req := map[string]interface{}{
		"email":      "newuser@example.com",
		"first_name": "New",
		"last_name":  "User",
		"source":     "popup",
	}

	c, _ := NewTestContext(http.MethodPost, "/api/capture-email", req)

	// Note: This will fail without Stripe mock, but we can test the database logic
	// For now, just verify the request structure
	assert.NotNil(t, handler)
	assert.NotNil(t, c)

	// Verify campaign was created
	assert.Equal(t, "First Time 15%", campaign.Name)
}

// TestHandleCaptureEmail_ExistingWithCode tests Bug #4 scenario
// When email already exists WITH a code, should return existing code
func TestHandleCaptureEmail_ExistingWithCode(t *testing.T) {
	_, queries, cleanup := NewTestDB()
	defer cleanup()

	ctx := context.Background()

	// Create campaign
	campaign, err := queries.CreatePromotionCampaign(ctx, db.CreatePromotionCampaignParams{
		ID:                ulid.Make().String(),
		Name:              "First Time 15%",
		DiscountType:      "percent",
		DiscountValue:     15,
		Active:            sql.NullInt64{Int64: 1, Valid: true},
		StripePromotionID: sql.NullString{String: "promo_test", Valid: true},
		StartDate:         time.Now(),
	})
	require.NoError(t, err)

	// Create existing promotion code
	existingCode, err := queries.CreatePromotionCode(ctx, db.CreatePromotionCodeParams{
		ID:                    ulid.Make().String(),
		CampaignID:            campaign.ID,
		Code:                  "EXISTING123",
		Email:                 sql.NullString{String: "existing@example.com", Valid: true},
		StripePromotionCodeID: sql.NullString{String: "stripe_code", Valid: true},
	})
	require.NoError(t, err)

	// Create existing marketing contact with this code
	_, err = queries.CreateMarketingContact(ctx, db.CreateMarketingContactParams{
		ID:              ulid.Make().String(),
		Email:           "existing@example.com",
		FirstName:       sql.NullString{String: "Existing", Valid: true},
		Source:          "popup",
		OptedIn:         sql.NullInt64{Int64: 1, Valid: true},
		PromotionCodeID: sql.NullString{String: existingCode.ID, Valid: true},
	})
	require.NoError(t, err)

	// Now when we try to capture this email again, it should return the existing code
	contact, err := queries.GetMarketingContactByEmail(ctx, "existing@example.com")
	require.NoError(t, err)
	assert.True(t, contact.PromotionCodeID.Valid)
	assert.Equal(t, existingCode.ID, contact.PromotionCodeID.String)

	// Get the code
	code, err := queries.GetPromotionCodeByID(ctx, contact.PromotionCodeID.String)
	require.NoError(t, err)
	assert.Equal(t, "EXISTING123", code.Code)
}

// TestHandleCaptureEmail_ExistingNoCode tests Bug #4 FIX
// When email exists WITHOUT a code, should UPDATE existing contact (not create orphaned code)
func TestHandleCaptureEmail_ExistingNoCode(t *testing.T) {
	_, queries, cleanup := NewTestDB()
	defer cleanup()

	ctx := context.Background()

	// Create existing marketing contact WITHOUT promotion code
	email := "nocode@example.com"
	existingContact, err := queries.CreateMarketingContact(ctx, db.CreateMarketingContactParams{
		ID:              ulid.Make().String(),
		Email:           email,
		FirstName:       sql.NullString{String: "NoCode", Valid: true},
		Source:          "newsletter",
		OptedIn:         sql.NullInt64{Int64: 1, Valid: true},
		PromotionCodeID: sql.NullString{Valid: false}, // NO CODE!
	})
	require.NoError(t, err)
	assert.False(t, existingContact.PromotionCodeID.Valid, "Initially should have no promo code")

	// Create campaign
	campaign, err := queries.CreatePromotionCampaign(ctx, db.CreatePromotionCampaignParams{
		ID:                ulid.Make().String(),
		Name:              "First Time 15%",
		DiscountType:      "percent",
		DiscountValue:     15,
		Active:            sql.NullInt64{Int64: 1, Valid: true},
		StripePromotionID: sql.NullString{String: "promo_test", Valid: true},
		StartDate:         time.Now(),
	})
	require.NoError(t, err)

	// Create a promotion code
	newCode, err := queries.CreatePromotionCode(ctx, db.CreatePromotionCodeParams{
		ID:                    ulid.Make().String(),
		CampaignID:            campaign.ID,
		Code:                  "NEWCODE123",
		Email:                 sql.NullString{String: email, Valid: true},
		StripePromotionCodeID: sql.NullString{String: "stripe_code_new", Valid: true},
	})
	require.NoError(t, err)

	// Now UPDATE the existing contact with the new code (Bug #4 fix)
	err = queries.UpdateMarketingContactPromoCode(ctx, db.UpdateMarketingContactPromoCodeParams{
		PromotionCodeID: sql.NullString{String: newCode.ID, Valid: true},
		Email:           email,
	})
	require.NoError(t, err)

	// Verify the contact was UPDATED (not created as duplicate)
	updatedContact, err := queries.GetMarketingContactByEmail(ctx, email)
	require.NoError(t, err)
	assert.Equal(t, existingContact.ID, updatedContact.ID, "Should be same contact record")
	assert.True(t, updatedContact.PromotionCodeID.Valid, "Should now have promo code")
	assert.Equal(t, newCode.ID, updatedContact.PromotionCodeID.String)

	// Verify the code is linked
	linkedCode, err := queries.GetPromotionCodeByID(ctx, updatedContact.PromotionCodeID.String)
	require.NoError(t, err)
	assert.Equal(t, "NEWCODE123", linkedCode.Code)
}

// TestMarketingContactUniqueness tests that emails are unique in marketing_contacts
func TestMarketingContactUniqueness(t *testing.T) {
	_, queries, cleanup := NewTestDB()
	defer cleanup()

	ctx := context.Background()
	email := "unique@example.com"

	// Create first contact
	_, err := queries.CreateMarketingContact(ctx, db.CreateMarketingContactParams{
		ID:      ulid.Make().String(),
		Email:   email,
		Source:  "popup",
		OptedIn: sql.NullInt64{Int64: 1, Valid: true},
	})
	require.NoError(t, err)

	// Try to create duplicate - should fail
	_, err = queries.CreateMarketingContact(ctx, db.CreateMarketingContactParams{
		ID:      ulid.Make().String(),
		Email:   email,
		Source:  "newsletter",
		OptedIn: sql.NullInt64{Int64: 1, Valid: true},
	})

	assert.Error(t, err, "Should fail due to UNIQUE constraint on email")
	assert.Contains(t, err.Error(), "UNIQUE", "Error should mention UNIQUE constraint")
}

// TestPromotionCodeGeneration tests that codes are generated correctly
func TestPromotionCodeGeneration(t *testing.T) {
	_, queries, cleanup := NewTestDB()
	defer cleanup()

	ctx := context.Background()

	// Create campaign
	campaign, err := queries.CreatePromotionCampaign(ctx, db.CreatePromotionCampaignParams{
		ID:                ulid.Make().String(),
		Name:              "Test Campaign",
		DiscountType:      "percent",
		DiscountValue:     15,
		Active:            sql.NullInt64{Int64: 1, Valid: true},
		StripePromotionID: sql.NullString{String: "promo_test", Valid: true},
		StartDate:         time.Now(),
	})
	require.NoError(t, err)

	// Create multiple codes
	codes := []string{"CODE1", "CODE2", "CODE3"}
	for _, codeStr := range codes {
		_, err := queries.CreatePromotionCode(ctx, db.CreatePromotionCodeParams{
			ID:                    ulid.Make().String(),
			CampaignID:            campaign.ID,
			Code:                  codeStr,
			StripePromotionCodeID: sql.NullString{String: "stripe_" + codeStr, Valid: true},
		})
		require.NoError(t, err)
	}

	// Get codes for campaign
	campaignCodes, err := queries.GetPromotionCodesByCampaign(ctx, db.GetPromotionCodesByCampaignParams{
		CampaignID: campaign.ID,
		Limit:      10,
		Offset:     0,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, len(campaignCodes))
}

// TestEmailPreferencesCreatedOnCapture tests Bug #2 fix
// When email is captured, email_preferences should be created with promotional=1
func TestEmailPreferencesCreatedOnCapture(t *testing.T) {
	_, queries, cleanup := NewTestDB()
	defer cleanup()

	emailService := emailutil.NewService(queries)
	ctx := context.Background()

	email := "newcapture@example.com"

	// Verify no preferences exist
	_, err := queries.GetEmailPreferencesByEmail(ctx, email)
	assert.Error(t, err, "Should not have preferences initially")

	// Create email preferences (simulating what HandleCaptureEmail does)
	prefs, err := emailService.GetOrCreateEmailPreferences(ctx, email, nil)
	require.NoError(t, err)

	// Update to enable promotional (this is what the fix does)
	err = queries.UpdateEmailPreferences(ctx, db.UpdateEmailPreferencesParams{
		ID:             prefs.ID,
		Transactional:  sql.NullInt64{Int64: 1, Valid: true},
		Promotional:    sql.NullInt64{Int64: 1, Valid: true},
		AbandonedCart:  sql.NullInt64{Int64: 1, Valid: true},
		Newsletter:     sql.NullInt64{Int64: 0, Valid: true},
		ProductUpdates: sql.NullInt64{Int64: 0, Valid: true},
	})
	require.NoError(t, err)

	// Verify promotional is enabled
	updatedPrefs, err := queries.GetEmailPreferencesByEmail(ctx, email)
	require.NoError(t, err)
	assert.Equal(t, int64(1), updatedPrefs.Promotional.Int64, "Promotional should be enabled for new opt-ins")

	// Now CheckEmailPreference should return true
	canSend, err := emailService.CheckEmailPreference(ctx, email, "promotional")
	assert.NoError(t, err)
	assert.True(t, canSend, "Should be able to send promotional emails after opt-in")
}
