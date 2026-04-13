package auth

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/edstardo/auth-base/pkg/db"
)

type SignupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func SignupHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if HandleCORS(w, r) {
			return
		}
		if r.Method != http.MethodPost {
			WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}

		var req SignupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
			return
		}

		email := strings.TrimSpace(strings.ToLower(req.Email))
		if !emailRegex.MatchString(email) {
			WriteError(w, http.StatusBadRequest, "invalid_email", "invalid email format")
			return
		}

		hash, err := HashPassword(req.Password)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_password", err.Error())
			return
		}

		user, err := db.CreateUser(database, email, hash)
		if err != nil {
			if errors.Is(err, db.ErrEmailAlreadyRegistered) {
				WriteError(w, http.StatusConflict, "email_already_exists", "email already registered")
				return
			}
			WriteError(w, http.StatusInternalServerError, "internal_error", "failed to create user")
			return
		}

		WriteJSON(w, http.StatusCreated, SignupResponse{
			ID:        user.ID,
			Email:     user.Email,
			CreatedAt: user.CreatedAt,
		})
	}
}
