package routes

import (
	"pesxchange-backend/config"
	"pesxchange-backend/handlers"
	"pesxchange-backend/middleware"
	"pesxchange-backend/services"

	"github.com/gofiber/fiber/v2"
)

func SetupAuthRoutes(api fiber.Router) {
	cfg := config.Load()
	userService := services.NewUserService()
	authService := services.NewAuthService(cfg, userService)
	authHandler := handlers.NewAuthHandler(authService, cfg)

	auth := api.Group("/auth")
	
	// Apply auth-specific rate limiting
	auth.Use(middleware.AuthRateLimit())
	auth.Use(middleware.ValidateJSON())

	// PESU authentication endpoint
	auth.Post("/pesu", authHandler.LoginWithPESU)
	
	// Check SRN endpoint
	auth.Get("/check-srn", authHandler.CheckSRN)
}

func SetupUserRoutes(api fiber.Router) {
	userService := services.NewUserService()
	userHandler := handlers.NewUserHandler(userService)

	users := api.Group("/users")
	
	// Get all users (admin only - not implemented)
	users.Get("/", userHandler.GetAllUsers)
}

func SetupProfileRoutes(api fiber.Router) {
	userService := services.NewUserService()
	userHandler := handlers.NewUserHandler(userService)

	profile := api.Group("/profile")
	
	// Public endpoints
	profile.Get("/:id", userHandler.GetProfile)  // Get user profile (public)
	
	// Protected route requiring authentication
	profile.Put("/:id", middleware.JWTAuth(), middleware.ValidateJSON(), userHandler.UpdateProfile)  // Update user profile
}

func SetupItemRoutes(api fiber.Router) {
	itemService := services.NewItemService()
	itemHandler := handlers.NewItemHandler(itemService)
	imageHandler := handlers.NewImageHandler()

	items := api.Group("/items")
	
	// Public endpoints
	items.Get("/", itemHandler.GetItems)      // Get all items with filters and pagination
	items.Get("/:id", itemHandler.GetItem)   // Get single item by ID
	items.Get("/:id/image/:index", itemHandler.GetItemImage) // Get item image
	items.Get("/seller/:sellerId", itemHandler.GetItemsBySeller) // Get items by seller ID
	
	// Protected routes requiring authentication
	items.Post("/", middleware.JWTAuth(), middleware.ValidateJSON(), itemHandler.CreateItem)           // Create new item
	items.Put("/:id", middleware.JWTAuth(), middleware.ValidateJSON(), itemHandler.UpdateItem)        // Update item
	items.Delete("/:id", middleware.JWTAuth(), itemHandler.DeleteItem)                                // Delete item
	
	// Image management routes
	items.Post("/upload-images", middleware.JWTAuth(), imageHandler.UploadImage)                      // Upload images to Supabase Storage
	items.Post("/convert-images", middleware.JWTAuth(), middleware.ValidateJSON(), imageHandler.ConvertBase64ToStorage) // Convert base64 to storage URLs
}

func SetupMessageRoutes(api fiber.Router) {
	messageService := services.NewMessageService()
	messageHandler := handlers.NewMessageHandler(messageService)

	// Protected message routes requiring authentication
	messages := api.Group("/messages")
	
	messages.Post("/", middleware.JWTAuth(), middleware.ValidateJSON(), messageHandler.SendMessage)            // Send a new message
	messages.Get("/", middleware.JWTAuth(), messageHandler.GetMessages)                                       // Get messages between users for an item
	messages.Put("/read", middleware.JWTAuth(), middleware.ValidateJSON(), messageHandler.MarkAsRead)         // Mark messages as read

	// Get active chats endpoint (protected)
	chats := api.Group("/active-chats")
	chats.Get("/", middleware.JWTAuth(), messageHandler.GetActiveChats)
}