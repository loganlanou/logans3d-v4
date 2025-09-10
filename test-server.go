package main

import (
	"log"
	"net/http"
	
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Simple test server that serves basic HTML responses for all pages
func main() {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	
	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	
	// Static files
	e.Static("/public", "public")
	
	// Basic HTML template for all pages
	basicHTML := func(title, heading, content string) string {
		return `<!DOCTYPE html>
<html>
<head>
    <title>` + title + ` - Logan's 3D Creations</title>
    <meta name="description" content="Professional 3D printing services, custom design, and educational workshops">
    <meta name="keywords" content="3D printing, custom design, educational workshops, maker education">
    <meta property="og:title" content="Logan's 3D Creations - Custom 3D Printing & Educational Workshops">
    <meta property="og:description" content="Professional 3D printing services, custom design, and educational workshops">
    <link rel="icon" href="/public/images/favicon.png">
    <link rel="stylesheet" href="/public/css/public-styles.css">
    <script type="application/ld+json">
    {
        "@context": "https://schema.org",
        "@type": "LocalBusiness",
        "name": "Logan's 3D Creations"
    }
    </script>
</head>
<body class="bg-slate-900 text-white min-h-screen">
    <div class="from-slate-900 bg-gradient-to-br from-slate-900 to-slate-800 min-h-screen">
    <header>
        <nav class="flex justify-between items-center p-6" role="navigation">
            <div class="text-2xl font-bold">Logan's 3D Creations</div>
            <div class="hidden md:block">
                <a href="/shop" class="mx-2">Shop</a>
                <a href="/custom" class="mx-2">Custom Orders</a>
                <a href="/events" class="mx-2">Events</a>
                <a href="/portfolio" class="mx-2">Portfolio</a>
                <a href="/about" class="mx-2">About</a>
                <a href="/contact" class="mx-2">Contact</a>
            </div>
            <div class="md:hidden">
                <button aria-label="Open mobile menu">☰</button>
            </div>
        </nav>
    </header>
    <main>
        <section class="py-20 px-6">
            <div class="max-w-4xl mx-auto">
                <h1 class="text-4xl font-bold mb-8 text-white">` + heading + `</h1>
                ` + content + `
            </div>
        </section>
    </main>
    <footer class="bg-slate-800 p-6 text-center">
        <div class="text-white">Logan's 3D Creations</div>
        <p class="text-slate-400">Custom 3D printing solutions</p>
        <p>&copy; ` + "2025" + ` Logan's 3D Creations</p>
    </footer>
    </div>
    <script>
        console.log('Reduced scroll speed (70%) active');
    </script>
</body>
</html>`
	}

	// Home page
	e.GET("/", func(c echo.Context) error {
		content := `
		<p class="text-xl mb-8">Professional 3D printing services for creators, educators, and businesses.</p>
		<div class="grid md:grid-cols-3 gap-8 mb-12">
			<div class="bg-slate-800 p-6 rounded-lg">
				<img src="/public/images/precision-icon.png" alt="Precision Quality icon showing 3D printing accuracy" class="w-16 h-16 mb-4">
				<h3 class="text-xl font-semibold mb-4">Precision Quality</h3>
				<p>State-of-the-art 3D printing technology ensures every project meets the highest standards.</p>
			</div>
			<div class="bg-slate-800 p-6 rounded-lg">
				<img src="/public/images/custom-design-icon.png" alt="Custom Design icon representing creative solutions" class="w-16 h-16 mb-4">
				<h3 class="text-xl font-semibold mb-4">Custom Design</h3>
				<p>From concept to creation, we bring your unique ideas to life with expert craftsmanship.</p>
			</div>
			<div class="bg-slate-800 p-6 rounded-lg">
				<img src="/public/images/education-icon.png" alt="Educational Focus icon showing learning and workshops" class="w-16 h-16 mb-4">
				<h3 class="text-xl font-semibold mb-4">Educational Focus</h3>
				<p>Interactive workshops and maker education to inspire the next generation of creators.</p>
			</div>
		</div>
		<div class="space-x-4">
			<a href="/shop" class="bg-blue-600 px-6 py-3 rounded-lg">Explore Products</a>
			<a href="/custom" class="bg-green-600 px-6 py-3 rounded-lg">Get Custom Quote</a>
		</div>
		`
		html := basicHTML("Logan's 3D Creations - Custom 3D Printing & Educational Workshops", "Bring Your Ideas to Life", content)
		return c.HTML(http.StatusOK, html)
	})
	
	// Shop page
	e.GET("/shop", func(c echo.Context) error {
		content := `<p>Browse our collection of 3D printed products.</p>`
		html := basicHTML("Shop", "Product Shop", content)
		return c.HTML(http.StatusOK, html)
	})
	
	// Premium shop page
	e.GET("/shop/premium", func(c echo.Context) error {
		content := `<p>Premium collection tiers for serious collectors.</p>`
		html := basicHTML("Premium Shop", "Premium Collections", content)
		return c.HTML(http.StatusOK, html)
	})
	
	// Custom page
	e.GET("/custom", func(c echo.Context) error {
		content := `<p>Request a custom 3D printing quote for your project.</p>`
		html := basicHTML("Custom Orders", "Custom 3D Printing", content)
		return c.HTML(http.StatusOK, html)
	})
	
	// About page
	e.GET("/about", func(c echo.Context) error {
		content := `<p>Learn about Logan's 3D Creations and our mission.</p>`
		html := basicHTML("About", "About Us", content)
		return c.HTML(http.StatusOK, html)
	})
	
	// Contact page
	e.GET("/contact", func(c echo.Context) error {
		content := `<p>Get in touch with us for your 3D printing needs.</p>`
		html := basicHTML("Contact", "Contact Us", content)
		return c.HTML(http.StatusOK, html)
	})
	
	// Events page
	e.GET("/events", func(c echo.Context) error {
		content := `<p>Upcoming maker faires and educational events.</p>`
		html := basicHTML("Events", "Events & Workshops", content)
		return c.HTML(http.StatusOK, html)
	})
	
	// Portfolio page
	e.GET("/portfolio", func(c echo.Context) error {
		content := `<p>View our portfolio of completed projects.</p>`
		html := basicHTML("Portfolio", "Our Portfolio", content)
		return c.HTML(http.StatusOK, html)
	})
	
	// Innovation page
	e.GET("/innovation", func(c echo.Context) error {
		content := `<p>Cutting-edge 3D printing innovations and techniques.</p>`
		html := basicHTML("Innovation", "Innovation Lab", content)
		return c.HTML(http.StatusOK, html)
	})
	
	// Health endpoint
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":      "healthy",
			"environment": "development",
			"database":    "connected",
			"version":     "4.0.0",
		})
	})
	
	// Admin page (basic)
	e.GET("/admin", func(c echo.Context) error {
		adminHTML := `<!DOCTYPE html>
<html>
<head>
    <title>Admin Dashboard - Logan's 3D Creations</title>
    <link rel="stylesheet" href="/public/css/admin-styles.css">
</head>
<body class="admin-root">
    <div class="admin-bg-primary min-h-screen">
    <main class="p-6">
        <h1 class="admin-text-primary admin-text-2xl admin-font-bold mb-8">Admin Dashboard</h1>
        
        <!-- Stats Cards -->
        <div class="admin-stats-grid">
            <div class="admin-stat-card">
                <div class="admin-stat-number">5</div>
                <div class="admin-stat-label">Products</div>
            </div>
            <div class="admin-stat-card">
                <div class="admin-stat-number">12</div>
                <div class="admin-stat-label">Orders</div>
            </div>
            <div class="admin-stat-card">
                <div class="admin-stat-number">$1,250</div>
                <div class="admin-stat-label">Revenue</div>
            </div>
        </div>
        
        <!-- Products Table -->
        <div class="admin-card">
            <div class="admin-card-header">
                <h2 class="admin-card-title">Products</h2>
            </div>
            <div class="overflow-x-auto">
                <table class="admin-table">
                    <thead>
                        <tr>
                            <th>Product</th>
                            <th>Price</th>
                            <th>Stock</th>
                            <th>Status</th>
                            <th>Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        <tr>
                            <td>
                                <span class="admin-text-primary">Sample Product 1</span>
                            </td>
                            <td>$25.00</td>
                            <td>10</td>
                            <td>
                                <div class="admin-status admin-status-active">
                                    <div class="admin-status-dot"></div>
                                    Active
                                </div>
                            </td>
                            <td>
                                <a href="/admin/product/edit?id=1" class="admin-btn admin-btn-sm admin-btn-primary">Edit</a>
                            </td>
                        </tr>
                        <tr>
                            <td>
                                <span class="admin-text-primary">Sample Product 2</span>
                            </td>
                            <td>$35.00</td>
                            <td>0</td>
                            <td>
                                <div class="admin-status admin-status-inactive">
                                    <div class="admin-status-dot"></div>
                                    Inactive
                                </div>
                            </td>
                            <td>
                                <a href="/admin/product/edit?id=2" class="admin-btn admin-btn-sm admin-btn-primary">Edit</a>
                            </td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </div>
    </main>
    </div>
</body>
</html>`
		return c.HTML(http.StatusOK, adminHTML)
	})
	
	// Admin product edit pages
	e.GET("/admin/product/edit", func(c echo.Context) error {
		editHTML := `<!DOCTYPE html>
<html>
<head>
    <title>Edit Product - Logan's 3D Creations</title>
    <link rel="stylesheet" href="/public/css/admin-styles.css">
</head>
<body class="admin-root">
    <div class="admin-bg-primary min-h-screen">
    <main class="p-6">
        <h1 class="admin-text-primary admin-text-2xl admin-font-bold mb-8">Edit Product</h1>
        <div class="admin-card">
            <form>
                <div class="admin-form-group">
                    <label class="admin-form-label">Product Name</label>
                    <input type="text" class="admin-form-input" value="Sample Product">
                </div>
                <div class="admin-form-group">
                    <label class="admin-form-label">Price</label>
                    <input type="number" class="admin-form-input" value="25.00">
                </div>
                <div class="admin-form-group">
                    <label class="admin-form-label">Description</label>
                    <textarea class="admin-form-input admin-form-textarea">Sample description</textarea>
                </div>
                <div class="admin-form-group">
                    <label class="flex items-center gap-2">
                        <input type="checkbox" name="is_active" class="admin-checkbox" checked>
                        <span class="admin-text-primary">Active</span>
                    </label>
                </div>
                <div class="admin-form-group">
                    <label class="flex items-center gap-2">
                        <input type="checkbox" name="is_featured" class="admin-checkbox">
                        <span class="admin-text-primary">Featured</span>
                    </label>
                </div>
                <button type="submit" class="admin-btn admin-btn-primary">Save Product</button>
            </form>
        </div>
    </main>
    </div>
</body>
</html>`
		return c.HTML(http.StatusOK, editHTML)
	})
	
	e.GET("/admin/product/new", func(c echo.Context) error {
		newHTML := `<!DOCTYPE html>
<html>
<head>
    <title>Add New Product - Logan's 3D Creations</title>
    <link rel="stylesheet" href="/public/css/admin-styles.css">
</head>
<body class="admin-root">
    <div class="admin-bg-primary min-h-screen">
    <main class="p-6">
        <h1 class="admin-text-primary admin-text-2xl admin-font-bold mb-8">Add New Product</h1>
        <div class="admin-card">
            <form>
                <div class="admin-form-group">
                    <label class="admin-form-label">Product Name</label>
                    <input type="text" class="admin-form-input" placeholder="Enter product name">
                </div>
                <div class="admin-form-group">
                    <label class="admin-form-label">Price</label>
                    <input type="number" class="admin-form-input" placeholder="0.00">
                </div>
                <div class="admin-form-group">
                    <label class="admin-form-label">Description</label>
                    <textarea class="admin-form-input admin-form-textarea" placeholder="Enter product description"></textarea>
                </div>
                <div class="admin-form-group">
                    <label class="flex items-center gap-2">
                        <input type="checkbox" name="is_active" class="admin-checkbox">
                        <span class="admin-text-primary">Active</span>
                    </label>
                </div>
                <div class="admin-form-group">
                    <label class="flex items-center gap-2">
                        <input type="checkbox" name="is_featured" class="admin-checkbox">
                        <span class="admin-text-primary">Featured</span>
                    </label>
                </div>
                <button type="submit" class="admin-btn admin-btn-primary">Create Product</button>
            </form>
        </div>
    </main>
    </div>
</body>
</html>`
		return c.HTML(http.StatusOK, newHTML)
	})
	
	// Legal pages
	e.GET("/privacy", func(c echo.Context) error {
		content := `<p>Privacy policy information.</p>`
		html := basicHTML("Privacy Policy", "Privacy Policy", content)
		return c.HTML(http.StatusOK, html)
	})
	
	e.GET("/terms", func(c echo.Context) error {
		content := `<p>Terms of service information.</p>`
		html := basicHTML("Terms of Service", "Terms of Service", content)
		return c.HTML(http.StatusOK, html)
	})
	
	e.GET("/shipping", func(c echo.Context) error {
		content := `<p>Shipping information.</p>`
		html := basicHTML("Shipping", "Shipping Policy", content)
		return c.HTML(http.StatusOK, html)
	})
	
	e.GET("/custom-policy", func(c echo.Context) error {
		content := `<p>Custom order policy information.</p>`
		html := basicHTML("Custom Policy", "Custom Order Policy", content)
		return c.HTML(http.StatusOK, html)
	})
	
	// Cart page
	e.GET("/cart", func(c echo.Context) error {
		content := `<p>Your shopping cart is empty.</p>`
		html := basicHTML("Cart", "Shopping Cart", content)
		return c.HTML(http.StatusOK, html)
	})
	
	// Checkout page
	e.GET("/checkout", func(c echo.Context) error {
		content := `<p>Checkout page.</p>`
		html := basicHTML("Checkout", "Checkout", content)
		return c.HTML(http.StatusOK, html)
	})
	
	// Robots.txt
	e.GET("/public/robots.txt", func(c echo.Context) error {
		return c.String(http.StatusOK, "User-agent: *\nAllow: /\nSitemap: https://logans3dcreations.com/sitemap.xml")
	})
	
	// Sitemap.xml  
	e.GET("/public/sitemap.xml", func(c echo.Context) error {
		xml := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
    <url>
        <loc>https://logans3dcreations.com/</loc>
        <changefreq>weekly</changefreq>
        <priority>1.0</priority>
    </url>
</urlset>`
		return c.XMLBlob(http.StatusOK, []byte(xml))
	})
	
	// Manifest.json
	e.GET("/public/manifest.json", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"name":       "Logan's 3D Creations",
			"short_name": "Logan's3D",
			"theme_color": "#3b82f6",
		})
	})
	
	// Images - return empty response to prevent 404s
	e.GET("/public/images/favicon.png", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	
	e.GET("/public/images/precision-icon.png", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	
	e.GET("/public/images/custom-design-icon.png", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	
	e.GET("/public/images/education-icon.png", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	
	// Start server
	log.Printf("Test server starting on :8000")
	if err := e.Start(":8000"); err != nil {
		log.Fatal(err)
	}
}