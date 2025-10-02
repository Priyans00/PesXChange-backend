package database

import (
	"context"
	"fmt"
	"log"

	"pesxchange-backend/config"

	"github.com/supabase-community/supabase-go"
)

var (
	Client *supabase.Client
)

// Initialize sets up the Supabase client
func Initialize(cfg *config.Config) error {
	var err error
	
	Client, err = supabase.NewClient(cfg.SupabaseURL, cfg.SupabaseAnonKey, &supabase.ClientOptions{
		Headers: map[string]string{
			"apikey": cfg.SupabaseAnonKey,
		},
	})
	
	if err != nil {
		return err
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