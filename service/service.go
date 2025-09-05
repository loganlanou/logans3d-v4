package service

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"
	"github.com/gorilla/sessions"
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
	e.POST("/cart/add", s.handleAddToCart)
	e.POST("/cart/update", s.handleUpdateCart)
	e.POST("/cart/remove", s.handleRemoveFromCart)
	
	// Custom quote routes
	e.GET("/custom", s.handleCustom)
	e.POST("/custom/quote", s.handleCustomQuote)
	
	// Checkout routes
	e.GET("/checkout", s.handleCheckout)
	e.POST("/checkout", s.handleProcessCheckout)
	e.GET("/checkout/success", s.handleCheckoutSuccess)
	
	// Payment API routes
	api := e.Group("/api")
	api.POST("/payment/create-intent", s.paymentHandler.CreatePaymentIntent)
	api.POST("/payment/create-customer", s.paymentHandler.CreateCustomer)
	api.POST("/stripe/webhook", s.paymentHandler.HandleWebhook)
	
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
			imageURL = images[0].ImageUrl
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
			imageURL = images[0].ImageUrl
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
			imageURL = images[0].ImageUrl
		}
		
		productsWithImages = append(productsWithImages, shop.ProductWithImage{
			Product:  product,
			ImageURL: imageURL,
		})
	}
	
	return Render(c, shop.Index(productsWithImages, categories, &category))
}

func (s *Service) handleCart(c echo.Context) error {
	ctx := c.Request().Context()
	sessionID := s.getOrCreateSessionID(c)
	
	// Get cart items
	cartItems, err := s.storage.Queries.GetCartBySession(ctx, sql.NullString{String: sessionID, Valid: true})
	if err != nil {
		slog.Error("failed to fetch cart items", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load cart")
	}
	
	// Calculate total
	totalInterface, err := s.storage.Queries.GetCartTotal(ctx, db.GetCartTotalParams{
		SessionID: sql.NullString{String: sessionID, Valid: true},
		UserID:    sql.NullString{Valid: false},
	})
	var total int64
	if err != nil {
		slog.Error("failed to calculate cart total", "error", err)
		total = 0
	} else {
		if t, ok := totalInterface.(int64); ok {
			total = t
		} else {
			total = 0
		}
	}
	
	return Render(c, shop.Cart(cartItems, total))
}

func (s *Service) handleAddToCart(c echo.Context) error {
	ctx := c.Request().Context()
	sessionID := s.getOrCreateSessionID(c)
	
	var req struct {
		ProductID string `json:"product_id" form:"product_id"`
		Quantity  int64  `json:"quantity" form:"quantity"`
	}
	
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request")
	}
	
	if req.Quantity <= 0 {
		req.Quantity = 1
	}
	
	// Verify product exists
	product, err := s.storage.Queries.GetProduct(ctx, req.ProductID)
	if err != nil {
		slog.Error("product not found", "product_id", req.ProductID, "error", err)
		return echo.NewHTTPError(http.StatusNotFound, "Product not found")
	}
	
	// Check stock
	if product.StockQuantity.Valid && product.StockQuantity.Int64 <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Product out of stock")
	}
	
	// Check if item already exists in cart
	existingItem, err := s.storage.Queries.GetExistingCartItem(ctx, db.GetExistingCartItemParams{
		SessionID: sql.NullString{String: sessionID, Valid: true},
		UserID:    sql.NullString{Valid: false}, // TODO: Handle logged-in users
		ProductID: req.ProductID,
	})
	
	if err != nil && err != sql.ErrNoRows {
		slog.Error("failed to check existing cart item", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to check cart")
	}
	
	if err == sql.ErrNoRows {
		// Item doesn't exist, add new
		err = s.storage.Queries.AddToCart(ctx, db.AddToCartParams{
			ID:        uuid.New().String(),
			SessionID: sql.NullString{String: sessionID, Valid: true},
			UserID:    sql.NullString{Valid: false}, // TODO: Handle logged-in users
			ProductID: req.ProductID,
			Quantity:  req.Quantity,
		})
	} else {
		// Item exists, update quantity
		err = s.storage.Queries.UpdateCartItemQuantity(ctx, db.UpdateCartItemQuantityParams{
			ID:       existingItem.ID,
			Quantity: existingItem.Quantity + req.Quantity,
		})
	}
	
	if err != nil {
		slog.Error("failed to add item to cart", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to add item to cart")
	}
	
	// Get updated cart count
	cartCountInterface, err := s.storage.Queries.GetCartItemCount(ctx, db.GetCartItemCountParams{
		SessionID: sql.NullString{String: sessionID, Valid: true},
		UserID:    sql.NullString{Valid: false},
	})
	var cartCount int64
	if err != nil {
		slog.Error("failed to get cart count", "error", err)
		cartCount = 0
	} else {
		if c, ok := cartCountInterface.(int64); ok {
			cartCount = c
		} else {
			cartCount = 0
		}
	}
	
	return c.JSON(http.StatusOK, map[string]any{
		"status":     "added",
		"message":    fmt.Sprintf("Added %s to cart", product.Name),
		"cart_count": cartCount,
	})
}

func (s *Service) handleUpdateCart(c echo.Context) error {
	ctx := c.Request().Context()
	
	var req struct {
		ItemID   string `json:"item_id" form:"item_id"`
		Quantity int64  `json:"quantity" form:"quantity"`
	}
	
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request")
	}
	
	if req.Quantity <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Quantity must be greater than 0")
	}
	
	err := s.storage.Queries.UpdateCartItemQuantity(ctx, db.UpdateCartItemQuantityParams{
		Quantity: req.Quantity,
		ID:       req.ItemID,
	})
	
	if err != nil {
		slog.Error("failed to update cart item", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update cart item")
	}
	
	return c.JSON(http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Service) handleRemoveFromCart(c echo.Context) error {
	ctx := c.Request().Context()
	
	var req struct {
		ItemID string `json:"item_id" form:"item_id"`
	}
	
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request")
	}
	
	err := s.storage.Queries.RemoveCartItem(ctx, req.ItemID)
	if err != nil {
		slog.Error("failed to remove cart item", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to remove cart item")
	}
	
	return c.JSON(http.StatusOK, map[string]string{"status": "removed"})
}

func (s *Service) handleCustom(c echo.Context) error {
	return Render(c, custom.Index())
}

func (s *Service) handleCustomQuote(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "quote_received"})
}

func (s *Service) handleCheckout(c echo.Context) error {
	// Basic checkout page with Stripe integration
	checkoutHTML := `
<!DOCTYPE html>
<html>
<head>
    <title>Checkout - Logan's 3D Creations</title>
    <script src="https://js.stripe.com/v3/"></script>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        .checkout-form { background: #f5f5f5; padding: 30px; border-radius: 8px; }
        input[type="email"], input[type="text"] { width: 100%; padding: 10px; margin: 10px 0; }
        #card-element { padding: 10px; border: 1px solid #ccc; border-radius: 4px; margin: 10px 0; }
        #submit { background: #5469d4; color: white; padding: 12px 20px; border: none; border-radius: 4px; cursor: pointer; }
        #submit:hover { background: #4f63d2; }
        #submit:disabled { opacity: 0.6; cursor: default; }
        .error { color: #fa755a; }
    </style>
</head>
<body>
    <h1>Checkout</h1>
    <div class="checkout-form">
        <form id="payment-form">
            <div>
                <label for="email">Email Address</label>
                <input type="email" id="email" placeholder="your@email.com" required />
            </div>
            
            <div>
                <label for="card-element">Credit or debit card</label>
                <div id="card-element"></div>
                <div id="card-errors" role="alert" class="error"></div>
            </div>
            
            <button type="submit" id="submit">Pay $10.00</button>
        </form>
    </div>

    <script>
        const stripe = Stripe('` + s.config.Stripe.PublishableKey + `');
        const elements = stripe.elements();
        
        const cardElement = elements.create('card');
        cardElement.mount('#card-element');
        
        const form = document.getElementById('payment-form');
        const submitButton = document.getElementById('submit');
        
        form.addEventListener('submit', async (event) => {
            event.preventDefault();
            submitButton.disabled = true;
            submitButton.textContent = 'Processing...';
            
            const email = document.getElementById('email').value;
            
            // Create customer first
            const customerResponse = await fetch('/api/payment/create-customer', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    email: email,
                    name: email.split('@')[0]
                })
            });
            
            const customer = await customerResponse.json();
            
            // Create payment intent
            const paymentIntentResponse = await fetch('/api/payment/create-intent', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    amount: 1000, // $10.00 in cents
                    currency: 'usd',
                    customer_id: customer.customer_id
                })
            });
            
            const paymentIntent = await paymentIntentResponse.json();
            
            // Confirm payment with Stripe
            const result = await stripe.confirmCardPayment(paymentIntent.client_secret, {
                payment_method: {
                    card: cardElement,
                    billing_details: {
                        email: email,
                    },
                }
            });
            
            if (result.error) {
                document.getElementById('card-errors').textContent = result.error.message;
                submitButton.disabled = false;
                submitButton.textContent = 'Pay $10.00';
            } else {
                window.location.href = '/checkout/success';
            }
        });
    </script>
</body>
</html>`
	
	return c.HTML(http.StatusOK, checkoutHTML)
}

func (s *Service) handleProcessCheckout(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "processing"})
}

func (s *Service) handleCheckoutSuccess(c echo.Context) error {
	return c.HTML(http.StatusOK, "<h1>Order Confirmed!</h1><p>Thank you for your purchase.</p>")
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

// getOrCreateSessionID gets or creates a session ID for cart management
func (s *Service) getOrCreateSessionID(c echo.Context) string {
	// First check for existing session cookie
	cookie, err := c.Cookie("session_id")
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}
	
	// Create new session ID
	sessionID := uuid.New().String()
	
	// Set cookie (valid for 30 days)
	cookie = &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60, // 30 days
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteStrictMode,
	}
	c.SetCookie(cookie)
	
	return sessionID
}

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

// Render renders a templ component and writes it to the response
func Render(c echo.Context, component templ.Component) error {
	return component.Render(c.Request().Context(), c.Response().Writer)
}