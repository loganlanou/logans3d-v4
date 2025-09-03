package service

import (
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/storage"
	"github.com/loganlanou/logans3d-v4/views/home"
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
	// Static files
	e.Static("/public", "public")
	
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
	return Render(c, home.Index())
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

// Render renders a templ component and writes it to the response
func Render(c echo.Context, component templ.Component) error {
	return component.Render(c.Request().Context(), c.Response().Writer)
}