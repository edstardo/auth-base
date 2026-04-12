package unit

import (
	"testing"

	"github.com/edstardo/auth-base/pkg/auth"
)

func TestHashPassword_Valid(t *testing.T) {
	hash, err := auth.HashPassword("validpass123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(hash) == 0 {
		t.Fatal("expected non-empty hash")
	}
	// bcrypt hashes start with $2a$ or $2b$
	if hash[0] != '$' {
		t.Fatalf("expected bcrypt format, got %q", hash)
	}
}

func TestHashPassword_TooShort(t *testing.T) {
	_, err := auth.HashPassword("short")
	if err == nil {
		t.Fatal("expected error for password < 8 chars")
	}
}

func TestHashPassword_TooLong(t *testing.T) {
	long := make([]byte, 51)
	for i := range long {
		long[i] = 'a'
	}
	_, err := auth.HashPassword(string(long))
	if err == nil {
		t.Fatal("expected error for password > 50 chars")
	}
}

func TestHashPassword_ExactMinLength(t *testing.T) {
	_, err := auth.HashPassword("12345678") // exactly 8
	if err != nil {
		t.Fatalf("expected no error for 8-char password, got %v", err)
	}
}

func TestHashPassword_ExactMaxLength(t *testing.T) {
	pw := make([]byte, 50)
	for i := range pw {
		pw[i] = 'a'
	}
	_, err := auth.HashPassword(string(pw))
	if err != nil {
		t.Fatalf("expected no error for 50-char password, got %v", err)
	}
}

func TestHashPassword_UniqueHashes(t *testing.T) {
	hash1, _ := auth.HashPassword("samepassword")
	hash2, _ := auth.HashPassword("samepassword")
	if hash1 == hash2 {
		t.Fatal("expected different hashes for same password (unique salt)")
	}
}

func TestVerifyPassword_Correct(t *testing.T) {
	password := "correctpass1"
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("hash failed: %v", err)
	}
	if !auth.VerifyPassword(hash, password) {
		t.Fatal("expected true for correct password")
	}
}

func TestVerifyPassword_Wrong(t *testing.T) {
	hash, err := auth.HashPassword("correctpass1")
	if err != nil {
		t.Fatalf("hash failed: %v", err)
	}
	if auth.VerifyPassword(hash, "wrongpassword") {
		t.Fatal("expected false for wrong password")
	}
}

func TestVerifyPassword_InvalidHash(t *testing.T) {
	if auth.VerifyPassword("not-a-real-hash", "anypassword") {
		t.Fatal("expected false for invalid hash")
	}
}
