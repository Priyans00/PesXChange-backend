package handlers

import (
	"strconv"
	"strings"

	"pesxchange-backend/middleware"
	"pesxchange-backend/models"
	"pesxchange-backend/services"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type ItemHandler struct {
	itemService *services.ItemService
	validator   *validator.Validate
}

func NewItemHandler(itemService *services.ItemService) *ItemHandler {
	return &ItemHandler{
		itemService: itemService,
		validator:   validator.New(),
	}
}

// CreateItem handles item creation
func (h *ItemHandler) CreateItem(c *fiber.Ctx) error {
	var req models.CreateItemRequest
	
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}
	
	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		// Create more user-friendly error messages
		var errorMsgs []string
		for _, err := range err.(validator.ValidationErrors) {
			switch err.Tag() {
			case "required":
				errorMsgs = append(errorMsgs, err.Field()+" is required")
			case "min":
				errorMsgs = append(errorMsgs, err.Field()+" must be at least "+err.Param()+" characters long")
			case "max":
				errorMsgs = append(errorMsgs, err.Field()+" must be less than "+err.Param()+" characters")
			case "gt":
				errorMsgs = append(errorMsgs, err.Field()+" must be greater than "+err.Param())
			case "oneof":
				errorMsgs = append(errorMsgs, err.Field()+" must be one of: "+err.Param())
			default:
				errorMsgs = append(errorMsgs, err.Field()+" is invalid")
			}
		}
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   strings.Join(errorMsgs, ", "),
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
	
	// Ensure the authenticated user can only create items for themselves
	userID := authenticatedUserID.(string)
	if req.SellerID != "" && req.SellerID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Error:   "You can only create items for yourself",
		})
	}
	
	// Set seller ID to authenticated user ID for security
	req.SellerID = userID
	
	item, err := h.itemService.CreateItem(c.Context(), &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to create item",
		})
	}
	
	return c.Status(fiber.StatusCreated).JSON(models.APIResponse{
		Success: true,
		Data:    item,
		Message: "Item created successfully",
	})
}

// GetItems handles item listing with filters
func (h *ItemHandler) GetItems(c *fiber.Ctx) error {
	limit, offset := middleware.ParsePagination(c)
	
	// Parse filters
	filters := make(map[string]interface{})
	
	if search := c.Query("search"); search != "" {
		filters["search"] = strings.TrimSpace(search)
	}
	
	if category := c.Query("category"); category != "" {
		filters["category"] = strings.TrimSpace(category)
	}
	
	if condition := c.Query("condition"); condition != "" {
		filters["condition"] = strings.TrimSpace(condition)
	}
	
	if location := c.Query("location"); location != "" {
		filters["location"] = strings.TrimSpace(location)
	}
	
	if sort := c.Query("sort"); sort != "" {
		validSorts := []string{"created_at", "price_asc", "price_desc", "title"}
		for _, valid := range validSorts {
			if sort == valid {
				filters["sort"] = sort
				break
			}
		}
	}
	
	// Parse price filters
	if minPriceStr := c.Query("min_price"); minPriceStr != "" {
		if minPrice, err := strconv.ParseFloat(minPriceStr, 64); err == nil && minPrice > 0 {
			filters["min_price"] = minPrice
		}
	}
	
	if maxPriceStr := c.Query("max_price"); maxPriceStr != "" {
		if maxPrice, err := strconv.ParseFloat(maxPriceStr, 64); err == nil && maxPrice > 0 {
			filters["max_price"] = maxPrice
		}
	}
	
	items, total, err := h.itemService.GetItems(c.Context(), limit, offset, filters)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to retrieve items",
		})
	}
	
	// Set cache headers for item listings (1 minute to keep data fresh)
	c.Set("Cache-Control", "public, max-age=60")
	c.Set("Connection", "keep-alive")
	
	return c.JSON(models.PaginatedResponse{
		Success: true,
		Data:    items,
		Pagination: models.Pagination{
			Limit:  limit,
			Offset: offset,
			Total:  total,
		},
	})
}

// GetItem handles single item retrieval
func (h *ItemHandler) GetItem(c *fiber.Ctx) error {
	itemID := c.Params("id")
	if itemID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Item ID is required",
		})
	}
	
	item, err := h.itemService.GetItemByID(c.Context(), itemID)
	if err != nil {
		if err.Error() == "item not found" {
			return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Error:   "Item not found",
			})
		}
		
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to get item",
		})
	}
	
	// Set cache headers for individual items (5 minutes)
	c.Set("Cache-Control", "public, max-age=300")
	c.Set("Connection", "keep-alive")
	
	return c.JSON(models.APIResponse{
		Success: true,
		Data:    item,
	})
}

// UpdateItem handles item updates
func (h *ItemHandler) UpdateItem(c *fiber.Ctx) error {
	itemID := c.Params("id")
	if itemID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Item ID is required",
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
	
	sellerID := authenticatedUserID.(string)
	
	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}
	
	item, err := h.itemService.UpdateItem(c.Context(), itemID, sellerID, updates)
	if err != nil {
		if err.Error() == "item not found" {
			return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Error:   "Item not found",
			})
		}
		if strings.Contains(err.Error(), "unauthorized") {
			return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
				Success: false,
				Error:   "You can only edit your own items",
			})
		}
		
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to update item",
		})
	}
	
	return c.JSON(models.APIResponse{
		Success: true,
		Data:    item,
		Message: "Item updated successfully",
	})
}

// DeleteItem handles item deletion
func (h *ItemHandler) DeleteItem(c *fiber.Ctx) error {
	itemID := c.Params("id")
	if itemID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Item ID is required",
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
	
	sellerID := authenticatedUserID.(string)
	
	err := h.itemService.DeleteItem(c.Context(), itemID, sellerID)
	if err != nil {
		if err.Error() == "item not found" {
			return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Error:   "Item not found",
			})
		}
		if strings.Contains(err.Error(), "unauthorized") {
			return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
				Success: false,
				Error:   "You can only delete your own items",
			})
		}
		
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to delete item",
		})
	}
	
	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Item deleted successfully",
	})
}

// GetItemImage serves individual item images
func (h *ItemHandler) GetItemImage(c *fiber.Ctx) error {
	itemID := c.Params("id")
	imageIndex := c.Params("index", "0")
	
	idx, err := strconv.Atoi(imageIndex)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid image index",
		})
	}
	
	// Get the item from database
	item, err := h.itemService.GetItemByID(c.Context(), itemID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Item not found",
		})
	}
	
	// Check if image index exists
	if idx >= len(item.Images) || idx < 0 {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Error:   "Image not found",
		})
	}
	
	imageData := item.Images[idx]
	
	// Check if it's base64 data
	if strings.HasPrefix(imageData, "data:image/") {
		// Parse base64 image
		parts := strings.Split(imageData, ",")
		if len(parts) != 2 {
			return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
				Success: false,
				Error:   "Invalid image format",
			})
		}
		
		// Extract content type
		header := parts[0]
		var contentType string
		if strings.Contains(header, "image/jpeg") || strings.Contains(header, "image/jpg") {
			contentType = "image/jpeg"
		} else if strings.Contains(header, "image/png") {
			contentType = "image/png"
		} else if strings.Contains(header, "image/webp") {
			contentType = "image/webp"
		} else {
			contentType = "image/jpeg" // default
		}
		
		// For now, return a placeholder response since serving large base64 images directly
		// is not recommended. In production, you should store images in file storage.
		return c.Status(fiber.StatusOK).JSON(models.APIResponse{
			Success: true,
			Data: map[string]interface{}{
				"message": "Image data available but too large for direct serving",
				"item_id": itemID,
				"index": idx,
				"type": contentType,
				"size": len(imageData),
			},
		})
	}
	
	// If it's already a URL, redirect to it
	return c.Redirect(imageData)
}

// GetItemsBySeller handles getting items by seller ID
func (h *ItemHandler) GetItemsBySeller(c *fiber.Ctx) error {
	sellerID := c.Params("sellerId")
	if sellerID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Seller ID is required",
		})
	}

	// Parse pagination parameters
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	// Validate pagination
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	items, err := h.itemService.GetItemsBySeller(c.Context(), sellerID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to retrieve seller items",
		})
	}

	// Set cache headers for seller items (2 minutes)
	c.Set("Cache-Control", "public, max-age=120")
	
	return c.JSON(models.APIResponse{
		Success: true,
		Data:    items,
		Message: "Seller items retrieved successfully",
	})
}