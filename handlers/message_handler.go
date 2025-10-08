package handlers

import (
	"pesxchange-backend/middleware"
	"pesxchange-backend/models"
	"pesxchange-backend/services"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type MessageHandler struct {
	messageService *services.MessageService
	validator      *validator.Validate
}

func NewMessageHandler(messageService *services.MessageService) *MessageHandler {
	return &MessageHandler{
		messageService: messageService,
		validator:      validator.New(),
	}
}

// SendMessage handles sending a new message
func (h *MessageHandler) SendMessage(c *fiber.Ctx) error {
	var req models.SendMessageRequest
	
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}
	
	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Validation failed: " + err.Error(),
		})
	}
	
	// Get authenticated user ID from JWT middleware
	senderID := c.Locals("userID")
	if senderID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.APIResponse{
			Success: false,
			Error:   "Authentication required",
		})
	}
	
	userID := senderID.(string)
	
	// Prevent self-messaging
	if userID == req.ReceiverID {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Cannot send message to yourself",
		})
	}
	
	message, err := h.messageService.SendMessage(c.Context(), userID, &req)
	if err != nil {
		status := fiber.StatusInternalServerError
		errorMsg := "Failed to send message"
		
		if err.Error() == "receiver not found" || err.Error() == "item not found" {
			status = fiber.StatusNotFound
			errorMsg = err.Error()
		} else if err.Error() == "item is not active" {
			status = fiber.StatusBadRequest
			errorMsg = err.Error()
		}
		
		return c.Status(status).JSON(models.APIResponse{
			Success: false,
			Error:   errorMsg,
		})
	}
	
	return c.Status(fiber.StatusCreated).JSON(models.APIResponse{
		Success: true,
		Data:    message,
		Message: "Message sent successfully",
	})
}

// GetMessages handles retrieving messages between users for an item
func (h *MessageHandler) GetMessages(c *fiber.Ctx) error {
	// Get authenticated user ID from JWT middleware
	authenticatedUserID := c.Locals("userID")
	if authenticatedUserID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.APIResponse{
			Success: false,
			Error:   "Authentication required",
		})
	}
	
	userID := authenticatedUserID.(string)
	
	otherUserID := c.Query("other_user_id")
	itemID := c.Query("item_id")
	
	if otherUserID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "other_user_id is required",
		})
	}
	
	// item_id is now optional - if not provided, get all messages between users
	limit, offset := middleware.ParsePagination(c)
	
	messages, err := h.messageService.GetMessages(c.Context(), userID, otherUserID, itemID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to get messages",
		})
	}
	
	// Mark messages as read
	go func() {
		h.messageService.MarkMessagesAsRead(c.Context(), userID, otherUserID, itemID)
	}()
	
	return c.JSON(models.APIResponse{
		Success: true,
		Data:    messages,
	})
}

// GetActiveChats handles retrieving all active conversations for a user
func (h *MessageHandler) GetActiveChats(c *fiber.Ctx) error {
	// Get authenticated user ID from JWT middleware
	authenticatedUserID := c.Locals("userID")
	if authenticatedUserID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.APIResponse{
			Success: false,
			Error:   "Authentication required",
		})
	}
	
	userID := authenticatedUserID.(string)
	
	chats, err := h.messageService.GetActiveChats(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to get active chats",
		})
	}
	
	return c.JSON(models.APIResponse{
		Success: true,
		Data:    chats,
	})
}

// MarkAsRead handles marking messages as read
func (h *MessageHandler) MarkAsRead(c *fiber.Ctx) error {
	// Get authenticated user ID from JWT middleware
	authenticatedUserID := c.Locals("userID")
	if authenticatedUserID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.APIResponse{
			Success: false,
			Error:   "Authentication required",
		})
	}
	
	userID := authenticatedUserID.(string)
	
	var req struct {
		OtherUserID string `json:"other_user_id" validate:"required"`
		ItemID      string `json:"item_id" validate:"required"`
	}
	
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}
	
	if err := h.validator.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Validation failed: " + err.Error(),
		})
	}
	
	err := h.messageService.MarkMessagesAsRead(c.Context(), userID, req.OtherUserID, req.ItemID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to mark messages as read",
		})
	}
	
	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Messages marked as read",
	})
}