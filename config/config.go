package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                string
	SupabaseURL         string
	SupabaseAnonKey     string
	SupabaseServiceKey  string
	JWTSecret           string
	AllowedOrigins      string
	Environment         string
	PESUAuthURL         string
	RateLimitMax        int
	RateLimitWindow     int
}

func Load() *Config {
	if err := godotenv.Load(); err != nil && os.Getenv("ENVIRONMENT") == "development" {
		log.Println("No .env file found, using environment variables")
	}

	rateLimitMax, _ := strconv.Atoi(getEnv("RATE_LIMIT_MAX", "100"))
	rateLimitWindow, _ := strconv.Atoi(getEnv("RATE_LIMIT_WINDOW", "3600"))

	// Validate required environment variables
	jwtSecret := getEnv("JWT_SECRET", "")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}
	if len(jwtSecret) < 32 {
		log.Fatal("JWT_SECRET must be at least 32 characters long")
	}

	supabaseURL := getEnv("SUPABASE_URL", "")
	supabaseAnonKey := getEnv("SUPABASE_ANON_KEY", "")
	if supabaseURL == "" || supabaseAnonKey == "" {
		log.Fatal("SUPABASE_URL and SUPABASE_ANON_KEY environment variables are required")
	}

	return &Config{
		Port:                getEnv("PORT", "8080"),
		SupabaseURL:         supabaseURL,
		SupabaseAnonKey:     supabaseAnonKey,
		SupabaseServiceKey:  getEnv("SUPABASE_SERVICE_KEY", ""),
		JWTSecret:           jwtSecret,
		AllowedOrigins:      getEnv("ALLOWED_ORIGINS", "http://localhost:3000"),
		Environment:         getEnv("ENVIRONMENT", "production"),
		PESUAuthURL:         getEnv("PESU_AUTH_URL", "https://pesu-auth.onrender.com"),
		RateLimitMax:        rateLimitMax,
		RateLimitWindow:     rateLimitWindow,
	}
}

func (c *Config) IsDevelopment() bool {
	return strings.ToLower(c.Environment) == "development"
}

func (c *Config) IsProduction() bool {
	return strings.ToLower(c.Environment) == "production"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}