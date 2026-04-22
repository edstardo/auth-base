.PHONY: help build run test coverage lint clean migrate docker-build docker-run docker-down

BINARY      := auth-service
MIGRATE_DSN := postgres://postgres:postgres@localhost:5432/auth_db?sslmode=disable

.DEFAULT_GOAL := help

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "Available targets:\n"} /^[a-zA-Z_-]+:.*##/ {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Compile the auth-service binary
	go build -o $(BINARY) .

run: build ## Build and run the service locally
	./$(BINARY)

test: ## Run all tests
	go test -v ./...

coverage: ## Run tests with coverage and generate HTML report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint: ## Format and vet code
	go fmt ./...
	go vet ./...

clean: ## Remove build artifacts
	rm -f $(BINARY) coverage.out coverage.html
	go clean

migrate: ## Apply database migrations (requires golang-migrate)
	@command -v migrate >/dev/null 2>&1 || { echo "golang-migrate not installed. See https://github.com/golang-migrate/migrate"; exit 1; }
	migrate -path migrations -database "$(MIGRATE_DSN)" up

docker-build: ## Build the Docker image
	docker build -t $(BINARY):latest .

docker-run: docker-build ## Start postgres + auth-service via docker-compose
	docker-compose up

docker-down: ## Stop docker-compose services
	docker-compose down
