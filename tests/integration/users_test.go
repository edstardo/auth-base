package integration

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"

	_ "github.com/lib/pq"

	"github.com/edstardo/auth-base/pkg/db"
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()

	host := envOrDefault("DB_HOST", "localhost")
	port := envOrDefault("DB_PORT", "5432")
	user := envOrDefault("DB_USER", "postgres")
	password := envOrDefault("DB_PASSWORD", "postgres")
	name := envOrDefault("DB_NAME", "auth_db")

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, name,
	)

	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("opening database: %v", err)
	}

	if err := conn.Ping(); err != nil {
		t.Skipf("skipping integration test: database not available: %v", err)
	}

	// Clean users table before each test
	_, err = conn.Exec("DELETE FROM users")
	if err != nil {
		t.Fatalf("cleaning users table: %v", err)
	}

	t.Cleanup(func() { conn.Close() })
	return conn
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func TestCreateUser_Success(t *testing.T) {
	conn := testDB(t)

	user, err := db.CreateUser(conn, "alice@example.com", "$2a$12$fakehashvalue")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if user.ID == "" {
		t.Fatal("expected non-empty UUID")
	}
	if user.Email != "alice@example.com" {
		t.Fatalf("expected email alice@example.com, got %s", user.Email)
	}
	if user.CreatedAt.IsZero() {
		t.Fatal("expected non-zero created_at")
	}
	if user.UpdatedAt.IsZero() {
		t.Fatal("expected non-zero updated_at")
	}
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	conn := testDB(t)

	_, err := db.CreateUser(conn, "dup@example.com", "$2a$12$fakehashvalue")
	if err != nil {
		t.Fatalf("first insert failed: %v", err)
	}

	_, err = db.CreateUser(conn, "dup@example.com", "$2a$12$anotherhash")
	if !errors.Is(err, db.ErrEmailAlreadyRegistered) {
		t.Fatalf("expected ErrEmailAlreadyRegistered, got %v", err)
	}
}

func TestFindUserByEmail_Success(t *testing.T) {
	conn := testDB(t)

	created, err := db.CreateUser(conn, "bob@example.com", "$2a$12$fakehashvalue")
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	found, err := db.FindUserByEmail(conn, "bob@example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if found.ID != created.ID {
		t.Fatalf("expected ID %s, got %s", created.ID, found.ID)
	}
	if found.Email != "bob@example.com" {
		t.Fatalf("expected email bob@example.com, got %s", found.Email)
	}
	if found.PasswordHash != "$2a$12$fakehashvalue" {
		t.Fatalf("expected password hash to match, got %s", found.PasswordHash)
	}
}

func TestFindUserByEmail_NotFound(t *testing.T) {
	conn := testDB(t)

	_, err := db.FindUserByEmail(conn, "nobody@example.com")
	if !errors.Is(err, db.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUserExists_True(t *testing.T) {
	conn := testDB(t)

	_, err := db.CreateUser(conn, "exists@example.com", "$2a$12$fakehashvalue")
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	exists, err := db.UserExists(conn, "exists@example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !exists {
		t.Fatal("expected user to exist")
	}
}

func TestUserExists_False(t *testing.T) {
	conn := testDB(t)

	exists, err := db.UserExists(conn, "ghost@example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exists {
		t.Fatal("expected user to not exist")
	}
}
