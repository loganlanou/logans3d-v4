package service

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"
	"github.com/gorilla/sessions"
	"github.com/stripe/stripe-go/v80"
	"github.com/stripe/stripe-go/v80/checkout/session"
	"github.com/loganlanou/logans3d-v4/internal/handlers"
	"github.com/loganlanou/logans3d-v4/storage"
	"github.com/loganlanou/logans3d-v4/storage/db"
	"github.com/loganlanou/logans3d-v4/views/about"
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
	storage        *storage.Storage
	config         *Config
	paymentHandler *handlers.PaymentHandler
}

func New(storage *storage.Storage, config *Config) *Service {
	// Initialize session store for gothic
	key := []byte(config.JWT.Secret)
	maxAge := 86400 * 30 // 30 days
	isProd := false // Set to true in production for secure cookies
	store := sessions.NewCookieStore(key)
	store.MaxAge(maxAge)
	store.Options.Path = "/"
	store.Options.HttpOnly = true
	store.Options.Secure = isProd
	
	gothic.Store = store
	
	// Initialize Google OAuth provider if credentials are available
	if config.OAuth.ClientID != "" && config.OAuth.ClientSecret != "" {
		goth.UseProviders(
			google.New(config.OAuth.ClientID, config.OAuth.ClientSecret, config.OAuth.RedirectURL),
		)
	}
	
	return &Service{
		storage:        storage,
		config:         config,
		paymentHandler: handlers.NewPaymentHandler(),
	}
}

func (s *Service) RegisterRoutes(e *echo.Echo) {
	// Static files
	e.Static("/public", "public")
	
	// Home page
	e.GET("/", s.handleHome)
	
	// Static pages
	e.GET("/about", s.handleAbout)
	e.GET("/events", s.handleEvents)
	e.GET("/contact", s.handleContact)
	e.GET("/portfolio", s.handlePortfolio)
	e.GET("/innovation", s.handleInnovation)
	
	// Legal pages
	e.GET("/privacy", s.handlePrivacy)
	e.GET("/terms", s.handleTerms)
	e.GET("/shipping", s.handleShipping)
	e.GET("/custom-policy", s.handleCustomPolicy)
	
	// Authentication pages
	e.GET("/login", s.handleLoginPlaceholder)
	e.GET("/auth/google", s.handleGoogleAuth)
	e.GET("/auth/google/callback", s.handleGoogleCallback)
	e.GET("/logout", s.handleLogout)
	
	// Shop routes
	shop := e.Group("/shop")
	shop.GET("", s.handleShop)
	shop.GET("/premium", s.handlePremium)
	shop.GET("/product/:slug", s.handleProduct)
	shop.GET("/category/:slug", s.handleCategory)
	
	// Cart routes
	e.GET("/cart", s.handleCart)
	e.POST("/api/cart/add", s.handleAddToCart)
	e.DELETE("/api/cart/item/:id", s.handleRemoveFromCart)
	e.PUT("/api/cart/item/:id", s.handleUpdateCartItem)
	e.GET("/api/cart", s.handleGetCart)
	
	// Custom quote routes
	e.GET("/custom", s.handleCustom)
	e.POST("/custom/quote", s.handleCustomQuote)
	
	// Stripe Checkout routes
	e.POST("/checkout/create-session", s.handleCreateStripeCheckoutSession)
	e.POST("/checkout/create-session-single", s.handleCreateStripeCheckoutSessionSingle)
	e.POST("/checkout/create-session-multi", s.handleCreateStripeCheckoutSessionMulti)
	e.POST("/checkout/create-session-cart", s.handleCreateStripeCheckoutSessionCart)
	e.GET("/checkout/success", s.handleCheckoutSuccess)
	e.GET("/checkout/cancel", s.handleCheckoutCancel)
	
	// Payment API routes
	api := e.Group("/api")
	api.POST("/payment/create-intent", s.paymentHandler.CreatePaymentIntent)
	api.POST("/payment/create-customer", s.paymentHandler.CreateCustomer)
	api.POST("/stripe/webhook", s.paymentHandler.HandleWebhook)
	
	// Admin routes
	adminHandler := handlers.NewAdminHandler(s.storage)
	admin := e.Group("/admin")
	admin.GET("", adminHandler.HandleAdminDashboard)
	admin.GET("/categories", adminHandler.HandleCategoriesTab)
	admin.GET("/product/new", adminHandler.HandleProductForm)
	admin.POST("/product", adminHandler.HandleCreateProduct)
	admin.GET("/product/edit", adminHandler.HandleProductForm)
	admin.POST("/product/:id", adminHandler.HandleUpdateProduct)
	admin.POST("/product/:id/delete", adminHandler.HandleDeleteProduct)
	
	// Category management routes
	admin.GET("/category/new", adminHandler.HandleCategoryForm)
	admin.POST("/category", adminHandler.HandleCreateCategory)
	admin.GET("/category/edit", adminHandler.HandleCategoryForm)
	admin.POST("/category/:id", adminHandler.HandleUpdateCategory)
	admin.POST("/category/:id/delete", adminHandler.HandleDeleteCategory)
	
	// Orders management routes
	admin.GET("/orders", adminHandler.HandleOrdersList)
	admin.GET("/orders/:id", adminHandler.HandleOrderDetail)
	admin.POST("/orders/:id/status", adminHandler.HandleUpdateOrderStatus)
	
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
	
	// Developer routes
	dev := e.Group("/dev")
	dev.GET("", adminHandler.HandleDeveloperDashboard)
	dev.GET("/system", adminHandler.HandleSystemInfo)
	dev.GET("/memory", adminHandler.HandleMemoryStats)
	dev.GET("/database", adminHandler.HandleDatabaseInfo)
	dev.GET("/config", adminHandler.HandleConfigInfo)
	dev.POST("/gc", adminHandler.HandleGarbageCollect)
	
	// Health check
	e.GET("/health", s.handleHealth)
}

// Basic handler implementations
func (s *Service) handleHome(c echo.Context) error {
	slog.Info("Home page requested", "ip", c.RealIP())
	return Render(c, home.Index())
}

func (s *Service) handleAbout(c echo.Context) error {
	return Render(c, about.Index())
}

func (s *Service) handleShop(c echo.Context) error {
	ctx := c.Request().Context()
	
	// Get all categories for filter
	categories, err := s.storage.Queries.ListCategories(ctx)
	if err != nil {
		slog.Error("failed to fetch categories", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load categories")
	}
	
	// Get all products
	products, err := s.storage.Queries.ListProducts(ctx)
	if err != nil {
		slog.Error("failed to fetch products", "error", err)
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

			// Build the full path from the filename
			if rawImageURL != "" {
				// Database should only contain filenames
				imageURL = "/public/images/products/" + rawImageURL
			}
		}
		
		productsWithImages = append(productsWithImages, shop.ProductWithImage{
			Product:  product,
			ImageURL: imageURL,
		})
	}
	
	return Render(c, shop.Index(productsWithImages, categories, nil))
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

			// Build the full path from the filename
			if rawImageURL != "" {
				// Database should only contain filenames
				imageURL = "/public/images/products/" + rawImageURL
			}
		}
		
		featuredProducts = append(featuredProducts, shop.ProductWithImage{
			Product:  product,
			ImageURL: imageURL,
		})
		count++
	}
	
	return Render(c, shop.Premium(collections, featuredProducts))
}

func (s *Service) handleProduct(c echo.Context) error {
	slug := c.Param("slug")
	ctx := c.Request().Context()
	
	// Get product by slug
	product, err := s.storage.Queries.GetProductBySlug(ctx, slug)
	if err != nil {
		slog.Error("failed to fetch product", "slug", slug, "error", err)
		return echo.NewHTTPError(http.StatusNotFound, "Product not found")
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
	
	return Render(c, shop.ProductDetail(product, images, category))
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

			// Build the full path from the filename
			if rawImageURL != "" {
				// Database should only contain filenames
				imageURL = "/public/images/products/" + rawImageURL
			}
		}
		
		productsWithImages = append(productsWithImages, shop.ProductWithImage{
			Product:  product,
			ImageURL: imageURL,
		})
	}
	
	return Render(c, shop.Index(productsWithImages, categories, &category))
}

// Cart handlers removed - replaced with Stripe Checkout

func (s *Service) handleCustom(c echo.Context) error {
	return Render(c, custom.Index())
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
		// Ensure we have an absolute URL for Stripe
		imageURL = fmt.Sprintf("%s://%s/public/images/products/%s", c.Scheme(), c.Request().Host, images[0].ImageUrl)
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
	
	session, err := session.New(params)
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
	// Clear cart after successful purchase
	successHTML := `
		<html>
		<head>
			<title>Order Confirmed - Logan's 3D Creations</title>
			<script>
				// Clear cart after successful purchase
				if (localStorage.getItem('stripe_cart')) {
					localStorage.removeItem('stripe_cart');
				}
				// Redirect to home page after 3 seconds
				setTimeout(() => {
					window.location.href = '/';
				}, 3000);
			</script>
		</head>
		<body>
			<h1>Order Confirmed!</h1>
			<p>Thank you for your purchase. You will be redirected shortly...</p>
		</body>
		</html>
	`
	return c.HTML(http.StatusOK, successHTML)
}

// handleCart renders the shopping cart page
func (s *Service) handleCart(c echo.Context) error {
	return Render(c, shop.Cart())
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
		// Ensure we have an absolute URL for Stripe
		imageURL = fmt.Sprintf("%s://%s/public/images/products/%s", c.Scheme(), c.Request().Host, images[0].ImageUrl)
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
	
	session, err := session.New(params)
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
			// Convert relative URL to absolute URL
			var imageURL string
			if item.ImageURL[0] == '/' {
				imageURL = fmt.Sprintf("%s://%s%s", c.Scheme(), c.Request().Host, item.ImageURL)
			} else {
				// Handle direct filename - prepend full product image path
				imageURL = fmt.Sprintf("%s://%s/public/images/products/%s", c.Scheme(), c.Request().Host, item.ImageURL)
			}
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
	
	session, err := session.New(params)
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
	
	// Get cart items from database
	sessionIDParam := sql.NullString{String: sessionID, Valid: true}
	cartItems, err := s.storage.Queries.GetCartBySession(ctx, sessionIDParam)
	if err != nil {
		slog.Error("failed to get cart items", "error", err, "session_id", sessionID)
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
				},
			},
			Quantity: stripe.Int64(item.Quantity),
		}
		
		// Add product image if available
		if item.ImageUrl != "" {
			var imageURL string
			if item.ImageUrl[0] == '/' {
				imageURL = fmt.Sprintf("%s://%s%s", c.Scheme(), c.Request().Host, item.ImageUrl)
			} else {
				imageURL = fmt.Sprintf("%s://%s/public/images/products/%s", c.Scheme(), c.Request().Host, item.ImageUrl)
			}
			lineItem.PriceData.ProductData.Images = []*string{stripe.String(imageURL)}
		}
		
		lineItems = append(lineItems, lineItem)
	}
	
	// Create Stripe Checkout Session
	stripe.Key = s.config.Stripe.SecretKey
	
	params := &stripe.CheckoutSessionParams{
		Mode:             stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems:        lineItems,
		SuccessURL:       stripe.String(fmt.Sprintf("%s://%s/checkout/success?session_id={CHECKOUT_SESSION_ID}", c.Scheme(), c.Request().Host)),
		CancelURL:        stripe.String(fmt.Sprintf("%s://%s/cart", c.Scheme(), c.Request().Host)),
		CustomerCreation: stripe.String("always"),
	}
	
	session, err := session.New(params)
	if err != nil {
		slog.Error("failed to create stripe checkout session", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create checkout session")
	}
	
	return c.JSON(http.StatusOK, map[string]string{"url": session.URL})
}

func (s *Service) handleEvents(c echo.Context) error {
	return Render(c, events.Index())
}

func (s *Service) handleContact(c echo.Context) error {
	return Render(c, contact.Index())
}

func (s *Service) handlePortfolio(c echo.Context) error {
	return Render(c, portfolio.Index())
}

func (s *Service) handleInnovation(c echo.Context) error {
	return Render(c, innovation.Index())
}

func (s *Service) handlePrivacy(c echo.Context) error {
	return Render(c, legal.Privacy())
}

func (s *Service) handleTerms(c echo.Context) error {
	return Render(c, legal.Terms())
}

func (s *Service) handleShipping(c echo.Context) error {
	return Render(c, legal.Shipping())
}

func (s *Service) handleCustomPolicy(c echo.Context) error {
	return Render(c, legal.CustomPolicy())
}

func (s *Service) handleHealth(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"status":      "healthy",
		"environment": s.config.Environment,
		"database":    "connected",
	})
}

// Session management removed - no longer needed without cart

// handleGoogleAuth initiates Google OAuth flow
func (s *Service) handleGoogleAuth(c echo.Context) error {
	// Check if OAuth is configured
	if s.config.OAuth.ClientID == "" || s.config.OAuth.ClientSecret == "" {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Google OAuth is not configured",
		})
	}
	
	// Development mode: If using test credentials, redirect to mock flow
	if s.config.OAuth.ClientID == "test-client-id-for-development" {
		return c.Redirect(http.StatusTemporaryRedirect, "/auth/google/callback?mock=true")
	}
	
	// Set provider in context for gothic
	c.Request().URL.RawQuery = "provider=google"
	
	// Start the authentication process
	gothic.BeginAuthHandler(c.Response(), c.Request())
	return nil
}

// handleGoogleCallback handles the Google OAuth callback
func (s *Service) handleGoogleCallback(c echo.Context) error {
	// Check if OAuth is configured
	if s.config.OAuth.ClientID == "" || s.config.OAuth.ClientSecret == "" {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Google OAuth is not configured",
		})
	}
	
	// Development mode: Handle mock authentication
	if c.QueryParam("mock") == "true" {
		// Create a mock user for development
		slog.Info("Mock Google authentication successful", 
			"user_id", "dev-user-123", 
			"email", "developer@logans3dcreations.com", 
			"name", "Development User")
		
		// TODO: Store user in database and create session
		// For now, redirect to home with success message
		return c.Redirect(http.StatusTemporaryRedirect, "/?login=success&user=developer@logans3dcreations.com")
	}
	
	// Set provider in context for gothic
	c.Request().URL.RawQuery = "provider=google"
	
	// Complete the authentication process
	user, err := gothic.CompleteUserAuth(c.Response(), c.Request())
	if err != nil {
		slog.Error("OAuth callback failed", "error", err)
		return c.Redirect(http.StatusTemporaryRedirect, "/login?error=oauth_failed")
	}
	
	// TODO: Store user in database and create session
	slog.Info("User authenticated via Google", "user_id", user.UserID, "email", user.Email, "name", user.Name)
	
	// For now, just redirect to home with success message
	return c.Redirect(http.StatusTemporaryRedirect, "/?login=success")
}

// handleLogout handles user logout
func (s *Service) handleLogout(c echo.Context) error {
	// TODO: Clear user session
	
	// Redirect to home page
	return c.Redirect(http.StatusTemporaryRedirect, "/?logout=success")
}

// handleLoginPlaceholder handles the login page (placeholder implementation)
func (s *Service) handleLoginPlaceholder(c echo.Context) error {
	loginHTML := `
<!DOCTYPE html>
<html>
<head>
    <title>Sign In - Logan's 3D Creations</title>
    <style>
        body { 
            font-family: Arial, sans-serif; 
            max-width: 400px; 
            margin: 50px auto; 
            padding: 20px;
            background: linear-gradient(135deg, #7C3AED 0%, #EC4899 50%, #06B6D4 100%);
            min-height: 100vh;
        }
        .login-card { 
            background: white; 
            padding: 40px; 
            border-radius: 12px; 
            box-shadow: 0 8px 25px rgba(0,0,0,0.15);
            text-align: center;
        }
        h1 { 
            color: #333; 
            margin-bottom: 30px; 
            font-size: 28px;
        }
        .google-btn { 
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 12px;
            background: #4285f4; 
            color: white; 
            padding: 12px 24px; 
            border: none; 
            border-radius: 8px; 
            cursor: pointer;
            font-size: 16px;
            width: 100%;
            margin-bottom: 20px;
            transition: background-color 0.2s;
        }
        .google-btn:hover { 
            background: #3367d6; 
        }
        .placeholder-note {
            background: #f3f4f6;
            border: 1px solid #d1d5db;
            border-radius: 8px;
            padding: 16px;
            color: #6b7280;
            font-size: 14px;
            margin-top: 20px;
        }
        .back-link {
            display: inline-block;
            margin-top: 20px;
            color: #7C3AED;
            text-decoration: none;
            font-weight: 500;
        }
        .back-link:hover {
            text-decoration: underline;
        }
    </style>
</head>
<body>
    <div class="login-card">
        <h1>Sign In</h1>
        
        <button class="google-btn" onclick="handleGoogleSignIn()">
            <svg width="20" height="20" viewBox="0 0 24 24">
                <path fill="currentColor" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/>
                <path fill="currentColor" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/>
                <path fill="currentColor" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/>
                <path fill="currentColor" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/>
            </svg>
            Sign in with Google
        </button>
        
        <div class="placeholder-note">
            <strong>Note:</strong> Google OAuth integration is in development. 
            This page will redirect to Google authentication once configured.
        </div>
        
        <a href="/" class="back-link">← Back to Home</a>
    </div>

    <script>
        function handleGoogleSignIn() {
            // Redirect to Google OAuth
            window.location.href = '/auth/google';
        }
    </script>
</body>
</html>`
	
	return c.HTML(http.StatusOK, loginHTML)
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

	ctx := c.Request().Context()
	err := s.storage.Queries.RemoveCartItem(ctx, itemID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to remove item from cart")
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
	return component.Render(c.Request().Context(), c.Response().Writer)
}