package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/edstardo/auth-base/pkg/auth"
	"github.com/edstardo/auth-base/pkg/db"
)

func postSignup(t *testing.T, handler http.HandlerFunc, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func decodeSignupResponse(t *testing.T, rr *httptest.ResponseRecorder) auth.SignupResponse {
	t.Helper()
	var resp auth.SignupResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	return resp
}

func decodeErrorResponse(t *testing.T, rr *httptest.ResponseRecorder) auth.ErrorResponse {
	t.Helper()
	var resp auth.ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding error response: %v", err)
	}
	return resp
}

func TestSignupHandler_Success(t *testing.T) {
	conn := testDB(t)
	handler := auth.SignupHandler(conn)

	rr := postSignup(t, handler, `{"email":"alice@example.com","password":"secure_password_123"}`)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	resp := decodeSignupResponse(t, rr)
	if resp.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if resp.Email != "alice@example.com" {
		t.Fatalf("expected email alice@example.com, got %s", resp.Email)
	}
	if resp.CreatedAt.IsZero() {
		t.Fatal("expected non-zero created_at")
	}

	// Verify persisted with hashed password (not plaintext).
	user, err := db.FindUserByEmail(conn, "alice@example.com")
	if err != nil {
		t.Fatalf("expected user to exist: %v", err)
	}
	if user.PasswordHash == "secure_password_123" {
		t.Fatal("password was stored as plaintext")
	}
	if !auth.VerifyPassword(user.PasswordHash, "secure_password_123") {
		t.Fatal("stored hash does not verify against original password")
	}
}

func TestSignupHandler_NormalizesEmail(t *testing.T) {
	conn := testDB(t)
	handler := auth.SignupHandler(conn)

	rr := postSignup(t, handler, `{"email":"  Alice@Example.COM  ","password":"secure_password_123"}`)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	resp := decodeSignupResponse(t, rr)
	if resp.Email != "alice@example.com" {
		t.Fatalf("expected normalized email, got %s", resp.Email)
	}
}

func TestSignupHandler_InvalidEmail(t *testing.T) {
	conn := testDB(t)
	handler := auth.SignupHandler(conn)

	rr := postSignup(t, handler, `{"email":"not-an-email","password":"secure_password_123"}`)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if got := decodeErrorResponse(t, rr).Error; got != "invalid_email" {
		t.Fatalf("expected error code invalid_email, got %s", got)
	}
}

func TestSignupHandler_ShortPassword(t *testing.T) {
	conn := testDB(t)
	handler := auth.SignupHandler(conn)

	rr := postSignup(t, handler, `{"email":"bob@example.com","password":"short"}`)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if got := decodeErrorResponse(t, rr).Error; got != "invalid_password" {
		t.Fatalf("expected error code invalid_password, got %s", got)
	}
}

func TestSignupHandler_DuplicateEmail(t *testing.T) {
	conn := testDB(t)
	handler := auth.SignupHandler(conn)

	body := `{"email":"dup@example.com","password":"secure_password_123"}`
	if rr := postSignup(t, handler, body); rr.Code != http.StatusCreated {
		t.Fatalf("first signup expected 201, got %d", rr.Code)
	}

	rr := postSignup(t, handler, body)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rr.Code)
	}
	if got := decodeErrorResponse(t, rr).Error; got != "email_already_exists" {
		t.Fatalf("expected error code email_already_exists, got %s", got)
	}
}

func TestSignupHandler_InvalidJSON(t *testing.T) {
	conn := testDB(t)
	handler := auth.SignupHandler(conn)

	req := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBufferString(`{"email":`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if got := decodeErrorResponse(t, rr).Error; got != "invalid_request" {
		t.Fatalf("expected error code invalid_request, got %s", got)
	}
}

func TestSignupHandler_MethodNotAllowed(t *testing.T) {
	conn := testDB(t)
	handler := auth.SignupHandler(conn)

	req := httptest.NewRequest(http.MethodGet, "/signup", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

func TestSignupHandler_CORSPreflight(t *testing.T) {
	conn := testDB(t)
	handler := auth.SignupHandler(conn)

	req := httptest.NewRequest(http.MethodOptions, "/signup", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected CORS origin *, got %q", got)
	}
}
