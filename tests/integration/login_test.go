package integration

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/edstardo/auth-base/pkg/auth"
	"github.com/edstardo/auth-base/pkg/config"
	"github.com/edstardo/auth-base/pkg/db"
)

const (
	testJWTSecret     = "test_secret_key_minimum_32_chars_long_xyz"
	testAccessExpiry  = 900
	testRefreshExpiry = 604800
)

func testJWTConfig() config.JWTConfig {
	return config.JWTConfig{
		Secret:        testJWTSecret,
		AccessExpiry:  testAccessExpiry,
		RefreshExpiry: testRefreshExpiry,
	}
}

func seedUser(t *testing.T, conn *sql.DB, email, password string) db.User {
	t.Helper()
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("hashing password: %v", err)
	}
	user, err := db.CreateUser(conn, email, hash)
	if err != nil {
		t.Fatalf("seeding user: %v", err)
	}
	return user
}

func postLogin(t *testing.T, handler http.HandlerFunc, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func decodeLoginResponse(t *testing.T, rr *httptest.ResponseRecorder) auth.LoginResponse {
	t.Helper()
	var resp auth.LoginResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	return resp
}

func TestLoginHandler_Success(t *testing.T) {
	conn := testDB(t)
	user := seedUser(t, conn, "alice@example.com", "secure_password_123")
	handler := auth.LoginHandler(conn, testJWTConfig())

	rr := postLogin(t, handler, `{"email":"alice@example.com","password":"secure_password_123"}`)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	resp := decodeLoginResponse(t, rr)
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatal("expected non-empty access and refresh tokens")
	}
	if resp.TokenType != "Bearer" {
		t.Fatalf("expected token type Bearer, got %s", resp.TokenType)
	}
	if resp.ExpiresIn != testAccessExpiry {
		t.Fatalf("expected expires_in %d, got %d", testAccessExpiry, resp.ExpiresIn)
	}

	accessClaims, err := auth.ValidateToken(resp.AccessToken, testJWTSecret)
	if err != nil {
		t.Fatalf("access token invalid: %v", err)
	}
	if accessClaims.Type != "access" {
		t.Fatalf("expected access type, got %s", accessClaims.Type)
	}
	if accessClaims.UserID != user.ID {
		t.Fatalf("expected user ID %s, got %s", user.ID, accessClaims.UserID)
	}
	if accessClaims.Email != "alice@example.com" {
		t.Fatalf("expected email alice@example.com, got %s", accessClaims.Email)
	}

	refreshClaims, err := auth.ValidateRefreshToken(resp.RefreshToken, testJWTSecret)
	if err != nil {
		t.Fatalf("refresh token invalid: %v", err)
	}
	if refreshClaims.UserID != user.ID {
		t.Fatalf("refresh token user ID mismatch: %s", refreshClaims.UserID)
	}
}

func TestLoginHandler_NormalizesEmail(t *testing.T) {
	conn := testDB(t)
	seedUser(t, conn, "alice@example.com", "secure_password_123")
	handler := auth.LoginHandler(conn, testJWTConfig())

	rr := postLogin(t, handler, `{"email":"  Alice@Example.COM  ","password":"secure_password_123"}`)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestLoginHandler_WrongPassword(t *testing.T) {
	conn := testDB(t)
	seedUser(t, conn, "alice@example.com", "secure_password_123")
	handler := auth.LoginHandler(conn, testJWTConfig())

	rr := postLogin(t, handler, `{"email":"alice@example.com","password":"wrong_password_abc"}`)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
	if got := decodeErrorResponse(t, rr).Error; got != "invalid_credentials" {
		t.Fatalf("expected error code invalid_credentials, got %s", got)
	}
}

func TestLoginHandler_UnknownEmail(t *testing.T) {
	conn := testDB(t)
	handler := auth.LoginHandler(conn, testJWTConfig())

	rr := postLogin(t, handler, `{"email":"ghost@example.com","password":"secure_password_123"}`)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
	if got := decodeErrorResponse(t, rr).Error; got != "invalid_credentials" {
		t.Fatalf("expected error code invalid_credentials, got %s", got)
	}
}

func TestLoginHandler_InvalidJSON(t *testing.T) {
	conn := testDB(t)
	handler := auth.LoginHandler(conn, testJWTConfig())

	rr := postLogin(t, handler, `{"email":`)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if got := decodeErrorResponse(t, rr).Error; got != "invalid_request" {
		t.Fatalf("expected error code invalid_request, got %s", got)
	}
}

func TestLoginHandler_MethodNotAllowed(t *testing.T) {
	conn := testDB(t)
	handler := auth.LoginHandler(conn, testJWTConfig())

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

func TestLoginHandler_CORSPreflight(t *testing.T) {
	conn := testDB(t)
	handler := auth.LoginHandler(conn, testJWTConfig())

	req := httptest.NewRequest(http.MethodOptions, "/login", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected CORS origin *, got %q", got)
	}
}
