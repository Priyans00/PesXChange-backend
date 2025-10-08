package handlers

import (
	"fmt"
	"pesxchange-backend/models"
	"pesxchange-backend/services"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type UserHandler struct {
	userService *services.UserService
	validator   *validator.Validate
}

func NewUserHandler(userService *services.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
		validator:   validator.New(),
	}
}

// GetProfile gets user profile by ID
func (h *UserHandler) GetProfile(c *fiber.Ctx) error {
	userID := c.Params("id")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "User ID is required",
		})
	}
	
	user, err := h.userService.GetUserByID(c.Context(), userID)
	if err != nil {
		if err.Error() == "user not found" {
			return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Error:   "User not found",
			})
		}
		
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to retrieve user profile",
		})
	}
	
	// Set cache headers for profile data (5 minutes)
	c.Set("Cache-Control", "public, max-age=300")
	c.Set("ETag", fmt.Sprintf("\"%s-%d\"", userID, user.UpdatedAt.Unix()))
	
	return c.JSON(models.APIResponse{
		Success: true,
		Data:    user,
	})
}

// UpdateProfile updates user profile
func (h *UserHandler) UpdateProfile(c *fiber.Ctx) error {
	userID := c.Params("id")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "User ID is required",
		})
	}
	
	// Get authenticated user ID from JWT middleware
	authenticatedUserID := c.Locals("userID")
	if authenticatedUserID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.APIResponse{
			Success: false,
			Error:   "Authentication required",
		})
	}
	
	// Ensure user can only update their own profile
	authUserID := authenticatedUserID.(string)
	if authUserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "You can only update your own profile",
		})
	}
	
	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}
	
	// Remove protected fields that shouldn't be updated directly
	delete(updates, "id")
	delete(updates, "srn")
	delete(updates, "created_at")
	delete(updates, "verified")
	delete(updates, "rating")
	
	user, err := h.userService.UpdateUserProfile(c.Context(), userID, updates)
	if err != nil {
		if err.Error() == "user not found" {
			return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Error:   "User not found",
			})
		}
		
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to update user profile",
		})
	}
	
	return c.JSON(models.APIResponse{
		Success: true,
		Data:    user,
		Message: "Profile updated successfully",
	})
}

// GetAllUsers gets all users with pagination
func (h *UserHandler) GetAllUsers(c *fiber.Ctx) error {
	// This would typically require admin authentication
	// For now, we'll return a basic response
	
	return c.JSON(models.APIResponse{
		Success: false,
		Error:   "Endpoint not implemented - requires admin authentication",
	})
}