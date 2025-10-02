package middleware

import (
	"log"
	"strconv"
	"time"

	"pesxchange-backend/config"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// ErrorHandler handles all errors
func ErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal server error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	// Only log detailed errors in development
	cfg := config.Load()
	if cfg.IsDevelopment() {
		log.Printf("Error [%s %s]: %v", c.Method(), c.Path(), err)
	} else {
		// In production, only log error codes and sanitized info
		log.Printf("Error [%d]: %s %s", code, c.Method(), c.Path())
	}

	return c.Status(code).JSON(fiber.Map{
		"error":   message,
		"success": false,
	})
}

// RateLimit creates a rate limiter middleware
func RateLimit() fiber.Handler {
	cfg := config.Load()
	return limiter.New(limiter.Config{
		Max:               cfg.RateLimitMax,
		Expiration:        time.Duration(cfg.RateLimitWindow) * time.Second,
		LimiterMiddleware: limiter.SlidingWindow{},
		KeyGenerator: func(c *fiber.Ctx) string {
			// Use X-Forwarded-For for proxy environments, fallback to IP
			if forwarded := c.Get("X-Forwarded-For"); forwarded != "" {
				return forwarded
			}
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   "Rate limit exceeded. Please try again later.",
				"success": false,
			})
		},
	})
}

// AuthRateLimit creates a stricter rate limiter for auth endpoints
func AuthRateLimit() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:               3, // Stricter limit for auth
		Expiration:        15 * time.Minute,
		LimiterMiddleware: limiter.SlidingWindow{},
		KeyGenerator: func(c *fiber.Ctx) string {
			key := c.IP()
			if forwarded := c.Get("X-Forwarded-For"); forwarded != "" {
				key = forwarded
			}
			return key + "-auth"
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   "Too many authentication attempts. Please wait 15 minutes before trying again.",
				"success": false,
			})
		},
	})
}

// Logger middleware for request logging
func Logger() fiber.Handler {
	cfg := config.Load()
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		
		// Only log in development or for errors
		if cfg.IsDevelopment() || c.Response().StatusCode() >= 400 {
			log.Printf("%s %s %d %v",
				c.Method(),
				c.Path(),
				c.Response().StatusCode(),
				time.Since(start),
			)
		}
		
		return err
	}
}

// ValidateJSON validates request content type
func ValidateJSON() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Method() == "POST" || c.Method() == "PUT" || c.Method() == "PATCH" {
			if c.Get("Content-Type") != "application/json" {
				return fiber.NewError(fiber.StatusBadRequest, "Content-Type must be application/json")
			}
		}
		return c.Next()
	}
}

// ParsePagination extracts pagination parameters
func ParsePagination(c *fiber.Ctx) (int, int) {
	limit, _ := strconv.Atoi(c.Query("limit", "12"))  // Reduce default to 12 for better performance
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	
	// Enforce limits
	if limit > 50 {  // Reduce max limit to 50
		limit = 50
	}
	if limit < 1 {
		limit = 12
	}
	if offset < 0 {
		offset = 0
	}
	
	return limit, offset
}