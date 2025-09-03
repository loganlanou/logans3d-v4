package service

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/storage"
)

type Service struct {
	storage *storage.Storage
	config  *Config
}

func New(storage *storage.Storage, config *Config) *Service {
	return &Service{
		storage: storage,
		config:  config,
	}
}

func (s *Service) RegisterRoutes(e *echo.Echo) {
	// Home page
	e.GET("/", s.handleHome)
	
	// Static pages
	e.GET("/about", s.handleAbout)
	e.GET("/events", s.handleEvents)
	e.GET("/contact", s.handleContact)
	e.GET("/portfolio", s.handlePortfolio)
	
	// Legal pages
	e.GET("/privacy", s.handlePrivacy)
	e.GET("/terms", s.handleTerms)
	e.GET("/shipping", s.handleShipping)
	e.GET("/custom-policy", s.handleCustomPolicy)
	
	// Shop routes
	shop := e.Group("/shop")
	shop.GET("", s.handleShop)
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
	
	// Health check
	e.GET("/health", s.handleHealth)
}

// Basic handler implementations
func (s *Service) handleHome(c echo.Context) error {
	slog.Info("Home page requested", "ip", c.RealIP())
	return c.HTML(http.StatusOK, `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Logan's 3D Creations - Welcome</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
        .container { max-width: 800px; margin: 0 auto; background: white; padding: 40px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #333; text-align: center; margin-bottom: 30px; }
        .status { background: #e8f5e8; padding: 20px; border-radius: 6px; border-left: 4px solid #4caf50; margin: 20px 0; }
        .nav { display: flex; gap: 15px; justify-content: center; margin: 30px 0; flex-wrap: wrap; }
        .nav a { background: #007bff; color: white; padding: 10px 20px; text-decoration: none; border-radius: 4px; }
        .nav a:hover { background: #0056b3; }
        .tech-stack { background: #f8f9fa; padding: 20px; border-radius: 6px; margin: 20px 0; }
        .tech-stack h3 { margin-top: 0; color: #495057; }
        .tech-list { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 10px; }
        .tech-item { background: white; padding: 10px; border-radius: 4px; border: 1px solid #dee2e6; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🎯 Logan's 3D Creations v4</h1>
        
        <div class="status">
            <strong>✅ Development Server Running!</strong><br>
            The foundation stack is successfully set up and working.
        </div>
        
        <div class="nav">
            <a href="/">Home</a>
            <a href="/about">About</a>
            <a href="/shop">Shop</a>
            <a href="/custom">Custom Orders</a>
            <a href="/events">Events</a>
            <a href="/contact">Contact</a>
            <a href="/portfolio">Portfolio</a>
        </div>
        
        <div class="tech-stack">
            <h3>🛠️ Technology Stack</h3>
            <div class="tech-list">
                <div class="tech-item"><strong>Backend:</strong> Go 1.25 + Echo v4.13</div>
                <div class="tech-item"><strong>Database:</strong> SQLite + SQLC + Goose</div>
                <div class="tech-item"><strong>Templates:</strong> Templ (type-safe)</div>
                <div class="tech-item"><strong>Frontend:</strong> Alpine.js + Tailwind CSS</div>
                <div class="tech-item"><strong>Development:</strong> Air (hot reload)</div>
                <div class="tech-item"><strong>Testing:</strong> Playwright E2E</div>
                <div class="tech-item"><strong>Deployment:</strong> Vercel</div>
                <div class="tech-item"><strong>Payments:</strong> Stripe</div>
            </div>
        </div>
        
        <p><strong>Next Steps:</strong></p>
        <ul>
            <li>✅ Project structure and configuration</li>
            <li>✅ Database schema and migrations</li>
            <li>✅ Basic web server with routing</li>
            <li>⏳ Frontend tooling (Tailwind CSS, PostCSS)</li>
            <li>⏳ Templ template system</li>
            <li>⏳ Complete welcome page</li>
            <li>⏳ E2E testing setup</li>
        </ul>
        
        <p style="text-align: center; margin-top: 40px; color: #6c757d;">
            <em>Pre-Phase 1 Foundation Setup - Session 1</em><br>
            Environment: Development | Port: 8000
        </p>
    </div>
</body>
</html>
	`)
}

func (s *Service) handleAbout(c echo.Context) error {
	return c.HTML(http.StatusOK, "<h1>About Logan's 3D Creations</h1><p>Coming soon...</p>")
}

func (s *Service) handleShop(c echo.Context) error {
	return c.HTML(http.StatusOK, "<h1>Shop</h1><p>Product catalog coming soon...</p>")
}

func (s *Service) handleProduct(c echo.Context) error {
	slug := c.Param("slug")
	return c.HTML(http.StatusOK, "<h1>Product: "+slug+"</h1><p>Product details coming soon...</p>")
}

func (s *Service) handleCategory(c echo.Context) error {
	slug := c.Param("slug")
	return c.HTML(http.StatusOK, "<h1>Category: "+slug+"</h1><p>Category products coming soon...</p>")
}

func (s *Service) handleCart(c echo.Context) error {
	return c.HTML(http.StatusOK, "<h1>Shopping Cart</h1><p>Cart functionality coming soon...</p>")
}

func (s *Service) handleAddToCart(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "added"})
}

func (s *Service) handleUpdateCart(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Service) handleRemoveFromCart(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "removed"})
}

func (s *Service) handleCustom(c echo.Context) error {
	return c.HTML(http.StatusOK, "<h1>Custom 3D Printing</h1><p>Quote form coming soon...</p>")
}

func (s *Service) handleCustomQuote(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "quote_received"})
}

func (s *Service) handleCheckout(c echo.Context) error {
	return c.HTML(http.StatusOK, "<h1>Checkout</h1><p>Stripe integration coming soon...</p>")
}

func (s *Service) handleProcessCheckout(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "processing"})
}

func (s *Service) handleCheckoutSuccess(c echo.Context) error {
	return c.HTML(http.StatusOK, "<h1>Order Confirmed!</h1><p>Thank you for your purchase.</p>")
}

func (s *Service) handleEvents(c echo.Context) error {
	return c.HTML(http.StatusOK, "<h1>Upcoming Events</h1><p>Event listings coming soon...</p>")
}

func (s *Service) handleContact(c echo.Context) error {
	return c.HTML(http.StatusOK, "<h1>Contact Us</h1><p>Contact form coming soon...</p>")
}

func (s *Service) handlePortfolio(c echo.Context) error {
	return c.HTML(http.StatusOK, "<h1>Portfolio</h1><p>Gallery coming soon...</p>")
}

func (s *Service) handlePrivacy(c echo.Context) error {
	return c.HTML(http.StatusOK, "<h1>Privacy Policy</h1><p>Legal content coming soon...</p>")
}

func (s *Service) handleTerms(c echo.Context) error {
	return c.HTML(http.StatusOK, "<h1>Terms of Service</h1><p>Legal content coming soon...</p>")
}

func (s *Service) handleShipping(c echo.Context) error {
	return c.HTML(http.StatusOK, "<h1>Shipping & Returns</h1><p>Policy content coming soon...</p>")
}

func (s *Service) handleCustomPolicy(c echo.Context) error {
	return c.HTML(http.StatusOK, "<h1>Custom Work Policy</h1><p>Policy content coming soon...</p>")
}

func (s *Service) handleHealth(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":      "healthy",
		"environment": s.config.Environment,
		"database":    "connected",
	})
}