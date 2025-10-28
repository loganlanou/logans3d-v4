package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/loganlanou/logans3d-v4/storage"
	stripego "github.com/stripe/stripe-go/v80"
	checkoutsession "github.com/stripe/stripe-go/v80/checkout/session"
)

func main() {
	dbPath := flag.String("db", "./data/database.db", "Path to SQLite database")
	stripeKey := flag.String("stripe-key", os.Getenv("STRIPE_SECRET_KEY"), "Stripe secret key")
	dryRun := flag.Bool("dry-run", false, "Dry run mode - don't update database")
	flag.Parse()

	if *stripeKey == "" {
		log.Fatal("Stripe secret key is required (--stripe-key or STRIPE_SECRET_KEY env var)")
	}

	// Initialize Stripe
	stripego.Key = *stripeKey

	// Open database
	storage, err := storage.New(*dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()
	queries := storage.Queries

	// Get all orders with Stripe checkout session IDs
	orders, err := queries.ListOrders(ctx)
	if err != nil {
		log.Fatalf("Failed to list orders: %v", err)
	}

	slog.Info("starting backfill", "total_orders", len(orders), "dry_run", *dryRun)

	processed := 0
	updated := 0
	skipped := 0
	errors := 0

	for _, order := range orders {
		processed++

		// Skip orders without a Stripe session ID
		if !order.StripeCheckoutSessionID.Valid || order.StripeCheckoutSessionID.String == "" {
			skipped++
			continue
		}

		// Skip orders that already have discount data
		if order.DiscountCents.Valid && order.DiscountCents.Int64 > 0 {
			skipped++
			slog.Debug("order already has discount data", "order_id", order.ID, "discount_cents", order.DiscountCents.Int64)
			continue
		}

		sessionID := order.StripeCheckoutSessionID.String
		slog.Info("processing order", "order_id", order.ID, "session_id", sessionID, "progress", fmt.Sprintf("%d/%d", processed, len(orders)))

		// Retrieve Stripe session with expanded details
		params := &stripego.CheckoutSessionParams{}
		params.AddExpand("total_details")
		params.AddExpand("total_details.breakdown")

		session, err := checkoutsession.Get(sessionID, params)
		if err != nil {
			slog.Error("failed to retrieve stripe session", "error", err, "session_id", sessionID, "order_id", order.ID)
			errors++
			// Add a small delay to avoid rate limiting on errors
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// Extract discount information
		discountCents := int64(0)
		promotionCode := sql.NullString{}
		promotionCodeID := sql.NullString{}

		if session.TotalDetails != nil && session.TotalDetails.AmountDiscount > 0 {
			discountCents = session.TotalDetails.AmountDiscount

			// Try to get promotion code from session
			if len(session.TotalDetails.Breakdown.Discounts) > 0 {
				firstDiscount := session.TotalDetails.Breakdown.Discounts[0]
				if firstDiscount.Discount != nil && firstDiscount.Discount.PromotionCode != nil {
					code := firstDiscount.Discount.PromotionCode.Code
					promotionCode = sql.NullString{String: code, Valid: true}

					// Try to find this code in our promotion_codes table
					promoRecord, err := queries.GetPromotionCodeByCode(ctx, code)
					if err == nil {
						promotionCodeID = sql.NullString{String: promoRecord.ID, Valid: true}
						slog.Info("promotion code matched to database", "promotion_code_id", promoRecord.ID, "order_id", order.ID)
					} else if err != sql.ErrNoRows {
						slog.Warn("failed to lookup promotion code in database", "error", err, "code", code)
					}
				}
			}
		}

		// Calculate original subtotal (before discount)
		originalSubtotalCents := order.SubtotalCents + discountCents

		if discountCents > 0 {
			slog.Info("found discount for order",
				"order_id", order.ID,
				"discount_cents", discountCents,
				"promotion_code", promotionCode.String,
				"original_subtotal_cents", originalSubtotalCents)

			if !*dryRun {
				// Update the order with discount information
				_, err = storage.DB().ExecContext(ctx, `
					UPDATE orders
					SET original_subtotal_cents = ?,
						discount_cents = ?,
						promotion_code = ?,
						promotion_code_id = ?,
						updated_at = CURRENT_TIMESTAMP
					WHERE id = ?
				`, originalSubtotalCents, discountCents, promotionCode, promotionCodeID, order.ID)

				if err != nil {
					slog.Error("failed to update order", "error", err, "order_id", order.ID)
					errors++
					continue
				}

				slog.Info("updated order with discount data", "order_id", order.ID)
			}

			updated++
		} else {
			slog.Debug("no discount found for order", "order_id", order.ID)
			skipped++
		}

		// Add a small delay to avoid Stripe rate limits (100 requests per second)
		time.Sleep(100 * time.Millisecond)
	}

	slog.Info("backfill complete",
		"total_orders", len(orders),
		"processed", processed,
		"updated", updated,
		"skipped", skipped,
		"errors", errors,
		"dry_run", *dryRun)

	if *dryRun {
		slog.Info("DRY RUN - no changes were made to the database")
	}
}
