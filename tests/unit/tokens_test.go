package unit

import (
	"strings"
	"testing"
	"time"

	"github.com/edstardo/auth-base/pkg/auth"
)

const testSecret = "test_secret_key_minimum_32_chars_long_xyz"

func TestGenerateAccessToken_Valid(t *testing.T) {
	token, err := auth.GenerateAccessToken("user-123", "user@example.com", testSecret, 15*time.Minute)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts, got %d", len(parts))
	}
}

func TestGenerateAccessToken_Claims(t *testing.T) {
	token, err := auth.GenerateAccessToken("user-123", "user@example.com", testSecret, 15*time.Minute)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	claims, err := auth.ValidateToken(token, testSecret)
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	if claims.UserID != "user-123" {
		t.Errorf("expected UserID user-123, got %s", claims.UserID)
	}
	if claims.Email != "user@example.com" {
		t.Errorf("expected Email user@example.com, got %s", claims.Email)
	}
	if claims.Type != "access" {
		t.Errorf("expected Type access, got %s", claims.Type)
	}
}

func TestGenerateRefreshToken_Valid(t *testing.T) {
	token, err := auth.GenerateRefreshToken("user-123", testSecret, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts, got %d", len(parts))
	}
}

func TestGenerateRefreshToken_Claims(t *testing.T) {
	token, err := auth.GenerateRefreshToken("user-123", testSecret, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	claims, err := auth.ValidateToken(token, testSecret)
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	if claims.UserID != "user-123" {
		t.Errorf("expected UserID user-123, got %s", claims.UserID)
	}
	if claims.Email != "" {
		t.Errorf("expected empty Email for refresh token, got %s", claims.Email)
	}
	if claims.Type != "refresh" {
		t.Errorf("expected Type refresh, got %s", claims.Type)
	}
}

func TestValidateToken_InvalidSignature(t *testing.T) {
	token, err := auth.GenerateAccessToken("user-123", "user@example.com", testSecret, 15*time.Minute)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	_, err = auth.ValidateToken(token, "wrong_secret_key_that_is_32_chars_long")
	if err == nil {
		t.Fatal("expected error for invalid signature")
	}
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	token, err := auth.GenerateAccessToken("user-123", "user@example.com", testSecret, -1*time.Minute)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	_, err = auth.ValidateToken(token, testSecret)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestValidateToken_MalformedToken(t *testing.T) {
	_, err := auth.ValidateToken("not.a.token", testSecret)
	if err == nil {
		t.Fatal("expected error for malformed token")
	}
}

func TestValidateRefreshToken_Valid(t *testing.T) {
	token, err := auth.GenerateRefreshToken("user-123", testSecret, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	claims, err := auth.ValidateRefreshToken(token, testSecret)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if claims.Type != "refresh" {
		t.Errorf("expected Type refresh, got %s", claims.Type)
	}
}

func TestValidateRefreshToken_RejectsAccessToken(t *testing.T) {
	token, err := auth.GenerateAccessToken("user-123", "user@example.com", testSecret, 15*time.Minute)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	_, err = auth.ValidateRefreshToken(token, testSecret)
	if err == nil {
		t.Fatal("expected error when validating access token as refresh token")
	}
}
