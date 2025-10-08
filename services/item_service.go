package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"pesxchange-backend/database"
	"pesxchange-backend/models"

	"github.com/google/uuid"
)

type ItemService struct{}

func NewItemService() *ItemService {
	return &ItemService{}
}

// CreateItem creates a new item listing
func (s *ItemService) CreateItem(ctx context.Context, req *models.CreateItemRequest) (*models.Item, error) {
	client := database.GetClient()
	
	now := time.Now()
	
	// Set default values to match Node.js API
	isAvailable := true
	if req.IsAvailable != nil {
		isAvailable = *req.IsAvailable
	}
	
	views := 0
	if req.Views != nil {
		views = *req.Views
	}
	
	// Set default location if empty
	location := strings.TrimSpace(req.Location)
	if location == "" {
		location = "PES University, Bangalore"
	}
	
	item := &models.Item{
		ID:          uuid.New().String(),
		Title:       strings.TrimSpace(req.Title),
		Description: strings.TrimSpace(req.Description),
		Price:       req.Price,
		Location:    location,
		Condition:   req.Condition,
		Images:      req.Images,
		Views:       views,
		IsAvailable: isAvailable,
		IsFeatured:  false,
		SellerID:    req.SellerID,
		CreatedAt:   now,
		UpdatedAt:   now,
		Category:    req.Category,
	}
	
	var newItems []models.Item
	_, _, err := client.From("items").
		Insert(item, false, "", "", "").
		Execute()
	
	if err != nil {
		return nil, fmt.Errorf("failed to create item: %w", err)
	}
	
	if len(newItems) > 0 {
		return &newItems[0], nil
	}
	return item, nil
}

// processItemImages handles image data to prevent huge responses - highly optimized
func (s *ItemService) processItemImages(items []models.Item) {
	for i := range items {
		// Quick check for empty images
		if len(items[i].Images) == 0 {
			items[i].ImageURLs = []string{}
			continue
		}
		
		// For performance, just set first 3 images max
		maxImages := len(items[i].Images)
		if maxImages > 3 {
			maxImages = 3 // Limit to 3 images for listing performance
		}
		
		processedImages := make([]string, 0, maxImages)
		for j := 0; j < maxImages; j++ {
			img := items[i].Images[j]
			
			// Skip large base64 images to prevent huge responses (legacy data)
			if len(img) > 500 && strings.HasPrefix(img, "data:image/") {
				continue
			} else {
				// Keep URLs and small images as is
				processedImages = append(processedImages, img)
			}
		}
		
		items[i].Images = processedImages
		items[i].ImageURLs = processedImages
	}
}

// GetItems retrieves items with pagination and filters  
func (s *ItemService) GetItems(ctx context.Context, limit, offset int, filters map[string]interface{}) ([]models.Item, int, error) {
	client := database.GetClient()
	
	// Select fields - cannot directly join with user_profile, will fetch seller info separately if needed
	query := client.From("items").Select("id,title,description,price,location,condition,seller_id,images,category,created_at,updated_at,is_available,views", "exact", false)
	
	// Apply search filter
	if search, ok := filters["search"].(string); ok && search != "" {
		query = query.Ilike("title", fmt.Sprintf("%%%s%%", search))
	}
	
	// Apply category filter
	if category, ok := filters["category"].(string); ok && category != "" {
		query = query.Eq("category", category)
	}
	
	// Apply condition filter
	if condition, ok := filters["condition"].(string); ok && condition != "" {
		query = query.Eq("condition", condition)
	}
	
	// Apply price range filters
	if minPrice, ok := filters["min_price"].(float64); ok && minPrice > 0 {
		query = query.Gte("price", fmt.Sprintf("%.2f", minPrice))
	}
	if maxPrice, ok := filters["max_price"].(float64); ok && maxPrice > 0 {
		query = query.Lte("price", fmt.Sprintf("%.2f", maxPrice))
	}
	
	// Apply location filter
	if location, ok := filters["location"].(string); ok && location != "" {
		query = query.Ilike("location", fmt.Sprintf("%%%s%%", location))
	}
	
	// Apply sorting
	sortBy := "created_at"
	ascending := false
	if sort, ok := filters["sort"].(string); ok {
		switch sort {
		case "price_asc":
			sortBy = "price"
			ascending = true
		case "price_desc":
			sortBy = "price"
			ascending = false
		case "title":
			sortBy = "title"
			ascending = true
		default:
			sortBy = "created_at"
			ascending = false
		}
	}
	
	if ascending {
		query = query.Order(sortBy, nil)
	} else {
		query = query.Order(sortBy, nil)
	}
	
	// Apply pagination
	query = query.Range(offset, offset+limit-1, "")
	
	var items []models.Item
	data, _, err := query.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get items: %w", err)
	}
	
	// Parse the response data into items slice
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, 0, fmt.Errorf("failed to parse items: %w", err)
	}
	
	// Process images to prevent huge responses
	s.processItemImages(items)
	
	// Add backward compatibility mapping
	for i := range items {
		items[i].ImageURLs = items[i].Images // Map images to image_urls for frontend compatibility
		// Note: categories array is not used in current schema, only single category field
	}
	
	// Get proper total count for pagination
	countQuery := client.From("items").Select("count", "exact", false)
	
	// Apply same filters for count
	if search, ok := filters["search"].(string); ok && search != "" {
		countQuery = countQuery.Ilike("title", fmt.Sprintf("%%%s%%", search))
	}
	if category, ok := filters["category"].(string); ok && category != "" {
		countQuery = countQuery.Eq("category", category)
	}
	if condition, ok := filters["condition"].(string); ok && condition != "" {
		countQuery = countQuery.Eq("condition", condition)
	}
	if minPrice, ok := filters["min_price"].(float64); ok && minPrice > 0 {
		countQuery = countQuery.Gte("price", fmt.Sprintf("%.2f", minPrice))
	}
	if maxPrice, ok := filters["max_price"].(float64); ok && maxPrice > 0 {
		countQuery = countQuery.Lte("price", fmt.Sprintf("%.2f", maxPrice))
	}
	if location, ok := filters["location"].(string); ok && location != "" {
		countQuery = countQuery.Ilike("location", fmt.Sprintf("%%%s%%", location))
	}
	
	countData, _, err := countQuery.Execute()
	if err != nil {
		// If count fails, use length as fallback
		return items, len(items), nil
	}
	
	totalCount := 0
	var countResult []map[string]interface{}
	if err := json.Unmarshal(countData, &countResult); err == nil && len(countResult) > 0 {
		if count, ok := countResult[0]["count"].(float64); ok {
			totalCount = int(count)
		}
	}
	
	// If count failed, fallback to length of current items
	if totalCount == 0 {
		totalCount = len(items)
	}
	
	return items, totalCount, nil
}

// GetItemByID retrieves a single item by ID with seller information
func (s *ItemService) GetItemByID(ctx context.Context, itemID string) (*models.Item, error) {
	client := database.GetClient()
	
	// Fetch item
	var items []models.Item
	data, _, err := client.From("items").
		Select("*", "exact", false).
		Eq("id", itemID).
		Execute()
	
	if err != nil {
		return nil, fmt.Errorf("failed to get item: %w", err)
	}
	
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("failed to parse item: %w", err)
	}
	
	if len(items) == 0 {
		return nil, fmt.Errorf("item not found")
	}
	
	item := &items[0]
	
	// Fetch seller information separately
	if item.SellerID != "" {
		var sellers []models.User
		sellerData, _, err := client.From("user_profiles").
			Select("id, nickname, name, email, avatar_url, rating, location, created_at", "exact", false).
			Eq("id", item.SellerID).
			Execute()
		
		if err == nil && len(sellerData) > 0 {
			if err := json.Unmarshal(sellerData, &sellers); err == nil && len(sellers) > 0 {
				item.Seller = &sellers[0]
			}
		}
	}
	
	// Add backward compatibility mapping
	item.ImageURLs = item.Images
	if item.Category != "" {
		item.Categories = []string{item.Category}
	}
	
	return item, nil
}

// IncrementViews increments the view count for an item
func (s *ItemService) IncrementViews(ctx context.Context, itemID string) error {
	client := database.GetClient()
	
	// Get current views count
	var items []models.Item
	data, _, err := client.From("items").
		Select("views", "exact", false).
		Eq("id", itemID).
		Execute()
	
	if err != nil {
		return fmt.Errorf("failed to get item views: %w", err)
	}
	
	if err := json.Unmarshal(data, &items); err != nil || len(items) == 0 {
		return fmt.Errorf("item not found")
	}
	
	// Increment views
	newViews := items[0].Views + 1
	updates := map[string]interface{}{
		"views": newViews,
	}
	
	_, _, err = client.From("items").
		Update(updates, "", "").
		Eq("id", itemID).
		Execute()
	
	if err != nil {
		return fmt.Errorf("failed to increment views: %w", err)
	}
	
	return nil
}

// UpdateItem updates an existing item
func (s *ItemService) UpdateItem(ctx context.Context, itemID, sellerID string, updates map[string]interface{}) (*models.Item, error) {
	client := database.GetClient()
	
	// Verify ownership
	var items []models.Item
	_, _, err := client.From("items").
		Select("seller_id", "exact", false).
		Eq("id", itemID).
		Execute()
	
	if err != nil {
		return nil, fmt.Errorf("failed to verify item ownership: %w", err)
	}
	
	if len(items) == 0 {
		return nil, fmt.Errorf("item not found")
	}
	
	if items[0].SellerID != sellerID {
		return nil, fmt.Errorf("unauthorized: not the item owner")
	}
	
	// Remove protected fields
	delete(updates, "id")
	delete(updates, "seller_id")
	delete(updates, "created_at")
	
	updates["updated_at"] = time.Now()
	
	var updatedItems []models.Item
	_, _, err = client.From("items").
		Update(updates, "", "").
		Eq("id", itemID).
		Execute()
	
	if err != nil {
		return nil, fmt.Errorf("failed to update item: %w", err)
	}
	
	if len(updatedItems) == 0 {
		return nil, fmt.Errorf("item not found")
	}
	
	return &updatedItems[0], nil
}

// DeleteItem deletes an item (soft delete by changing status)
func (s *ItemService) DeleteItem(ctx context.Context, itemID, sellerID string) error {
	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}
	
	_, err := s.UpdateItem(ctx, itemID, sellerID, updates)
	return err
}

// GetItemsBySeller retrieves items by seller ID
func (s *ItemService) GetItemsBySeller(ctx context.Context, sellerID string, limit, offset int) ([]models.Item, error) {
	client := database.GetClient()
	
	var items []models.Item
	data, _, err := client.From("items").
		Select("*", "exact", false).
		Eq("seller_id", sellerID).
		Order("created_at", nil).
		Range(offset, offset+limit-1, "").
		Execute()
	
	if err != nil {
		return nil, fmt.Errorf("failed to get seller items: %w", err)
	}
	
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("failed to parse seller items: %w", err)
	}
	
	// Fetch seller information once for all items (they all have the same seller)
	if len(items) > 0 && sellerID != "" {
		var sellers []models.User
		sellerData, _, err := client.From("user_profiles").
			Select("id, nickname, name, avatar_url, rating", "exact", false).
			Eq("id", sellerID).
			Execute()
		
		if err == nil && len(sellerData) > 0 {
			if err := json.Unmarshal(sellerData, &sellers); err == nil && len(sellers) > 0 {
				// Attach seller info to all items
				for i := range items {
					items[i].Seller = &sellers[0]
				}
			}
		}
	}
	
	// Process images to prevent huge responses
	s.processItemImages(items)
	
	// Add backward compatibility mapping
	for i := range items {
		items[i].ImageURLs = items[i].Images // Map images to image_urls for frontend compatibility
		// Note: categories array is not used in current schema, only single category field
	}
	
	return items, nil
}