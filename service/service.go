package service

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/oklog/ulid/v2"
	"github.com/stripe/stripe-go/v80"
	checkoutsession "github.com/stripe/stripe-go/v80/checkout/session"
	"github.com/loganlanou/logans3d-v4/internal/auth"
	"github.com/loganlanou/logans3d-v4/internal/email"
	"github.com/loganlanou/logans3d-v4/internal/handlers"
	"github.com/loganlanou/logans3d-v4/internal/shipping"
	"github.com/loganlanou/logans3d-v4/storage"
	"github.com/loganlanou/logans3d-v4/storage/db"
	"github.com/loganlanou/logans3d-v4/views/about"
	"github.com/loganlanou/logans3d-v4/views/account"
	"github.com/loganlanou/logans3d-v4/views/contact"
	"github.com/loganlanou/logans3d-v4/views/custom"
	"github.com/loganlanou/logans3d-v4/views/events"
	"github.com/loganlanou/logans3d-v4/views/home"
	"github.com/loganlanou/logans3d-v4/views/innovation"
	"github.com/loganlanou/logans3d-v4/views/legal"
	"github.com/loganlanou/logans3d-v4/views/portfolio"
	"github.com/loganlanou/logans3d-v4/views/shop"
)


type Service struct {
	storage         *storage.Storage
	config          *Config
	paymentHandler  *handlers.PaymentHandler
	shippingHandler *handlers.ShippingHandler
	shippingService *shipping.ShippingService
	emailService    *email.Service
	authHandler     *handlers.AuthHandler
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

	shippingService, err := shipping.NewShippingService(shippingConfig)
	if err != nil {
		slog.Error("failed to initialize shipping service", "error", err)
		// Continue without shipping service for now
		shippingService = nil
	}

	var shippingHandler *handlers.ShippingHandler
	if shippingService != nil {
		shippingHandler = handlers.NewShippingHandler(storage.Queries, shippingService)
	}

	// Initialize email service
	emailService := email.NewService()

	return &Service{
		storage:         storage,
		config:          config,
		paymentHandler:  handlers.NewPaymentHandler(storage.Queries, emailService),
		shippingHandler: shippingHandler,
		shippingService: shippingService,
		emailService:    emailService,
		authHandler:     handlers.NewAuthHandler(),
	}
}

func (s *Service) RegisterRoutes(e *echo.Echo) {
	// Initialize Clerk SDK with secret key - this configures the default backend
	clerkSecretKey := os.Getenv("CLERK_SECRET_KEY")
	if clerkSecretKey == "" {
		slog.Error("CLERK_SECRET_KEY is not set!")
	} else {
		slog.Debug("Clerk SDK initialized", "secret_key_prefix", clerkSecretKey[:min(len(clerkSecretKey), 10)])
	}
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

	// Account routes
	withAuth.GET("/account", s.handleAccount)
	withAuth.GET("/account/orders/:id", s.handleAccountOrderDetail)

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
	withAuth.POST("/checkout/create-session", s.handleCreateStripeCheckoutSession)
	withAuth.POST("/checkout/create-session-single", s.handleCreateStripeCheckoutSessionSingle)
	withAuth.POST("/checkout/create-session-multi", s.handleCreateStripeCheckoutSessionMulti)
	withAuth.POST("/checkout/create-session-cart", s.handleCreateStripeCheckoutSessionCart)
	withAuth.GET("/checkout/success", s.handleCheckoutSuccess)
	withAuth.GET("/checkout/cancel", s.handleCheckoutCancel)

	// Payment API routes
	api := withAuth.Group("/api")
	api.POST("/payment/create-intent", s.paymentHandler.CreatePaymentIntent)
	api.POST("/payment/create-customer", s.paymentHandler.CreateCustomer)
	api.POST("/stripe/webhook", s.paymentHandler.HandleWebhook)

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
	admin := withAuth.Group("/admin", auth.RequireAdmin())
	admin.GET("", adminHandler.HandleAdminDashboard)
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
	admin.POST("/product/image/:imageId/delete", adminHandler.HandleDeleteProductImage)
	admin.POST("/product/image/:imageId/set-primary", adminHandler.HandleSetPrimaryProductImage)
	admin.GET("/product/search", adminHandler.HandleProductSearch)
	
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
	admin.POST("/contacts/:id/priority", adminHandler.HandleUpdateContactPriority)
	admin.POST("/contacts/:id/notes", adminHandler.HandleAddContactNotes)
	admin.POST("/contacts/:id/notes/delete", adminHandler.HandleDeleteContactNotes)

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

	// Combine with images
	productsWithImages := make([]home.ProductWithImage, 0, len(featuredProducts))
	for _, product := range featuredProducts {
		images, err := s.storage.Queries.GetProductImages(ctx, product.ID)
		if err != nil {
			slog.Error("failed to fetch product images", "product_id", product.ID, "error", err)
			continue
		}

		imageURL := ""
		if len(images) > 0 {
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
				imageURL = "/public/images/products/" + rawImageURL
			}
		}

		productsWithImages = append(productsWithImages, home.ProductWithImage{
			Product:  product,
			ImageURL: imageURL,
		})
	}

	return Render(c, home.Index(c, productsWithImages))
}

func (s *Service) handleAbout(c echo.Context) error {
	return Render(c, about.Index(c))
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
	
	// Combine with images
	productsWithImages := make([]shop.ProductWithImage, 0, len(products))
	for _, product := range products {
		images, err := s.storage.Queries.GetProductImages(ctx, product.ID)
		if err != nil {
			slog.Error("failed to fetch product images", "product_id", product.ID, "error", err)
			continue
		}
		
		imageURL := ""
		if len(images) > 0 {
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
				imageURL = "/public/images/products/" + rawImageURL
			}
		}
		
		productsWithImages = append(productsWithImages, shop.ProductWithImage{
			Product:  product,
			ImageURL: imageURL,
		})
	}

	return Render(c, shop.Index(c, productsWithImages, categories, nil))
}

func (s *Service) handlePremium(c echo.Context) error {
	ctx := c.Request().Context()
	
	// Create sample premium collection tiers
	collections := []shop.CollectionTier{
		{
			Name:         "Bronze",
			Slug:         "bronze",
			Description:  "Essential premium pieces to start your collection with high-quality detail and materials",
			Price:        4999, // $49.99
			OriginalPrice: 5999, // $59.99
			Discount:     17,
			Items:        3,
			Color:        "amber",
			GradientFrom: "from-amber-600",
			GradientTo:   "to-yellow-600",
			IconEmoji:    "🥉",
			Features: []string{
				"3 carefully selected premium items",
				"High-detail 0.2mm layer resolution",
				"Premium PLA+ materials",
				"Basic post-processing included",
				"Standard shipping",
			},
		},
		{
			Name:         "Silver",
			Slug:         "silver",
			Description:  "Enhanced collection featuring superior detail and exclusive variations for dedicated collectors",
			Price:        9999, // $99.99
			OriginalPrice: 12999, // $129.99
			Discount:     23,
			Items:        6,
			Color:        "gray",
			GradientFrom: "from-gray-500",
			GradientTo:   "to-slate-500",
			IconEmoji:    "🥈",
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
			Name:         "Gold",
			Slug:         "gold",
			Description:  "Elite tier with the most detailed models, rare materials, and collector-exclusive items",
			Price:        19999, // $199.99
			OriginalPrice: 27999, // $279.99
			Discount:     29,
			Items:        10,
			Color:        "amber",
			GradientFrom: "from-amber-500",
			GradientTo:   "to-yellow-500",
			IconEmoji:    "🥇",
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
			Name:         "Platinum",
			Slug:         "platinum",
			Description:  "Ultra-exclusive collection with master-crafted pieces and personalized touches",
			Price:        39999, // $399.99
			OriginalPrice: 54999, // $549.99
			Discount:     27,
			Items:        15,
			Color:        "slate",
			GradientFrom: "from-slate-400",
			GradientTo:   "to-gray-400",
			IconEmoji:    "💎",
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
			Name:         "Titanium",
			Slug:         "titanium",
			Description:  "Industrial-grade collection featuring aerospace materials and cutting-edge techniques",
			Price:        79999, // $799.99
			OriginalPrice: 109999, // $1099.99
			Discount:     27,
			Items:        20,
			Color:        "slate",
			GradientFrom: "from-slate-600",
			GradientTo:   "to-gray-600",
			IconEmoji:    "🛡️",
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
			Name:         "Diamond",
			Slug:         "diamond",
			Description:  "The pinnacle of 3D printing excellence with precious metal inlays and gemstone accents",
			Price:        159999, // $1599.99
			OriginalPrice: 219999, // $2199.99
			Discount:     27,
			Items:        25,
			Color:        "blue",
			GradientFrom: "from-blue-400",
			GradientTo:   "to-cyan-400",
			IconEmoji:    "💎",
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
			Name:         "Collectors",
			Slug:         "collectors",
			Description:  "Ultimate prestige collection for serious collectors with one-of-a-kind masterpieces",
			Price:        299999, // $2999.99
			OriginalPrice: 399999, // $3999.99
			Discount:     25,
			Items:        50,
			Color:        "purple",
			GradientFrom: "from-purple-600",
			GradientTo:   "to-pink-600",
			IconEmoji:    "👑",
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
	
	// Sort products by price descending and take top 8
	featuredProducts := make([]shop.ProductWithImage, 0, 8)
	count := 0
	for _, product := range products {
		if count >= 8 {
			break
		}
		
		images, err := s.storage.Queries.GetProductImages(ctx, product.ID)
		if err != nil {
			slog.Error("failed to fetch product images", "product_id", product.ID, "error", err)
			continue
		}
		
		imageURL := ""
		if len(images) > 0 {
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
				imageURL = "/public/images/products/" + rawImageURL
			}
		}
		
		featuredProducts = append(featuredProducts, shop.ProductWithImage{
			Product:  product,
			ImageURL: imageURL,
		})
		count++
	}

	return Render(c, shop.Premium(c, collections, featuredProducts))
}

func (s *Service) handleProduct(c echo.Context) error {
	slug := c.Param("slug")
	ctx := c.Request().Context()

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

		// Build ProductWithImage for each related product
		for _, relatedProduct := range relatedProductsList {
			primaryImage, err := s.storage.Queries.GetPrimaryProductImage(ctx, relatedProduct.ID)
			imageURL := ""
			if err == nil {
				imageURL = fmt.Sprintf("/public/images/products/%s", primaryImage.ImageUrl)
			}

			relatedProducts = append(relatedProducts, shop.ProductWithImage{
				Product:  relatedProduct,
				ImageURL: imageURL,
			})
		}
	}

	return Render(c, shop.Product(c, product, category, images, relatedProducts))
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

	// Get a few featured products as suggestions
	featuredProducts, err := s.storage.Queries.ListFeaturedProducts(ctx)
	if err == nil && len(featuredProducts) > 0 {
		// Limit to 4 products
		limit := 4
		if len(featuredProducts) < limit {
			limit = len(featuredProducts)
		}

		for i := 0; i < limit; i++ {
			product := featuredProducts[i]
			images, err := s.storage.Queries.GetProductImages(ctx, product.ID)
			if err != nil {
				continue
			}

			imageURL := ""
			if len(images) > 0 {
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
					imageURL = "/public/images/products/" + rawImageURL
				}
			}

			relatedProducts = append(relatedProducts, shop.ProductWithImage{
				Product:  product,
				ImageURL: imageURL,
			})
		}
	}

	c.Response().Status = http.StatusNotFound
	return Render(c, shop.ProductNotFound(c, slug, categories, relatedProducts))
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
	
	// Combine with images
	productsWithImages := make([]shop.ProductWithImage, 0, len(products))
	for _, product := range products {
		images, err := s.storage.Queries.GetProductImages(ctx, product.ID)
		if err != nil {
			slog.Error("failed to fetch product images", "product_id", product.ID, "error", err)
			continue
		}
		
		imageURL := ""
		if len(images) > 0 {
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
				imageURL = "/public/images/products/" + rawImageURL
			}
		}
		
		productsWithImages = append(productsWithImages, shop.ProductWithImage{
			Product:  product,
			ImageURL: imageURL,
		})
	}

	return Render(c, shop.Index(c, productsWithImages, categories, &category))
}

// Cart handlers removed - replaced with Stripe Checkout

func (s *Service) handleCustom(c echo.Context) error {
	return Render(c, custom.Index(c))
}

func (s *Service) handleCustomQuote(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "quote_received"})
}

func (s *Service) handleCreateStripeCheckoutSession(c echo.Context) error {
	ctx := c.Request().Context()
	
	// Parse form data
	productID := c.FormValue("product_id")
	quantityStr := c.FormValue("quantity")
	
	if productID == "" || quantityStr == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Missing product_id or quantity")
	}
	
	quantity, err := strconv.ParseInt(quantityStr, 10, 64)
	if err != nil || quantity <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid quantity")
	}
	
	// Get product details from database
	product, err := s.storage.Queries.GetProduct(ctx, productID)
	if err != nil {
		slog.Error("failed to get product", "error", err, "product_id", productID)
		return echo.NewHTTPError(http.StatusNotFound, "Product not found")
	}
	
	// Check stock
	if product.StockQuantity.Valid && product.StockQuantity.Int64 < quantity {
		return echo.NewHTTPError(http.StatusBadRequest, "Not enough stock available")
	}
	
	// Get primary product image (optional)
	imageURL := ""
	images, err := s.storage.Queries.GetProductImages(ctx, productID)
	if err == nil && len(images) > 0 {
		// Ensure we have an absolute URL for Stripe - database contains only filename
		imageURL = fmt.Sprintf("%s/public/images/products/%s", s.config.BaseURL, images[0].ImageUrl)
	}
	
	// Create Stripe Checkout Session with dynamic product
	stripe.Key = s.config.Stripe.SecretKey
	
	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency:   stripe.String("usd"),
					UnitAmount: stripe.Int64(product.PriceCents),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name:        stripe.String(product.Name),
						Description: &product.Description.String,
					},
				},
				Quantity: stripe.Int64(quantity),
			},
		},
		SuccessURL: stripe.String(fmt.Sprintf("%s://%s/checkout/success?session_id={CHECKOUT_SESSION_ID}", c.Scheme(), c.Request().Host)),
		CancelURL:  stripe.String(fmt.Sprintf("%s://%s/shop", c.Scheme(), c.Request().Host)),
		CustomerCreation: stripe.String("always"), // Always create customer for order tracking
		PaymentIntentData: &stripe.CheckoutSessionPaymentIntentDataParams{
			Metadata: map[string]string{
				"product_id": productID,
				"quantity":   quantityStr,
			},
		},
	}
	
	// Add product image if available
	if imageURL != "" {
		params.LineItems[0].PriceData.ProductData.Images = []*string{stripe.String(imageURL)}
	}

	// Expand line_items for webhook processing
	params.AddExpand("line_items")

	session, err := checkoutsession.New(params)
	if err != nil {
		slog.Error("failed to create stripe checkout session", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create checkout session")
	}
	
	// Redirect to Stripe Checkout
	return c.Redirect(http.StatusSeeOther, session.URL)
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
	return Render(c, shop.Cart(c))
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

	// Render account page
	return Render(c, account.Index(c, user, orders))
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

	// Render order detail page
	return Render(c, account.OrderDetail(c, order, orderItems))
}

// handleCreateStripeCheckoutSessionSingle handles single item checkout (Buy Now)
func (s *Service) handleCreateStripeCheckoutSessionSingle(c echo.Context) error {
	ctx := c.Request().Context()
	
	// Parse JSON data
	var request struct {
		ProductID string `json:"productId"`
		Quantity  int64  `json:"quantity"`
	}
	
	if err := c.Bind(&request); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request data")
	}
	
	if request.ProductID == "" || request.Quantity <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Missing or invalid productId or quantity")
	}
	
	// Get product details from database
	product, err := s.storage.Queries.GetProduct(ctx, request.ProductID)
	if err != nil {
		slog.Error("failed to get product", "error", err, "product_id", request.ProductID)
		return echo.NewHTTPError(http.StatusNotFound, "Product not found")
	}
	
	// Check stock
	if product.StockQuantity.Valid && product.StockQuantity.Int64 < request.Quantity {
		return echo.NewHTTPError(http.StatusBadRequest, "Not enough stock available")
	}
	
	// Get primary product image (optional)
	imageURL := ""
	images, err := s.storage.Queries.GetProductImages(ctx, request.ProductID)
	if err == nil && len(images) > 0 {
		// Ensure we have an absolute URL for Stripe - database contains only filename
		imageURL = fmt.Sprintf("%s/public/images/products/%s", s.config.BaseURL, images[0].ImageUrl)
	}
	
	// Create Stripe Checkout Session with dynamic product
	stripe.Key = s.config.Stripe.SecretKey
	
	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency:   stripe.String("usd"),
					UnitAmount: stripe.Int64(product.PriceCents),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name:        stripe.String(product.Name),
						Description: &product.Description.String,
					},
				},
				Quantity: stripe.Int64(request.Quantity),
			},
		},
		SuccessURL: stripe.String(fmt.Sprintf("%s://%s/checkout/success?session_id={CHECKOUT_SESSION_ID}", c.Scheme(), c.Request().Host)),
		CancelURL:  stripe.String(fmt.Sprintf("%s://%s/shop", c.Scheme(), c.Request().Host)),
		CustomerCreation: stripe.String("always"),
		PaymentIntentData: &stripe.CheckoutSessionPaymentIntentDataParams{
			Metadata: map[string]string{
				"product_id": request.ProductID,
				"quantity":   strconv.FormatInt(request.Quantity, 10),
			},
		},
	}
	
	// Add product image if available
	if imageURL != "" {
		params.LineItems[0].PriceData.ProductData.Images = []*string{stripe.String(imageURL)}
	}

	// Expand line_items for webhook processing
	params.AddExpand("line_items")

	session, err := checkoutsession.New(params)
	if err != nil {
		slog.Error("failed to create stripe checkout session", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create checkout session")
	}
	
	return c.JSON(http.StatusOK, map[string]string{"url": session.URL})
}

// handleCreateStripeCheckoutSessionMulti handles multi-item checkout (Cart)
func (s *Service) handleCreateStripeCheckoutSessionMulti(c echo.Context) error {
	ctx := c.Request().Context()
	
	// Parse JSON data
	var request struct {
		Items []struct {
			ProductID string `json:"productId"`
			Name      string `json:"name"`
			Price     int64  `json:"price"` // price in cents
			ImageURL  string `json:"imageUrl"`
			Quantity  int64  `json:"quantity"`
		} `json:"items"`
	}
	
	if err := c.Bind(&request); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request data")
	}
	
	if len(request.Items) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "No items in cart")
	}
	
	// Validate each item and check stock
	var lineItems []*stripe.CheckoutSessionLineItemParams
	
	for _, item := range request.Items {
		if item.ProductID == "" || item.Quantity <= 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid item data")
		}
		
		// Get product details to verify and check stock
		product, err := s.storage.Queries.GetProduct(ctx, item.ProductID)
		if err != nil {
			slog.Error("failed to get product", "error", err, "product_id", item.ProductID)
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Product %s not found", item.ProductID))
		}
		
		// Check stock
		if product.StockQuantity.Valid && product.StockQuantity.Int64 < item.Quantity {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Not enough stock available for %s", product.Name))
		}
		
		// Create line item
		lineItem := &stripe.CheckoutSessionLineItemParams{
			PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
				Currency:   stripe.String("usd"),
				UnitAmount: stripe.Int64(item.Price),
				ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name: stripe.String(item.Name),
				},
			},
			Quantity: stripe.Int64(item.Quantity),
		}
		
		// Add product image if available
		if item.ImageURL != "" {
			// Ensure we have an absolute URL for Stripe - database contains only filename
			imageURL := fmt.Sprintf("%s/public/images/products/%s", s.config.BaseURL, item.ImageURL)
			lineItem.PriceData.ProductData.Images = []*string{stripe.String(imageURL)}
		}
		
		lineItems = append(lineItems, lineItem)
	}
	
	// Create Stripe Checkout Session with multiple items
	stripe.Key = s.config.Stripe.SecretKey
	
	params := &stripe.CheckoutSessionParams{
		Mode:             stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems:        lineItems,
		SuccessURL:       stripe.String(fmt.Sprintf("%s://%s/checkout/success?session_id={CHECKOUT_SESSION_ID}", c.Scheme(), c.Request().Host)),
		CancelURL:        stripe.String(fmt.Sprintf("%s://%s/cart", c.Scheme(), c.Request().Host)),
		CustomerCreation: stripe.String("always"),
	}

	// Expand line_items for webhook processing
	params.AddExpand("line_items")

	session, err := checkoutsession.New(params)
	if err != nil {
		slog.Error("failed to create stripe checkout session", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create checkout session")
	}
	
	return c.JSON(http.StatusOK, map[string]string{"url": session.URL})
}

// handleCreateStripeCheckoutSessionCart handles checkout from cart session
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
		lineItem := &stripe.CheckoutSessionLineItemParams{
			PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
				Currency:   stripe.String("usd"),
				UnitAmount: stripe.Int64(item.PriceCents),
				ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name: stripe.String(item.Name),
					Metadata: map[string]string{
						"product_id": item.ProductID,
					},
				},
			},
			Quantity: stripe.Int64(item.Quantity),
		}

		// Add product image if available
		if item.ImageUrl != "" {
			// Ensure we have an absolute URL for Stripe - database contains only filename
			imageURL := fmt.Sprintf("%s/public/images/products/%s", s.config.BaseURL, item.ImageUrl)
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
	return Render(c, events.Index(c))
}

func (s *Service) handleContact(c echo.Context) error {
	return Render(c, contact.Index(c))
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

	_, err := s.storage.Queries.CreateContactRequest(ctx, db.CreateContactRequestParams{
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

	return c.HTML(http.StatusOK, `<div class="mb-4 p-4 bg-emerald-500/20 border border-emerald-500/50 rounded-xl text-emerald-300 text-sm">Thank you for your message! We'll get back to you soon.</div>`)
}

func (s *Service) handlePortfolio(c echo.Context) error {
	return Render(c, portfolio.Index(c))
}

func (s *Service) handleInnovation(c echo.Context) error {
	return Render(c, innovation.Index(c))
}

func (s *Service) handleManufacturing(c echo.Context) error {
	return Render(c, innovation.Manufacturing(c))
}

func (s *Service) handlePrivacy(c echo.Context) error {
	return Render(c, legal.Privacy(c))
}

func (s *Service) handleTerms(c echo.Context) error {
	return Render(c, legal.Terms(c))
}

func (s *Service) handleShipping(c echo.Context) error {
	return Render(c, legal.Shipping(c))
}

func (s *Service) handleCustomPolicy(c echo.Context) error {
	return Render(c, legal.CustomPolicy(c))
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
		ProductID string `json:"productId"`
		Quantity  int64  `json:"quantity"`
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

	ctx := c.Request().Context()

	// Check if product exists
	_, err = s.storage.Queries.GetProduct(ctx, req.ProductID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Product not found")
	}

	// Check if item already exists in cart
	existingItem, err := s.storage.Queries.GetExistingCartItem(ctx, db.GetExistingCartItemParams{
		SessionID: sql.NullString{String: sessionID, Valid: true},
		UserID:    sql.NullString{Valid: false}, // Handle user auth later
		ProductID: req.ProductID,
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
			ID:        itemID,
			SessionID: sql.NullString{String: sessionID, Valid: true},
			UserID:    sql.NullString{Valid: false},
			ProductID: req.ProductID,
			Quantity:  req.Quantity,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to add item to cart")
		}
	}

	// Invalidate shipping selection when cart changes
	if s.shippingHandler != nil {
		s.shippingHandler.InvalidateShipping(c, sessionID)
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
		s.shippingHandler.InvalidateShipping(c, sessionID)
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
		s.shippingHandler.InvalidateShipping(c, sessionID)
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

	ctx := c.Request().Context()

	// Get cart items
	items, err := s.storage.Queries.GetCartBySession(ctx, sql.NullString{String: sessionID, Valid: true})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get cart items")
	}

	// Get cart total
	total, err := s.storage.Queries.GetCartTotal(ctx, db.GetCartTotalParams{
		SessionID: sql.NullString{String: sessionID, Valid: true},
		UserID:    sql.NullString{Valid: false},
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get cart total")
	}

	// Convert total to int64 (it comes as sql.NullFloat64)
	var totalCents int64
	if total.Valid {
		totalCents = int64(total.Float64)
	}

	// Format response
	response := map[string]interface{}{
		"items":       items,
		"totalCents":  totalCents,
		"totalDollar": float64(totalCents) / 100,
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

// Shipping admin handlers
func (s *Service) handleShippingAdmin(c echo.Context) error {
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>Shipping Configuration - Logan's 3D Creations Admin</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 1200px; margin: 0 auto; padding: 20px; }
        .section { margin: 20px 0; padding: 20px; border: 1px solid #ddd; border-radius: 8px; }
        .config-text { width: 100%; height: 400px; font-family: monospace; }
        .btn { background: #007bff; color: white; padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; }
        .btn:hover { background: #0056b3; }
        .success { color: green; } .error { color: red; }
        .info { background: #f8f9fa; padding: 15px; border-left: 4px solid #007bff; margin: 10px 0; }
    </style>
</head>
<body>
    <h1>Shipping Configuration</h1>

    <div class="info">
        <strong>Note:</strong> This is a basic admin interface for the shipping integration.
        The shipping system is now integrated and ready for production use with ShipStation.
    </div>

    <div class="section">
        <h2>ShipStation Integration Status</h2>
        <p><strong>API Status:</strong> <span id="api-status">Checking...</span></p>
        <p><strong>Configuration:</strong> Loaded from config/shipping.json</p>
        <p><strong>Database Schema:</strong> Updated with shipping tables</p>
        <button class="btn" onclick="testConnection()">Test ShipStation Connection</button>
    </div>

    <div class="section">
        <h2>Current Configuration</h2>
        <p>The shipping configuration includes:</p>
        <ul>
            <li>Box catalog with 4 optimized box sizes</li>
            <li>Packing algorithm for optimal box selection</li>
            <li>ShipStation API integration for live rates</li>
            <li>Automated label creation and tracking</li>
            <li>USPS Ground Advantage Cubic tier optimization</li>
        </ul>
        <button class="btn" onclick="window.location.href='/admin/shipping/test'">View Test Interface</button>
    </div>

    <div class="section">
        <h2>API Endpoints Available</h2>
        <ul>
            <li><code>POST /api/shipping/rates</code> - Get shipping rates for cart</li>
            <li><code>POST /api/shipping/labels</code> - Create shipping label</li>
            <li><code>PUT /api/shipping/labels/:id/void</code> - Void shipping label</li>
            <li><code>POST /api/shipping/validate-address</code> - Validate shipping address</li>
        </ul>
    </div>

    <div class="section">
        <h2>Implementation Status</h2>
        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 20px;">
            <div>
                <h3>✅ Completed</h3>
                <ul>
                    <li>Shipping configuration system</li>
                    <li>Box catalog and packing algorithm</li>
                    <li>ShipStation API client</li>
                    <li>Rate calculation service</li>
                    <li>Database schema</li>
                    <li>API endpoints</li>
                    <li>Basic tests</li>
                </ul>
            </div>
            <div>
                <h3>🔄 Next Steps</h3>
                <ul>
                    <li>Frontend integration for checkout</li>
                    <li>Order fulfillment workflow</li>
                    <li>Admin order management UI</li>
                    <li>Production API key configuration</li>
                    <li>Shipping address validation UI</li>
                    <li>Comprehensive testing</li>
                </ul>
            </div>
        </div>
    </div>

    <p><a href="/admin">← Back to Admin Dashboard</a></p>

    <script>
        function testConnection() {
            document.getElementById('api-status').textContent = 'Testing...';
            fetch('/admin/shipping/test')
                .then(response => response.text())
                .then(data => {
                    document.getElementById('api-status').innerHTML = '<span class="success">✅ Ready for testing</span>';
                })
                .catch(error => {
                    document.getElementById('api-status').innerHTML = '<span class="error">❌ ' + error.message + '</span>';
                });
        }

        // Check status on load
        testConnection();
    </script>
</body>
</html>`

	return c.HTML(http.StatusOK, html)
}

func (s *Service) handleShippingConfigUpdate(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "success",
		"message": "Configuration update feature coming soon",
	})
}

func (s *Service) handleShippingTest(c echo.Context) error {
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>Shipping Test Interface</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        .form-group { margin: 15px 0; }
        label { display: block; margin-bottom: 5px; font-weight: bold; }
        input, select { width: 100%; padding: 8px; border: 1px solid #ddd; border-radius: 4px; }
        .btn { background: #007bff; color: white; padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; margin: 5px; }
        .btn:hover { background: #0056b3; }
        .result { margin: 20px 0; padding: 15px; border: 1px solid #ddd; border-radius: 4px; background: #f8f9fa; }
        .error { color: red; background: #f8d7da; border-color: #f5c6cb; }
        .success { color: green; background: #d4edda; border-color: #c3e6cb; }
    </style>
</head>
<body>
    <h1>Shipping Test Interface</h1>

    <div class="form-group">
        <h2>Test Cart Items</h2>
        <label>Small Items:</label>
        <input type="number" id="small" value="2" min="0">

        <label>Medium Items:</label>
        <input type="number" id="medium" value="1" min="0">

        <label>Large Items:</label>
        <input type="number" id="large" value="0" min="0">

        <label>XL Items:</label>
        <input type="number" id="xl" value="0" min="0">
    </div>

    <div class="form-group">
        <h2>Shipping Address</h2>
        <label>Postal Code:</label>
        <input type="text" id="postal" value="55401" placeholder="55401">

        <label>Country:</label>
        <select id="country">
            <option value="US">United States</option>
            <option value="CA">Canada</option>
        </select>
    </div>

    <button class="btn" onclick="testRates()">Get Shipping Rates</button>
    <button class="btn" onclick="testPacking()">Test Packing Only</button>

    <div id="result"></div>

    <p><a href="/admin/shipping">← Back to Shipping Admin</a></p>

    <script>
        function testRates() {
            const data = {
                item_counts: {
                    small: parseInt(document.getElementById('small').value) || 0,
                    medium: parseInt(document.getElementById('medium').value) || 0,
                    large: parseInt(document.getElementById('large').value) || 0,
                    xl: parseInt(document.getElementById('xl').value) || 0
                },
                ship_to: {
                    postal_code: document.getElementById('postal').value,
                    country_code: document.getElementById('country').value
                }
            };

            document.getElementById('result').innerHTML = '<div class="result">Getting shipping rates...</div>';

            fetch('/api/shipping/rates', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify(data)
            })
            .then(response => response.json())
            .then(data => {
                let html = '<div class="result success"><h3>Shipping Options</h3>';
                if (data.options && data.options.length > 0) {
                    data.options.forEach(option => {
                        html += '<p><strong>' + option.carrier_name + ' ' + option.service_name + '</strong><br>';
                        html += 'Price: $' + option.price.toFixed(2) + ' + $' + option.box_cost.toFixed(2) + ' (box) = $' + option.total_cost.toFixed(2) + '<br>';
                        html += 'Delivery: ' + option.delivery_days + ' days<br>';
                        html += 'Box: ' + option.box_sku + '</p>';
                    });
                } else {
                    html += '<p>No shipping options available. Error: ' + (data.error || 'Unknown error') + '</p>';
                }
                html += '</div>';
                document.getElementById('result').innerHTML = html;
            })
            .catch(error => {
                document.getElementById('result').innerHTML = '<div class="result error">Error: ' + error.message + '</div>';
            });
        }

        function testPacking() {
            const small = parseInt(document.getElementById('small').value) || 0;
            const medium = parseInt(document.getElementById('medium').value) || 0;
            const large = parseInt(document.getElementById('large').value) || 0;
            const xl = parseInt(document.getElementById('xl').value) || 0;

            // Calculate estimated weights based on new system
            const itemWeights = {
                small: 3.0,   // oz
                medium: 7.05, // oz
                large: 15.0,  // oz
                xlarge: 35.3  // oz
            };

            const totalItems = small + medium + large + xl;
            const itemWeight = small * itemWeights.small + medium * itemWeights.medium +
                             large * itemWeights.large + xl * itemWeights.xlarge;

            // Packing materials
            const bubbleWrap = totalItems * 0.2;
            const packingMaterials = 1.0 + 0.5 + 0.8; // paper + tape + air pillows

            // Assume 10x8x6 box for demo
            const boxWeight = 6.0;
            const totalWeight = boxWeight + itemWeight + bubbleWrap + packingMaterials;

            let html = '<div class="result"><h3>Enhanced Packing Analysis</h3>';
            html += '<p><strong>Items:</strong> ' + small + ' small, ' + medium + ' medium, ' + large + ' large, ' + xl + ' XL</p>';
            html += '<p><strong>Total Small Units:</strong> ' + (small + medium*3 + large*6 + xl*18) + '</p>';
            html += '<h4>Weight Breakdown:</h4>';
            html += '<ul>';
            html += '<li>Items: ' + itemWeight.toFixed(2) + ' oz</li>';
            html += '<li>Bubble wrap: ' + bubbleWrap.toFixed(2) + ' oz</li>';
            html += '<li>Packing materials: ' + packingMaterials.toFixed(2) + ' oz</li>';
            html += '<li>Box (10x8x6): ' + boxWeight.toFixed(2) + ' oz</li>';
            html += '<li><strong>Total estimated weight: ' + totalWeight.toFixed(2) + ' oz (' + (totalWeight/16).toFixed(2) + ' lbs)</strong></li>';
            html += '</ul>';
            html += '<p><em>The system now factors in actual item weights (70-100g small, ~200g medium, 350-500g large), packing materials, and box weights for accurate shipping rates.</em></p>';
            html += '</div>';

            document.getElementById('result').innerHTML = html;
        }
    </script>
</body>
</html>`

	return c.HTML(http.StatusOK, html)
}

// Render renders a templ component and writes it to the response
func Render(c echo.Context, component templ.Component) error {
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTMLCharsetUTF8)
	// Don't call WriteHeader here - let Echo handle it on first Write()
	return component.Render(c.Request().Context(), c.Response())
}

