package stripe

import (
	"fmt"
	"time"

	"github.com/stripe/stripe-go/v80"
	"github.com/stripe/stripe-go/v80/coupon"
	"github.com/stripe/stripe-go/v80/promotioncode"
)

// CreatePromotionCampaign creates a Stripe promotion (coupon)
func CreatePromotionCampaign(name string, discountType string, discountValue int64) (*stripe.Coupon, error) {
	params := &stripe.CouponParams{}
	params.Name = stripe.String(name)
	params.Duration = stripe.String(string(stripe.CouponDurationOnce)) // One-time use

	switch discountType {
	case "percentage":
		params.PercentOff = stripe.Float64(float64(discountValue))
	case "fixed_amount":
		params.AmountOff = stripe.Int64(discountValue)
		params.Currency = stripe.String(string(stripe.CurrencyUSD))
	default:
		return nil, fmt.Errorf("invalid discount type: %s", discountType)
	}

	return coupon.New(params)
}

// CreateUniquePromotionCode creates a unique promotion code for a specific email
func CreateUniquePromotionCode(couponID string, code string, email string, expiresInDays int) (*stripe.PromotionCode, error) {
	params := &stripe.PromotionCodeParams{
		Coupon: stripe.String(couponID),
		Code:   stripe.String(code),
	}

	// Set expiration
	if expiresInDays > 0 {
		expiresAt := time.Now().AddDate(0, 0, expiresInDays).Unix()
		params.ExpiresAt = stripe.Int64(expiresAt)
	}

	// Restrict to specific customer email
	params.Restrictions = &stripe.PromotionCodeRestrictionsParams{
		FirstTimeTransaction: stripe.Bool(true), // Only for first-time customers
	}

	// Set max redemptions to 1
	params.MaxRedemptions = stripe.Int64(1)

	return promotioncode.New(params)
}

// ValidatePromotionCode checks if a promotion code is valid and active
func ValidatePromotionCode(code string) (*stripe.PromotionCode, error) {
	// Search for promotion code by code
	params := &stripe.PromotionCodeListParams{}
	params.Code = stripe.String(code)
	params.Active = stripe.Bool(true)

	iter := promotioncode.List(params)
	if iter.Next() {
		promoCode := iter.PromotionCode()

		// Check if expired
		if promoCode.ExpiresAt > 0 && time.Now().Unix() > promoCode.ExpiresAt {
			return nil, fmt.Errorf("promotion code has expired")
		}

		// Check if max redemptions reached
		if promoCode.MaxRedemptions > 0 && promoCode.TimesRedeemed >= promoCode.MaxRedemptions {
			return nil, fmt.Errorf("promotion code has reached maximum redemptions")
		}

		return promoCode, nil
	}

	if err := iter.Err(); err != nil {
		return nil, err
	}

	return nil, fmt.Errorf("promotion code not found or inactive")
}

// GetPromotionCodeByID retrieves a promotion code by its ID
func GetPromotionCodeByID(id string) (*stripe.PromotionCode, error) {
	return promotioncode.Get(id, nil)
}
