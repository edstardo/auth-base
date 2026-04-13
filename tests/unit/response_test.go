package unit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/edstardo/auth-base/pkg/auth"
)

func TestWriteJSON_SetsHeadersStatusAndBody(t *testing.T) {
	rr := httptest.NewRecorder()
	payload := auth.LogoutResponse{Message: "ok"}

	auth.WriteJSON(rr, http.StatusAccepted, payload)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}
	if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Fatalf("expected CORS origin *, got %q", origin)
	}

	var got auth.LogoutResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("expected valid JSON body, got error: %v", err)
	}
	if got.Message != "ok" {
		t.Fatalf("expected message %q, got %q", "ok", got.Message)
	}
}

func TestWriteJSON_NilPayloadWritesNoBody(t *testing.T) {
	rr := httptest.NewRecorder()

	auth.WriteJSON(rr, http.StatusNoContent, nil)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rr.Code)
	}
	if rr.Body.Len() != 0 {
		t.Fatalf("expected empty body, got %q", rr.Body.String())
	}
}

func TestWriteError_EncodesErrorResponse(t *testing.T) {
	rr := httptest.NewRecorder()

	auth.WriteError(rr, http.StatusBadRequest, "invalid_email", "email format is invalid")

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var got auth.ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("expected valid JSON body, got error: %v", err)
	}
	if got.Error != "invalid_email" {
		t.Fatalf("expected error code %q, got %q", "invalid_email", got.Error)
	}
	if got.Message != "email format is invalid" {
		t.Fatalf("expected message %q, got %q", "email format is invalid", got.Message)
	}
}

func TestHandleCORS_OptionsRequestHandled(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/signup", nil)

	handled := auth.HandleCORS(rr, req)

	if !handled {
		t.Fatal("expected HandleCORS to return true for OPTIONS")
	}
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rr.Code)
	}
	if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Fatalf("expected CORS origin *, got %q", origin)
	}
	if methods := rr.Header().Get("Access-Control-Allow-Methods"); methods == "" {
		t.Fatal("expected Access-Control-Allow-Methods to be set")
	}
}

func TestHandleCORS_NonOptionsPassthrough(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/signup", nil)

	handled := auth.HandleCORS(rr, req)

	if handled {
		t.Fatal("expected HandleCORS to return false for non-OPTIONS")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("expected untouched default status 200, got %d", rr.Code)
	}
}
