package service

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/storage"
	"github.com/loganlanou/logans3d-v4/storage/db"
	"github.com/loganlanou/logans3d-v4/views/about"
	"github.com/loganlanou/logans3d-v4/views/contact"
	"github.com/loganlanou/logans3d-v4/views/events"
	"github.com/loganlanou/logans3d-v4/views/home"
	"github.com/loganlanou/logans3d-v4/views/legal"
	// "github.com/loganlanou/logans3d-v4/views/portfolio"
	"github.com/loganlanou/logans3d-v4/views/shop"
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
	
	return Render(c, shop.Index(productsWithImages, categories))
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
	
	return Render(c, shop.Index(productsWithImages, categories))
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
	return Render(c, events.Index())
}

func (s *Service) handleContact(c echo.Context) error {
	return Render(c, contact.Index())
}

func (s *Service) handlePortfolio(c echo.Context) error {
	return c.HTML(200, "<h1>Portfolio</h1><p>Coming soon...</p>")
	// return Render(c, portfolio.Index())
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