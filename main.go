package main

import (
	"log"
	"strings"
	"time"

	"pesxchange-backend/config"
	"pesxchange-backend/database"
	"pesxchange-backend/middleware"
	"pesxchange-backend/routes"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/helmet/v2"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	if err := database.Initialize(cfg); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Initialize Fiber app with balanced settings for development and production
	readTimeout := 30 * time.Second  // Default for production
	writeTimeout := 30 * time.Second // Default for production
	if cfg.IsDevelopment() {
		readTimeout = 60 * time.Second   // More lenient for development
		writeTimeout = 60 * time.Second  // More lenient for development
	}
	
	app := fiber.New(fiber.Config{
		ErrorHandler:      middleware.ErrorHandler,
		ReadBufferSize:    8192,  // 8KB - good balance
		WriteBufferSize:   8192,  // 8KB - good balance
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       120 * time.Second,   // 2 minutes - longer idle timeout
		BodyLimit:         2 * 1024 * 1024, // 2MB - security limit
		DisableKeepalive:  false, // Keep connections alive
		ServerHeader:      "",    // Hide server information
		AppName:           "PesXChange API",
		EnablePrintRoutes: cfg.IsDevelopment(), // Show routes in development
	})

	// Security middleware with optimized settings
	app.Use(helmet.New(helmet.Config{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "DENY",
		HSTSMaxAge:            31536000, // 1 year
		ContentSecurityPolicy: "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'",
	}))
	
	// Improved CORS middleware to prevent connection issues
	app.Use(func(c *fiber.Ctx) error {
		origin := c.Get("Origin")
		
		// Parse allowed origins from config
		allowedOrigins := strings.Split(cfg.AllowedOrigins, ",")
		for i, o := range allowedOrigins {
			allowedOrigins[i] = strings.TrimSpace(o)
		}
		
		// In development, be more permissive to prevent connection issues
		isAllowed := cfg.IsDevelopment()
		if !isAllowed {
			for _, allowed := range allowedOrigins {
				if origin == allowed {
					isAllowed = true
					break
				}
			}
		}
		
		// Always set basic CORS headers to prevent connection resets
		c.Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS,PATCH")
		c.Set("Access-Control-Allow-Headers", "Origin,Content-Type,Accept,Authorization,X-Requested-With,X-User-ID")
		c.Set("Access-Control-Max-Age", "86400")
		c.Set("Vary", "Origin")
		
		// Set origin-specific headers if allowed
		if isAllowed && origin != "" {
			c.Set("Access-Control-Allow-Origin", origin)
			c.Set("Access-Control-Allow-Credentials", "true")
		} else if cfg.IsDevelopment() {
			// In development, allow localhost variants
			c.Set("Access-Control-Allow-Origin", "*")
		}
		
		// Handle preflight OPTIONS requests
		if c.Method() == "OPTIONS" {
			return c.SendStatus(204)
		}
		
		return c.Next()
	})

	// Request logging (only in development or for errors)
	app.Use(middleware.Logger())
	
	// Add keep-alive and connection handling
	app.Use(func(c *fiber.Ctx) error {
		// Set keep-alive headers to prevent connection resets
		c.Set("Connection", "keep-alive")
		c.Set("Keep-Alive", "timeout=5, max=1000")
		return c.Next()
	})
	
	// Rate limiting (only for API routes)
	apiGroup := app.Group("/api")
	apiGroup.Use(middleware.RateLimit())

	// Global OPTIONS handler for any missed preflight requests
	app.Options("/*", func(c *fiber.Ctx) error {
		return c.SendStatus(204)
	})

	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"service": "pesxchange-backend",
		})
	})

	// Health check endpoint under API
	apiGroup.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"service": "pesxchange-backend",
		})
	})
	
	// Temporary debug route to test JWT auth
	apiGroup.Get("/test-auth", middleware.JWTAuth(), func(c *fiber.Ctx) error {
		userID := c.Locals("userID")
		return c.JSON(fiber.Map{
			"message": "Authentication successful",
			"userID": userID,
		})
	})
	
	// Setup routes with the configured API group
	routes.SetupAuthRoutes(apiGroup)
	routes.SetupUserRoutes(apiGroup)
	routes.SetupItemRoutes(apiGroup)
	routes.SetupMessageRoutes(apiGroup)
	routes.SetupProfileRoutes(apiGroup)

	// Start server
	port := cfg.Port
	if cfg.IsDevelopment() {
		log.Printf("Server starting on port %s", port)
	}
	log.Fatal(app.Listen(":" + port))
}