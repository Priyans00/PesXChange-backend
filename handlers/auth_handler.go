package handlers

import (
	"strings"
	
	"pesxchange-backend/config"
	"pesxchange-backend/models"
	"pesxchange-backend/services"
	"pesxchange-backend/utils"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	authService *services.AuthService
	validator   *validator.Validate
	config      *config.Config
}

func NewAuthHandler(authService *services.AuthService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		validator:   validator.New(),
		config:      cfg,
	}
}

// LoginWithPESU handles PESU authentication - matches Next.js API exactly
func (h *AuthHandler) LoginWithPESU(c *fiber.Ctx) error {
	var req models.PESUAuthRequest
	
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}
	
	// Basic validation - no complex validation needed, just like Next.js version
	if req.Username == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Username and password are required",
		})
	}
	
	// Authenticate with PESU and create/update user
	user, err := h.authService.AuthenticateWithPESU(c.Context(), &req)
	if err != nil {
		// Match the error handling from Next.js version
		status := fiber.StatusInternalServerError
		errorMsg := "Internal server error"
		
		if strings.Contains(err.Error(), "authentication failed") || 
		   strings.Contains(err.Error(), "invalid SRN format") {
			status = fiber.StatusUnauthorized
			errorMsg = err.Error()
		} else if strings.Contains(err.Error(), "authentication service unavailable") {
			status = fiber.StatusServiceUnavailable
			errorMsg = "Authentication service unavailable"
		} else if strings.Contains(err.Error(), "Unable to connect") {
			status = fiber.StatusServiceUnavailable
			errorMsg = "Unable to connect to authentication service. Please try again later."
		}
		
		return c.Status(status).JSON(fiber.Map{
			"error": errorMsg,
		})
	}
	
	// Generate JWT token for the authenticated user
	token, err := utils.GenerateJWT(user, h.config)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate authentication token",
		})
	}
	
	// Return user object with authentication token
	return c.JSON(fiber.Map{
		"user": fiber.Map{
			"id":      user.ID,
			"srn":     user.SRN,
			"name":    user.Name,
			"email":   user.Email,
			"profile": user, // The full user object serves as the profile
		},
		"token": token, // JWT token for API authentication
	})
}

// CheckSRN checks if SRN exists in database
func (h *AuthHandler) CheckSRN(c *fiber.Ctx) error {
	srn := c.Query("srn")
	if srn == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "SRN parameter is required",
		})
	}
	
	if !h.authService.ValidateSRN(srn) {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid SRN format",
		})
	}
	
	userService := services.NewUserService()
	exists, err := userService.CheckSRNExists(c.Context(), srn)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to check SRN",
		})
	}
	
	return c.JSON(models.APIResponse{
		Success: true,
		Data: fiber.Map{
			"exists": exists,
			"srn":    srn,
		},
	})
}