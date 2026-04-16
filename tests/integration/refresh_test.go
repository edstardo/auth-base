package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/edstardo/auth-base/pkg/auth"
)

func postRefresh(t *testing.T, handler http.HandlerFunc, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func decodeRefreshResponse(t *testing.T, rr *httptest.ResponseRecorder) auth.RefreshResponse {
	t.Helper()
	var resp auth.RefreshResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	return resp
}

func issueRefreshToken(t *testing.T, userID string, expiry time.Duration) string {
	t.Helper()
	token, err := auth.GenerateRefreshToken(userID, testJWTSecret, expiry)
	if err != nil {
		t.Fatalf("generating refresh token: %v", err)
	}
	return token
}

func TestRefreshHandler_Success(t *testing.T) {
	conn := testDB(t)
	user := seedUser(t, conn, "alice@example.com", "secure_password_123")
	handler := auth.RefreshHandler(conn, testJWTConfig())

	refreshToken := issueRefreshToken(t, user.ID, time.Duration(testRefreshExpiry)*time.Second)

	body, err := json.Marshal(auth.RefreshRequest{RefreshToken: refreshToken})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	rr := postRefresh(t, handler, string(body))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	resp := decodeRefreshResponse(t, rr)
	if resp.AccessToken == "" {
		t.Fatal("expected non-empty access token")
	}
	if resp.TokenType != "Bearer" {
		t.Fatalf("expected token type Bearer, got %s", resp.TokenType)
	}
	if resp.ExpiresIn != testAccessExpiry {
		t.Fatalf("expected expires_in %d, got %d", testAccessExpiry, resp.ExpiresIn)
	}

	claims, err := auth.ValidateToken(resp.AccessToken, testJWTSecret)
	if err != nil {
		t.Fatalf("access token invalid: %v", err)
	}
	if claims.Type != "access" {
		t.Fatalf("expected access type, got %s", claims.Type)
	}
	if claims.UserID != user.ID {
		t.Fatalf("expected user ID %s, got %s", user.ID, claims.UserID)
	}
	if claims.Email != user.Email {
		t.Fatalf("expected email %s, got %s", user.Email, claims.Email)
	}
}

func TestRefreshHandler_TrimsWhitespace(t *testing.T) {
	conn := testDB(t)
	user := seedUser(t, conn, "alice@example.com", "secure_password_123")
	handler := auth.RefreshHandler(conn, testJWTConfig())

	refreshToken := issueRefreshToken(t, user.ID, time.Duration(testRefreshExpiry)*time.Second)
	body, _ := json.Marshal(auth.RefreshRequest{RefreshToken: "  " + refreshToken + "  "})

	rr := postRefresh(t, handler, string(body))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestRefreshHandler_EmptyToken(t *testing.T) {
	conn := testDB(t)
	handler := auth.RefreshHandler(conn, testJWTConfig())

	rr := postRefresh(t, handler, `{"refresh_token":"   "}`)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
	if got := decodeErrorResponse(t, rr).Error; got != "invalid_token" {
		t.Fatalf("expected error code invalid_token, got %s", got)
	}
}

func TestRefreshHandler_MissingTokenField(t *testing.T) {
	conn := testDB(t)
	handler := auth.RefreshHandler(conn, testJWTConfig())

	rr := postRefresh(t, handler, `{}`)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
	if got := decodeErrorResponse(t, rr).Error; got != "invalid_token" {
		t.Fatalf("expected error code invalid_token, got %s", got)
	}
}

func TestRefreshHandler_MalformedToken(t *testing.T) {
	conn := testDB(t)
	handler := auth.RefreshHandler(conn, testJWTConfig())

	rr := postRefresh(t, handler, `{"refresh_token":"not.a.valid.jwt"}`)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
	if got := decodeErrorResponse(t, rr).Error; got != "invalid_token" {
		t.Fatalf("expected error code invalid_token, got %s", got)
	}
}

func TestRefreshHandler_WrongSecret(t *testing.T) {
	conn := testDB(t)
	user := seedUser(t, conn, "alice@example.com", "secure_password_123")
	handler := auth.RefreshHandler(conn, testJWTConfig())

	// Sign token with a different secret so signature verification fails.
	badToken, err := auth.GenerateRefreshToken(user.ID, "a_completely_different_secret_value", time.Hour)
	if err != nil {
		t.Fatalf("generating token: %v", err)
	}
	body, _ := json.Marshal(auth.RefreshRequest{RefreshToken: badToken})

	rr := postRefresh(t, handler, string(body))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
	if got := decodeErrorResponse(t, rr).Error; got != "invalid_token" {
		t.Fatalf("expected error code invalid_token, got %s", got)
	}
}

func TestRefreshHandler_ExpiredToken(t *testing.T) {
	conn := testDB(t)
	user := seedUser(t, conn, "alice@example.com", "secure_password_123")
	handler := auth.RefreshHandler(conn, testJWTConfig())

	expiredToken := issueRefreshToken(t, user.ID, -time.Hour)
	body, _ := json.Marshal(auth.RefreshRequest{RefreshToken: expiredToken})

	rr := postRefresh(t, handler, string(body))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
	if got := decodeErrorResponse(t, rr).Error; got != "invalid_token" {
		t.Fatalf("expected error code invalid_token, got %s", got)
	}
}

func TestRefreshHandler_RejectsAccessToken(t *testing.T) {
	conn := testDB(t)
	user := seedUser(t, conn, "alice@example.com", "secure_password_123")
	handler := auth.RefreshHandler(conn, testJWTConfig())

	accessToken, err := auth.GenerateAccessToken(user.ID, user.Email, testJWTSecret, time.Hour)
	if err != nil {
		t.Fatalf("generating access token: %v", err)
	}
	body, _ := json.Marshal(auth.RefreshRequest{RefreshToken: accessToken})

	rr := postRefresh(t, handler, string(body))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
	if got := decodeErrorResponse(t, rr).Error; got != "invalid_token" {
		t.Fatalf("expected error code invalid_token, got %s", got)
	}
}

func TestRefreshHandler_UserNoLongerExists(t *testing.T) {
	conn := testDB(t)
	handler := auth.RefreshHandler(conn, testJWTConfig())

	// Valid refresh token with a user ID that does not exist in the database.
	token := issueRefreshToken(t, "00000000-0000-0000-0000-000000000000", time.Hour)
	body, _ := json.Marshal(auth.RefreshRequest{RefreshToken: token})

	rr := postRefresh(t, handler, string(body))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
	if got := decodeErrorResponse(t, rr).Error; got != "invalid_token" {
		t.Fatalf("expected error code invalid_token, got %s", got)
	}
}

func TestRefreshHandler_InvalidJSON(t *testing.T) {
	conn := testDB(t)
	handler := auth.RefreshHandler(conn, testJWTConfig())

	rr := postRefresh(t, handler, `{"refresh_token":`)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if got := decodeErrorResponse(t, rr).Error; got != "invalid_request" {
		t.Fatalf("expected error code invalid_request, got %s", got)
	}
}

func TestRefreshHandler_MethodNotAllowed(t *testing.T) {
	conn := testDB(t)
	handler := auth.RefreshHandler(conn, testJWTConfig())

	req := httptest.NewRequest(http.MethodGet, "/refresh", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

func TestRefreshHandler_CORSPreflight(t *testing.T) {
	conn := testDB(t)
	handler := auth.RefreshHandler(conn, testJWTConfig())

	req := httptest.NewRequest(http.MethodOptions, "/refresh", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected CORS origin *, got %q", got)
	}
}

func TestRefreshHandler_DifferentFromPreviousAccessToken(t *testing.T) {
	conn := testDB(t)
	user := seedUser(t, conn, "alice@example.com", "secure_password_123")
	handler := auth.RefreshHandler(conn, testJWTConfig())

	prevAccess, err := auth.GenerateAccessToken(user.ID, user.Email, testJWTSecret, time.Duration(testAccessExpiry)*time.Second)
	if err != nil {
		t.Fatalf("generating initial access token: %v", err)
	}

	// Ensure issuance timestamps differ so token strings don't collide.
	time.Sleep(time.Second)

	refreshToken := issueRefreshToken(t, user.ID, time.Duration(testRefreshExpiry)*time.Second)
	body, _ := json.Marshal(auth.RefreshRequest{RefreshToken: refreshToken})
	rr := postRefresh(t, handler, string(body))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if got := decodeRefreshResponse(t, rr).AccessToken; got == prevAccess {
		t.Fatal("expected refreshed access token to differ from previous")
	}
}
