package middleware

import (
	"strings"

	"pesxchange-backend/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gofiber/fiber/v2"
)

// JWTClaims represents JWT token claims
type JWTClaims struct {
	UserID string `json:"user_id"`
	SRN    string `json:"srn"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// JWTAuth creates a JWT authentication middleware
func JWTAuth() fiber.Handler {
	cfg := config.Load()
	
	return func(c *fiber.Ctx) error {
		// Get token from Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Authorization header required",
				"success": false,
			})
		}
		
		// Check if it starts with "Bearer "
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Invalid authorization header format",
				"success": false,
			})
		}
		
		// Parse and validate token
		token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method is specifically HS256
			if token.Method != jwt.SigningMethodHS256 {
				return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid signing method")
			}
			return []byte(cfg.JWTSecret), nil
		})
		
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Invalid token",
				"success": false,
			})
		}
		
		// Extract and validate claims
		if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
			// Validate token issuer
			if claims.Issuer != "pesxchange-backend" {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error":   "Invalid token issuer",
					"success": false,
				})
			}
			
			// Validate required claims
			if claims.UserID == "" || claims.SRN == "" {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error":   "Invalid token claims",
					"success": false,
				})
			}
			
			// Set user information in context
			c.Locals("userID", claims.UserID)
			c.Locals("userSRN", claims.SRN)
			c.Locals("userName", claims.Name)
			c.Locals("userEmail", claims.Email)
			
			return c.Next()
		}
		
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "Invalid token claims",
			"success": false,
		})
	}
}

// OptionalJWTAuth creates an optional JWT authentication middleware
func OptionalJWTAuth() fiber.Handler {
	cfg := config.Load()
	
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Next() // Continue without authentication
		}
		
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			return c.Next() // Continue without authentication
		}
		
		token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid signing method")
			}
			return []byte(cfg.JWTSecret), nil
		})
		
		if err == nil {
			if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid && 
				claims.Issuer == "pesxchange-backend" && 
				claims.UserID != "" && claims.SRN != "" {
				c.Locals("userID", claims.UserID)
				c.Locals("userSRN", claims.SRN)
				c.Locals("userName", claims.Name)
				c.Locals("userEmail", claims.Email)
			}
		}
		
		return c.Next()
	}
}