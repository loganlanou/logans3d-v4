package email

import (
	"bytes"
	"fmt"
	"html/template"
	"log/slog"
	"net/smtp"
	"os"
	"strconv"
)

// Service handles email sending via Brevo SMTP
type Service struct {
	host     string
	port     int
	username string
	password string
	from     string
}

// NewService creates a new email service configured with Brevo SMTP
func NewService() *Service {
	port, err := strconv.Atoi(os.Getenv("BREVO_SMTP_PORT"))
	if err != nil {
		port = 587 // default
	}

	return &Service{
		host:     os.Getenv("BREVO_SMTP_HOST"),
		port:     port,
		username: os.Getenv("BREVO_SMTP_HOST"), // Brevo uses SMTP host as username
		password: os.Getenv("BREVO_SMTP_KEY"),
		from:     os.Getenv("EMAIL_FROM"),
	}
}

// Email represents an email message
type Email struct {
	To      []string
	Subject string
	Body    string
	IsHTML  bool
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
