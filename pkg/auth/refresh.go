package auth

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/edstardo/auth-base/pkg/config"
	"github.com/edstardo/auth-base/pkg/db"
)

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func RefreshHandler(database *sql.DB, jwtCfg config.JWTConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if HandleCORS(w, r) {
			return
		}
		if r.Method != http.MethodPost {
			WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}

		var req RefreshRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
			return
		}

		token := strings.TrimSpace(req.RefreshToken)
		if token == "" {
			WriteError(w, http.StatusUnauthorized, "invalid_token", "refresh token is required")
			return
		}

		claims, err := ValidateRefreshToken(token, jwtCfg.Secret)
		if err != nil {
			WriteError(w, http.StatusUnauthorized, "invalid_token", "invalid or expired refresh token")
			return
		}

		user, err := db.FindUserByID(database, claims.UserID)
		if err != nil {
			if errors.Is(err, db.ErrUserNotFound) {
				WriteError(w, http.StatusUnauthorized, "invalid_token", "invalid or expired refresh token")
				return
			}
			WriteError(w, http.StatusInternalServerError, "internal_error", "failed to process refresh")
			return
		}

		accessExpiry := time.Duration(jwtCfg.AccessExpiry) * time.Second
		accessToken, err := GenerateAccessToken(user.ID, user.Email, jwtCfg.Secret, accessExpiry)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", "failed to issue access token")
			return
		}

		WriteJSON(w, http.StatusOK, RefreshResponse{
			AccessToken: accessToken,
			ExpiresIn:   jwtCfg.AccessExpiry,
			TokenType:   "Bearer",
		})
	}
}
