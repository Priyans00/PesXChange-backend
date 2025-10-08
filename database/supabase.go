package database

import (
	"context"
	"fmt"
	"log"

	"pesxchange-backend/config"

	"github.com/supabase-community/supabase-go"
)

var (
	Client        *supabase.Client
	ServiceClient *supabase.Client // Client with service key for storage operations
)

// Initialize sets up the Supabase client
func Initialize(cfg *config.Config) error {
	var err error
	
	// Initialize regular client with anon key
	Client, err = supabase.NewClient(cfg.SupabaseURL, cfg.SupabaseAnonKey, &supabase.ClientOptions{
		Headers: map[string]string{
			"apikey": cfg.SupabaseAnonKey,
		},
	})
	
	if err != nil {
		return err
	}

	// Initialize service client with service key for storage operations
	if cfg.SupabaseServiceKey != "" {
		ServiceClient, err = supabase.NewClient(cfg.SupabaseURL, cfg.SupabaseServiceKey, &supabase.ClientOptions{
			Headers: map[string]string{
				"apikey": cfg.SupabaseServiceKey,
			},
		})
		
		if err != nil {
			log.Printf("Warning: Failed to initialize service client: %v", err)
			ServiceClient = Client // Fallback to regular client
		}
	} else {
		log.Println("Warning: SUPABASE_SERVICE_KEY not set, using anon key for storage operations")
		ServiceClient = Client // Use anon key client
	}

	// Only log in development
	if cfg.IsDevelopment() {
		log.Println("Database client initialized")
	}
	return nil
}

// GetClient returns the initialized Supabase client
func GetClient() *supabase.Client {
	return Client
}

// GetStorageClient returns the service client for storage operations
// Falls back to regular client if service client is not available
func GetStorageClient() *supabase.Client {
	if ServiceClient != nil {
		return ServiceClient
	}
	return Client
}

// HealthCheck verifies the database connection
func HealthCheck(ctx context.Context) error {
	if Client == nil {
		return ErrClientNotInitialized
	}

	// Simple query to check connection
	_, _, err := Client.From("user_profiles").Select("id", "exact", false).Limit(1, "").Execute()
	return err
}

// Error definitions
var (
	ErrClientNotInitialized = fmt.Errorf("supabase client not initialized")
)