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

type UserService struct{}

func NewUserService() *UserService {
	return &UserService{}
}

// UpsertUser creates or updates a user profile in Supabase
func (s *UserService) UpsertUser(ctx context.Context, profile *models.PESUProfile) (*models.User, error) {
	client := database.GetClient()
	
	// First, check if user exists by SRN
	var existingUsers []models.User
	data, _, err := client.From("user_profiles").
		Select("*", "exact", false).
		Eq("srn", profile.SRN).
		Execute()
	
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	
	if err := json.Unmarshal(data, &existingUsers); err != nil {
		return nil, fmt.Errorf("failed to parse existing user: %w", err)
	}
	
	now := time.Now()
	
	// If user exists, update their information
	if len(existingUsers) > 0 {
		existingUser := existingUsers[0]
		
		// Update with latest profile information
		updatedUser := &models.User{
			ID:          existingUser.ID, // Keep existing ID
			SRN:         profile.SRN,
			PRN:         profile.PRN,
			Name:        profile.Name,
			Email:       profile.Email,
			Phone:       profile.Phone,
			Bio:         existingUser.Bio,         // Keep existing bio
			AvatarURL:   existingUser.AvatarURL,   // Keep existing avatar
			Program:     profile.Program,
			Branch:      profile.Branch,
			Semester:    profile.Semester,
			Section:     profile.Section,
			CampusCode:  &profile.CampusCode,
			Campus:      profile.Campus,
			Rating:      existingUser.Rating,      // Keep existing rating
			Verified:    true, // PESU authenticated users are verified
			Location:    existingUser.Location,    // Keep existing location
			CreatedAt:   existingUser.CreatedAt,   // Keep original creation time
			UpdatedAt:   now,
			LastLogin:   &now,
			Nickname:    existingUser.Nickname,    // Keep existing nickname
		}
		
		// Update the user in the database
		_, _, err = client.From("user_profiles").
			Update(updatedUser, "", "").
			Eq("id", existingUser.ID).
			Execute()
		
		if err != nil {
			return nil, fmt.Errorf("failed to update user: %w", err)
		}
		
		return updatedUser, nil
	}
	
	// User doesn't exist, create new user
	newUser := &models.User{
		ID:          uuid.New().String(),
		SRN:         profile.SRN,
		PRN:         profile.PRN,
		Name:        profile.Name,
		Email:       profile.Email,
		Phone:       profile.Phone,
		Bio:         "", // Empty bio initially
		AvatarURL:   "", // Empty avatar URL initially
		Program:     profile.Program,
		Branch:      profile.Branch,
		Semester:    profile.Semester,
		Section:     profile.Section,
		CampusCode:  &profile.CampusCode,
		Campus:      profile.Campus,
		Rating:      0.0,  // Default rating
		Verified:    true, // PESU authenticated users are verified
		Location:    "PES University, Bangalore", // Default location
		CreatedAt:   now,
		UpdatedAt:   now,
		LastLogin:   &now,
		Nickname:    "", // Empty nickname initially
	}
	
	// Insert new user into database
	_, _, err = client.From("user_profiles").
		Insert(newUser, false, "", "", "").
		Execute()
	
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	
	return newUser, nil
}

// GetUserByID retrieves a user by ID
func (s *UserService) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	client := database.GetClient()
	
	var users []models.User
	data, _, err := client.From("user_profiles").
		Select("*", "exact", false).
		Eq("id", userID).
		Execute()
	
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	
	if err := json.Unmarshal(data, &users); err != nil {
		return nil, fmt.Errorf("failed to parse user: %w", err)
	}
	
	if len(users) == 0 {
		return nil, fmt.Errorf("user not found")
	}
	
	return &users[0], nil
}

// GetUserBySRN retrieves a user by SRN
func (s *UserService) GetUserBySRN(ctx context.Context, srn string) (*models.User, error) {
	client := database.GetClient()
	
	var users []models.User
	data, _, err := client.From("users").
		Select("*", "exact", false).
		Eq("srn", srn).
		Execute()
	
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	
	if err := json.Unmarshal(data, &users); err != nil {
		return nil, fmt.Errorf("failed to parse user: %w", err)
	}
	
	if len(users) == 0 {
		return nil, fmt.Errorf("user not found")
	}
	
	return &users[0], nil
}

// CheckSRNExists checks if an SRN exists in the database
func (s *UserService) CheckSRNExists(ctx context.Context, srn string) (bool, error) {
	client := database.GetClient()
	
	var users []models.User
	data, _, err := client.From("users").
		Select("id", "exact", false).
		Eq("srn", srn).
		Limit(1, "").
		Execute()
	
	if err != nil {
		return false, fmt.Errorf("failed to check SRN: %w", err)
	}
	
	if err := json.Unmarshal(data, &users); err != nil {
		return false, fmt.Errorf("failed to parse SRN check: %w", err)
	}
	
	return len(users) > 0, nil
}

// UpdateUserProfile updates user profile information
func (s *UserService) UpdateUserProfile(ctx context.Context, userID string, updates map[string]interface{}) (*models.User, error) {
	client := database.GetClient()
	
	updates["updated_at"] = time.Now()
	
	var updatedUsers []models.User
	data, _, err := client.From("user_profiles").
		Update(updates, "", "").
		Eq("id", userID).
		Execute()
	
	if err != nil {
		return nil, fmt.Errorf("failed to update user profile: %w", err)
	}
	
	if err := json.Unmarshal(data, &updatedUsers); err != nil {
		return nil, fmt.Errorf("failed to parse updated user: %w", err)
	}
	
	if len(updatedUsers) == 0 {
		return nil, fmt.Errorf("user not found")
	}
	
	return &updatedUsers[0], nil
}