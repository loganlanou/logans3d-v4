package jobs

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/loganlanou/logans3d-v4/internal/email"
	"github.com/loganlanou/logans3d-v4/storage"
	"github.com/loganlanou/logans3d-v4/storage/db"
)

const (
	// EmailSendInterval is how often we check for emails to send (15 minutes)
	EmailSendInterval = 15 * time.Minute

	// Email timing offsets
	Email1HrOffset     = "-1 hour -5 minutes"   // Send 1hr after abandonment (with 5min buffer)
	Email1HrMinOffset  = "-1 hour -20 minutes"  // Don't send earlier than 1hr 20min ago
	Email24HrOffset    = "-24 hours -5 minutes" // Send 24hr after abandonment
	Email24HrMinOffset = "-24 hours -30 minutes"
	Email72HrOffset    = "-72 hours -5 minutes" // Send 72hr after abandonment
	Email72HrMinOffset = "-72 hours -30 minutes"
)

type AbandonedCartEmailSender struct {
	storage      *storage.Storage
	emailService *email.Service
	ticker       *time.Ticker
	done         chan bool
}

func NewAbandonedCartEmailSender(storage *storage.Storage, emailService *email.Service) *AbandonedCartEmailSender {
	return &AbandonedCartEmailSender{
		storage:      storage,
		emailService: emailService,
		done:         make(chan bool),
	}
}

// Start begins the email sending background job
func (s *AbandonedCartEmailSender) Start(ctx context.Context) {
	slog.Info("starting abandoned cart email sender", "interval", EmailSendInterval)

	// Run immediately on start
	s.sendRecoveryEmails(ctx)

	// Then run on interval
	s.ticker = time.NewTicker(EmailSendInterval)

	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.sendRecoveryEmails(ctx)
			case <-s.done:
				slog.Info("abandoned cart email sender stopped")
				return
			}
		}
	}()
}

// Stop stops the background job
func (s *AbandonedCartEmailSender) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
	close(s.done)
}

// sendRecoveryEmails sends all pending recovery emails
func (s *AbandonedCartEmailSender) sendRecoveryEmails(ctx context.Context) {
	slog.Debug("checking for recovery emails to send")

	// Send 1-hour emails
	sent1hr := s.sendEmailsForAttemptType(ctx, "email_1hr", Email1HrOffset, Email1HrMinOffset)

	// Send 24-hour emails
	sent24hr := s.sendEmailsForAttemptType(ctx, "email_24hr", Email24HrOffset, Email24HrMinOffset)

	// Send 72-hour emails
	sent72hr := s.sendEmailsForAttemptType(ctx, "email_72hr", Email72HrOffset, Email72HrMinOffset)

	total := sent1hr + sent24hr + sent72hr
	if total > 0 {
		slog.Info("recovery emails sent", "1hr", sent1hr, "24hr", sent24hr, "72hr", sent72hr, "total", total)
	} else {
		slog.Debug("no recovery emails to send")
	}
}

// sendEmailsForAttemptType sends emails for a specific attempt type
func (s *AbandonedCartEmailSender) sendEmailsForAttemptType(ctx context.Context, attemptType string, timeOffset string, minTimeOffset string) int {
	// Get carts that need this email
	carts, err := s.storage.Queries.GetCartsNeedingRecoveryEmail(ctx, db.GetCartsNeedingRecoveryEmailParams{
		AttemptType:   attemptType,
		TimeOffset:    timeOffset,
		MinTimeOffset: minTimeOffset,
	})
	if err != nil {
		slog.Error("failed to get carts needing recovery email", "attempt_type", attemptType, "error", err)
		return 0
	}

	if len(carts) == 0 {
		return 0
	}

	slog.Debug("found carts needing recovery email", "attempt_type", attemptType, "count", len(carts))

	sentCount := 0
	for _, cart := range carts {
		// Skip carts without email
		if !cart.CustomerEmail.Valid || cart.CustomerEmail.String == "" {
			continue
		}

		// Get cart snapshots
		snapshots, err := s.storage.Queries.GetCartSnapshotsByAbandonedCartID(ctx, cart.ID)
		if err != nil {
			slog.Error("failed to get cart snapshots", "cart_id", cart.ID, "error", err)
			continue
		}

		// Convert snapshots to email items
		items := make([]email.AbandonedCartItem, 0, len(snapshots))
		for _, snapshot := range snapshots {
			items = append(items, email.AbandonedCartItem{
				ProductName:  snapshot.ProductName,
				ProductImage: snapshot.ProductImageUrl.String,
				Quantity:     snapshot.Quantity,
				UnitPrice:    snapshot.UnitPriceCents,
			})
		}

		// Create email data
		customerName := "Customer"
		if cart.CustomerName.Valid && cart.CustomerName.String != "" {
			customerName = cart.CustomerName.String
		}

		trackingToken := uuid.New().String()

		// Get promo code if available (for 24hr and 72hr emails only)
		// Only include promo code if user has opted into promotional emails
		var promoCode, promoExpires string
		if attemptType == "email_24hr" || attemptType == "email_72hr" {
			// Check if user has opted into promotional emails
			canSendPromo, err := s.emailService.CheckEmailPreference(ctx, cart.CustomerEmail.String, "promotional")
			if err != nil {
				slog.Warn("failed to check promotional preference, not including promo code", "email", cart.CustomerEmail.String, "error", err)
			} else if canSendPromo {
				// User is opted in, get promo code
				cartWithPromo, err := s.storage.Queries.GetAbandonedCartWithPromoCode(ctx, cart.ID)
				if err == nil && cartWithPromo.PromoCode.Valid {
					promoCode = cartWithPromo.PromoCode.String
					if cartWithPromo.PromoExpiresAt.Valid {
						promoExpires = cartWithPromo.PromoExpiresAt.Time.Format("Jan 2, 2006")
					}
				}
			}
		}

		emailData := &email.AbandonedCartData{
			CustomerName:  customerName,
			CustomerEmail: cart.CustomerEmail.String,
			CartValue:     cart.CartValueCents,
			ItemCount:     cart.ItemCount,
			Items:         items,
			TrackingToken: trackingToken,
			AbandonedAt:   cart.AbandonedAt.Format("January 2, 2006 at 3:04 PM"),
			PromoCode:     promoCode,
			PromoExpires:  promoExpires,
		}

		// Send the email
		err = s.emailService.SendAbandonedCartRecoveryEmail(emailData, attemptType)
		if err != nil {
			slog.Error("failed to send recovery email", "cart_id", cart.ID, "attempt_type", attemptType, "error", err)

			// Create failed recovery attempt record
			s.createRecoveryAttempt(ctx, cart.ID, attemptType, trackingToken, "failed")
			continue
		}

		// Create successful recovery attempt record
		s.createRecoveryAttempt(ctx, cart.ID, attemptType, trackingToken, "sent")

		// Update cart status to contacted
		err = s.storage.Queries.MarkCartAsContacted(ctx, cart.ID)
		if err != nil {
			slog.Error("failed to mark cart as contacted", "cart_id", cart.ID, "error", err)
		}

		sentCount++
		slog.Info("sent recovery email", "cart_id", cart.ID, "email", cart.CustomerEmail.String, "attempt_type", attemptType)
	}

	return sentCount
}

// createRecoveryAttempt creates a record of a recovery email attempt
func (s *AbandonedCartEmailSender) createRecoveryAttempt(ctx context.Context, cartID string, attemptType string, trackingToken string, status string) {
	attemptID := uuid.New().String()

	subject := ""
	switch attemptType {
	case "email_1hr":
		subject = "You left something in your cart!"
	case "email_24hr":
		subject = "Still interested in your cart?"
	case "email_72hr":
		subject = "Last chance to complete your order!"
	}

	_, err := s.storage.Queries.CreateRecoveryAttempt(ctx, db.CreateRecoveryAttemptParams{
		ID:              attemptID,
		AbandonedCartID: cartID,
		AttemptType:     attemptType,
		SentAt:          time.Now(),
		EmailSubject:    sql.NullString{String: subject, Valid: true},
		TrackingToken:   sql.NullString{String: trackingToken, Valid: true},
		Status:          sql.NullString{String: status, Valid: true},
	})
	if err != nil {
		slog.Error("failed to create recovery attempt record", "cart_id", cartID, "error", err)
	}
}
