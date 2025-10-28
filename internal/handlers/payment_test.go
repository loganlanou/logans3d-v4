package handlers

import (
	"context"
	"database/sql"
	"testing"
	"time"

	emailutil "github.com/loganlanou/logans3d-v4/internal/email"
	"github.com/loganlanou/logans3d-v4/storage/db"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	stripe "github.com/stripe/stripe-go/v80"
)

// createMockCheckoutSession creates a mock Stripe checkout session for testing
func createMockCheckoutSession(promoCode string, discountAmount int64) *stripe.CheckoutSession {
	session := &stripe.CheckoutSession{
		ID:          "cs_test_123",
		AmountTotal: 1500, // $15.00
		CustomerDetails: &stripe.CheckoutSessionCustomerDetails{
			Email: "test@example.com",
			Name:  "Test User",
		},
		LineItems: &stripe.LineItemList{
			Data: []*stripe.LineItem{
				{
					Price: &stripe.Price{
						Product: &stripe.Product{
							ID: "prod_test",
						},
					},
					Quantity: 1,
				},
			},
		},
	}

	// Add discount information if promo code provided
	if promoCode != "" && discountAmount > 0 {
		session.TotalDetails = &stripe.CheckoutSessionTotalDetails{
			AmountDiscount: discountAmount,
			Breakdown: &stripe.CheckoutSessionTotalDetailsBreakdown{
				Discounts: []*stripe.CheckoutSessionTotalDetailsBreakdownDiscount{
					{
						Amount: discountAmount,
						Discount: &stripe.Discount{
							Coupon: &stripe.Coupon{
								AmountOff: int64(discountAmount),
								Currency:  "usd",
							},
							PromotionCode: &stripe.PromotionCode{
								Code: promoCode,
							},
						},
					},
				},
			},
		}
	}

	return session
}

// createMockCheckoutSessionPercentOff creates a mock session with percent-off discount
func createMockCheckoutSessionPercentOff(promoCode string, percentOff int64) *stripe.CheckoutSession {
	session := &stripe.CheckoutSession{
		ID:          "cs_test_123",
		AmountTotal: 1275, // $12.75 (after 15% discount)
		CustomerDetails: &stripe.CheckoutSessionCustomerDetails{
			Email: "test@example.com",
			Name:  "Test User",
		},
		LineItems: &stripe.LineItemList{
			Data: []*stripe.LineItem{
				{
					Price: &stripe.Price{
						Product: &stripe.Product{
							ID: "prod_test",
						},
					},
					Quantity: 1,
				},
			},
		},
		TotalDetails: &stripe.CheckoutSessionTotalDetails{
			AmountDiscount: 225, // $2.25 discount (15% of $15)
			Breakdown: &stripe.CheckoutSessionTotalDetailsBreakdown{
				Discounts: []*stripe.CheckoutSessionTotalDetailsBreakdownDiscount{
					{
						Amount: 225,
						Discount: &stripe.Discount{
							Coupon: &stripe.Coupon{
								PercentOff: float64(percentOff),
							},
							PromotionCode: &stripe.PromotionCode{
								Code: promoCode,
							},
						},
					},
				},
			},
		},
	}

	return session
}

// TestGetOrCreateCampaignForDiscount_AmountOff tests creating campaign for fixed amount discount
func TestGetOrCreateCampaignForDiscount_AmountOff(t *testing.T) {
	_, queries, cleanup := NewTestDB()
	defer cleanup()

	emailService := emailutil.NewService(queries)
	handler := NewPaymentHandler(queries, emailService)
	ctx := context.Background()

	// Create mock discount breakdown with $5 off
	discount := &stripe.CheckoutSessionTotalDetailsBreakdownDiscount{
		Amount: 500, // $5.00
		Discount: &stripe.Discount{
			Coupon: &stripe.Coupon{
				AmountOff: 500,
				Currency:  "usd",
			},
		},
	}

	// First call should create campaign
	campaign1, err := handler.getOrCreateCampaignForDiscount(ctx, discount)
	require.NoError(t, err)
	assert.Equal(t, "External Stripe - $5.00 Off", campaign1.Name)
	assert.Equal(t, "amount", campaign1.DiscountType)
	assert.Equal(t, int64(500), campaign1.DiscountValue)

	// Second call with same discount should return existing campaign
	campaign2, err := handler.getOrCreateCampaignForDiscount(ctx, discount)
	require.NoError(t, err)
	assert.Equal(t, campaign1.ID, campaign2.ID, "Should return same campaign")
}

// TestGetOrCreateCampaignForDiscount_PercentOff tests creating campaign for percent discount
func TestGetOrCreateCampaignForDiscount_PercentOff(t *testing.T) {
	_, queries, cleanup := NewTestDB()
	defer cleanup()

	emailService := emailutil.NewService(queries)
	handler := NewPaymentHandler(queries, emailService)
	ctx := context.Background()

	// Create mock discount breakdown with 15% off
	discount := &stripe.CheckoutSessionTotalDetailsBreakdownDiscount{
		Amount: 225, // Actual amount discounted
		Discount: &stripe.Discount{
			Coupon: &stripe.Coupon{
				PercentOff: 15.0,
			},
		},
	}

	// Create campaign
	campaign, err := handler.getOrCreateCampaignForDiscount(ctx, discount)
	require.NoError(t, err)
	assert.Equal(t, "External Stripe - 15% Off", campaign.Name)
	assert.Equal(t, "percentage", campaign.DiscountType)
	assert.Equal(t, int64(15), campaign.DiscountValue)
}

// TestCreateExternalPromotionCode_New tests creating a new external promotion code
func TestCreateExternalPromotionCode_New(t *testing.T) {
	_, queries, cleanup := NewTestDB()
	defer cleanup()

	emailService := emailutil.NewService(queries)
	handler := NewPaymentHandler(queries, emailService)
	ctx := context.Background()

	// Create mock discount breakdown
	discount := &stripe.CheckoutSessionTotalDetailsBreakdownDiscount{
		Amount: 500,
		Discount: &stripe.Discount{
			Coupon: &stripe.Coupon{
				AmountOff: 500,
				Currency:  "usd",
			},
			PromotionCode: &stripe.PromotionCode{
				Code: "TEST123",
			},
		},
	}

	// Create promotion code
	promoCode, err := handler.createExternalPromotionCode(ctx, "TEST123", discount)
	require.NoError(t, err)
	assert.Equal(t, "TEST123", promoCode.Code)
	assert.NotEmpty(t, promoCode.CampaignID)

	// Verify code exists in database
	retrievedCode, err := queries.GetPromotionCodeByCode(ctx, "TEST123")
	require.NoError(t, err)
	assert.Equal(t, "TEST123", retrievedCode.Code)
	assert.Equal(t, promoCode.ID, retrievedCode.ID)
}

// TestCreateExternalPromotionCode_Duplicate tests handling duplicate code creation
func TestCreateExternalPromotionCode_Duplicate(t *testing.T) {
	_, queries, cleanup := NewTestDB()
	defer cleanup()

	emailService := emailutil.NewService(queries)
	handler := NewPaymentHandler(queries, emailService)
	ctx := context.Background()

	discount := &stripe.CheckoutSessionTotalDetailsBreakdownDiscount{
		Amount: 500,
		Discount: &stripe.Discount{
			Coupon: &stripe.Coupon{
				AmountOff: 500,
				Currency:  "usd",
			},
			PromotionCode: &stripe.PromotionCode{
				Code: "DUPLICATE123",
			},
		},
	}

	// Create first code
	promoCode1, err := handler.createExternalPromotionCode(ctx, "DUPLICATE123", discount)
	require.NoError(t, err)

	// Try to create duplicate - should handle gracefully by returning existing
	promoCode2, err := handler.createExternalPromotionCode(ctx, "DUPLICATE123", discount)
	require.NoError(t, err)
	assert.Equal(t, promoCode1.ID, promoCode2.ID, "Should return existing code")
}

// TestPromotionCodeExtraction_WithCode tests extracting promo code from session
func TestPromotionCodeExtraction_WithCode(t *testing.T) {
	_, queries, cleanup := NewTestDB()
	defer cleanup()

	ctx := context.Background()

	// Create mock session with promotion code
	session := createMockCheckoutSession("TEST003", 500)

	// Verify session structure
	require.NotNil(t, session.TotalDetails)
	require.NotNil(t, session.TotalDetails.Breakdown)
	require.Len(t, session.TotalDetails.Breakdown.Discounts, 1)

	discount := session.TotalDetails.Breakdown.Discounts[0]
	require.NotNil(t, discount.Discount)
	require.NotNil(t, discount.Discount.PromotionCode)
	assert.Equal(t, "TEST003", discount.Discount.PromotionCode.Code)

	// Test code doesn't exist yet
	_, err := queries.GetPromotionCodeByCode(ctx, "TEST003")
	assert.Error(t, err, "Code should not exist yet")
}

// TestPromotionCodeExtraction_WithoutCode tests session without promotion code
func TestPromotionCodeExtraction_WithoutCode(t *testing.T) {
	cleanup := func() {}
	defer cleanup()

	// Create mock session without promotion code
	session := createMockCheckoutSession("", 0)

	// Verify no discount information
	if session.TotalDetails != nil {
		assert.Nil(t, session.TotalDetails.Breakdown)
	}
}

// TestCampaignNaming tests campaign name generation for different discount types
func TestCampaignNaming(t *testing.T) {
	_, queries, cleanup := NewTestDB()
	defer cleanup()

	emailService := emailutil.NewService(queries)
	handler := NewPaymentHandler(queries, emailService)
	ctx := context.Background()

	testCases := []struct {
		name          string
		discount      *stripe.CheckoutSessionTotalDetailsBreakdownDiscount
		expectedName  string
		expectedType  string
		expectedValue int64
	}{
		{
			name: "Fixed $10 discount",
			discount: &stripe.CheckoutSessionTotalDetailsBreakdownDiscount{
				Amount: 1000,
				Discount: &stripe.Discount{
					Coupon: &stripe.Coupon{
						AmountOff: 1000,
						Currency:  "usd",
					},
				},
			},
			expectedName:  "External Stripe - $10.00 Off",
			expectedType:  "amount",
			expectedValue: 1000,
		},
		{
			name: "20% discount",
			discount: &stripe.CheckoutSessionTotalDetailsBreakdownDiscount{
				Amount: 400,
				Discount: &stripe.Discount{
					Coupon: &stripe.Coupon{
						PercentOff: 20.0,
					},
				},
			},
			expectedName:  "External Stripe - 20% Off",
			expectedType:  "percentage",
			expectedValue: 20,
		},
		{
			name: "5% discount",
			discount: &stripe.CheckoutSessionTotalDetailsBreakdownDiscount{
				Amount: 100,
				Discount: &stripe.Discount{
					Coupon: &stripe.Coupon{
						PercentOff: 5.0,
					},
				},
			},
			expectedName:  "External Stripe - 5% Off",
			expectedType:  "percentage",
			expectedValue: 5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			campaign, err := handler.getOrCreateCampaignForDiscount(ctx, tc.discount)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedName, campaign.Name)
			assert.Equal(t, tc.expectedType, campaign.DiscountType)
			assert.Equal(t, tc.expectedValue, campaign.DiscountValue)
		})
	}
}

// TestPromotionCodeInOrderCreation tests that promotion codes are properly stored in orders
func TestPromotionCodeInOrderCreation(t *testing.T) {
	_, queries, cleanup := NewTestDB()
	defer cleanup()

	ctx := context.Background()

	// Create a campaign first
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

	// Create a promotion code
	promoCode, err := queries.CreatePromotionCode(ctx, db.CreatePromotionCodeParams{
		ID:                    ulid.Make().String(),
		CampaignID:            campaign.ID,
		Code:                  "ORDER123",
		StripePromotionCodeID: sql.NullString{String: "stripe_order123", Valid: true},
	})
	require.NoError(t, err)

	// Create a test user first (required for foreign key)
	user, err := CreateTestUser(queries)
	require.NoError(t, err)

	// Create an order with the promotion code
	order, err := queries.CreateOrder(ctx, db.CreateOrderParams{
		ID:                      ulid.Make().String(),
		UserID:                  user.ID,
		CustomerEmail:           "customer@example.com",
		CustomerName:            "Test Customer",
		SubtotalCents:           1275,                                    // $12.75 after discount
		OriginalSubtotalCents:   sql.NullInt64{Int64: 1500, Valid: true}, // $15.00 original
		DiscountCents:           sql.NullInt64{Int64: 225, Valid: true},  // $2.25 discount
		PromotionCode:           sql.NullString{String: "ORDER123", Valid: true},
		PromotionCodeID:         sql.NullString{String: promoCode.ID, Valid: true},
		TaxCents:                0,
		ShippingCents:           0,
		TotalCents:              1275,
		StripeCheckoutSessionID: sql.NullString{String: "cs_test", Valid: true},
		Status:                  sql.NullString{String: "pending", Valid: true},
	})
	require.NoError(t, err)

	// Verify order has promotion code
	assert.True(t, order.PromotionCode.Valid)
	assert.Equal(t, "ORDER123", order.PromotionCode.String)
	assert.True(t, order.PromotionCodeID.Valid)
	assert.Equal(t, promoCode.ID, order.PromotionCodeID.String)
	assert.True(t, order.DiscountCents.Valid)
	assert.Equal(t, int64(225), order.DiscountCents.Int64)
	assert.True(t, order.OriginalSubtotalCents.Valid)
	assert.Equal(t, int64(1500), order.OriginalSubtotalCents.Int64)
}

// TestBreakdownExpansion tests that breakdown must be explicitly expanded
func TestBreakdownExpansion(t *testing.T) {
	t.Run("With breakdown expanded", func(t *testing.T) {
		session := createMockCheckoutSession("TEST003", 500)

		require.NotNil(t, session.TotalDetails)
		require.NotNil(t, session.TotalDetails.Breakdown, "Breakdown should be present when expanded")
		require.Len(t, session.TotalDetails.Breakdown.Discounts, 1)

		code := session.TotalDetails.Breakdown.Discounts[0].Discount.PromotionCode.Code
		assert.Equal(t, "TEST003", code)
	})

	t.Run("Without breakdown expanded (simulated)", func(t *testing.T) {
		// Simulate what Stripe returns when breakdown is NOT expanded
		session := &stripe.CheckoutSession{
			ID: "cs_test_123",
			TotalDetails: &stripe.CheckoutSessionTotalDetails{
				AmountDiscount: 500, // Discount amount is present
				Breakdown:      nil, // But breakdown is nil!
			},
		}

		assert.NotNil(t, session.TotalDetails)
		assert.Equal(t, int64(500), session.TotalDetails.AmountDiscount, "Discount amount is visible")
		assert.Nil(t, session.TotalDetails.Breakdown, "Breakdown is nil without expansion")

		// This is why we need to expand "total_details.breakdown" explicitly
	})
}
