// Package auth provides JWT token management and authentication utilities.
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	// ErrInvalidToken is returned when a token is invalid
	ErrInvalidToken = errors.New("invalid token")
	// ErrExpiredToken is returned when a token has expired
	ErrExpiredToken = errors.New("token has expired")
)

// JWTClaims represents the JWT claims structure used for authentication tokens.
// It includes the user ID and username along with standard JWT registered claims.
type JWTClaims struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string     `json:"username"`
	jwt.RegisteredClaims
}

// TokenManager handles JWT token generation and validation operations.
// It manages tokens using a secret key for signing and verification.
type TokenManager struct {
	secretKey []byte
}

// NewTokenManager creates a new TokenManager with the provided secret key.
// The secret key is used to sign and verify JWT tokens.
func NewTokenManager(secretKey string) *TokenManager {
	return &TokenManager{
		secretKey: []byte(secretKey),
	}
}

// GenerateToken generates a JWT token for a user.
// The token is valid for 24 hours and includes the user ID and username in the claims.
// Returns the token string and any error that occurred during generation.
func (tm *TokenManager) GenerateToken(userID uuid.UUID, username string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour) // Token valid for 24 hours
	
	claims := &JWTClaims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "terminal.sh",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(tm.secretKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token string and returns the claims if valid.
// Returns ErrInvalidToken if the token is malformed or invalid, or ErrExpiredToken
// if the token has expired. Returns the claims and nil error if validation succeeds.
func (tm *TokenManager) ValidateToken(tokenString string) (*JWTClaims, error) {
	claims := &JWTClaims{}
	
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return tm.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

