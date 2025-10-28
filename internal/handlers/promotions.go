package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/internal/email"
	stripeutil "github.com/loganlanou/logans3d-v4/internal/stripe"
	"github.com/loganlanou/logans3d-v4/storage/db"
	"github.com/oklog/ulid/v2"
)

type PromotionsHandler struct {
	queries      *db.Queries
	emailService *email.Service
}

func NewPromotionsHandler(queries *db.Queries, emailService *email.Service) *PromotionsHandler {
	return &PromotionsHandler{
		queries:      queries,
		emailService: emailService,
	}
}

// HandleCaptureEmail handles email capture from popup and issues a promotion code
func (h *PromotionsHandler) HandleCaptureEmail(c echo.Context) error {
	var req struct {
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Source    string `json:"source"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	if req.Email == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Email required"})
	}

	ctx := context.Background()

	// Check if email already exists
	var existingContact db.MarketingContact
	var err error
	existingContact, err = h.queries.GetMarketingContactByEmail(ctx, req.Email)
	if err == nil {
		// Email already captured, update popup_shown_at
		_ = h.queries.UpdatePopupShownAt(ctx, req.Email)

		// If they already have a code, return it
		if existingContact.PromotionCodeID.Valid {
			code, err := h.queries.GetPromotionCodeByID(ctx, existingContact.PromotionCodeID.String)
			if err == nil {
				return c.JSON(http.StatusOK, map[string]interface{}{
					"success": true,
					"code":    code.Code,
					"message": "Welcome back! Here's your discount code.",
				})
			}
		}
		// Note: If existing contact has no code, we'll create one and update the record below
	}

	// Get or create first-time customer campaign
	campaign, err := h.getOrCreateFirstTimeCampaign(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create campaign"})
	}

	// Generate unique code
	codeStr := h.generateUniqueCode(req.Email)

	// Create Stripe promotion code
	stripePromoCode, err := stripeutil.CreateUniquePromotionCode(
		campaign.StripePromotionID.String,
		codeStr,
		req.Email,
		30, // 30 days expiration
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create promotion code"})
	}

	// Create promotion code in database
	promoCode, err := h.queries.CreatePromotionCode(ctx, db.CreatePromotionCodeParams{
		ID:                   ulid.Make().String(),
		CampaignID:           campaign.ID,
		Code:                 codeStr,
		StripePromotionCodeID: sql.NullString{String: stripePromoCode.ID, Valid: true},
		Email:                sql.NullString{String: req.Email, Valid: true},
		UserID:               sql.NullString{},
		MaxUses:              sql.NullInt64{Int64: 1, Valid: true},
		ExpiresAt:            sql.NullTime{Time: time.Now().AddDate(0, 0, 30), Valid: true},
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save promotion code"})
	}

	// Create or update marketing contact
	if req.Source == "" {
		req.Source = "popup"
	}

	// If we found an existing contact earlier (without a code), update it
	if existingContact.Email != "" {
		// Update existing contact with new promotion code
		err = h.queries.UpdateMarketingContactPromoCode(ctx, db.UpdateMarketingContactPromoCodeParams{
			PromotionCodeID: sql.NullString{String: promoCode.ID, Valid: true},
			Email:           req.Email,
		})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update contact"})
		}
	} else {
		// Create new marketing contact
		_, err = h.queries.CreateMarketingContact(ctx, db.CreateMarketingContactParams{
			ID:              ulid.Make().String(),
			Email:           req.Email,
			FirstName:       sql.NullString{String: req.FirstName, Valid: req.FirstName != ""},
			LastName:        sql.NullString{String: req.LastName, Valid: req.LastName != ""},
			Source:          req.Source,
			OptedIn:         sql.NullInt64{Int64: 1, Valid: true},
			PromotionCodeID: sql.NullString{String: promoCode.ID, Valid: true},
		})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save contact"})
		}
	}

	// Mark popup as shown for this email
	_ = h.queries.UpdatePopupShownAt(ctx, req.Email)

	// Create email preferences with promotional opt-in (user explicitly opted in via popup)
	prefs, _ := h.emailService.GetOrCreateEmailPreferences(ctx, req.Email, nil)
	// Ensure promotional emails are enabled
	if prefs != nil {
		_ = h.queries.UpdateEmailPreferences(ctx, db.UpdateEmailPreferencesParams{
			ID:            prefs.ID,
			Transactional: sql.NullInt64{Int64: 1, Valid: true},
			Promotional:   sql.NullInt64{Int64: 1, Valid: true},
			AbandonedCart: sql.NullInt64{Int64: 1, Valid: true},
			Newsletter:    sql.NullInt64{Int64: 0, Valid: true},
			ProductUpdates: sql.NullInt64{Int64: 0, Valid: true},
		})
	}

	// Send welcome email with code
	go h.sendWelcomeEmail(req.Email, req.FirstName, codeStr)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"code":    codeStr,
		"message": "Check your email for your discount code!",
	})
}

// HandleValidateCode validates a promotion code
func (h *PromotionsHandler) HandleValidateCode(c echo.Context) error {
	code := c.Param("code")
	if code == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Code required"})
	}

	ctx := context.Background()

	// Check database first
	promoCode, err := h.queries.GetPromotionCodeByCode(ctx, code)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Invalid promotion code"})
	}

	// Check expiration
	if promoCode.ExpiresAt.Valid && time.Now().After(promoCode.ExpiresAt.Time) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Code has expired"})
	}

	// Check max uses
	if promoCode.MaxUses.Valid && promoCode.CurrentUses.Valid && promoCode.CurrentUses.Int64 >= promoCode.MaxUses.Int64 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Code has been fully redeemed"})
	}

	// Get campaign details
	campaign, err := h.queries.GetPromotionCampaignByID(ctx, promoCode.CampaignID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get campaign"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"valid":          true,
		"code":           promoCode.Code,
		"discount_type":  campaign.DiscountType,
		"discount_value": campaign.DiscountValue,
	})
}

// getOrCreateFirstTimeCampaign gets or creates the first-time customer campaign
func (h *PromotionsHandler) getOrCreateFirstTimeCampaign(ctx context.Context) (*db.PromotionCampaign, error) {
	// Try to get existing campaign
	campaigns, err := h.queries.GetActivePromotionCampaigns(ctx)
	if err == nil {
		for _, campaign := range campaigns {
			if campaign.Name == "First-Time Customer" {
				return &campaign, nil
			}
		}
	}

	// Create new campaign
	stripeCoupon, err := stripeutil.CreatePromotionCampaign("First-Time Customer", "percentage", 15)
	if err != nil {
		return nil, err
	}

	campaign, err := h.queries.CreatePromotionCampaign(ctx, db.CreatePromotionCampaignParams{
		ID:                ulid.Make().String(),
		Name:              "First-Time Customer",
		Description:       sql.NullString{String: "15% off for first-time customers", Valid: true},
		DiscountType:      "percentage",
		DiscountValue:     15,
		StripePromotionID: sql.NullString{String: stripeCoupon.ID, Valid: true},
		StartDate:         time.Now(),
		EndDate:           sql.NullTime{},
		MaxUses:           sql.NullInt64{},
		Active:            sql.NullInt64{Int64: 1, Valid: true},
	})

	return &campaign, err
}

// generateUniqueCode generates a unique promotion code based on email
func (h *PromotionsHandler) generateUniqueCode(email string) string {
	// Use first part of email + timestamp suffix
	parts := strings.Split(email, "@")
	prefix := strings.ToUpper(parts[0])
	if len(prefix) > 8 {
		prefix = prefix[:8]
	}

	// Add random suffix
	suffix := fmt.Sprintf("%d", time.Now().Unix()%10000)
	return fmt.Sprintf("%s%s", prefix, suffix)
}

// sendWelcomeEmail sends welcome email with promotion code
func (h *PromotionsHandler) sendWelcomeEmail(emailAddr, firstName, code string) {
	ctx := context.Background()

	// Check if user has opted out
	canSend, _ := h.emailService.CheckEmailPreference(ctx, emailAddr, "promotional")
	if !canSend {
		return
	}

	// Get or create preferences to get unsubscribe token
	prefs, err := h.emailService.GetOrCreateEmailPreferences(ctx, emailAddr, nil)
	if err != nil {
		return
	}

	var unsubscribeToken string
	if prefs != nil && prefs.UnsubscribeToken.Valid {
		unsubscribeToken = prefs.UnsubscribeToken.String
	}

	// Render and send email
	data := &email.WelcomeCouponData{
		CustomerName: firstName,
		Email:        emailAddr,
		PromoCode:    code,
		DiscountText: "15% off",
		ExpiresAt:    time.Now().AddDate(0, 0, 30).Format("January 2, 2006"),
	}

	html, err := email.RenderWelcomeCouponEmailWithToken(data, unsubscribeToken)
	if err != nil {
		return
	}

	emailMsg := &email.Email{
		To:      []string{emailAddr},
		Subject: "Welcome! Here's 15% off your first order",
		Body:    html,
		IsHTML:  true,
	}

	h.emailService.Send(emailMsg)

	// Log the send
	h.emailService.LogEmailSend(ctx, emailAddr, "promotional", emailMsg.Subject, "welcome_coupon", "", map[string]interface{}{
		"promo_code": code,
	})
}
