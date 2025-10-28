package email

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/smtp"
	"os"
	"strconv"

	"github.com/loganlanou/logans3d-v4/storage/db"
	"github.com/oklog/ulid/v2"
)

// Service handles email sending via Brevo SMTP
type Service struct {
	host     string
	port     int
	username string
	password string
	from     string
	queries  *db.Queries
}

// NewService creates a new email service configured with Brevo SMTP
func NewService(queries *db.Queries) *Service {
	port, err := strconv.Atoi(os.Getenv("BREVO_SMTP_PORT"))
	if err != nil {
		port = 587 // default
	}

	return &Service{
		host:     os.Getenv("BREVO_SMTP_HOST"),
		port:     port,
		username: os.Getenv("BREVO_SMTP_LOGIN"),
		password: os.Getenv("BREVO_SMTP_KEY"),
		from:     os.Getenv("EMAIL_FROM"),
		queries:  queries,
	}
}

// GenerateUnsubscribeToken generates a secure random token for unsubscribe links
func GenerateUnsubscribeToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CheckEmailPreference checks if a user has opted in to receive a specific email type
// Returns true if the user can receive the email, false otherwise
func (s *Service) CheckEmailPreference(ctx context.Context, email string, emailType string) (bool, error) {
	if s.queries == nil {
		// If no database connection, allow all emails (backward compatibility)
		return true, nil
	}

	// Transactional emails always allowed (order confirmations, etc.)
	if emailType == "transactional" {
		return true, nil
	}

	// Check if preferences exist for this email
	prefs, err := s.queries.GetEmailPreferencesByEmail(ctx, email)
	if err != nil {
		if err == sql.ErrNoRows {
			// No preferences set - default to allowing abandoned cart, denying others
			if emailType == "abandoned_cart" {
				return true, nil
			}
			return false, nil
		}
		return false, fmt.Errorf("failed to get email preferences: %w", err)
	}

	// Check the specific preference
	switch emailType {
	case "abandoned_cart":
		return prefs.AbandonedCart.Valid && prefs.AbandonedCart.Int64 == 1, nil
	case "promotional":
		return prefs.Promotional.Valid && prefs.Promotional.Int64 == 1, nil
	case "newsletter":
		return prefs.Newsletter.Valid && prefs.Newsletter.Int64 == 1, nil
	case "product_updates":
		return prefs.ProductUpdates.Valid && prefs.ProductUpdates.Int64 == 1, nil
	default:
		return false, nil
	}
}

// LogEmailSend logs an email send to the database
func (s *Service) LogEmailSend(ctx context.Context, recipientEmail, emailType, subject, templateName, trackingToken string, metadata map[string]interface{}) error {
	if s.queries == nil {
		// If no database connection, skip logging (backward compatibility)
		return nil
	}

	var metadataJSON sql.NullString
	if metadata != nil {
		jsonBytes, err := json.Marshal(metadata)
		if err != nil {
			slog.Warn("failed to marshal email metadata", "error", err)
		} else {
			metadataJSON = sql.NullString{String: string(jsonBytes), Valid: true}
		}
	}

	var trackingTokenNullable sql.NullString
	if trackingToken != "" {
		trackingTokenNullable = sql.NullString{String: trackingToken, Valid: true}
	}

	_, err := s.queries.CreateEmailHistory(ctx, db.CreateEmailHistoryParams{
		ID:             ulid.Make().String(),
		UserID:         sql.NullString{}, // Will be set by caller if known
		RecipientEmail: recipientEmail,
		EmailType:      emailType,
		Subject:        subject,
		TemplateName:   templateName,
		TrackingToken:  trackingTokenNullable,
		Metadata:       metadataJSON,
	})

	if err != nil {
		slog.Error("failed to log email send", "error", err, "email", recipientEmail, "type", emailType)
		return fmt.Errorf("failed to log email: %w", err)
	}

	return nil
}

// GetOrCreateEmailPreferences gets existing preferences or creates new ones with default values
func (s *Service) GetOrCreateEmailPreferences(ctx context.Context, email string, userID *string) (*db.EmailPreference, error) {
	if s.queries == nil {
		return nil, fmt.Errorf("database queries not available")
	}

	// Try to get existing preferences
	prefs, err := s.queries.GetEmailPreferencesByEmail(ctx, email)
	if err == nil {
		return &prefs, nil
	}

	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get email preferences: %w", err)
	}

	// Create new preferences with default values
	token, err := GenerateUnsubscribeToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate unsubscribe token: %w", err)
	}

	var userIDNullable sql.NullString
	if userID != nil {
		userIDNullable = sql.NullString{String: *userID, Valid: true}
	}

	newPrefs, err := s.queries.GetOrCreateEmailPreferences(ctx, db.GetOrCreateEmailPreferencesParams{
		ID:               ulid.Make().String(),
		UserID:           userIDNullable,
		Email:            email,
		UnsubscribeToken: sql.NullString{String: token, Valid: true},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create email preferences: %w", err)
	}

	return &newPrefs, nil
}

// Email represents an email message
type Email struct {
	To      []string
	Subject string
	Body    string
	IsHTML  bool
	ReplyTo string
}

// Send sends an email via Brevo SMTP
func (s *Service) Send(email *Email) error {
	// Validate configuration
	if s.host == "" || s.password == "" || s.from == "" {
		return fmt.Errorf("email service not configured: missing BREVO_SMTP_HOST, BREVO_SMTP_KEY, or EMAIL_FROM")
	}

	// Build email message
	var msg bytes.Buffer
	msg.WriteString(fmt.Sprintf("From: %s\r\n", s.from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", email.To[0]))
	if email.ReplyTo != "" {
		msg.WriteString(fmt.Sprintf("Reply-To: %s\r\n", email.ReplyTo))
	}
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", email.Subject))

	if email.IsHTML {
		msg.WriteString("MIME-Version: 1.0\r\n")
		msg.WriteString("Content-Type: text/html; charset=\"utf-8\"\r\n")
	}

	msg.WriteString("\r\n")
	msg.WriteString(email.Body)

	// Set up authentication
	auth := smtp.PlainAuth("", s.username, s.password, s.host)

	// Send email
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	err := smtp.SendMail(addr, auth, s.from, email.To, msg.Bytes())
	if err != nil {
		slog.Error("failed to send email", "error", err, "to", email.To)
		return fmt.Errorf("failed to send email: %w", err)
	}

	slog.Info("email sent successfully", "to", email.To, "subject", email.Subject)
	return nil
}

// OrderData contains all the data needed for order emails
type OrderData struct {
	OrderID           string
	CustomerName      string
	CustomerEmail     string
	OrderDate         string
	Items             []OrderItem
	SubtotalCents     int64
	TaxCents          int64
	ShippingCents     int64
	TotalCents        int64
	ShippingAddress   Address
	BillingAddress    Address
	PaymentIntentID   string
}

// OrderItem represents a single item in an order
type OrderItem struct {
	ProductName  string
	ProductImage string
	Quantity     int64
	PriceCents   int64
	TotalCents   int64
}

// Address represents a shipping or billing address
type Address struct {
	Name       string
	Line1      string
	Line2      string
	City       string
	State      string
	PostalCode string
	Country    string
}

// FormatCents converts cents to dollar string (e.g., 1234 -> "$12.34")
func FormatCents(cents int64) string {
	dollars := float64(cents) / 100.0
	return fmt.Sprintf("$%.2f", dollars)
}

// SendOrderConfirmation sends an order confirmation email to the customer
func (s *Service) SendOrderConfirmation(data *OrderData) error {
	// Render the full email (content + base template)
	html, err := RenderCustomerOrderEmail(data)
	if err != nil {
		return err
	}

	email := &Email{
		To:      []string{data.CustomerEmail},
		Subject: fmt.Sprintf("Order Confirmation - Order #%s", data.OrderID),
		Body:    html,
		IsHTML:  true,
	}

	return s.Send(email)
}

// SendOrderNotificationToAdmin sends an order notification to the admin/internal email
func (s *Service) SendOrderNotificationToAdmin(data *OrderData) error {
	// Render the full email (content + base template)
	html, err := RenderAdminOrderEmail(data)
	if err != nil {
		return err
	}

	internalEmail := os.Getenv("EMAIL_TO_INTERNAL")
	if internalEmail == "" {
		internalEmail = "prints@logans3dcreations.com"
	}

	email := &Email{
		To:      []string{internalEmail},
		Subject: fmt.Sprintf("New Order Received - Order #%s", data.OrderID),
		Body:    html,
		IsHTML:  true,
	}

	return s.Send(email)
}

// RenderCustomerOrderEmail renders the customer order email template for preview
func RenderCustomerOrderEmail(data *OrderData) (string, error) {
	// Render the content section
	tmpl := template.Must(template.New("customer").Funcs(template.FuncMap{
		"FormatCents": FormatCents,
	}).Parse(customerOrderContentTemplate))

	var content bytes.Buffer
	if err := tmpl.Execute(&content, data); err != nil {
		return "", fmt.Errorf("failed to render customer email content: %w", err)
	}

	// Wrap in base template
	subject := fmt.Sprintf("Order Confirmation - Order #%s", data.OrderID)
	return WrapEmailContent(content.String(), subject)
}

// RenderAdminOrderEmail renders the admin order email template for preview
func RenderAdminOrderEmail(data *OrderData) (string, error) {
	// Render the content section
	tmpl := template.Must(template.New("admin").Funcs(template.FuncMap{
		"FormatCents": FormatCents,
	}).Parse(adminOrderContentTemplate))

	var content bytes.Buffer
	if err := tmpl.Execute(&content, data); err != nil {
		return "", fmt.Errorf("failed to render admin email content: %w", err)
	}

	// Wrap in base template
	subject := fmt.Sprintf("New Order Received - Order #%s", data.OrderID)
	return WrapEmailContent(content.String(), subject)
}

// ContactRequestData contains all data for contact request emails
type ContactRequestData struct {
	ID                  string
	FirstName           string
	LastName            string
	Email               string
	Phone               string
	Subject             string
	Message             string
	NewsletterSubscribe bool
	IPAddress           string
	UserAgent           string
	Referrer            string
	SubmittedAt         string
}

// SendContactRequestNotification sends a contact request notification to admin
func (s *Service) SendContactRequestNotification(data *ContactRequestData) error {
	html, err := RenderContactRequestEmail(data)
	if err != nil {
		return err
	}

	internalEmail := os.Getenv("EMAIL_TO_INTERNAL")
	if internalEmail == "" {
		internalEmail = "prints@logans3dcreations.com"
	}

	email := &Email{
		To:      []string{internalEmail},
		Subject: fmt.Sprintf("New Contact Request - %s", data.Subject),
		Body:    html,
		IsHTML:  true,
	}

	if data.Email != "" {
		email.ReplyTo = data.Email
	}

	return s.Send(email)
}

// RenderContactRequestEmail renders the contact request email template
func RenderContactRequestEmail(data *ContactRequestData) (string, error) {
	tmpl := template.Must(template.New("contact").Parse(contactRequestContentTemplate))

	var content bytes.Buffer
	if err := tmpl.Execute(&content, data); err != nil {
		return "", fmt.Errorf("failed to render contact request email content: %w", err)
	}

	subject := fmt.Sprintf("New Contact Request - %s", data.Subject)
	return WrapEmailContent(content.String(), subject)
}

// AbandonedCartItem represents an item in an abandoned cart
type AbandonedCartItem struct {
	ProductName  string
	ProductImage string
	Quantity     int64
	UnitPrice    int64
}

// AbandonedCartData contains all data for abandoned cart recovery emails
type AbandonedCartData struct {
	CustomerName  string
	CustomerEmail string
	CartValue     int64
	ItemCount     int64
	Items         []AbandonedCartItem
	TrackingToken string
	AbandonedAt   string
}

// SendAbandonedCartRecoveryEmail sends a recovery email to a customer
func (s *Service) SendAbandonedCartRecoveryEmail(data *AbandonedCartData, attemptType string) error {
	var html string
	var err error
	var subject string

	switch attemptType {
	case "email_1hr":
		html, err = RenderAbandonedCartRecovery1Hr(data)
		subject = "You left something in your cart!"
	case "email_24hr":
		html, err = RenderAbandonedCartRecovery24Hr(data)
		subject = "Still interested in your cart?"
	case "email_72hr":
		html, err = RenderAbandonedCartRecovery72Hr(data)
		subject = "Last chance to complete your order!"
	default:
		return fmt.Errorf("unknown attempt type: %s", attemptType)
	}

	if err != nil {
		return err
	}

	email := &Email{
		To:      []string{data.CustomerEmail},
		Subject: subject,
		Body:    html,
		IsHTML:  true,
	}

	return s.Send(email)
}

// RenderAbandonedCartRecovery1Hr renders the 1-hour recovery email
func RenderAbandonedCartRecovery1Hr(data *AbandonedCartData) (string, error) {
	tmpl := template.Must(template.New("abandoned_1hr").Funcs(template.FuncMap{
		"FormatCents": FormatCents,
		"ne":          func(a, b int64) bool { return a != b },
	}).Parse(abandonedCartRecovery1HrTemplate))

	var content bytes.Buffer
	if err := tmpl.Execute(&content, data); err != nil {
		return "", fmt.Errorf("failed to render 1hr recovery email: %w", err)
	}

	return WrapEmailContent(content.String(), "You left something in your cart!")
}

// RenderAbandonedCartRecovery24Hr renders the 24-hour recovery email
func RenderAbandonedCartRecovery24Hr(data *AbandonedCartData) (string, error) {
	tmpl := template.Must(template.New("abandoned_24hr").Funcs(template.FuncMap{
		"FormatCents": FormatCents,
		"ne":          func(a, b int64) bool { return a != b },
	}).Parse(abandonedCartRecovery24HrTemplate))

	var content bytes.Buffer
	if err := tmpl.Execute(&content, data); err != nil {
		return "", fmt.Errorf("failed to render 24hr recovery email: %w", err)
	}

	return WrapEmailContent(content.String(), "Still interested in your cart?")
}

// RenderAbandonedCartRecovery72Hr renders the 72-hour recovery email
func RenderAbandonedCartRecovery72Hr(data *AbandonedCartData) (string, error) {
	tmpl := template.Must(template.New("abandoned_72hr").Funcs(template.FuncMap{
		"FormatCents": FormatCents,
		"ne":          func(a, b int64) bool { return a != b },
	}).Parse(abandonedCartRecovery72HrTemplate))

	var content bytes.Buffer
	if err := tmpl.Execute(&content, data); err != nil {
		return "", fmt.Errorf("failed to render 72hr recovery email: %w", err)
	}

	return WrapEmailContent(content.String(), "Last chance to complete your order!")
}
