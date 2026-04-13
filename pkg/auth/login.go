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

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func LoginHandler(database *sql.DB, jwtCfg config.JWTConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if HandleCORS(w, r) {
			return
		}
		if r.Method != http.MethodPost {
			WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}

		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
			return
		}

		email := strings.TrimSpace(strings.ToLower(req.Email))

		user, err := db.FindUserByEmail(database, email)
		if err != nil {
			if errors.Is(err, db.ErrUserNotFound) {
				WriteError(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
				return
			}
			WriteError(w, http.StatusInternalServerError, "internal_error", "failed to process login")
			return
		}

		if !VerifyPassword(user.PasswordHash, req.Password) {
			WriteError(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
			return
		}

		accessExpiry := time.Duration(jwtCfg.AccessExpiry) * time.Second
		refreshExpiry := time.Duration(jwtCfg.RefreshExpiry) * time.Second

		accessToken, err := GenerateAccessToken(user.ID, user.Email, jwtCfg.Secret, accessExpiry)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", "failed to issue access token")
			return
		}
		refreshToken, err := GenerateRefreshToken(user.ID, jwtCfg.Secret, refreshExpiry)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", "failed to issue refresh token")
			return
		}

		WriteJSON(w, http.StatusOK, LoginResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresIn:    jwtCfg.AccessExpiry,
			TokenType:    "Bearer",
		})
	}
}
