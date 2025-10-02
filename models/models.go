package models

import (
	"time"
)

// PESUProfile represents a PESU student profile
type PESUProfile struct {
	Name       string `json:"name" validate:"required"`
	PRN        string `json:"prn" validate:"required"`
	SRN        string `json:"srn" validate:"required,srn"`
	Program    string `json:"program" validate:"required"`
	Branch     string `json:"branch" validate:"required"`
	Semester   string `json:"semester" validate:"required"`
	Section    string `json:"section" validate:"required"`
	Email      string `json:"email" validate:"required,email"`
	Phone      string `json:"phone"`
	CampusCode int    `json:"campus_code"`
	Campus     string `json:"campus"`
}

// PESUAuthRequest represents authentication request
type PESUAuthRequest struct {
	Username string `json:"username" validate:"required,srn"`
	Password string `json:"password" validate:"required,min=1"`
}

// PESUAuthResponse represents authentication response from PESU API
type PESUAuthResponse struct {
	Status    bool         `json:"status"`
	Profile   *PESUProfile `json:"profile,omitempty"`
	Message   string       `json:"message"`
	Timestamp string       `json:"timestamp"`
}

// User represents a user in the system - matches user_profiles table exactly
type User struct {
	ID          string     `json:"id" db:"id"`
	SRN         string     `json:"srn" db:"srn"`
	PRN         string     `json:"prn" db:"prn"`
	Name        string     `json:"name" db:"name"`
	Email       string     `json:"email" db:"email"`
	Phone       string     `json:"phone" db:"phone"`
	Bio         string     `json:"bio" db:"bio"`
	AvatarURL   string     `json:"avatar_url" db:"avatar_url"`
	Program     string     `json:"program" db:"program"`
	Branch      string     `json:"branch" db:"branch"`
	Semester    string     `json:"semester" db:"semester"`
	Section     string     `json:"section" db:"section"`
	CampusCode  *int       `json:"campus_code" db:"campus_code"`
	Campus      string     `json:"campus" db:"campus"`
	Rating      float64    `json:"rating" db:"rating"`
	Verified    bool       `json:"verified" db:"verified"`
	Location    string     `json:"location" db:"location"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	LastLogin   *time.Time `json:"last_login" db:"last_login"`
	Nickname    string     `json:"nickname" db:"nickname"`
}

// Item represents an item for sale - matches items table exactly
type Item struct {
	ID          string    `json:"id" db:"id"`
	Title       string    `json:"title" db:"title" validate:"required,min=3,max=100"`
	Description string    `json:"description" db:"description" validate:"required,min=10,max=1000"`
	Price       float64   `json:"price" db:"price" validate:"required,gt=0"`
	Location    string    `json:"location" db:"location" validate:"required"`
	Year        *int      `json:"year" db:"year"`
	Condition   string    `json:"condition" db:"condition" validate:"required,oneof=New Like\ New Good Fair Poor"`
	CategoryID  *string   `json:"category_id" db:"category_id"`
	Images      []string  `json:"images" db:"images"`
	Views       int       `json:"views" db:"views"`
	IsAvailable bool      `json:"is_available" db:"is_available"`
	IsFeatured  bool      `json:"is_featured" db:"is_featured"`
	SellerID    string    `json:"seller_id" db:"seller_id"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
	Category    string    `json:"category" db:"category"`
	
	// Legacy field for backward compatibility with frontend
	ImageURLs   []string  `json:"image_urls,omitempty"`
	Categories  []string  `json:"categories,omitempty"`
	
	// Joined fields
	Seller *User `json:"seller,omitempty"`
}

// CreateItemRequest represents item creation request - matches Node.js API
type CreateItemRequest struct {
	Title       string   `json:"title" validate:"required,min=3,max=100"`
	Description string   `json:"description" validate:"required,min=10,max=1000"`
	Price       float64  `json:"price" validate:"required,gt=0"`
	Location    string   `json:"location" validate:"required"`
	Condition   string   `json:"condition" validate:"required,oneof=New Like\ New Good Fair Poor"`
	Category    string   `json:"category"`
	Images      []string `json:"images"`
	SellerID    string   `json:"seller_id" validate:"required"`
	IsAvailable *bool    `json:"is_available"`
	Views       *int     `json:"views"`
}

// Message represents a chat message
type Message struct {
	ID         string    `json:"id" db:"id"`
	SenderID   string    `json:"sender_id" db:"sender_id"`
	ReceiverID string    `json:"receiver_id" db:"receiver_id"`
	Message    string    `json:"message" db:"message" validate:"required,min=1,max=1000"`
	IsRead     bool      `json:"is_read" db:"is_read"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	
	// Legacy field for backward compatibility
	Content    string    `json:"content,omitempty"`
	ReadAt     *time.Time `json:"read_at,omitempty"`
	
	// Joined fields
	Sender   *User `json:"sender,omitempty"`
	Receiver *User `json:"receiver,omitempty"`
	Item     *Item `json:"item,omitempty"`
}

// SendMessageRequest represents message sending request - matches Node.js API
type SendMessageRequest struct {
	ReceiverID string `json:"receiver_id" validate:"required"`
	Message    string `json:"message" validate:"required,min=1,max=1000"`
}

// Chat represents a conversation between two users
type Chat struct {
	ID           string    `json:"id"`
	User1ID      string    `json:"user1_id"`
	User2ID      string    `json:"user2_id"`
	LastMessage  *Message  `json:"last_message"`
	UnreadCount  int       `json:"unread_count"`
	UpdatedAt    time.Time `json:"updated_at"`
	
	// Joined fields
	OtherUser *User `json:"other_user,omitempty"`
}

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// PaginatedResponse represents paginated API response
type PaginatedResponse struct {
	Success    bool        `json:"success"`
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

// Pagination represents pagination metadata
type Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}