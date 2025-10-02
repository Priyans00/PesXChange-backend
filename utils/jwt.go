package utils

import (
	"time"

	"pesxchange-backend/config"
	"pesxchange-backend/middleware"
	"pesxchange-backend/models"

	"github.com/golang-jwt/jwt/v5"
)

// GenerateJWT generates a JWT token for a user
func GenerateJWT(user *models.User, cfg *config.Config) (string, error) {
	claims := &middleware.JWTClaims{
		UserID: user.ID,
		SRN:    user.SRN,
		Name:   user.Name,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // 24 hours
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "pesxchange-backend",
			Subject:   user.ID,
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		return "", err
	}
	
	return tokenString, nil
}

// RefreshJWT generates a refresh token with longer expiration
func RefreshJWT(user *models.User, cfg *config.Config) (string, error) {
	claims := &middleware.JWTClaims{
		UserID: user.ID,
		SRN:    user.SRN,
		Name:   user.Name,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)), // 7 days
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "pesxchange-backend",
			Subject:   user.ID,
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		return "", err
	}
	
	return tokenString, nil
}