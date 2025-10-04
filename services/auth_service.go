package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"pesxchange-backend/config"
	"pesxchange-backend/models"
)

type AuthService struct {
	config      *config.Config
	userService *UserService
}

func NewAuthService(cfg *config.Config, userService *UserService) *AuthService {
	return &AuthService{
		config:      cfg,
		userService: userService,
	}
}

// ValidateSRN validates SRN format
func (s *AuthService) ValidateSRN(srn string) bool {
	srnPattern := regexp.MustCompile(`^PES\d{1}[A-Z]{2}\d{2}[A-Z]{2}\d{3}$`)
	return srnPattern.MatchString(strings.ToUpper(srn))
}

// AuthenticateWithPESU authenticates user with PESU API
func (s *AuthService) AuthenticateWithPESU(ctx context.Context, req *models.PESUAuthRequest) (*models.User, error) {
	// Validate SRN format
	if !s.ValidateSRN(req.Username) {
		return nil, fmt.Errorf("invalid SRN format")
	}
	
	// Basic input validation - trim whitespace but don't remove valid characters
	username := strings.TrimSpace(req.Username)
	password := strings.TrimSpace(req.Password)
	
	// Check for empty inputs
	if username == "" || password == "" {
		return nil, fmt.Errorf("username and password are required")
	}
	
	// Limit length for security (reasonable limits)
	if len(username) > 20 || len(password) > 128 {
		return nil, fmt.Errorf("input length exceeds maximum allowed")
	}
	
	// Prepare request to PESU API
	authReq := map[string]interface{}{
		"username": strings.ToUpper(username),
		"password": password,
		"profile":  true,
	}
	
	jsonData, err := json.Marshal(authReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// Create HTTP client with optimized timeout and transport settings
	client := &http.Client{
		Timeout: 15 * time.Second, // Reduced timeout for better performance
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     30 * time.Second,
		},
	}
	
	// Make request to PESU API
	authURL := fmt.Sprintf("%s/authenticate", s.config.PESUAuthURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", authURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "PesXChange-Backend/1.0")
	
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to authentication service: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("authentication service unavailable (status: %d)", resp.StatusCode)
	}
	
	// Parse response
	var authResp models.PESUAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, fmt.Errorf("invalid authentication response: %w", err)
	}
	
	if !authResp.Status {
		return nil, fmt.Errorf("authentication failed: %s", authResp.Message)
	}
	
	if authResp.Profile == nil {
		return nil, fmt.Errorf("profile information not available")
	}
	
	// Create or update user profile
	user, err := s.userService.UpsertUser(ctx, authResp.Profile)
	if err != nil {
		return nil, fmt.Errorf("failed to create/update user profile: %w", err)
	}
	
	return user, nil
}