package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

var (
	ErrEmailAlreadyRegistered = errors.New("email already registered")
	ErrUserNotFound           = errors.New("user not found")
)

type User struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func CreateUser(db *sql.DB, email, passwordHash string) (User, error) {
	var user User
	err := db.QueryRow(
		`INSERT INTO users (email, password_hash)
		 VALUES ($1, $2)
		 RETURNING id, email, password_hash, created_at, updated_at`,
		email, passwordHash,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return User{}, ErrEmailAlreadyRegistered
		}
		return User{}, fmt.Errorf("creating user: %w", err)
	}
	return user, nil
}

func FindUserByEmail(db *sql.DB, email string) (User, error) {
	var user User
	err := db.QueryRow(
		`SELECT id, email, password_hash, created_at, updated_at
		 FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("finding user by email: %w", err)
	}
	return user, nil
}

func UserExists(db *sql.DB, email string) (bool, error) {
	var exists bool
	err := db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`,
		email,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("checking user exists: %w", err)
	}
	return exists, nil
}
