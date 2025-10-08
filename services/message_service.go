package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"pesxchange-backend/database"
	"pesxchange-backend/models"

	"github.com/google/uuid"
)

type MessageService struct{}

func NewMessageService() *MessageService {
	return &MessageService{}
}

// SendMessage sends a new message
func (s *MessageService) SendMessage(ctx context.Context, senderID string, req *models.SendMessageRequest) (*models.Message, error) {
	client := database.GetClient()
	
	// Validate that receiver exists
	if err := s.validateMessageRequest(ctx, req); err != nil {
		return nil, err
	}
	
	now := time.Now()
	
	// Create message data map for insert
	messageData := map[string]interface{}{
		"sender_id":   senderID,
		"receiver_id": req.ReceiverID,
		"message":     req.Message,
		"is_read":     false,
		"created_at":  now.Format(time.RFC3339),
	}
	
	// Only include item_id if provided and not empty
	if req.ItemID != "" {
		messageData["item_id"] = req.ItemID
	}
	
	data, _, err := client.From("messages").
		Insert(messageData, false, "", "", "").
		Execute()
	
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}
	
	// Parse the response data
	if data != nil && len(data) > 0 {
		var messages []models.Message
		if err := json.Unmarshal(data, &messages); err == nil && len(messages) > 0 {
			return &messages[0], nil
		}
	}
	
	// Fallback: create message from the data we inserted
	messageID := uuid.New().String()
	if data != nil && len(data) > 0 {
		// Try to extract ID from response
		var responseData []map[string]interface{}
		if err := json.Unmarshal(data, &responseData); err == nil && len(responseData) > 0 {
			if id, ok := responseData[0]["id"].(string); ok {
				messageID = id
			}
		}
	}
	
	message := &models.Message{
		ID:         messageID,
		SenderID:   senderID,
		ReceiverID: req.ReceiverID,
		Message:    req.Message,
		IsRead:     false,
		CreatedAt:  now,
	}
	if req.ItemID != "" {
		message.ItemID = &req.ItemID
	}
	
	return message, nil
}

// GetMessages retrieves messages between two users for a specific item (or all messages if no item specified)
func (s *MessageService) GetMessages(ctx context.Context, userID, otherUserID, itemID string, limit, offset int) ([]models.Message, error) {
	client := database.GetClient()
	
	query := client.From("messages").
		Select("*", "exact", false).
		Or(fmt.Sprintf("sender_id.eq.%s,receiver_id.eq.%s", userID, otherUserID), "").
		Order("created_at", nil)
	
	// Only filter by item_id if provided
	if itemID != "" {
		query = query.Eq("item_id", itemID)
	}
	
	data, _, err := query.Range(offset, offset+limit-1, "").Execute()
	
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	
	var messages []models.Message
	if data != nil {
		if err := json.Unmarshal(data, &messages); err != nil {
			return nil, fmt.Errorf("failed to parse messages: %w", err)
		}
	}
	
	return messages, nil
}

// GetActiveChats retrieves all active conversations for a user
func (s *MessageService) GetActiveChats(ctx context.Context, userID string) ([]models.Chat, error) {
	client := database.GetClient()
	
	// Get latest message for each conversation
	data, _, err := client.From("messages").
		Select("*", "exact", false).
		Or(fmt.Sprintf("sender_id.eq.%s,receiver_id.eq.%s", userID, userID), "").
		Order("created_at", nil).
		Execute()
	
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	
	var messages []models.Message
	if data != nil {
		if err := json.Unmarshal(data, &messages); err != nil {
			return nil, fmt.Errorf("failed to parse messages: %w", err)
		}
	}
	
	// Group messages by conversation (other_user + item)
	chatMap := make(map[string]*models.Chat)
	
	for _, msg := range messages {
		var otherUserID string
		var otherUser *models.User
		
		if msg.SenderID == userID {
			otherUserID = msg.ReceiverID
			otherUser = msg.Receiver
		} else {
			otherUserID = msg.SenderID
			otherUser = msg.Sender
		}
		
		chatKey := fmt.Sprintf("%s-%s", userID, otherUserID)
		
		if _, exists := chatMap[chatKey]; !exists {
			chatMap[chatKey] = &models.Chat{
				ID:          chatKey,
				User1ID:     userID,
				User2ID:     otherUserID,
				LastMessage: &msg,
				UpdatedAt:   msg.CreatedAt,
				OtherUser:   otherUser,
				UnreadCount: 0, // TODO: Calculate unread count
			}
		}
	}
	
	// Convert map to slice
	chats := make([]models.Chat, 0, len(chatMap))
	for _, chat := range chatMap {
		chats = append(chats, *chat)
	}
	
	return chats, nil
}

// MarkMessagesAsRead marks messages as read
func (s *MessageService) MarkMessagesAsRead(ctx context.Context, userID, otherUserID, itemID string) error {
	client := database.GetClient()
	
	now := time.Now()
	updates := map[string]interface{}{
		"read_at": now,
	}
	
	_, _, err := client.From("messages").
		Update(updates, "", "").
		Eq("receiver_id", userID).
		Eq("sender_id", otherUserID).
		Eq("item_id", itemID).
		Is("read_at", "null").
		Execute()
	
	if err != nil {
		return fmt.Errorf("failed to mark messages as read: %w", err)
	}
	
	return nil
}

// validateMessageRequest validates the message request
func (s *MessageService) validateMessageRequest(ctx context.Context, req *models.SendMessageRequest) error {
	client := database.GetClient()
	
	// Check if receiver exists in user_profiles table
	data, _, err := client.From("user_profiles").
		Select("id", "exact", false).
		Eq("id", req.ReceiverID).
		Execute()
	
	if err != nil {
		return fmt.Errorf("failed to validate receiver: %w", err)
	}
	
	if data == nil || len(data) == 0 {
		return fmt.Errorf("receiver not found")
	}
	
	return nil
}