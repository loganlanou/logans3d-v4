package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/loganlanou/logans3d-v4/internal/stripe"
	"github.com/loganlanou/logans3d-v4/storage"
	"github.com/loganlanou/logans3d-v4/storage/db"
	"github.com/oklog/ulid/v2"
)

const (
	// AbandonmentThreshold is the time after which a cart is considered abandoned (30 minutes)
	AbandonmentThreshold = 30 * time.Minute

	// DetectionInterval is how often we check for abandoned carts (5 minutes)
	DetectionInterval = 5 * time.Minute
)

type AbandonedCartDetector struct {
	storage *storage.Storage
	ticker  *time.Ticker
	done    chan bool
}

func NewAbandonedCartDetector(storage *storage.Storage) *AbandonedCartDetector {
	return &AbandonedCartDetector{
		storage: storage,
		done:    make(chan bool),
	}
}

// Start begins the abandoned cart detection background job
func (d *AbandonedCartDetector) Start(ctx context.Context) {
	slog.Info("starting abandoned cart detector", "interval", DetectionInterval, "threshold", AbandonmentThreshold)

	// Run immediately on start
	d.detectAbandonedCarts(ctx)

	// Then run on interval
	d.ticker = time.NewTicker(DetectionInterval)

	go func() {
		for {
			select {
			case <-d.ticker.C:
				d.detectAbandonedCarts(ctx)
			case <-d.done:
				slog.Info("abandoned cart detector stopped")
				return
			}
		}
	}()
}

// Stop stops the background job
func (d *AbandonedCartDetector) Stop() {
	if d.ticker != nil {
		d.ticker.Stop()
	}
	close(d.done)
}

// detectAbandonedCarts finds and marks carts as abandoned
func (d *AbandonedCartDetector) detectAbandonedCarts(ctx context.Context) {
	slog.Debug("running abandoned cart detection")

	// Find carts that haven't been updated in 30+ minutes
	cutoffTime := time.Now().Add(-AbandonmentThreshold)

	// Query to find potentially abandoned carts
	// We need to check both session-based and user-based carts
	query := `
		SELECT
			COALESCE(ci.session_id, '') as session_id,
			COALESCE(ci.user_id, '') as user_id,
			MAX(ci.updated_at) as last_update,
			COUNT(DISTINCT ci.id) as item_count,
			SUM(p.price_cents * ci.quantity) as cart_value
		FROM cart_items ci
		JOIN products p ON ci.product_id = p.id
		WHERE ci.updated_at < ?
		GROUP BY COALESCE(ci.session_id, ''), COALESCE(ci.user_id, '')
		HAVING item_count > 0
	`

	rows, err := d.storage.DB().QueryContext(ctx, query, cutoffTime)
	if err != nil {
		slog.Error("failed to query potentially abandoned carts", "error", err)
		return
	}
	defer rows.Close()

	var processedCount int
	var newAbandonedCount int

	for rows.Next() {
		var sessionID, userID string
		var lastUpdateStr string
		var itemCount int64
		var cartValue int64

		if err := rows.Scan(&sessionID, &userID, &lastUpdateStr, &itemCount, &cartValue); err != nil {
			slog.Error("failed to scan abandoned cart row", "error", err)
			continue
		}

		// Parse the timestamp string from SQLite
		lastUpdate, err := time.Parse("2006-01-02 15:04:05", lastUpdateStr)
		if err != nil {
			slog.Error("failed to parse last_update timestamp", "error", err, "value", lastUpdateStr)
			continue
		}

		processedCount++

		// Check if this cart was already marked as abandoned
		var checkErr error

		if sessionID != "" {
			_, checkErr = d.storage.Queries.GetAbandonedCartBySession(ctx, sql.NullString{String: sessionID, Valid: true})
		} else if userID != "" {
			// Check by user_id if no session_id
			_, checkErr = d.storage.Queries.GetAbandonedCartByUser(ctx, sql.NullString{String: userID, Valid: true})
		}

		// If no existing abandoned cart record, create one
		if checkErr == sql.ErrNoRows {
			err := d.createAbandonedCartRecord(ctx, sessionID, userID, itemCount, cartValue, lastUpdate)
			if err != nil {
				slog.Error("failed to create abandoned cart record", "error", err, "session_id", sessionID, "user_id", userID)
				continue
			}
			newAbandonedCount++
			slog.Info("detected new abandoned cart", "session_id", sessionID, "user_id", userID, "value", cartValue, "items", itemCount)
		}
	}

	if newAbandonedCount > 0 {
		slog.Info("abandoned cart detection complete", "processed", processedCount, "new_abandoned", newAbandonedCount)
	} else {
		slog.Debug("abandoned cart detection complete", "processed", processedCount, "new_abandoned", newAbandonedCount)
	}
}

func (d *AbandonedCartDetector) createAbandonedCartRecord(
	ctx context.Context,
	sessionID string,
	userID string,
	itemCount int64,
	cartValue int64,
	abandonedAt time.Time,
) error {
	// Get customer info if available
	var customerEmail, customerName sql.NullString
	if userID != "" {
		user, err := d.storage.Queries.GetUser(ctx, userID)
		if err == nil {
			customerEmail = sql.NullString{String: user.Email, Valid: true}
			customerName = sql.NullString{String: user.FullName, Valid: true}
		}
	}

	// Create abandoned cart record
	cartID := uuid.New().String()
	_, err := d.storage.Queries.CreateAbandonedCart(ctx, db.CreateAbandonedCartParams{
		ID:             cartID,
		SessionID:      sql.NullString{String: sessionID, Valid: sessionID != ""},
		UserID:         sql.NullString{String: userID, Valid: userID != ""},
		CustomerEmail:  customerEmail,
		CustomerName:   customerName,
		CartValueCents: cartValue,
		ItemCount:      itemCount,
		AbandonedAt:    abandonedAt,
		Status:         sql.NullString{String: "active", Valid: true},
	})
	if err != nil {
		return err
	}

	// Check if customer is a first-time customer and generate promo code
	if customerEmail.Valid && customerEmail.String != "" {
		promoCodeID := d.generatePromoCodeForFirstTimer(ctx, userID, customerEmail.String)
		if promoCodeID != "" {
			// Link promo code to abandoned cart
			err := d.storage.Queries.UpdateAbandonedCartPromoCode(ctx, db.UpdateAbandonedCartPromoCodeParams{
				PromotionCodeID: sql.NullString{String: promoCodeID, Valid: true},
				ID:              cartID,
			})
			if err != nil {
				slog.Error("failed to link promo code to abandoned cart", "error", err, "cart_id", cartID, "promo_code_id", promoCodeID)
			} else {
				slog.Info("linked 5% promo code to abandoned cart", "cart_id", cartID, "email", customerEmail.String)
			}
		}
	}

	// Create snapshots of cart items
	err = d.createCartSnapshots(ctx, cartID, sessionID, userID)
	if err != nil {
		slog.Error("failed to create cart snapshots", "error", err, "cart_id", cartID)
		// Don't fail the whole operation if snapshots fail
	}

	return nil
}

// generatePromoCodeForFirstTimer checks if user has never purchased and generates a 5% promo code
func (d *AbandonedCartDetector) generatePromoCodeForFirstTimer(ctx context.Context, userID, email string) string {
	// Check if user has made any purchases
	hasPurchased, err := d.storage.Queries.HasUserMadePurchase(ctx, db.HasUserMadePurchaseParams{
		UserID:        sql.NullString{String: userID, Valid: userID != ""},
		CustomerEmail: sql.NullString{String: email, Valid: true},
	})
	if err != nil {
		slog.Error("failed to check user purchase history", "error", err, "user_id", userID, "email", email)
		return ""
	}

	// Only generate code for first-time customers
	if hasPurchased {
		return ""
	}

	// Get or create abandoned cart campaign
	campaign, err := d.getOrCreateAbandonedCartCampaign(ctx)
	if err != nil {
		slog.Error("failed to get abandoned cart campaign", "error", err)
		return ""
	}

	// Generate unique code (CART5-XXXX format)
	codeStr := fmt.Sprintf("CART5-%s", ulid.Make().String()[0:8])

	// Create Stripe promotion code
	stripePromoCode, err := stripe.CreateUniquePromotionCode(
		campaign.StripePromotionID.String,
		codeStr,
		email,
		10, // 10 days expiration
	)
	if err != nil {
		slog.Error("failed to create Stripe promotion code", "error", err, "code", codeStr)
		return ""
	}

	// Create promotion code in database
	promoCode, err := d.storage.Queries.CreatePromotionCode(ctx, db.CreatePromotionCodeParams{
		ID:                    ulid.Make().String(),
		CampaignID:            campaign.ID,
		Code:                  codeStr,
		StripePromotionCodeID: sql.NullString{String: stripePromoCode.ID, Valid: true},
		Email:                 sql.NullString{String: email, Valid: true},
		UserID:                sql.NullString{String: userID, Valid: userID != ""},
		MaxUses:               sql.NullInt64{Int64: 1, Valid: true},
		ExpiresAt:             sql.NullTime{Time: time.Now().AddDate(0, 0, 10), Valid: true},
	})
	if err != nil {
		slog.Error("failed to create promotion code in database", "error", err, "code", codeStr)
		return ""
	}

	return promoCode.ID
}

// getOrCreateAbandonedCartCampaign gets or creates the 5% abandoned cart campaign
func (d *AbandonedCartDetector) getOrCreateAbandonedCartCampaign(ctx context.Context) (*db.PromotionCampaign, error) {
	// Try to get existing campaign
	campaigns, err := d.storage.Queries.GetActivePromotionCampaigns(ctx)
	if err == nil {
		for _, campaign := range campaigns {
			if campaign.Name == "Abandoned Cart Recovery - 5% Off" {
				return &campaign, nil
			}
		}
	}

	// Create new campaign
	stripeCoupon, err := stripe.CreatePromotionCampaign("Abandoned Cart Recovery - 5% Off", "percentage", 5)
	if err != nil {
		return nil, err
	}

	campaign, err := d.storage.Queries.CreatePromotionCampaign(ctx, db.CreatePromotionCampaignParams{
		ID:                ulid.Make().String(),
		Name:              "Abandoned Cart Recovery - 5% Off",
		Description:       sql.NullString{String: "5% discount for first-time customers who abandoned their cart", Valid: true},
		DiscountType:      "percentage",
		DiscountValue:     5,
		StripePromotionID: sql.NullString{String: stripeCoupon.ID, Valid: true},
		StartDate:         time.Now(),
		EndDate:           sql.NullTime{},
		MaxUses:           sql.NullInt64{},
		Active:            sql.NullInt64{Int64: 1, Valid: true},
	})

	return &campaign, err
}

func (d *AbandonedCartDetector) createCartSnapshots(ctx context.Context, abandonedCartID, sessionID, userID string) error {
	// Handle session-based carts
	if sessionID != "" {
		cartItems, err := d.storage.Queries.GetCartBySession(ctx, sql.NullString{String: sessionID, Valid: true})
		if err != nil {
			return err
		}

		for _, item := range cartItems {
			snapshotID := uuid.New().String()
			unitPrice := priceToInt64(item.PriceCents)
			totalPrice := unitPrice * item.Quantity
			name := item.Name
			if item.VariantName != "" {
				name = fmt.Sprintf("%s (%s)", item.Name, item.VariantName)
			}
			variantSKU := item.VariantSku

			err = d.storage.Queries.CreateCartSnapshot(ctx, db.CreateCartSnapshotParams{
				ID:              snapshotID,
				AbandonedCartID: abandonedCartID,
				ProductID:       item.ProductID,
				ProductName:     name,
				ProductSku:      sql.NullString{String: variantSKU, Valid: variantSKU != ""},
				ProductImageUrl: sql.NullString{String: item.ImageUrl, Valid: item.ImageUrl != ""},
				Quantity:        item.Quantity,
				UnitPriceCents:  unitPrice,
				TotalPriceCents: totalPrice,
			})
			if err != nil {
				slog.Error("failed to create cart snapshot", "error", err, "product_id", item.ProductID)
				continue
			}
		}
	} else if userID != "" {
		// Handle user-based carts
		cartItems, err := d.storage.Queries.GetCartByUser(ctx, sql.NullString{String: userID, Valid: true})
		if err != nil {
			return err
		}

		for _, item := range cartItems {
			snapshotID := uuid.New().String()
			unitPrice := priceToInt64(item.PriceCents)
			totalPrice := unitPrice * item.Quantity
			name := item.Name
			if item.VariantName != "" {
				name = fmt.Sprintf("%s (%s)", item.Name, item.VariantName)
			}
			variantSKU := item.VariantSku

			err = d.storage.Queries.CreateCartSnapshot(ctx, db.CreateCartSnapshotParams{
				ID:              snapshotID,
				AbandonedCartID: abandonedCartID,
				ProductID:       item.ProductID,
				ProductName:     name,
				ProductSku:      sql.NullString{String: variantSKU, Valid: variantSKU != ""},
				ProductImageUrl: sql.NullString{String: item.ImageUrl, Valid: item.ImageUrl != ""},
				Quantity:        item.Quantity,
				UnitPriceCents:  unitPrice,
				TotalPriceCents: totalPrice,
			})
			if err != nil {
				slog.Error("failed to create cart snapshot", "error", err, "product_id", item.ProductID)
				continue
			}
		}
	}

	return nil
}

func priceToInt64(v interface{}) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case float64:
		return int64(val)
	case int:
		return int64(val)
	case sql.NullFloat64:
		if val.Valid {
			return int64(val.Float64)
		}
	case sql.NullInt64:
		if val.Valid {
			return val.Int64
		}
	}
	return 0
}

// CleanupExpiredCarts marks old abandoned carts as expired and deletes very old ones
func (d *AbandonedCartDetector) CleanupExpiredCarts(ctx context.Context) {
	slog.Debug("running abandoned cart cleanup")

	// Mark carts older than 30 days as expired
	err := d.storage.Queries.MarkExpiredAbandonedCarts(ctx)
	if err != nil {
		slog.Error("failed to mark expired carts", "error", err)
	}

	// Delete carts older than 90 days
	err = d.storage.Queries.DeleteOldAbandonedCarts(ctx)
	if err != nil {
		slog.Error("failed to delete old carts", "error", err)
	}

	slog.Debug("abandoned cart cleanup complete")
}
