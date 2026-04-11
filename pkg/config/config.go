package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

const minJWTSecretLength = 32

type Config struct {
	DB      DBConfig
	JWT     JWTConfig
	Service ServiceConfig
}

type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
}

type JWTConfig struct {
	Secret        string
	AccessExpiry  int
	RefreshExpiry int
}

type ServiceConfig struct {
	Port     string
	LogLevel string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	dbPort, err := getEnvInt("DB_PORT", 5432)
	if err != nil {
		return nil, fmt.Errorf("DB_PORT: %w", err)
	}

	accessExpiry, err := getEnvInt("JWT_ACCESS_EXPIRY", 900)
	if err != nil {
		return nil, fmt.Errorf("JWT_ACCESS_EXPIRY: %w", err)
	}

	refreshExpiry, err := getEnvInt("JWT_REFRESH_EXPIRY", 604800)
	if err != nil {
		return nil, fmt.Errorf("JWT_REFRESH_EXPIRY: %w", err)
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, errors.New("JWT_SECRET is required")
	}
	if len(secret) < minJWTSecretLength {
		return nil, fmt.Errorf("JWT_SECRET must be at least %d characters", minJWTSecretLength)
	}

	return &Config{
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     dbPort,
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Name:     getEnv("DB_NAME", "auth_db"),
		},
		JWT: JWTConfig{
			Secret:        secret,
			AccessExpiry:  accessExpiry,
			RefreshExpiry: refreshExpiry,
		},
		Service: ServiceConfig{
			Port:     getEnv("SERVICE_PORT", "8000"),
			LogLevel: getEnv("LOG_LEVEL", "info"),
		},
	}, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("invalid int %q", v)
	}
	return n, nil
}
