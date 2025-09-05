package service

import (
	"os"
	"strconv"
)

type Config struct {
	Environment string
	Port        string
	BaseURL     string
	DBPath      string
	
	// JWT configuration
	JWTSecret string
	
	// Google OAuth configuration
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	
	JWT struct {
		Secret string
	}
	
	OAuth struct {
		ClientID     string
		ClientSecret string
		RedirectURL  string
	}
	
	Stripe struct {
		PublishableKey string
		SecretKey      string
		WebhookSecret  string
	}
	
	Email struct {
		From     string
		Provider string
		APIKey   string
	}
	
	Upload struct {
		MaxSize int64
		Dir     string
	}
	
	Admin struct {
		Username string
		Password string
	}
}

func LoadConfig() (*Config, error) {
	config := &Config{
		Environment: getEnv("ENVIRONMENT", "development"),
		Port:        getEnv("PORT", "8000"),
		BaseURL:     getEnv("BASE_URL", "http://localhost:8000"),
		DBPath:      getEnv("DB_PATH", "./db/logans3d.db"),
	}
	
	// JWT
	config.JWTSecret = getEnv("JWT_SECRET", "development-secret")
	config.JWT.Secret = getEnv("JWT_SECRET", "development-secret")
	
	// OAuth
	config.GoogleClientID = getEnv("GOOGLE_CLIENT_ID", "")
	config.GoogleClientSecret = getEnv("GOOGLE_CLIENT_SECRET", "")
	config.GoogleRedirectURL = getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8000/auth/google/callback")
	config.OAuth.ClientID = getEnv("GOOGLE_CLIENT_ID", "")
	config.OAuth.ClientSecret = getEnv("GOOGLE_CLIENT_SECRET", "")
	config.OAuth.RedirectURL = getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8000/auth/google/callback")
	
	// Stripe
	config.Stripe.PublishableKey = getEnv("STRIPE_PUBLISHABLE_KEY", "")
	config.Stripe.SecretKey = getEnv("STRIPE_SECRET_KEY", "")
	config.Stripe.WebhookSecret = getEnv("STRIPE_WEBHOOK_SECRET", "")
	
	// Email
	config.Email.From = getEnv("EMAIL_FROM", "noreply@logans3dcreations.com")
	config.Email.Provider = getEnv("EMAIL_PROVIDER", "sendgrid")
	config.Email.APIKey = getEnv("EMAIL_API_KEY", "")
	
	// Upload
	maxSize := getEnv("UPLOAD_MAX_SIZE", "104857600") // 100MB default
	if size, err := strconv.ParseInt(maxSize, 10, 64); err == nil {
		config.Upload.MaxSize = size
	} else {
		config.Upload.MaxSize = 104857600
	}
	config.Upload.Dir = getEnv("UPLOAD_DIR", "./public/uploads")
	
	// Admin
	config.Admin.Username = getEnv("ADMIN_USERNAME", "admin")
	config.Admin.Password = getEnv("ADMIN_PASSWORD", "password")
	
	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}