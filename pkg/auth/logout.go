package auth

import (
	"encoding/json"
	"net/http"
)

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func LogoutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if HandleCORS(w, r) {
			return
		}
		if r.Method != http.MethodPost {
			WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}

		var req LogoutRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
			return
		}

		WriteJSON(w, http.StatusOK, LogoutResponse{
			Message: "Logged out successfully",
		})
	}
}
