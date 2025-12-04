package service

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/internal/auth"
	"github.com/loganlanou/logans3d-v4/internal/email"
	"github.com/loganlanou/logans3d-v4/internal/handlers"
	"github.com/loganlanou/logans3d-v4/internal/jobs"
	"github.com/loganlanou/logans3d-v4/internal/recaptcha"
	"github.com/loganlanou/logans3d-v4/internal/shipping"
	"github.com/loganlanou/logans3d-v4/internal/utils"
	"github.com/loganlanou/logans3d-v4/storage"
	"github.com/loganlanou/logans3d-v4/storage/db"
	"github.com/loganlanou/logans3d-v4/views/about"
	"github.com/loganlanou/logans3d-v4/views/account"
	"github.com/loganlanou/logans3d-v4/views/contact"
	"github.com/loganlanou/logans3d-v4/views/custom"
	"github.com/loganlanou/logans3d-v4/views/events"
	"github.com/loganlanou/logans3d-v4/views/home"
	"github.com/loganlanou/logans3d-v4/views/innovation"
	"github.com/loganlanou/logans3d-v4/views/layout"
	"github.com/loganlanou/logans3d-v4/views/legal"
	"github.com/loganlanou/logans3d-v4/views/portfolio"
	"github.com/loganlanou/logans3d-v4/views/shop"
	"github.com/oklog/ulid/v2"
	"github.com/stripe/stripe-go/v80"
	checkoutsession "github.com/stripe/stripe-go/v80/checkout/session"
)

type Service struct {
	storage                  *storage.Storage
	config                   *Config
	paymentHandler           *handlers.PaymentHandler
	shippingHandler          *handlers.ShippingHandler
	shippingService          *shipping.ShippingService
	emailService             *email.Service
	authHandler              *handlers.AuthHandler
	abandonedCartDetector    *jobs.AbandonedCartDetector
	abandonedCartEmailSender *jobs.AbandonedCartEmailSender
	ogImageRefresher         *jobs.OGImageRefresher
}

func New(storage *storage.Storage, config *Config) *Service {
	// Initialize shipping service - load from database instead of file
	ctx := context.Background()
	shippingConfig, err := shipping.LoadShippingConfigFromDB(ctx, storage.Queries)
	if err != nil {
		slog.Warn("failed to load shipping config from database, using defaults", "error", err)
		shippingConfig = shipping.CreateDefaultConfig()
	} else {
		slog.Info("loaded shipping configuration from database", "num_boxes", len(shippingConfig.Boxes))
	}

	shippingService, err := shipping.NewShippingService(shippingConfig, storage.Queries)
	if err != nil {
		slog.Error("failed to initialize shipping service", "error", err)
		// Continue without shipping service for now
		shippingService = nil
	}

	var shippingHandler *handlers.ShippingHandler
	if shippingService != nil {
		shippingHandler = handlers.NewShippingHandler(storage.Queries, shippingService)
	}

	// Initialize email service with database queries
	emailService := email.NewService(storage.Queries)

	// Initialize abandoned cart detector
	abandonedCartDetector := jobs.NewAbandonedCartDetector(storage)
	// Start the detector
	abandonedCartDetector.Start(ctx)

	// Initialize abandoned cart email sender
	abandonedCartEmailSender := jobs.NewAbandonedCartEmailSender(storage, emailService)
	// Start the email sender
	abandonedCartEmailSender.Start(ctx)

	// Initialize OG image refresher (runs once at startup in background)
	ogImageRefresher := jobs.NewOGImageRefresherWithAI(storage, os.Getenv("GEMINI_API_KEY"))
	ogImageRefresher.Start(ctx)

	return &Service{
		storage:                  storage,
		config:                   config,
		paymentHandler:           handlers.NewPaymentHandler(storage.Queries, emailService),
		shippingHandler:          shippingHandler,
		shippingService:          shippingService,
		emailService:             emailService,
		authHandler:              handlers.NewAuthHandler(),
		abandonedCartDetector:    abandonedCartDetector,
		abandonedCartEmailSender: abandonedCartEmailSender,
		ogImageRefresher:         ogImageRefresher,
	}
}

func (s *Service) RegisterRoutes(e *echo.Echo) {
	// Initialize Clerk SDK with secret key - this configures the default backend
	// Note: CLERK_SECRET_KEY is validated in main.go before app starts
	// Tests don't need this key since they don't call Clerk APIs
	clerkSecretKey := os.Getenv("CLERK_SECRET_KEY")
	clerk.SetKey(clerkSecretKey)

	// Static files - no auth middleware
	e.Static("/public", "public")

	// Logout - no auth middleware (must clear cookies without re-authentication)
	e.GET("/logout", s.authHandler.HandleLogout)

	// All other routes get auth middleware
	withAuth := e.Group("")
	withAuth.Use(auth.ClerkHandshakeMiddleware())
	withAuth.Use(auth.ClerkAuthMiddleware(s.storage))

	// Auth routes (public) - Clerk JavaScript SDK components
	withAuth.GET("/login", s.authHandler.HandleLogin)
	withAuth.GET("/signup", s.authHandler.HandleSignUp)
	withAuth.GET("/sign-up", s.authHandler.HandleSignUp) // Support both /signup and /sign-up

	// Home page
	withAuth.GET("/", s.handleHome)

	// Static pages
	withAuth.GET("/about", s.handleAbout)
	withAuth.GET("/events", s.handleEvents)
	withAuth.GET("/contact", s.handleContact)
	withAuth.POST("/contact/submit", s.handleContactSubmit)
	withAuth.GET("/portfolio", s.handlePortfolio)
	withAuth.GET("/innovation", s.handleInnovation)
	withAuth.GET("/innovation/manufacturing", s.handleManufacturing)

	// Legal pages
	withAuth.GET("/privacy", s.handlePrivacy)
	withAuth.GET("/terms", s.handleTerms)
	withAuth.GET("/shipping", s.handleShipping)
	withAuth.GET("/custom-policy", s.handleCustomPolicy)

	// Shop routes
	shop := withAuth.Group("/shop")
	shop.GET("", s.handleShop)
	shop.GET("/premium", s.handlePremium)
	shop.GET("/product/:slug", s.handleProduct)
	shop.GET("/category/:slug", s.handleCategory)

	// Cart routes
	withAuth.GET("/cart", s.handleCart)

	// Email preferences handler (needed for account routes)
	emailPrefsHandler := handlers.NewEmailPreferencesHandler(s.storage.Queries)

	// Account routes
	withAuth.GET("/account", s.handleAccount)
	withAuth.GET("/account/orders/:id", s.handleAccountOrderDetail)
	withAuth.GET("/account/email-preferences", emailPrefsHandler.HandleEmailPreferencesPage)

	// Redirect for backward compatibility
	withAuth.GET("/email-preferences", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/account/email-preferences")
	})

	// Cart API - all routes public for now
	withAuth.GET("/api/cart", s.handleGetCart)
	withAuth.POST("/api/cart/add", s.handleAddToCart)
	withAuth.DELETE("/api/cart/item/:id", s.handleRemoveFromCart)
	withAuth.PUT("/api/cart/item/:id", s.handleUpdateCartItem)
	withAuth.POST("/api/cart/validate", s.handleValidateCartSession)

	// Custom quote routes
	withAuth.GET("/custom", s.handleCustom)
	withAuth.POST("/custom/quote", s.handleCustomQuote)

	// Stripe Checkout routes
	withAuth.POST("/checkout/create-session-cart", s.handleCreateStripeCheckoutSessionCart)
	withAuth.GET("/checkout/success", s.handleCheckoutSuccess)
	withAuth.GET("/checkout/cancel", s.handleCheckoutCancel)

	// Payment API routes
	api := withAuth.Group("/api")
	api.POST("/payment/create-intent", s.paymentHandler.CreatePaymentIntent)
	api.POST("/payment/create-customer", s.paymentHandler.CreateCustomer)
	api.POST("/stripe/webhook", s.paymentHandler.HandleWebhook)

	// Email preferences routes (public - accessible via token)
	e.GET("/unsubscribe/:token", emailPrefsHandler.HandleUnsubscribe)
	api.GET("/email-preferences", emailPrefsHandler.HandleGetEmailPreferences)
	api.PUT("/email-preferences", emailPrefsHandler.HandleUpdateEmailPreferences)

	// Open Graph image generation
	geminiAPIKey := os.Getenv("GEMINI_API_KEY")
	ogImageHandler := handlers.NewOGImageHandlerWithAI(s.storage, geminiAPIKey)
	api.GET("/og-image/multi/:product_id", ogImageHandler.HandleGenerateMultiVariantOGImage) // Must be before :product_id route
	api.GET("/og-image/:product_id", ogImageHandler.HandleGenerateOGImage)
	api.GET("/carousel/:product_id", ogImageHandler.HandleDownloadCarouselImages) // Instagram carousel ZIP download

	// Promotion routes (public)
	promotionsHandler := handlers.NewPromotionsHandler(s.storage.Queries, s.emailService)
	adminPromotionsHandler := handlers.NewAdminPromotionsHandler(s.storage.Queries)
	api.POST("/promotions/capture-email", promotionsHandler.HandleCaptureEmail)
	api.GET("/promotions/validate/:code", promotionsHandler.HandleValidateCode)
	api.GET("/promotions/popup-status", adminPromotionsHandler.HandlePopupStatus)

	// Shipping API routes
	if s.shippingHandler != nil {
		api.POST("/shipping/rates", s.shippingHandler.GetShippingRates)
		api.POST("/shipping/selection", s.shippingHandler.SaveShippingSelection)
		api.GET("/shipping/selection", s.shippingHandler.GetShippingSelection)
		api.POST("/shipping/labels", s.shippingHandler.CreateLabel)
		api.PUT("/shipping/labels/:labelId/void", s.shippingHandler.VoidLabel)
		api.GET("/shipping/labels/:labelId/download", s.shippingHandler.DownloadLabel)
		api.POST("/shipping/validate-address", s.shippingHandler.ValidateAddress)
	}

	// Admin routes - protected with RequireAdmin middleware
	// Initialize admin handler with all required services
	adminHandler := handlers.NewAdminHandler(s.storage, s.shippingService, s.emailService)

	// Cart recovery email tracking - uses adminHandler but no auth required (customers click from email)
	withAuth.GET("/cart/recover", adminHandler.HandleRecoveryEmailTracking)

	admin := withAuth.Group("/admin", auth.RequireAdmin())
	admin.GET("", adminHandler.HandleAdminDashboard)
	admin.GET("/products", adminHandler.HandleProductsList)
	admin.GET("/categories", adminHandler.HandleCategoriesTab)
	admin.GET("/product/new", adminHandler.HandleProductForm)
	admin.POST("/product", adminHandler.HandleCreateProduct)
	admin.GET("/product/edit", adminHandler.HandleProductForm)
	admin.POST("/product/:id", adminHandler.HandleUpdateProduct)
	admin.POST("/product/:id/delete", adminHandler.HandleDeleteProduct)
	admin.POST("/product/:id/toggle-featured", adminHandler.HandleToggleProductFeatured)
	admin.POST("/product/:id/toggle-premium", adminHandler.HandleToggleProductPremium)
	admin.POST("/product/:id/toggle-active", adminHandler.HandleToggleProductActive)
	admin.POST("/product/:id/toggle-new", adminHandler.HandleToggleProductNew)
	admin.DELETE("/product/image/:imageId/delete", adminHandler.HandleDeleteProductImage)
	admin.PUT("/product/image/:imageId/set-primary", adminHandler.HandleSetPrimaryProductImage)
	admin.POST("/style-image/:imageId/primary", adminHandler.HandleSetPrimaryStyleImage)
	admin.POST("/product/:id/styles", adminHandler.HandleCreateProductStyle)
	admin.POST("/product/:id/sizes", adminHandler.HandleSaveProductSizes)
	admin.POST("/product/:id/skus", adminHandler.HandleCreateProductSKU)

	// Style panel routes (for admin SKU management UI)
	admin.GET("/style/:styleId/panel", adminHandler.HandleGetStylePanel)
	admin.PUT("/sku/:skuId/price", adminHandler.HandleUpdateSkuPrice)
	admin.PUT("/sku/:skuId/stock", adminHandler.HandleUpdateSkuStock)
	admin.PUT("/sku/:skuId/active", adminHandler.HandleToggleSkuActive)
	admin.POST("/style/:styleId/sku", adminHandler.HandleAddStyleSku)
	admin.DELETE("/sku/:skuId", adminHandler.HandleDeleteSkuFromPanel)
	admin.POST("/style/:styleId/set-primary", adminHandler.HandleSetPrimaryStyleFromPanel)
	admin.DELETE("/style/:styleId", adminHandler.HandleDeleteStyleFromPanel)
	admin.POST("/style-image/:imageId/set-primary", adminHandler.HandleSetPrimaryStyleImageFromPanel)
	admin.DELETE("/style-image/:imageId", adminHandler.HandleDeleteStyleImageFromPanel)
	admin.POST("/style/:styleId/images", adminHandler.HandleAddStyleImages)

	admin.GET("/product/search", adminHandler.HandleProductSearch)
	admin.GET("/product/:id/row", adminHandler.HandleGetProductRow)

	// AI Background generation routes
	aiBackgroundHandler := handlers.NewAIBackgroundHandler(s.storage, geminiAPIKey)
	admin.POST("/product/:id/generate-background", aiBackgroundHandler.HandleGenerateAIBackground)
	admin.GET("/product/:id/pending-backgrounds", aiBackgroundHandler.HandleGetPendingBackgrounds)
	admin.POST("/pending-background/:id/approve", aiBackgroundHandler.HandleApproveBackground)
	admin.POST("/pending-background/:id/reject", aiBackgroundHandler.HandleRejectBackground)
	admin.GET("/product/:id/edit-row", adminHandler.HandleGetProductEditRow)
	admin.PUT("/product/:id/inline", adminHandler.HandleUpdateProductInline)
	admin.GET("/product/:id/row-mobile", adminHandler.HandleGetProductRowMobile)
	admin.GET("/product/:id/edit-row-mobile", adminHandler.HandleGetProductEditRowMobile)
	admin.PUT("/product/:id/inline-mobile", adminHandler.HandleUpdateProductInlineMobile)

	// Category management routes
	admin.GET("/category/new", adminHandler.HandleCategoryForm)
	admin.POST("/category", adminHandler.HandleCreateCategory)
	admin.GET("/category/edit", adminHandler.HandleCategoryForm)
	admin.POST("/category/:id", adminHandler.HandleUpdateCategory)
	admin.POST("/category/:id/delete", adminHandler.HandleDeleteCategory)

	// Orders management routes
	admin.GET("/orders", adminHandler.HandleOrdersList)
	admin.GET("/orders/search", adminHandler.HandleOrderSearch)
	admin.GET("/orders/:id", adminHandler.HandleOrderDetail)
	admin.POST("/orders/:id/status", adminHandler.HandleUpdateOrderStatus)
	admin.GET("/orders/:id/tracking/lookup", adminHandler.HandleGetOrderTrackingLookup)
	admin.GET("/orders/:id/shipping/rates", adminHandler.HandleGetOrderShippingRates)
	admin.POST("/orders/:id/shipping/buy-label", adminHandler.HandleBuyShippingLabel)

	// Quotes management routes
	admin.GET("/quotes", adminHandler.HandleQuotesList)
	admin.GET("/quotes/:id", adminHandler.HandleQuoteDetail)
	admin.POST("/quotes/:id", adminHandler.HandleUpdateQuote)

	// Events management routes
	admin.GET("/events", adminHandler.HandleEventsList)
	admin.GET("/events/new", adminHandler.HandleEventForm)
	admin.POST("/events", adminHandler.HandleCreateEvent)
	admin.GET("/events/edit", adminHandler.HandleEventForm)
	admin.POST("/events/:id", adminHandler.HandleUpdateEvent)
	admin.POST("/events/:id/delete", adminHandler.HandleDeleteEvent)

	// Contact requests management routes
	admin.GET("/contacts", adminHandler.HandleContactsList)
	admin.GET("/contacts/table", adminHandler.HandleContactsTable)
	admin.GET("/contacts/:id", adminHandler.HandleContactDetail)
	admin.POST("/contacts/:id/status", adminHandler.HandleUpdateContactStatus)
	admin.POST("/contacts/bulk/status", adminHandler.HandleBulkUpdateContactStatus)
	admin.POST("/contacts/:id/priority", adminHandler.HandleUpdateContactPriority)
	admin.POST("/contacts/:id/notes", adminHandler.HandleAddContactNotes)
	admin.POST("/contacts/:id/notes/delete", adminHandler.HandleDeleteContactNotes)

	// Abandoned Carts management routes
	admin.GET("/abandoned-carts", adminHandler.HandleAbandonedCartsDashboard)
	admin.GET("/abandoned-carts/export", adminHandler.HandleExportAbandonedCarts)
	admin.GET("/abandoned-carts/:id", adminHandler.HandleAbandonedCartDetail)
	admin.POST("/abandoned-carts/:id/send-email", adminHandler.HandleSendRecoveryEmail)
	admin.POST("/abandoned-carts/:id/notes", adminHandler.HandleUpdateCartNotes)
	admin.POST("/abandoned-carts/:id/recover", adminHandler.HandleMarkCartRecovered)

	// Email management routes
	emailHandler := handlers.NewAdminEmailsHandler(s.storage.Queries)
	admin.GET("/emails", emailHandler.HandleEmailHistory)

	// Promotion management routes
	promotionsAdminHandler := handlers.NewAdminPromotionsHandler(s.storage.Queries)
	admin.GET("/promotions", promotionsAdminHandler.HandlePromotionsList)
	admin.GET("/promotions/:id", promotionsAdminHandler.HandlePromotionDetail)

	// Social Media management routes
	admin.GET("/social-media", adminHandler.HandleAdminSocialMedia)
	admin.POST("/social-media/generate/:product_id", adminHandler.HandleGeneratePostsForProduct)
	admin.GET("/social-media/product/:product_id", adminHandler.HandleSocialMediaProductView)
	admin.POST("/social-media/update-status", adminHandler.HandleUpdatePostStatus)
	admin.POST("/social-media/bulk-generate", adminHandler.HandleBulkGeneratePosts)
	admin.POST("/social-media/delete-pending", adminHandler.HandleDeleteAllPendingPosts)

	// Cart management routes
	cartHandler := handlers.NewCartHandler(s.storage)
	admin.GET("/carts", cartHandler.HandleCartsList)
	admin.GET("/carts/:id", cartHandler.HandleCartDetail)

	// User management routes
	userHandler := handlers.NewUserHandler(s.storage)
	admin.GET("/users", userHandler.HandleUsersList)
	admin.GET("/users/:id", userHandler.HandleUserDetail)

	// Shipping management routes
	admin.GET("/shipping/boxes", adminHandler.HandleShippingTab)
	admin.GET("/shipping/boxes/new", adminHandler.HandleBoxForm)
	admin.POST("/shipping/boxes", adminHandler.HandleCreateBox)
	admin.GET("/shipping/boxes/edit/:sku", adminHandler.HandleBoxForm)
	admin.POST("/shipping/boxes/:sku", adminHandler.HandleUpdateBox)
	admin.POST("/shipping/boxes/delete/:sku", adminHandler.HandleDeleteBox)
	admin.GET("/shipping/config", adminHandler.HandleShippingConfig)
	admin.POST("/shipping/config", adminHandler.HandleSaveShippingConfig)
	admin.GET("/shipping/settings", adminHandler.HandleShippingSettings)
	admin.POST("/shipping/settings", adminHandler.HandleSaveShippingSettings)

	// Email preview routes
	admin.GET("/email-preview", adminHandler.HandleEmailPreview)
	admin.GET("/email-preview/customer", adminHandler.HandleEmailPreviewCustomer)
	admin.GET("/email-preview/admin", adminHandler.HandleEmailPreviewAdmin)
	admin.POST("/email-preview/send-test", adminHandler.HandleSendTestEmail)

	// Developer routes - protected with RequireAdmin middleware
	dev := withAuth.Group("/dev", auth.RequireAdmin())
	// Page routes
	dev.GET("", adminHandler.HandleDeveloperDashboard)
	dev.GET("/system", adminHandler.HandleDevSystem)
	dev.GET("/database", adminHandler.HandleDevDatabase)
	dev.GET("/memory", adminHandler.HandleDevMemory)
	dev.GET("/logs", adminHandler.HandleDevLogs)
	dev.GET("/config", adminHandler.HandleDevConfig)
	// API routes
	dev.POST("/gc", adminHandler.HandleGarbageCollect)
	dev.GET("/logs/stream", adminHandler.HandleLogStream)
	dev.GET("/logs/tail", adminHandler.HandleLogTail)
	dev.POST("/logs/clear", adminHandler.HandleLogClear)

	// Health check - no auth
	e.GET("/health", s.handleHealth)

	// Facebook domain verification
	e.GET("/ov2w2j24qs2aozezx1wy0xyv0cf963.html", func(c echo.Context) error {
		return c.String(http.StatusOK, "ov2w2j24qs2aozezx1wy0xyv0cf963")
	})
}

// getProductImageURL returns the primary image URL for a product, correctly handling variants
func (s *Service) getProductImageURL(ctx context.Context, product db.Product) string {
	// For products with variants, prefer the AI-generated multi-variant OG image
	if product.HasVariants.Valid && product.HasVariants.Bool {
		// Check if multi-variant OG image exists
		multiOGPath := fmt.Sprintf("public/og-images/product-%s-multi.png", product.ID)
		if _, err := os.Stat(multiOGPath); err == nil {
			return "/" + multiOGPath
		}

		// Fall back to primary style's primary image
		styles, err := s.storage.Queries.GetProductStyles(ctx, product.ID)
		if err == nil && len(styles) > 0 {
			// First style is primary (ordered by is_primary DESC)
			primaryStyle := styles[0]
			styleImage, err := s.storage.Queries.GetPrimaryStyleImage(ctx, primaryStyle.ID)
			if err == nil && styleImage.ImageUrl != "" {
				return "/public/images/products/styles/" + styleImage.ImageUrl
			}
		}
	}

	// Fall back to regular product images
	images, err := s.storage.Queries.GetProductImages(ctx, product.ID)
	if err != nil || len(images) == 0 {
		return ""
	}

	// Get the primary image or the first one
	rawImageURL := ""
	for _, img := range images {
		if img.IsPrimary.Valid && img.IsPrimary.Bool {
			rawImageURL = img.ImageUrl
			break
		}
	}
	if rawImageURL == "" {
		rawImageURL = images[0].ImageUrl
	}

	if rawImageURL != "" {
		return "/public/images/products/" + rawImageURL
	}
	return ""
}

// Basic handler implementations
func (s *Service) handleHome(c echo.Context) error {
	ctx := c.Request().Context()
	slog.Info("Home page requested", "ip", c.RealIP())

	// Get featured products
	featuredProducts, err := s.storage.Queries.ListFeaturedProducts(ctx)
	if err != nil {
		slog.Error("failed to fetch featured products", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load featured products")
	}
	slog.Debug("fetched featured products", "count", len(featuredProducts))

	// Combine with images (handles variants correctly)
	productsWithImages := make([]home.ProductWithImage, 0, len(featuredProducts))
	for _, product := range featuredProducts {
		productsWithImages = append(productsWithImages, home.ProductWithImage{
			Product:  product,
			ImageURL: s.getProductImageURL(ctx, product),
		})
	}

	// Build PageMeta for home page
	meta := layout.NewPageMeta(c, s.storage.Queries)
	meta.Title = "Logan's 3D Creations - Custom 3D Printed Collectibles & Dinosaurs"
	meta.Description = "Discover unique 3D printed collectibles, dinosaurs, and custom creations. High-quality prints with expert craftsmanship and attention to detail."
	meta.Keywords = []string{"3D printing", "custom collectibles", "dinosaur models", "3D printed art", "collectible figurines"}
	meta.OGType = "website"

	return Render(c, home.Index(c, meta, productsWithImages))
}

func (s *Service) handleAbout(c echo.Context) error {
	meta := layout.NewPageMeta(c, s.storage.Queries)
	meta.Title = "About - Logan's 3D Creations"
	meta.Description = "Learn about Logan's 3D Creations, our mission to bring creativity to life through 3D printing, and our commitment to quality craftsmanship."
	meta.Keywords = []string{"about us", "3D printing company", "Logan's 3D Creations story", "mission"}
	meta.OGType = "website"
	return Render(c, about.Index(c, meta))
}

func (s *Service) handleShop(c echo.Context) error {
	ctx := c.Request().Context()

	// Get all categories for filter
	categories, err := s.storage.Queries.ListCategories(ctx)
	if err != nil {
		slog.Error("failed to fetch categories", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load categories")
	}
	slog.Debug("fetched categories", "count", len(categories))

	// Get all products
	products, err := s.storage.Queries.ListProducts(ctx)
	if err != nil {
		slog.Error("failed to fetch products", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load products")
	}
	slog.Debug("fetched products", "count", len(products))

	// Combine with images (handles variants correctly)
	productsWithImages := make([]shop.ProductWithImage, 0, len(products))
	for _, product := range products {
		productsWithImages = append(productsWithImages, shop.ProductWithImage{
			Product:  product,
			ImageURL: s.getProductImageURL(ctx, product),
		})
	}

	// Build PageMeta for shop page
	meta := layout.NewPageMeta(c, s.storage.Queries)
	meta.Title = "Shop - 3D Printed Collectibles & Dinosaurs | Logan's 3D Creations"
	meta.Description = "Browse our collection of unique 3D printed collectibles, dinosaurs, and custom creations. High-quality prints available for purchase."
	meta.Keywords = []string{"buy 3D prints", "3D printed collectibles", "dinosaur models for sale", "custom 3D printing", "collectible figurines shop"}
	meta.OGType = "website"

	return Render(c, shop.Index(c, meta, productsWithImages, categories, nil))
}

func (s *Service) handlePremium(c echo.Context) error {
	ctx := c.Request().Context()

	// Create sample premium collection tiers
	collections := []shop.CollectionTier{
		{
			Name:          "Bronze",
			Slug:          "bronze",
			Description:   "Essential premium pieces to start your collection with high-quality detail and materials",
			Price:         4999, // $49.99
			OriginalPrice: 5999, // $59.99
			Discount:      17,
			Items:         3,
			Color:         "amber",
			GradientFrom:  "from-amber-600",
			GradientTo:    "to-yellow-600",
			IconEmoji:     "🥉",
			Features: []string{
				"3 carefully selected premium items",
				"High-detail 0.2mm layer resolution",
				"Premium PLA+ materials",
				"Basic post-processing included",
				"Standard shipping",
			},
		},
		{
			Name:          "Silver",
			Slug:          "silver",
			Description:   "Enhanced collection featuring superior detail and exclusive variations for dedicated collectors",
			Price:         9999,  // $99.99
			OriginalPrice: 12999, // $129.99
			Discount:      23,
			Items:         6,
			Color:         "gray",
			GradientFrom:  "from-gray-500",
			GradientTo:    "to-slate-500",
			IconEmoji:     "🥈",
			Features: []string{
				"6 premium items with exclusive variants",
				"Ultra-high 0.15mm layer resolution",
				"Premium PETG and ABS materials",
				"Professional finishing included",
				"Priority shipping",
				"Collectible packaging",
			},
		},
		{
			Name:          "Gold",
			Slug:          "gold",
			Description:   "Elite tier with the most detailed models, rare materials, and collector-exclusive items",
			Price:         19999, // $199.99
			OriginalPrice: 27999, // $279.99
			Discount:      29,
			Items:         10,
			Color:         "amber",
			GradientFrom:  "from-amber-500",
			GradientTo:    "to-yellow-500",
			IconEmoji:     "🥇",
			Features: []string{
				"10 premium items including limited editions",
				"Microscopic 0.1mm layer resolution",
				"Specialty resins and metal-filled filaments",
				"Expert hand-finishing and detailing",
				"Express shipping with insurance",
				"Luxury presentation boxes",
				"Certificate of authenticity",
			},
		},
		{
			Name:          "Platinum",
			Slug:          "platinum",
			Description:   "Ultra-exclusive collection with master-crafted pieces and personalized touches",
			Price:         39999, // $399.99
			OriginalPrice: 54999, // $549.99
			Discount:      27,
			Items:         15,
			Color:         "slate",
			GradientFrom:  "from-slate-400",
			GradientTo:    "to-gray-400",
			IconEmoji:     "💎",
			Features: []string{
				"15 premium items with custom options",
				"Museum-quality 0.05mm precision",
				"Exotic materials including carbon fiber",
				"Master artisan finishing and painting",
				"White-glove delivery service",
				"Heirloom-quality presentation",
				"Personalized engraving available",
				"Exclusive access to pre-releases",
			},
		},
		{
			Name:          "Titanium",
			Slug:          "titanium",
			Description:   "Industrial-grade collection featuring aerospace materials and cutting-edge techniques",
			Price:         79999,  // $799.99
			OriginalPrice: 109999, // $1099.99
			Discount:      27,
			Items:         20,
			Color:         "slate",
			GradientFrom:  "from-slate-600",
			GradientTo:    "to-gray-600",
			IconEmoji:     "🛡️",
			Features: []string{
				"20 premium items with industrial materials",
				"Aerospace-grade titanium components",
				"Multi-material hybrid construction",
				"Professional-grade surface treatments",
				"Insured courier delivery worldwide",
				"Collector's vault storage box",
				"Numbered limited edition pieces",
				"Direct access to master craftsman",
				"Annual exclusive release preview",
			},
		},
		{
			Name:          "Diamond",
			Slug:          "diamond",
			Description:   "The pinnacle of 3D printing excellence with precious metal inlays and gemstone accents",
			Price:         159999, // $1599.99
			OriginalPrice: 219999, // $2199.99
			Discount:      27,
			Items:         25,
			Color:         "blue",
			GradientFrom:  "from-blue-400",
			GradientTo:    "to-cyan-400",
			IconEmoji:     "💎",
			Features: []string{
				"25 masterpiece items with precious accents",
				"Real gold and silver inlay options",
				"Swarovski crystal detail work",
				"Museum curator-level preservation",
				"Personal delivery by master craftsman",
				"Custom display case included",
				"Investment-grade documentation",
				"Lifetime warranty and restoration",
				"VIP studio tour and consultation",
				"Bespoke commission privileges",
			},
		},
		{
			Name:          "Collectors",
			Slug:          "collectors",
			Description:   "Ultimate prestige collection for serious collectors with one-of-a-kind masterpieces",
			Price:         299999, // $2999.99
			OriginalPrice: 399999, // $3999.99
			Discount:      25,
			Items:         50,
			Color:         "purple",
			GradientFrom:  "from-purple-600",
			GradientTo:    "to-pink-600",
			IconEmoji:     "👑",
			Features: []string{
				"50 unique collector pieces - never reproduced",
				"Collaboration with renowned artists",
				"Mixed media incorporating rare materials",
				"Individual artist signature and provenance",
				"Private viewing and selection process",
				"Museum-quality archival storage",
				"Comprehensive insurance coverage",
				"Collector network membership",
				"Priority access to artist collaborations",
				"Legacy collection management services",
			},
		},
	}

	// Get some featured premium products (top 8 most expensive)
	products, err := s.storage.Queries.ListProducts(ctx)
	if err != nil {
		slog.Error("failed to fetch products", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load products")
	}

	// Sort products by price descending and take top 8 (handles variants correctly)
	featuredProducts := make([]shop.ProductWithImage, 0, 8)
	for i, product := range products {
		if i >= 8 {
			break
		}
		featuredProducts = append(featuredProducts, shop.ProductWithImage{
			Product:  product,
			ImageURL: s.getProductImageURL(ctx, product),
		})
	}

	// Build page metadata
	meta := layout.NewPageMeta(c, s.storage.Queries)
	meta.Title = "Premium Collections - Logan's 3D Creations"
	meta.Description = "Discover our premium collection tiers with the most detailed models, bundled discounts, and exclusive variations."
	meta.Keywords = []string{"premium 3D prints", "collector items", "detailed models", "bundle discounts", "exclusive collections"}

	return Render(c, shop.Premium(c, collections, featuredProducts, meta))
}

func (s *Service) handleProduct(c echo.Context) error {
	slug := c.Param("slug")
	ctx := c.Request().Context()

	// Parse variant query params for sharing/pre-selection
	selectedColorID := c.QueryParam("color")
	selectedSizeID := c.QueryParam("size")

	// Get product by slug (only active products)
	product, err := s.storage.Queries.GetProductBySlug(ctx, slug)
	if err != nil {
		// Product not found or inactive - show shopping-specific 404
		slog.Info("product not found or inactive", "slug", slug)
		return s.handleProductNotFound(c, slug)
	}

	// Get product images
	images, err := s.storage.Queries.GetProductImages(ctx, product.ID)
	if err != nil {
		slog.Error("failed to fetch product images", "product_id", product.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load product images")
	}

	// Get category
	var category db.Category
	if product.CategoryID.Valid {
		category, err = s.storage.Queries.GetCategory(ctx, product.CategoryID.String)
		if err != nil {
			slog.Error("failed to fetch category", "category_id", product.CategoryID.String, "error", err)
			// Continue with empty category rather than failing
			category = db.Category{Name: "Uncategorized", Slug: "uncategorized"}
		}
	} else {
		category = db.Category{Name: "Uncategorized", Slug: "uncategorized"}
	}

	// Get related products from the same category
	var relatedProducts []shop.ProductWithImage
	if product.CategoryID.Valid {
		relatedProductsList, err := s.storage.Queries.ListRelatedProducts(ctx, db.ListRelatedProductsParams{
			CategoryID: product.CategoryID,
			ID:         product.ID,
			Limit:      4,
		})
		if err != nil {
			slog.Warn("failed to fetch related products", "product_id", product.ID, "error", err)
			relatedProductsList = []db.Product{}
		}

		// Build ProductWithImage for each related product (handles variants correctly)
		for _, relatedProduct := range relatedProductsList {
			relatedProducts = append(relatedProducts, shop.ProductWithImage{
				Product:  relatedProduct,
				ImageURL: s.getProductImageURL(ctx, relatedProduct),
			})
		}
	}

	// Load variant data if applicable
	var variantData *shop.ProductVariantData
	if product.HasVariants.Valid && product.HasVariants.Bool {
		data := shop.ProductVariantData{
			BasePriceCents: product.PriceCents,
		}

		styles, err := s.storage.Queries.GetProductVariantStyles(ctx, product.ID)
		if err != nil {
			slog.Error("failed to load product styles", "error", err, "product_id", product.ID)
		}

		for _, style := range styles {
			styleImages, _ := s.storage.Queries.GetProductStyleImages(ctx, style.ID)
			imagePaths := make([]string, 0, len(styleImages))
			for _, img := range styleImages {
				imagePaths = append(imagePaths, fmt.Sprintf("/public/images/products/styles/%s", img.ImageUrl))
			}
			if len(imagePaths) == 0 {
				for _, img := range images {
					imagePaths = append(imagePaths, fmt.Sprintf("/public/images/products/%s", img.ImageUrl))
				}
			}

			sizes, _ := s.storage.Queries.GetProductVariantSizesForStyle(ctx, db.GetProductVariantSizesForStyleParams{
				ProductID:      product.ID,
				ProductStyleID: style.ID,
			})

			sizeOptions := make([]shop.VariantSizeOption, 0, len(sizes))
			for _, size := range sizes {
				sizeOptions = append(sizeOptions, shop.VariantSizeOption{
					ValueID:              size.SizeID,
					Value:                size.SizeName,
					DisplayName:          size.SizeDisplayName,
					SkuID:                size.ProductSkuID,
					SKU:                  size.Sku,
					PriceAdjustmentCents: int64FromNull(size.PriceAdjustmentCents),
					StockQuantity:        int64FromNull(size.StockQuantity),
				})
			}

			primaryImage := ""
			if style.PrimaryImage != "" {
				primaryImage = fmt.Sprintf("/public/images/products/styles/%s", style.PrimaryImage)
			}
			if primaryImage == "" && len(imagePaths) > 0 {
				primaryImage = imagePaths[0]
			}

			data.Colors = append(data.Colors, shop.VariantColorOption{
				ID:           style.ID,
				Value:        style.Name,
				DisplayName:  style.Name,
				PrimaryImage: primaryImage,
				Images:       imagePaths,
				Sizes:        sizeOptions,
			})
		}

		if len(data.Colors) > 0 {
			variantData = &data
		}
	}

	// Build complete PageMeta with SEO data
	meta := layout.NewPageMeta(c, s.storage.Queries).
		FromProduct(product)

	// Add primary product image if available
	if len(images) > 0 {
		// Find primary image or use first one
		primaryImageFilename := ""
		for _, img := range images {
			if img.IsPrimary.Valid && img.IsPrimary.Bool {
				primaryImageFilename = img.ImageUrl
				break
			}
		}
		if primaryImageFilename == "" {
			primaryImageFilename = images[0].ImageUrl
		}
		meta = meta.WithProductImage(primaryImageFilename)
	}

	// Use dynamically generated OG image with text overlay
	// For multi-variant products, use the multi-variant OG image showing all colors
	var ogImageURL string
	if product.HasVariants.Valid && product.HasVariants.Bool && variantData != nil && len(variantData.Colors) > 1 {
		ogImageURL = fmt.Sprintf("/api/og-image/multi/%s", product.ID)
	} else {
		ogImageURL = fmt.Sprintf("/api/og-image/%s", product.ID)
	}
	meta = meta.WithOGImage(ogImageURL)

	// Add category for breadcrumbs and schema
	meta = meta.WithCategories([]db.Category{category})

	// If variant params provided, validate and apply variant-specific meta for sharing
	if selectedColorID != "" && selectedSizeID != "" && variantData != nil {
		// Find the matching color and size in variant data
		for _, color := range variantData.Colors {
			if color.ID == selectedColorID {
				for _, size := range color.Sizes {
					if size.ValueID == selectedSizeID {
						// Valid variant - build VariantInfo and apply to meta
						variantInfo := layout.VariantInfo{
							StyleID:      color.ID,
							StyleName:    color.DisplayName,
							SizeID:       size.ValueID,
							SizeName:     size.DisplayName,
							SkuID:        size.SkuID,
							SKU:          size.SKU,
							PriceCents:   variantData.BasePriceCents + size.PriceAdjustmentCents,
							PrimaryImage: color.PrimaryImage,
						}
						meta = meta.WithVariant(variantInfo)
						slog.Debug("applied variant to page meta",
							"product_id", product.ID,
							"style", color.DisplayName,
							"size", size.DisplayName)
						break
					}
				}
				break
			}
		}
	}

	return Render(c, shop.Product(c, meta, product, category, images, relatedProducts, variantData))
}

func (s *Service) handleProductNotFound(c echo.Context, slug string) error {
	ctx := c.Request().Context()

	// Get all categories for browsing
	categories, err := s.storage.Queries.ListCategories(ctx)
	if err != nil {
		slog.Error("failed to fetch categories", "error", err)
		categories = []db.Category{}
	}

	// Try to find the product regardless of active status to get its category
	var relatedProducts []shop.ProductWithImage

	// Get a few featured products as suggestions (handles variants correctly)
	featuredProducts, err := s.storage.Queries.ListFeaturedProducts(ctx)
	if err == nil && len(featuredProducts) > 0 {
		limit := 4
		if len(featuredProducts) < limit {
			limit = len(featuredProducts)
		}
		for i := 0; i < limit; i++ {
			product := featuredProducts[i]
			relatedProducts = append(relatedProducts, shop.ProductWithImage{
				Product:  product,
				ImageURL: s.getProductImageURL(ctx, product),
			})
		}
	}

	// Build page metadata
	meta := layout.NewPageMeta(c, s.storage.Queries)
	meta.Title = "Product Not Available - Logan's 3D Creations"
	meta.Description = "This product is no longer available. Browse our other amazing 3D printed creations."
	meta.Keywords = []string{"3D printed", "collectibles", "custom printing"}

	c.Response().Status = http.StatusNotFound
	return Render(c, shop.ProductNotFound(c, slug, categories, relatedProducts, meta))
}

func (s *Service) handleCategory(c echo.Context) error {
	slug := c.Param("slug")
	ctx := c.Request().Context()

	// Get category by slug
	category, err := s.storage.Queries.GetCategoryBySlug(ctx, slug)
	if err != nil {
		slog.Error("failed to fetch category", "slug", slug, "error", err)
		return echo.NewHTTPError(http.StatusNotFound, "Category not found")
	}

	// Get all categories for filter
	categories, err := s.storage.Queries.ListCategories(ctx)
	if err != nil {
		slog.Error("failed to fetch categories", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load categories")
	}

	// Get products in this category
	products, err := s.storage.Queries.ListProductsByCategory(ctx, sql.NullString{String: category.ID, Valid: true})
	if err != nil {
		slog.Error("failed to fetch products", "category_id", category.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load products")
	}

	// Combine with images (handles variants correctly)
	productsWithImages := make([]shop.ProductWithImage, 0, len(products))
	for _, product := range products {
		productsWithImages = append(productsWithImages, shop.ProductWithImage{
			Product:  product,
			ImageURL: s.getProductImageURL(ctx, product),
		})
	}

	// Build PageMeta for category page
	meta := layout.NewPageMeta(c, s.storage.Queries)
	meta.Title = fmt.Sprintf("%s - 3D Printed Collectibles | Logan's 3D Creations", category.Name)
	if category.Description.Valid {
		meta.Description = category.Description.String
	} else {
		meta.Description = fmt.Sprintf("Browse our collection of 3D printed %s. High-quality prints with expert craftsmanship.", category.Name)
	}
	meta.Keywords = []string{"3D printed " + category.Name, category.Name + " collectibles", "buy 3D prints", "custom 3D printing"}
	meta.OGType = "website"

	return Render(c, shop.Index(c, meta, productsWithImages, categories, &category))
}

// Cart handlers removed - replaced with Stripe Checkout

func (s *Service) handleCustom(c echo.Context) error {
	meta := layout.NewPageMeta(c, s.storage.Queries)
	meta.Title = "Custom 3D Printing Services | Logan's 3D Creations"
	meta.Description = "Request custom 3D printing services. We bring your ideas to life with precision and quality craftsmanship."
	meta.Keywords = []string{"custom 3D printing", "personalized printing", "custom designs", "3D print service"}
	meta.OGType = "website"
	return Render(c, custom.Index(c, meta))
}

func (s *Service) handleCustomQuote(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "quote_received"})
}

func (s *Service) handleCheckoutCancel(c echo.Context) error {
	return c.Redirect(http.StatusSeeOther, "/shop")
}

func (s *Service) handleCheckoutSuccess(c echo.Context) error {
	ctx := c.Request().Context()
	sessionID := c.QueryParam("session_id")

	if sessionID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Missing session_id")
	}

	// Retrieve the Stripe checkout session
	stripe.Key = s.config.Stripe.SecretKey
	params := &stripe.CheckoutSessionParams{}
	params.AddExpand("line_items")
	params.AddExpand("line_items.data.price.product")
	params.AddExpand("total_details.breakdown")
	session, err := checkoutsession.Get(sessionID, params)
	if err != nil {
		slog.Error("failed to retrieve stripe session", "error", err, "session_id", sessionID)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve checkout session")
	}

	// Check if order already exists (from webhook)
	order, err := s.storage.Queries.GetOrderByStripeSessionID(ctx, sql.NullString{String: sessionID, Valid: true})
	if err == sql.ErrNoRows {
		// Order doesn't exist yet - webhook might not have fired
		// Create the order here (idempotency is already in handleCheckoutCompleted)
		slog.Info("order not found, creating from success page", "session_id", sessionID)

		// Call the same handler used by webhooks
		if createErr := s.paymentHandler.HandleCheckoutCompleted(c, session); createErr != nil {
			slog.Error("failed to create order from success page", "error", createErr, "session_id", sessionID)
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create order - please contact support")
		}

		// Fetch the newly created order
		order, err = s.storage.Queries.GetOrderByStripeSessionID(ctx, sql.NullString{String: sessionID, Valid: true})
		if err != nil {
			slog.Error("failed to fetch newly created order", "error", err, "session_id", sessionID)
			return echo.NewHTTPError(http.StatusInternalServerError, "Order created but cannot be displayed - please contact support")
		}
	} else if err != nil {
		slog.Error("failed to query order", "error", err, "session_id", sessionID)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve order")
	}

	// Redirect directly to order detail page
	orderURL := "/account/orders/" + order.ID
	return c.Redirect(http.StatusSeeOther, orderURL)
}

// handleCart renders the shopping cart page
func (s *Service) handleCart(c echo.Context) error {
	// Build page metadata
	meta := layout.NewPageMeta(c, s.storage.Queries)
	meta.Title = "Shopping Cart - Logan's 3D Creations"
	meta.Description = "Review your items and proceed to checkout"
	meta.Keywords = []string{"shopping cart", "checkout", "3D printed items"}

	return Render(c, shop.Cart(c, meta))
}

// handleAccount renders the account page with profile and order history
func (s *Service) handleAccount(c echo.Context) error {
	// Check authentication
	if !auth.IsAuthenticated(c) {
		// Redirect to login with return URL
		return c.Redirect(http.StatusFound, "/login?redirect_url=/account")
	}

	// Get user from context
	user, ok := auth.GetDBUser(c)
	if !ok {
		slog.Error("authenticated user not found in context")
		return echo.NewHTTPError(http.StatusUnauthorized, "User not found")
	}

	ctx := c.Request().Context()

	// Fetch user's orders
	orders, err := s.storage.Queries.ListOrdersByUser(ctx, user.ID)
	if err != nil {
		slog.Error("failed to fetch user orders", "error", err, "user_id", user.ID)
		// Don't fail - just show empty orders
		orders = []db.Order{}
	}

	// Fetch buy-again items (products from past orders that are still available)
	buyAgainItems, err := s.storage.Queries.GetBuyAgainItems(ctx, user.ID)
	if err != nil {
		slog.Debug("failed to fetch buy-again items", "error", err, "user_id", user.ID)
		// Don't fail - just show empty buy-again section
		buyAgainItems = []db.GetBuyAgainItemsRow{}
	}

	// Build page metadata
	meta := layout.NewPageMeta(c, s.storage.Queries)
	meta.Title = "My Account - Logan's 3D Creations"
	meta.Description = "Manage your account and view order history"

	// Render account page
	return Render(c, account.Index(c, user, orders, buyAgainItems, meta))
}

func (s *Service) handleAccountOrderDetail(c echo.Context) error {
	// Check authentication
	if !auth.IsAuthenticated(c) {
		return c.Redirect(http.StatusFound, "/login?redirect_url=/account")
	}

	// Get user from context
	user, ok := auth.GetDBUser(c)
	if !ok {
		slog.Error("authenticated user not found in context")
		return echo.NewHTTPError(http.StatusUnauthorized, "User not found")
	}

	ctx := c.Request().Context()
	orderID := c.Param("id")

	// Fetch the order
	order, err := s.storage.Queries.GetOrder(ctx, orderID)
	if err != nil {
		slog.Error("failed to fetch order", "error", err, "order_id", orderID)
		return echo.NewHTTPError(http.StatusNotFound, "Order not found")
	}

	// Verify the order belongs to the user
	if order.UserID != user.ID {
		slog.Error("user attempted to access order they don't own", "user_id", user.ID, "order_id", orderID)
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}

	// Fetch order items
	orderItems, err := s.storage.Queries.GetOrderItems(ctx, orderID)
	if err != nil {
		slog.Error("failed to fetch order items", "error", err, "order_id", orderID)
		orderItems = []db.OrderItem{}
	}

	// Enrich order items with product availability and images
	itemsWithProduct := make([]account.OrderItemWithProduct, len(orderItems))
	for i, item := range orderItems {
		itemsWithProduct[i] = account.OrderItemWithProduct{
			Item: item,
		}

		// Try to get product info
		product, err := s.storage.Queries.GetProduct(ctx, item.ProductID)
		if err != nil {
			slog.Debug("product not found for order item", "product_id", item.ProductID)
			continue
		}

		itemsWithProduct[i].ProductSlug = product.Slug
		itemsWithProduct[i].IsAvailable = product.IsActive.Valid && product.IsActive.Bool

		// Get product image - check for style images first if there's a SKU
		if item.ProductSkuID.Valid && item.ProductSkuID.String != "" {
			sku, err := s.storage.Queries.GetProductSku(ctx, item.ProductSkuID.String)
			if err == nil && sku.ProductStyleID != "" {
				styleImages, err := s.storage.Queries.GetProductStyleImages(ctx, sku.ProductStyleID)
				if err == nil && len(styleImages) > 0 {
					itemsWithProduct[i].ImageURL = "/public/images/products/styles/" + styleImages[0].ImageUrl
					continue
				}
			}
		}

		// Fall back to regular product images
		images, err := s.storage.Queries.GetProductImages(ctx, item.ProductID)
		if err == nil && len(images) > 0 {
			itemsWithProduct[i].ImageURL = "/public/images/products/" + images[0].ImageUrl
		}
	}

	// Build page metadata
	meta := layout.NewPageMeta(c, s.storage.Queries)
	meta.Title = fmt.Sprintf("Order #%s - Logan's 3D Creations", order.ID[:8])
	meta.Description = "View order details and tracking information"

	// Render order detail page
	return Render(c, account.OrderDetail(c, order, itemsWithProduct, meta))
}

// handleCreateStripeCheckoutSessionCart handles checkout from cart session
//
// INTENTIONAL: This handler does NOT block checkout when stock is zero.
// Products with zero stock can still be purchased - the customer will see
// extended shipping times (e.g., "Ships in 2-3 weeks") in their cart and order.
// Stock is decremented in the webhook handler with protection against going negative.
// This is a business decision to allow pre-orders/backorders rather than losing sales.
func (s *Service) handleCreateStripeCheckoutSessionCart(c echo.Context) error {
	ctx := c.Request().Context()

	// Get session ID from cookie
	sessionID, err := s.getOrCreateSessionID(c)
	if err != nil {
		slog.Error("failed to get session ID", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Session error")
	}

	// Get shipping selection from database
	shippingSelection, err := s.storage.Queries.GetSessionShippingSelection(ctx, sessionID)
	if err == sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{
			"error": "Please select shipping before checkout",
		})
	}
	if err != nil {
		slog.Error("failed to get shipping selection", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get shipping selection")
	}

	// Validate shipping is still valid
	if !shippingSelection.IsValid.Valid || !shippingSelection.IsValid.Bool {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{
			"error": "Shipping selection is no longer valid. Please select shipping again.",
		})
	}

	// SECURITY: Get authenticated user - checkout requires authentication
	user, ok := auth.GetDBUser(c)
	if !ok {
		slog.Error("checkout attempted by unauthenticated user")
		return echo.NewHTTPError(http.StatusUnauthorized, "Authentication required")
	}

	// SECURITY: Transfer any session cart items to the authenticated user
	// This ensures items added before login are associated with the user
	transferErr := s.storage.Queries.TransferCartToUser(ctx, db.TransferCartToUserParams{
		UserID:    sql.NullString{String: user.ID, Valid: true},
		SessionID: sql.NullString{String: sessionID, Valid: true},
	})
	if transferErr != nil {
		slog.Error("failed to transfer cart to user", "error", transferErr, "user_id", user.ID)
		// Don't fail - continue with checkout
	}

	// SECURITY: Get cart items by user_id to ensure user owns the cart
	// This prevents a user from checking out with another user's cart
	cartItems, err := s.storage.Queries.GetCartByUser(ctx, sql.NullString{String: user.ID, Valid: true})
	if err != nil {
		slog.Error("failed to get cart items", "error", err, "user_id", user.ID)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch cart")
	}

	if len(cartItems) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Cart is empty")
	}

	// Convert cart items to Stripe line items
	var lineItems []*stripe.CheckoutSessionLineItemParams

	for _, item := range cartItems {
		product, err := s.storage.Queries.GetProduct(ctx, item.ProductID)
		if err != nil {
			slog.Error("failed to load product for cart item", "error", err, "product_id", item.ProductID)
			return echo.NewHTTPError(http.StatusBadRequest, "One of your items is no longer available")
		}

		var sku *db.ProductSku
		if item.ProductSkuID.Valid && item.ProductSkuID.String != "" {
			skuRecord, skuErr := s.storage.Queries.GetProductSkuForProduct(ctx, db.GetProductSkuForProductParams{
				ID:        item.ProductSkuID.String,
				ProductID: item.ProductID,
			})
			if skuErr != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "A selected variant is no longer available")
			}
			if skuRecord.IsActive.Valid && !skuRecord.IsActive.Bool {
				return echo.NewHTTPError(http.StatusBadRequest, "A selected variant is unavailable")
			}
			sku = &skuRecord
		}

		variantName, imageURL, attrs, effectivePrice, err := s.buildSkuPresentation(ctx, product, sku)
		if err != nil {
			slog.Error("failed to build variant presentation", "error", err, "product_id", product.ID)
			return echo.NewHTTPError(http.StatusInternalServerError, "Unable to prepare checkout")
		}

		metadata := map[string]string{
			"product_id": item.ProductID,
		}
		if sku != nil {
			metadata["sku_id"] = sku.ID
			if sku.Sku != "" {
				metadata["sku"] = sku.Sku
			}
			for key, val := range attrs {
				metadata[key] = val
			}
		}

		lineItem := &stripe.CheckoutSessionLineItemParams{
			PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
				Currency:   stripe.String("usd"),
				UnitAmount: stripe.Int64(effectivePrice),
				ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name:     stripe.String(variantName),
					Metadata: metadata,
				},
			},
			Quantity: stripe.Int64(item.Quantity),
		}

		// Add product image if available
		if imageURL != "" {
			lineItem.PriceData.ProductData.Images = []*string{stripe.String(imageURL)}
		}

		lineItems = append(lineItems, lineItem)
	}

	// Add shipping as a line item
	deliveryDaysText := ""
	if shippingSelection.DeliveryDays.Valid && shippingSelection.DeliveryDays.Int64 > 0 {
		deliveryDaysText = fmt.Sprintf("Estimated delivery: %d business days", shippingSelection.DeliveryDays.Int64)
	}

	shippingLineItem := &stripe.CheckoutSessionLineItemParams{
		PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
			Currency:   stripe.String("usd"),
			UnitAmount: stripe.Int64(shippingSelection.PriceCents),
			ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
				Name:        stripe.String(fmt.Sprintf("Shipping - %s %s", shippingSelection.CarrierName, shippingSelection.ServiceName)),
				Description: stripe.String(deliveryDaysText),
			},
		},
		Quantity: stripe.Int64(1),
	}
	lineItems = append(lineItems, shippingLineItem)

	// Create Stripe Checkout Session
	stripe.Key = s.config.Stripe.SecretKey

	params := &stripe.CheckoutSessionParams{
		Mode:             stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems:        lineItems,
		SuccessURL:       stripe.String(fmt.Sprintf("%s://%s/checkout/success?session_id={CHECKOUT_SESSION_ID}", c.Scheme(), c.Request().Host)),
		CancelURL:        stripe.String(fmt.Sprintf("%s://%s/cart", c.Scheme(), c.Request().Host)),
		CustomerCreation: stripe.String("always"),

		// Enable automatic tax calculation
		AutomaticTax: &stripe.CheckoutSessionAutomaticTaxParams{
			Enabled: stripe.Bool(true),
		},

		// Collect shipping address for tax calculation
		ShippingAddressCollection: &stripe.CheckoutSessionShippingAddressCollectionParams{
			AllowedCountries: []*string{stripe.String("US")},
		},

		// Enable promotion code input in Stripe checkout
		AllowPromotionCodes: stripe.Bool(true),
	}

	// Store shipment_id and user_id in metadata for label creation and order linking after payment
	// SECURITY: user.ID is validated above - this ensures the order is linked to the correct user
	params.Metadata = map[string]string{
		"session_id":  sessionID,
		"shipment_id": shippingSelection.ShipmentID,
		"rate_id":     shippingSelection.RateID,
		"user_id":     user.ID,
	}

	// Expand line_items and product metadata for webhook processing
	params.AddExpand("line_items")
	params.AddExpand("line_items.data.price.product")

	session, err := checkoutsession.New(params)
	if err != nil {
		slog.Error("failed to create stripe checkout session", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create checkout session")
	}

	return c.JSON(http.StatusOK, map[string]string{"url": session.URL})
}

func (s *Service) handleEvents(c echo.Context) error {
	meta := layout.NewPageMeta(c, s.storage.Queries)
	meta.Title = "Events & Workshops | Logan's 3D Creations"
	meta.Description = "Join us for 3D printing workshops, events, and educational programs. Learn hands-on 3D printing skills."
	meta.Keywords = []string{"3D printing events", "workshops", "educational programs", "maker events"}
	meta.OGType = "website"
	return Render(c, events.Index(c, meta))
}

func (s *Service) handleContact(c echo.Context) error {
	meta := layout.NewPageMeta(c, s.storage.Queries)
	meta.Title = "Contact Us | Logan's 3D Creations"
	meta.Description = "Get in touch with Logan's 3D Creations. We're here to answer your questions about our 3D printing services."
	meta.Keywords = []string{"contact", "get in touch", "3D printing questions", "customer support"}
	meta.OGType = "website"
	return Render(c, contact.Index(c, meta))
}

func (s *Service) handleContactSubmit(c echo.Context) error {
	ctx := c.Request().Context()

	firstName := c.FormValue("first_name")
	lastName := c.FormValue("last_name")
	emailAddr := c.FormValue("email")
	phone := c.FormValue("phone")
	subject := c.FormValue("subject")
	message := c.FormValue("message")
	newsletter := c.FormValue("newsletter") == "true"
	recaptchaToken := c.FormValue("g-recaptcha-response")

	// Validate reCAPTCHA
	valid, score, err := recaptcha.IsValid(recaptchaToken)
	if err != nil {
		slog.Error("recaptcha verification error", "error", err)
		return c.HTML(http.StatusBadRequest, `<div class="mb-4 p-4 bg-red-500/20 border border-red-500/50 rounded-xl text-red-300 text-sm">reCAPTCHA verification failed. Please try again.</div>`)
	}

	if !valid {
		slog.Debug("recaptcha verification failed", "score", score)
		return c.HTML(http.StatusBadRequest, `<div class="mb-4 p-4 bg-red-500/20 border border-red-500/50 rounded-xl text-red-300 text-sm">reCAPTCHA verification failed. Please try again.</div>`)
	}

	if strings.TrimSpace(firstName) == "" || strings.TrimSpace(lastName) == "" {
		return c.HTML(http.StatusBadRequest, `<div class="mb-4 p-4 bg-red-500/20 border border-red-500/50 rounded-xl text-red-300 text-sm">First and last name are required.</div>`)
	}

	if strings.TrimSpace(emailAddr) == "" && strings.TrimSpace(phone) == "" {
		return c.HTML(http.StatusBadRequest, `<div class="mb-4 p-4 bg-red-500/20 border border-red-500/50 rounded-xl text-red-300 text-sm">Please provide at least an email address or phone number.</div>`)
	}

	if strings.TrimSpace(subject) == "" || strings.TrimSpace(message) == "" {
		return c.HTML(http.StatusBadRequest, `<div class="mb-4 p-4 bg-red-500/20 border border-red-500/50 rounded-xl text-red-300 text-sm">Subject and message are required.</div>`)
	}

	id := ulid.Make().String()

	ipAddress := c.RealIP()
	userAgent := c.Request().UserAgent()
	referrer := c.Request().Referer()

	emailNull := sql.NullString{}
	if strings.TrimSpace(emailAddr) != "" {
		emailNull = sql.NullString{String: emailAddr, Valid: true}
	}

	phoneNull := sql.NullString{}
	if strings.TrimSpace(phone) != "" {
		phoneNull = sql.NullString{String: phone, Valid: true}
	}

	_, err = s.storage.Queries.CreateContactRequest(ctx, db.CreateContactRequestParams{
		ID:                  id,
		FirstName:           firstName,
		LastName:            lastName,
		Email:               emailNull,
		Phone:               phoneNull,
		Subject:             subject,
		Message:             message,
		NewsletterSubscribe: sql.NullBool{Bool: newsletter, Valid: true},
		IpAddress:           sql.NullString{String: ipAddress, Valid: true},
		UserAgent:           sql.NullString{String: userAgent, Valid: true},
		Referrer:            sql.NullString{String: referrer, Valid: true},
		Status:              sql.NullString{String: "new", Valid: true},
		Priority:            sql.NullString{String: "normal", Valid: true},
		RecaptchaScore:      sql.NullFloat64{Float64: score, Valid: true},
	})

	if err != nil {
		slog.Error("failed to create contact request", "error", err)
		return c.HTML(http.StatusInternalServerError, `<div class="mb-4 p-4 bg-red-500/20 border border-red-500/50 rounded-xl text-red-300 text-sm">Failed to submit contact request. Please try again.</div>`)
	}

	go func() {
		emailData := &email.ContactRequestData{
			ID:                  id,
			FirstName:           firstName,
			LastName:            lastName,
			Email:               emailAddr,
			Phone:               phone,
			Subject:             subject,
			Message:             message,
			NewsletterSubscribe: newsletter,
			IPAddress:           ipAddress,
			UserAgent:           userAgent,
			Referrer:            referrer,
			SubmittedAt:         time.Now().Format("January 2, 2006 at 3:04 PM MST"),
		}

		if err := s.emailService.SendContactRequestNotification(emailData); err != nil {
			slog.Error("failed to send contact request notification", "error", err, "contact_id", id)
		}
	}()

	return c.HTML(http.StatusOK, `<div class="mb-4 p-4 bg-emerald-500/20 border border-emerald-500/50 rounded-xl text-emerald-300 text-sm">Thank you! We've received your request and will get back to you soon.</div>`)
}

func (s *Service) handlePortfolio(c echo.Context) error {
	meta := layout.NewPageMeta(c, s.storage.Queries)
	meta.Title = "Portfolio | Logan's 3D Creations"
	meta.Description = "Explore our portfolio of 3D printing projects, custom designs, and creative works."
	meta.Keywords = []string{"3D printing portfolio", "project gallery", "custom designs"}
	meta.OGType = "website"
	return Render(c, portfolio.Index(c, meta))
}

func (s *Service) handleInnovation(c echo.Context) error {
	meta := layout.NewPageMeta(c, s.storage.Queries)
	meta.Title = "Innovation | Logan's 3D Creations"
	meta.Description = "Discover the innovative technologies and techniques we use in 3D printing."
	meta.Keywords = []string{"3D printing innovation", "technology", "advanced printing"}
	meta.OGType = "website"
	return Render(c, innovation.Index(c, meta))
}

func (s *Service) handleManufacturing(c echo.Context) error {
	meta := layout.NewPageMeta(c, s.storage.Queries)
	meta.Title = "Manufacturing | Logan's 3D Creations"
	meta.Description = "Learn about our manufacturing process and capabilities for 3D printing projects."
	meta.Keywords = []string{"3D printing manufacturing", "production", "capabilities"}
	meta.OGType = "website"
	return Render(c, innovation.Manufacturing(c, meta))
}

func (s *Service) handlePrivacy(c echo.Context) error {
	meta := layout.NewPageMeta(c, s.storage.Queries)
	meta.Title = "Privacy Policy | Logan's 3D Creations"
	meta.Description = "Read our privacy policy to understand how we protect and handle your personal information."
	meta.Keywords = []string{"privacy policy", "data protection", "user privacy"}
	meta.OGType = "website"
	return Render(c, legal.Privacy(c, meta))
}

func (s *Service) handleTerms(c echo.Context) error {
	meta := layout.NewPageMeta(c, s.storage.Queries)
	meta.Title = "Terms of Service | Logan's 3D Creations"
	meta.Description = "Review our terms of service for using Logan's 3D Creations website and services."
	meta.Keywords = []string{"terms of service", "terms and conditions", "user agreement"}
	meta.OGType = "website"
	return Render(c, legal.Terms(c, meta))
}

func (s *Service) handleShipping(c echo.Context) error {
	meta := layout.NewPageMeta(c, s.storage.Queries)
	meta.Title = "Shipping Policy | Logan's 3D Creations"
	meta.Description = "Learn about our shipping policies, delivery times, and shipping costs."
	meta.Keywords = []string{"shipping policy", "delivery", "shipping costs"}
	meta.OGType = "website"
	return Render(c, legal.Shipping(c, meta))
}

func (s *Service) handleCustomPolicy(c echo.Context) error {
	meta := layout.NewPageMeta(c, s.storage.Queries)
	meta.Title = "Custom Order Policy | Logan's 3D Creations"
	meta.Description = "Understand our custom order policy, lead times, and requirements for custom 3D printing projects."
	meta.Keywords = []string{"custom order policy", "custom printing terms", "order requirements"}
	meta.OGType = "website"
	return Render(c, legal.CustomPolicy(c, meta))
}

func (s *Service) handleHealth(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"status":      "healthy",
		"environment": s.config.Environment,
		"database":    "connected",
	})
}

// Cart API Handlers

// handleAddToCart adds an item to the cart
func (s *Service) handleAddToCart(c echo.Context) error {
	var req struct {
		ProductID    string `json:"productId"`
		ProductSkuID string `json:"productSkuId"`
		Quantity     int64  `json:"quantity"`
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request format")
	}

	if req.ProductID == "" || req.Quantity <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Missing or invalid productId or quantity")
	}

	// Get or create session ID
	sessionID, err := s.getOrCreateSessionID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create session")
	}

	// Check if user is authenticated
	user, isAuthenticated := auth.GetDBUser(c)
	var userID string
	if isAuthenticated {
		userID = user.ID
	}

	ctx := c.Request().Context()

	// Check if product exists
	product, err := s.storage.Queries.GetProduct(ctx, req.ProductID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Product not found")
	}

	hasVariants := product.HasVariants.Valid && product.HasVariants.Bool

	// When product has variants, require a specific SKU
	if hasVariants {
		if req.ProductSkuID == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "Please select a variant before adding to cart")
		}
		sku, skuErr := s.storage.Queries.GetProductSkuForProduct(ctx, db.GetProductSkuForProductParams{
			ID:        req.ProductSkuID,
			ProductID: req.ProductID,
		})
		if skuErr != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid variant selection")
		}
		if sku.IsActive.Valid && !sku.IsActive.Bool {
			return echo.NewHTTPError(http.StatusBadRequest, "This variant is unavailable")
		}
	}

	// Check if item already exists in cart
	existingItem, err := s.storage.Queries.GetExistingCartItem(ctx, db.GetExistingCartItemParams{
		SessionID:    sql.NullString{String: sessionID, Valid: !isAuthenticated},
		UserID:       sql.NullString{String: userID, Valid: isAuthenticated},
		ProductID:    req.ProductID,
		ProductSkuID: sql.NullString{String: req.ProductSkuID, Valid: req.ProductSkuID != ""},
	})

	if err == nil {
		// Item exists, update quantity
		newQuantity := existingItem.Quantity + req.Quantity

		err = s.storage.Queries.UpdateCartItemQuantity(ctx, db.UpdateCartItemQuantityParams{
			ID:       existingItem.ID,
			Quantity: newQuantity,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update cart item")
		}
	} else {
		// Item doesn't exist, add new item
		itemID := uuid.New().String()
		err = s.storage.Queries.AddToCart(ctx, db.AddToCartParams{
			ID:           itemID,
			SessionID:    sql.NullString{String: sessionID, Valid: !isAuthenticated},
			UserID:       sql.NullString{String: userID, Valid: isAuthenticated},
			ProductID:    req.ProductID,
			ProductSkuID: sql.NullString{String: req.ProductSkuID, Valid: req.ProductSkuID != ""},
			Quantity:     req.Quantity,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to add item to cart")
		}
	}

	// Invalidate shipping selection when cart changes
	if s.shippingHandler != nil {
		if err := s.shippingHandler.InvalidateShipping(c, sessionID); err != nil {
			slog.Error("failed to invalidate shipping after cart change", "error", err, "session_id", sessionID)
		}
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Item added to cart successfully",
	})
}

// handleRemoveFromCart removes an item from the cart
func (s *Service) handleRemoveFromCart(c echo.Context) error {
	itemID := c.Param("id")
	if itemID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Item ID is required")
	}

	// Get session ID for invalidation
	sessionID, err := s.getOrCreateSessionID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get session")
	}

	ctx := c.Request().Context()
	err = s.storage.Queries.RemoveCartItem(ctx, itemID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to remove item from cart")
	}

	// Invalidate shipping selection when cart changes
	if s.shippingHandler != nil {
		if err := s.shippingHandler.InvalidateShipping(c, sessionID); err != nil {
			slog.Error("failed to invalidate shipping after cart change", "error", err, "session_id", sessionID)
		}
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Item removed from cart successfully",
	})
}

// handleUpdateCartItem updates the quantity of an item in the cart
func (s *Service) handleUpdateCartItem(c echo.Context) error {
	itemID := c.Param("id")
	if itemID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Item ID is required")
	}

	var req struct {
		Quantity int64 `json:"quantity"`
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request format")
	}

	if req.Quantity < 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid quantity")
	}

	// Get session ID for invalidation
	sessionID, err := s.getOrCreateSessionID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get session")
	}

	ctx := c.Request().Context()

	if req.Quantity == 0 {
		// Remove item if quantity is 0
		err := s.storage.Queries.RemoveCartItem(ctx, itemID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to remove item from cart")
		}
	} else {
		// Update quantity
		err := s.storage.Queries.UpdateCartItemQuantity(ctx, db.UpdateCartItemQuantityParams{
			ID:       itemID,
			Quantity: req.Quantity,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update cart item")
		}
	}

	// Invalidate shipping selection when cart changes
	if s.shippingHandler != nil {
		if err := s.shippingHandler.InvalidateShipping(c, sessionID); err != nil {
			slog.Error("failed to invalidate shipping after cart change", "error", err, "session_id", sessionID)
		}
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Cart updated successfully",
	})
}

// handleGetCart returns the current cart contents
func (s *Service) handleGetCart(c echo.Context) error {
	sessionID, err := s.getOrCreateSessionID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create session")
	}

	// Check if user is authenticated
	user, isAuthenticated := auth.GetDBUser(c)
	var userID string
	if isAuthenticated {
		userID = user.ID
	}

	ctx := c.Request().Context()

	// Get cart items
	var items interface{}
	if isAuthenticated {
		items, err = s.storage.Queries.GetCartByUser(ctx, sql.NullString{String: userID, Valid: true})
	} else {
		items, err = s.storage.Queries.GetCartBySession(ctx, sql.NullString{String: sessionID, Valid: true})
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get cart items")
	}

	// Get cart total
	total, err := s.storage.Queries.GetCartTotal(ctx, db.GetCartTotalParams{
		SessionID: sql.NullString{String: sessionID, Valid: !isAuthenticated},
		UserID:    sql.NullString{String: userID, Valid: isAuthenticated},
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get cart total")
	}

	// Convert total to int64 (it comes as sql.NullFloat64)
	var totalCents int64
	if total.Valid {
		totalCents = int64(total.Float64)
	}

	// Format response with shipping config for frontend
	response := map[string]interface{}{
		"items":       items,
		"totalCents":  totalCents,
		"totalDollar": float64(totalCents) / 100,
		"shippingConfig": map[string]string{
			"inStockMessage":    utils.ShippingTimeInStock,
			"outOfStockMessage": utils.ShippingTimeOutOfStock,
		},
	}

	return c.JSON(http.StatusOK, response)
}

// handleValidateCartSession checks if the current cart session should be cleared
// This happens when the user has completed checkout
func (s *Service) handleValidateCartSession(c echo.Context) error {
	// Get session cookie to check if checkout was completed
	cookie, err := c.Cookie("session_id")
	if err != nil || cookie.Value == "" {
		// No session, nothing to validate
		return c.JSON(http.StatusOK, map[string]bool{"should_clear": false})
	}

	sessionID := cookie.Value
	ctx := c.Request().Context()

	// Check if cart is empty
	cartItems, err := s.storage.Queries.GetCartBySession(ctx, sql.NullString{String: sessionID, Valid: true})
	if err != nil || len(cartItems) == 0 {
		// Cart is already empty, tell frontend to clear localStorage
		return c.JSON(http.StatusOK, map[string]bool{"should_clear": true})
	}

	// Cart has items, it's valid
	return c.JSON(http.StatusOK, map[string]bool{"should_clear": false})
}

// getOrCreateSessionID gets existing session ID or creates new one
func (s *Service) getOrCreateSessionID(c echo.Context) (string, error) {
	cookie, err := c.Cookie("session_id")
	if err != nil || cookie.Value == "" {
		// Create new session ID
		sessionID := uuid.New().String()

		// Set session cookie
		newCookie := &http.Cookie{
			Name:     "session_id",
			Value:    sessionID,
			Path:     "/",
			MaxAge:   86400 * 30, // 30 days
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		}
		c.SetCookie(newCookie)

		return sessionID, nil
	}

	return cookie.Value, nil
}

// Render renders a templ component and writes it to the response
func Render(c echo.Context, component templ.Component) error {
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTMLCharsetUTF8)
	// Don't call WriteHeader here - let Echo handle it on first Write()
	return component.Render(c.Request().Context(), c.Response())
}

// buildSkuPresentation prepares display data for Stripe and UI when a SKU is present.
func (s *Service) buildSkuPresentation(ctx context.Context, product db.Product, sku *db.ProductSku) (string, string, map[string]string, int64, error) {
	attrs := map[string]string{}

	if sku != nil {
		// Get style name
		if style, err := s.storage.Queries.GetProductStyle(ctx, sku.ProductStyleID); err == nil {
			attrs["style"] = style.Name
		}
		// Get size name
		if size, err := s.storage.Queries.GetSize(ctx, sku.SizeID); err == nil {
			attrs["size"] = size.DisplayName
		}
	}

	var variantParts []string
	if style, ok := attrs["style"]; ok {
		variantParts = append(variantParts, style)
	}
	if size, ok := attrs["size"]; ok {
		variantParts = append(variantParts, size)
	}

	variantName := product.Name
	if len(variantParts) > 0 {
		variantName = fmt.Sprintf("%s - %s", product.Name, strings.Join(variantParts, ", "))
	}

	imageURL := ""
	if sku != nil {
		// Get primary image for the style
		if img, err := s.storage.Queries.GetPrimaryStyleImage(ctx, sku.ProductStyleID); err == nil {
			imageURL = fmt.Sprintf("%s/public/images/products/styles/%s", s.config.BaseURL, img.ImageUrl)
		}
	}
	if imageURL == "" {
		if primary, err := s.storage.Queries.GetPrimaryProductImage(ctx, product.ID); err == nil {
			imageURL = fmt.Sprintf("%s/public/images/products/%s", s.config.BaseURL, primary.ImageUrl)
		}
	}

	effectivePrice := product.PriceCents
	if sku != nil {
		effectivePrice += int64FromNull(sku.PriceAdjustmentCents)
	}

	return variantName, imageURL, attrs, effectivePrice, nil
}

func int64FromNull(n sql.NullInt64) int64 {
	if n.Valid {
		return n.Int64
	}
	return 0
}
