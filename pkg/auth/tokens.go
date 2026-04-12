package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims represents the JWT claims for both access and refresh tokens.
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email,omitempty"`
	Type   string `json:"type"`
	jwt.RegisteredClaims
}

// GenerateAccessToken creates a signed JWT access token.
func GenerateAccessToken(userID, email, secret string, expiry time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Email:  email,
		Type:   "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GenerateRefreshToken creates a signed JWT refresh token.
func GenerateRefreshToken(userID, secret string, expiry time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Type:   "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateToken parses and validates a JWT token, returning its claims.
func ValidateToken(tokenString, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parsing token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// ValidateRefreshToken validates a token and checks that it is a refresh token.
func ValidateRefreshToken(tokenString, secret string) (*Claims, error) {
	claims, err := ValidateToken(tokenString, secret)
	if err != nil {
		return nil, err
	}
	if claims.Type != "refresh" {
		return nil, fmt.Errorf("token is not a refresh token")
	}
	return claims, nil
}
