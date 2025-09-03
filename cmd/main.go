package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/loganlanou/logans3d-v4/service"
	"github.com/loganlanou/logans3d-v4/storage"
)

func main() {
	// slog is configured in slog.go via init()

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

	// Middleware
	e.Use(middleware.Logger())
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