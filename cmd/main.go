package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/loganlanou/logans3d-v4/service"
	"github.com/loganlanou/logans3d-v4/storage"
	"github.com/loganlanou/logans3d-v4/views/errors"
)

func main() {
	// slog is configured in slog.go via init()

	// Validate required environment variables
	validateRequiredEnvVars()

	// Load configuration
	config, err := service.LoadConfig()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Initialize database
	db, err := storage.New(config.DBPath)
	if err != nil {
		slog.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Initialize Echo
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Custom error handler for 404 and other errors
	e.HTTPErrorHandler = customHTTPErrorHandler

	// Middleware
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: `{"time":"${time_rfc3339}","id":"${id}","remote_ip":"${remote_ip}","host":"${host}","method":"${method}","uri":"${uri}","user_agent":"${user_agent}","status":${status},"error":"${error}","latency":${latency},"latency_human":"${latency_human}","bytes_in":${bytes_in},"bytes_out":${bytes_out}}` + "\n",
	}))
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Custom slog request middleware
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			slog.Info("request handled",
				"method", c.Request().Method,
				"path", c.Request().URL.Path,
				"status", c.Response().Status,
				"duration", time.Since(start),
				"ip", c.RealIP(),
			)

			return err
		}
	})

	// Custom middleware for security headers
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Set security headers
			c.Response().Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			c.Response().Header().Set("X-Content-Type-Options", "nosniff")
			c.Response().Header().Set("X-Frame-Options", "DENY")
			c.Response().Header().Set("X-XSS-Protection", "1; mode=block")
			return next(c)
		}
	})

	// Static files
	e.Static("/public", "public")

	// Initialize service and register routes
	svc := service.New(db, config)
	svc.RegisterRoutes(e)

	// Start server
	addr := fmt.Sprintf(":%s", config.Port)
	url := fmt.Sprintf("http://localhost:%s", config.Port)

	slog.Info("🚀 Logan's 3D Creations v4 starting",
		"url", url,
		"port", config.Port,
		"environment", config.Environment,
		"database", config.DBPath,
	)

	if err := e.Start(addr); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func customHTTPErrorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
	}

	if code == http.StatusNotFound {
		// Render custom 404 page
		path := c.Request().URL.Path
		slog.Info("404 not found", "path", path)
		c.Response().Status = http.StatusNotFound
		if renderErr := errors.NotFound(c, path).Render(c.Request().Context(), c.Response()); renderErr != nil {
			slog.Error("failed to render 404 page", "error", renderErr)
			c.String(http.StatusNotFound, "Page not found")
		}
		return
	}

	if code == http.StatusUnauthorized {
		// Render custom 401 page with auto-refresh capability
		var attemptedPath string
		if path, ok := c.Get("attempted_path").(string); ok && path != "" {
			attemptedPath = path
		} else {
			attemptedPath = c.Request().URL.Path
			if c.Request().URL.RawQuery != "" {
				attemptedPath += "?" + c.Request().URL.RawQuery
			}
		}

		hasClientCookie := false
		if val, ok := c.Get("has_client_cookie").(bool); ok {
			hasClientCookie = val
		}

		slog.Info("401 unauthorized", "path", attemptedPath, "has_client_cookie", hasClientCookie)
		c.Response().Status = http.StatusUnauthorized
		if renderErr := errors.Unauthorized(c, hasClientCookie, attemptedPath).Render(c.Request().Context(), c.Response()); renderErr != nil {
			slog.Error("failed to render 401 page", "error", renderErr)
			c.String(http.StatusUnauthorized, "Unauthorized")
		}
		return
	}

	// For other errors, use Echo's default error handler
	c.Logger().Error(err)
	if !c.Response().Committed {
		if c.Request().Method == http.MethodHead {
			c.NoContent(code)
		} else {
			c.String(code, http.StatusText(code))
		}
	}
}

func validateRequiredEnvVars() {
	requiredVars := []string{
		"CLERK_SECRET_KEY",
		"CLERK_PUBLISHABLE_KEY",
	}

	var missing []string
	for _, envVar := range requiredVars {
		if os.Getenv(envVar) == "" {
			missing = append(missing, envVar)
		}
	}

	if len(missing) > 0 {
		slog.Error("missing required environment variables",
			"missing", strings.Join(missing, ", "),
			"hint", "add these to .envrc and run 'direnv allow'",
		)
		fmt.Fprintf(os.Stderr, "\nRequired environment variables missing:\n")
		for _, v := range missing {
			fmt.Fprintf(os.Stderr, "  - %s\n", v)
		}
		fmt.Fprintf(os.Stderr, "\nAdd these to .envrc and run 'direnv allow'\n\n")
		os.Exit(1)
	}
}
