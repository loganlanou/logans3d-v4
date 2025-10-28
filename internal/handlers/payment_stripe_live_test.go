package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	stripe "github.com/stripe/stripe-go/v80"
	checkoutsession "github.com/stripe/stripe-go/v80/checkout/session"
	promotioncode "github.com/stripe/stripe-go/v80/promotioncode"
	"github.com/stretchr/testify/require"
)

// TestLiveStripeSession queries the actual Stripe API to see what data is returned
// This test is SKIPPED by default and only runs when STRIPE_LIVE_TEST=1
//
// Usage:
//   STRIPE_LIVE_TEST=1 STRIPE_SESSION_ID=cs_test_xxx go test -v ./internal/handlers -run TestLiveStripeSession
func TestLiveStripeSession(t *testing.T) {
	if os.Getenv("STRIPE_LIVE_TEST") != "1" {
		t.Skip("Skipping live Stripe API test. Set STRIPE_LIVE_TEST=1 to enable.")
	}

	// Get session ID from environment (or use the most recent one)
	sessionID := os.Getenv("STRIPE_SESSION_ID")
	if sessionID == "" {
		sessionID = "cs_test_b1P3JynPoQpgLKtmbh8SWYkFsqCKZYcUFR6Qn9WgTy57tMWq0QjnW16n19" // Default to recent checkout
	}

	// Set Stripe API key
	stripeKey := os.Getenv("STRIPE_SECRET_KEY")
	require.NotEmpty(t, stripeKey, "STRIPE_SECRET_KEY environment variable must be set")
	stripe.Key = stripeKey

	t.Logf("Querying Stripe session: %s", sessionID)

	// Test different expansion strategies
	testCases := []struct {
		name      string
		expansions []string
	}{
		{
			name: "Expand total_details.breakdown only",
			expansions: []string{
				"total_details.breakdown",
			},
		},
		{
			name: "Expand total_details.breakdown and line_items",
			expansions: []string{
				"total_details.breakdown",
				"line_items",
			},
		},
		{
			name: "Expand total_details.breakdown.discounts",
			expansions: []string{
				"total_details.breakdown.discounts",
			},
		},
		{
			name: "Expand total_details.breakdown.discounts.discount",
			expansions: []string{
				"total_details.breakdown.discounts.discount",
			},
		},
		{
			name: "Expand total_details.breakdown.discounts.discount.promotion_code",
			expansions: []string{
				"total_details.breakdown.discounts.discount.promotion_code",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := &stripe.CheckoutSessionParams{}
			for _, exp := range tc.expansions {
				params.AddExpand(exp)
			}

			session, err := checkoutsession.Get(sessionID, params)
			require.NoError(t, err, "Failed to retrieve session from Stripe")

			t.Logf("\n========== %s ==========", tc.name)
			t.Logf("Session ID: %s", session.ID)
			t.Logf("Customer Email: %s", session.CustomerDetails.Email)
			t.Logf("Amount Total: %d", session.AmountTotal)

			if session.TotalDetails != nil {
				t.Logf("Amount Discount: %d", session.TotalDetails.AmountDiscount)
				t.Logf("Has Breakdown: %v", session.TotalDetails.Breakdown != nil)

				if session.TotalDetails.Breakdown != nil {
					t.Logf("Number of Discounts: %d", len(session.TotalDetails.Breakdown.Discounts))

					for i, discount := range session.TotalDetails.Breakdown.Discounts {
						t.Logf("\n--- Discount #%d ---", i)
						t.Logf("  Amount: %d", discount.Amount)
						t.Logf("  Has Discount Object: %v", discount.Discount != nil)

						if discount.Discount != nil {
							t.Logf("  Discount ID: %s", discount.Discount.ID)
							t.Logf("  Has Coupon: %v", discount.Discount.Coupon != nil)
							t.Logf("  Has PromotionCode: %v", discount.Discount.PromotionCode != nil)

							if discount.Discount.Coupon != nil {
								t.Logf("  Coupon ID: %s", discount.Discount.Coupon.ID)
								t.Logf("  Coupon Name: %s", discount.Discount.Coupon.Name)
								if discount.Discount.Coupon.AmountOff > 0 {
									t.Logf("  Amount Off: %d %s", discount.Discount.Coupon.AmountOff, discount.Discount.Coupon.Currency)
								}
								if discount.Discount.Coupon.PercentOff > 0 {
									t.Logf("  Percent Off: %.0f%%", discount.Discount.Coupon.PercentOff)
								}
							}

							if discount.Discount.PromotionCode != nil {
								t.Logf("  PromotionCode ID: %s", discount.Discount.PromotionCode.ID)
								t.Logf("  PromotionCode Code: '%s'", discount.Discount.PromotionCode.Code)
								t.Logf("  PromotionCode Active: %v", discount.Discount.PromotionCode.Active)

								// Print the entire promotion code object as JSON for inspection
								promoJSON, _ := json.MarshalIndent(discount.Discount.PromotionCode, "    ", "  ")
								t.Logf("  Full PromotionCode Object:\n%s", string(promoJSON))
							}
						}
					}
				}
			}
		})
	}

	// Final test: Print the entire session as JSON
	t.Run("Full Session JSON", func(t *testing.T) {
		params := &stripe.CheckoutSessionParams{}
		params.AddExpand("total_details.breakdown")
		params.AddExpand("line_items")

		session, err := checkoutsession.Get(sessionID, params)
		require.NoError(t, err)

		sessionJSON, err := json.MarshalIndent(session, "", "  ")
		require.NoError(t, err)

		t.Logf("\n========== FULL SESSION JSON ==========\n%s\n", string(sessionJSON))
	})
}

// TestLiveStripePromotionCodeRetrieval tests retrieving a promotion code by ID
// This helps us understand if we need to make a separate API call to get the code string
//
// Usage:
//   STRIPE_LIVE_TEST=1 STRIPE_PROMO_CODE_ID=promo_xxx go test -v ./internal/handlers -run TestLiveStripePromotionCodeRetrieval
func TestLiveStripePromotionCodeRetrieval(t *testing.T) {
	if os.Getenv("STRIPE_LIVE_TEST") != "1" {
		t.Skip("Skipping live Stripe API test. Set STRIPE_LIVE_TEST=1 to enable.")
	}

	promoCodeID := os.Getenv("STRIPE_PROMO_CODE_ID")
	if promoCodeID == "" {
		t.Skip("Skipping promotion code retrieval test. Set STRIPE_PROMO_CODE_ID to test.")
	}

	stripeKey := os.Getenv("STRIPE_SECRET_KEY")
	require.NotEmpty(t, stripeKey, "STRIPE_SECRET_KEY environment variable must be set")
	stripe.Key = stripeKey

	t.Logf("Retrieving promotion code: %s", promoCodeID)

	promoCode, err := promotioncode.Get(promoCodeID, nil)
	require.NoError(t, err)

	t.Logf("\n========== PROMOTION CODE DETAILS ==========")
	t.Logf("ID: %s", promoCode.ID)
	t.Logf("Code: '%s'", promoCode.Code)
	t.Logf("Active: %v", promoCode.Active)
	t.Logf("Coupon ID: %s", promoCode.Coupon.ID)

	promoJSON, err := json.MarshalIndent(promoCode, "", "  ")
	require.NoError(t, err)
	t.Logf("\nFull Object:\n%s", string(promoJSON))
}

// TestParsePromoCodeFromLogs helps parse and understand the structure from logs
// This is a utility test to help visualize what we're seeing in the application logs
func TestParsePromoCodeFromLogs(t *testing.T) {
	// This test demonstrates the expected structure based on Stripe docs
	t.Log("Expected Structure:")
	t.Log("session.TotalDetails.Breakdown.Discounts[0].Discount.PromotionCode.Code")
	t.Log("")
	t.Log("If PromotionCode.Code is empty, possible reasons:")
	t.Log("1. PromotionCode needs additional expansion")
	t.Log("2. Need to retrieve PromotionCode by ID separately")
	t.Log("3. Code is in a different field (check Coupon.Name or Coupon.ID)")
	t.Log("4. Stripe API version mismatch")
	t.Log("")
	t.Log("Run the live test to see actual values:")
	t.Log("  STRIPE_LIVE_TEST=1 go test -v ./internal/handlers -run TestLiveStripeSession")
}

// Helper function to demonstrate how to extract promo code
func Example_extractPromoCodeFromSession() {
	// This is example code showing the expected pattern
	session := &stripe.CheckoutSession{
		TotalDetails: &stripe.CheckoutSessionTotalDetails{
			Breakdown: &stripe.CheckoutSessionTotalDetailsBreakdown{
				Discounts: []*stripe.CheckoutSessionTotalDetailsBreakdownDiscount{
					{
						Discount: &stripe.Discount{
							PromotionCode: &stripe.PromotionCode{
								Code:   "TEST003",
								ID:     "promo_xxx",
								Active: true,
							},
						},
					},
				},
			},
		},
	}

	if session.TotalDetails != nil && session.TotalDetails.Breakdown != nil &&
		len(session.TotalDetails.Breakdown.Discounts) > 0 {
		firstDiscount := session.TotalDetails.Breakdown.Discounts[0]
		if firstDiscount.Discount != nil && firstDiscount.Discount.PromotionCode != nil {
			code := firstDiscount.Discount.PromotionCode.Code
			fmt.Printf("Promotion Code: %s\n", code)
			// Output: Promotion Code: TEST003
		}
	}
}
