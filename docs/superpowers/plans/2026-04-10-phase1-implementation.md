# Phase 1 Auth Service Implementation Plan

**Goal:** Build a standalone Go auth service with signup/login/refresh/logout endpoints, JWT tokens, PostgreSQL, and local Docker deployment.

**Architecture:** 
- Modular design: config → database → auth logic → HTTP handlers
- Stateless JWT tokens (no session storage needed in Phase 1)
- Clean separation: business logic in `pkg/auth/`, database in `pkg/db/`, config in `pkg/config/`
- TDD approach for critical paths (password hashing, token generation, user queries)

**Tech Stack:** Go 1.21, PostgreSQL 14, bcrypt, golang-jwt/jwt/v5, golang-migrate, Docker Compose

**Testing Strategy:** After each task, you will test the code yourself (unit tests, manual curl/Postman requests, database checks) and commit when satisfied. No auto-commits.

---

## Progress

### Phase 1A: Core Infrastructure
- ✅ Task 1: Initialize Go Project & Dependencies
- ✅ Task 2: Configuration Package
- ✅ Task 3: PostgreSQL Connection & Migrations Setup

### Phase 1B: Auth Logic (Unit Tests)
- ✅ Task 4: Password Hashing with Bcrypt & Tests
- ✅ Task 5: JWT Token Generation & Validation with Tests

### Phase 1C: Database & HTTP Handlers
- ✅ Task 6: User Database Operations
- ✅ Task 7: Response Helpers & CORS
- ✅ Task 8: Signup Handler
- ✅ Task 9: Login Handler
- ✅ Task 10: Refresh Token Handler
- ⏳ Task 11: Logout Handler

### Phase 1D: Deployment & Polish
- ⏳ Task 12: Dockerfile & Docker Compose Setup
- ⏳ Task 13: Makefile with Common Commands
- ⏳ Task 14: Postman Collection with All Endpoints
- ⏳ Task 15: Setup Documentation (Optional Polish)

---

## File Structure Overview

```
auth-base/
├── main.go                          # Entry point, router setup
├── go.mod / go.sum                  # Dependencies
├── Makefile                         # Build/test/run commands
├── Dockerfile                       # Container build
├── docker-compose.yml               # Local dev environment (Go + Postgres)
├── .env.example                     # Env var template
├── pkg/
│   ├── config/
│   │   └── config.go                # Load config from env vars
│   ├── db/
│   │   ├── postgres.go              # DB connection pool
│   │   └── users.go                 # User CRUD operations
│   └── auth/
│       ├── password.go              # Bcrypt hashing/validation
│       ├── tokens.go                # JWT generation/validation
│       ├── response.go              # Response helpers with CORS
│       ├── signup.go                # Signup handler
│       ├── login.go                 # Login handler
│       ├── refresh.go               # Refresh token handler
│       └── logout.go                # Logout handler
├── migrations/
│   ├── 000001_initial_schema.up.sql
│   └── 000001_initial_schema.down.sql
├── tests/
│   ├── integration/
│   │   └── auth_integration_test.go
│   └── unit/
│       ├── password_test.go
│       └── tokens_test.go
└── postman/
    └── auth-service.postman_collection.json
```

---

# Phase 1A: Core Infrastructure

## Task 1: Initialize Go Project & Dependencies

**Goal:** Set up Go module and add all required dependencies.

**Files to Create:**
- `go.mod` (via `go mod init`)
- `main.go` (minimal entry point with health check endpoint)

**Files to Update:**
- None

**Testable Outcome:**
- ✅ `go mod init github.com/eds/auth-base` succeeds
- ✅ Dependencies installed: jwt/v5, bcrypt, pq, godotenv
- ✅ `go build -o auth-service` compiles successfully
- ✅ `./auth-service` starts and listens on port 8000
- ✅ `curl http://localhost:8000/health` returns JSON response

**Implementation Notes:**
- Initialize module, download dependencies (jwt, bcrypt, pq, godotenv)
- Create minimal main.go with health check endpoint that returns `{"message":"Auth service running"}`
- Service should read SERVICE_PORT from env or default to 8000

**Testing Commands:**
```bash
go build -o auth-service
./auth-service
# In another terminal:
curl http://localhost:8000/health
```

---

## Task 2: Configuration Package

**Goal:** Create config package that loads and validates environment variables.

**Files to Create:**
- `pkg/config/config.go`
- `.env.example`

**Files to Update:**
- None yet

**Testable Outcome:**
- ✅ Config struct loads from environment variables with defaults
- ✅ JWT_SECRET validation: required, minimum 32 characters
- ✅ Database connection parameters configurable
- ✅ `go build ./pkg/config` compiles without errors

**Implementation Notes:**
- Config struct with DB, JWT, and Service fields
- Load() function that reads env vars using godotenv
- Validate JWT_SECRET is present and ≥32 chars (fail early if not)
- Return defaults for DB_HOST (localhost), DB_PORT (5432), SERVICE_PORT (8000), etc.
- Support both .env file (optional) and direct env vars

**Testing Commands:**
```bash
export JWT_SECRET=test_secret_key_minimum_32_chars_long_xyz
go build ./pkg/config
```

---

## Task 3: PostgreSQL Connection & Migrations Setup

**Goal:** Set up PostgreSQL connection pool and database migration files.

**Files to Create:**
- `pkg/db/postgres.go`
- `migrations/000001_initial_schema.up.sql`
- `migrations/000001_initial_schema.down.sql`
- `docker-compose.yml` (PostgreSQL service only for now)

**Files to Update:**
- None

**Testable Outcome:**
- ✅ `docker-compose up postgres -d` starts PostgreSQL
- ✅ Migration files contain valid SQL syntax
- ✅ Users table created with fields: id (UUID PK), email (unique), password_hash, created_at, updated_at
- ✅ Email index created for fast lookups
- ✅ `go build ./pkg/db` compiles

**Implementation Notes:**
- NewPostgresConnection() function that creates connection pool and pings database
- Set MaxOpenConns=25, MaxIdleConns=5
- Up migration: CREATE TABLE users with proper schema
- Down migration: DROP TABLE and index
- docker-compose.yml with postgres:14-alpine, health check, volume for persistence

**Testing Commands:**
```bash
docker-compose up postgres -d
docker-compose ps  # Check healthy status
migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/auth_db?sslmode=disable" up
psql -h localhost -U postgres -d auth_db -c "\dt"  # List tables
```

---

# Phase 1B: Auth Logic (Unit Tests)

## Task 4: Password Hashing with Bcrypt & Tests

**Goal:** Implement password hashing/verification with bcrypt and write comprehensive unit tests.

**Files to Create:**
- `pkg/auth/password.go`
- `tests/unit/password_test.go`

**Files to Update:**
- None

**Testable Outcome:**
- ✅ `go test ./tests/unit -run TestHashPassword -v` all pass
- ✅ `go test ./tests/unit -run TestVerifyPassword -v` all pass
- ✅ `go test ./tests/unit -run TestPasswordValidation -v` all pass
- ✅ Hash length is ~60 characters (bcrypt format)
- ✅ Password length validation: reject <8 chars and >256 chars

**Implementation Notes:**
- HashPassword(password) returns hash or error
- VerifyPassword(hash, password) returns bool
- Constants: MinPasswordLength=8, MaxPasswordLength=256, BcryptCost=12
- Unit tests: valid hash, verify correct/wrong password, length validation

**Testing Commands:**
```bash
go test ./tests/unit -run TestHash -v
go test ./tests/unit -run TestVerify -v
go test ./tests/unit -v -run "TestHashPassword|TestVerifyPassword|TestPasswordValidation"
```

---

## Task 5: JWT Token Generation & Validation with Tests

**Goal:** Implement JWT token generation, validation, and claims structure with unit tests.

**Files to Create:**
- `pkg/auth/tokens.go`
- `tests/unit/tokens_test.go`

**Files to Update:**
- None

**Testable Outcome:**
- ✅ `go test ./tests/unit -run TestGenerate -v` all pass
- ✅ `go test ./tests/unit -run TestValidate -v` all pass
- ✅ Generated tokens have 3 parts (header.payload.signature)
- ✅ Access tokens include: sub (user ID), email, type=access, exp, iat
- ✅ Refresh tokens include: sub (user ID), type=refresh, exp, iat
- ✅ Token validation rejects invalid signature
- ✅ Token validation rejects expired tokens

**Implementation Notes:**
- Claims struct with UserID, Email, Type (access/refresh), and jwt.RegisteredClaims
- GenerateAccessToken(userID, email, secret, expiry) returns signed JWT
- GenerateRefreshToken(userID, secret, expiry) returns signed JWT
- ValidateToken(tokenString, secret) parses and validates signature/expiry, returns Claims
- ValidateRefreshToken() validates token has type=refresh
- Use HS256 algorithm, sign with secret key

**Testing Commands:**
```bash
go test ./tests/unit -run TestGenerate -v
go test ./tests/unit -run TestValidate -v
go test ./tests/unit -v
```

---

# Phase 1C: Database & HTTP Handlers

## Task 6: User Database Operations

**Goal:** Implement user CRUD operations with parameterized SQL queries.

**Files to Create:**
- `pkg/db/users.go`

**Files to Update:**
- None

**Testable Outcome:**
- ✅ `go build ./pkg/db` compiles
- ✅ CreateUser(db, email, passwordHash) inserts user and returns User struct
- ✅ FindUserByEmail(db, email) retrieves user or returns error
- ✅ UserExists(db, email) returns bool
- ✅ All queries use parameterized statements ($1, $2 placeholders)
- ✅ Duplicate email insert returns "email already registered" error
- ✅ Non-existent email lookup returns "user not found" error

**Implementation Notes:**
- User struct with ID, Email, PasswordHash, CreatedAt, UpdatedAt fields
- All DB queries use $1, $2 style parameterized queries (no string concatenation)
- CreateUser uses RETURNING to get generated UUID, timestamps
- FindUserByEmail returns User or error
- UserExists uses SELECT EXISTS pattern
- Handle pq constraint errors for duplicate email

**Testing Commands:**
```bash
go build ./pkg/db
# Manual testing via psql after database is set up
```

---

## Task 7: Response Helpers & CORS

**Goal:** Create HTTP response helpers with consistent error format and CORS headers.

**Files to Create:**
- `pkg/auth/response.go`

**Files to Update:**
- None

**Testable Outcome:**
- ✅ `go build ./pkg/auth` compiles
- ✅ Response structs defined: SignupResponse, LoginResponse, RefreshResponse, LogoutResponse, ErrorResponse
- ✅ WriteJSON() sets Content-Type, CORS headers, writes JSON
- ✅ WriteError() writes error with error code and message
- ✅ HandleCORS() responds to OPTIONS requests with headers
- ✅ All responses have `Access-Control-Allow-Origin: *`

**Implementation Notes:**
- ErrorResponse struct with error (code) and message fields
- Response structs for each endpoint (signup, login, refresh, logout)
- WriteJSON() helper that sets CORS headers and encodes JSON
- WriteError() helper that uses WriteJSON with ErrorResponse
- HandleCORS() for preflight requests
- Consistent error response format across all endpoints

**Testing Commands:**
```bash
go build ./pkg/auth
```

---

## Task 8: Signup Handler

**Goal:** Implement POST /signup endpoint with email validation, password hashing, user creation.

**Files to Create:**
- `pkg/auth/signup.go`

**Files to Update:**
- `main.go` (add signup route and database initialization)

**Testable Outcome:**
- ✅ POST /signup accepts { email, password }
- ✅ Valid request returns 201 with id, email, created_at
- ✅ Invalid email format returns 400 "invalid_email"
- ✅ Short password (<8 chars) returns 400 "invalid_password"
- ✅ Duplicate email returns 409 "email_already_exists"
- ✅ User created in database with hashed password
- ✅ Postman request succeeds and shows response
- ✅ psql query shows user row with UUID id and hashed password

**Implementation Notes:**
- SignupRequest struct with email, password
- Email validation regex: ^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$
- Call HashPassword() to hash before storing
- Call CreateUser() to insert into database
- Handle all error cases with appropriate status codes
- Update main.go to: load config, connect to database, register signup handler

**Testing Commands:**
```bash
export JWT_SECRET=test_secret_key_minimum_32_chars_long_xyz
go build -o auth-service
./auth-service

# Test valid signup
curl -X POST http://localhost:8000/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","password":"secure_password_123"}'

# Test invalid email
curl -X POST http://localhost:8000/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"not-an-email","password":"secure_password_123"}'

# Verify in database
psql -h localhost -U postgres -d auth_db -c "SELECT id, email FROM users;"
```

---

## Task 9: Login Handler

**Goal:** Implement POST /login endpoint with credential verification and token issuance.

**Files to Create:**
- `pkg/auth/login.go`

**Files to Update:**
- `main.go` (add login route)

**Testable Outcome:**
- ✅ POST /login accepts { email, password }
- ✅ Valid credentials return 200 with access_token, refresh_token, expires_in, token_type
- ✅ Non-existent email returns 401 "invalid_credentials"
- ✅ Wrong password returns 401 "invalid_credentials" (no user enumeration)
- ✅ Tokens are valid JWTs (can decode at jwt.io)
- ✅ Access token has correct expiry (900 seconds default)
- ✅ Refresh token has correct expiry (604800 seconds default)
- ✅ Postman request succeeds and returns tokens
- ✅ Tokens can be saved in Postman environment for next task

**Implementation Notes:**
- LoginRequest struct with email, password
- Call FindUserByEmail() to retrieve user
- Call VerifyPassword() to validate password
- Call GenerateAccessToken() and GenerateRefreshToken()
- Return both tokens with expires_in set to JWT_ACCESS_EXPIRY in seconds
- Generic error message on failure (no specifics to prevent enumeration)

**Testing Commands:**
```bash
# Test valid login
curl -X POST http://localhost:8000/login \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","password":"secure_password_123"}'

# Save the tokens (you'll use them in Task 10)

# Test wrong password
curl -X POST http://localhost:8000/login \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","password":"wrong"}'

# Decode access_token at https://jwt.io
```

---

## Task 10: Refresh Token Handler

**Goal:** Implement POST /refresh endpoint to issue new access token from refresh token.

**Files to Create:**
- `pkg/auth/refresh.go`

**Files to Update:**
- `main.go` (add refresh route)

**Testable Outcome:**
- ✅ POST /refresh accepts { refresh_token }
- ✅ Valid refresh token returns 200 with new access_token, expires_in, token_type
- ✅ Invalid/malformed token returns 401 "invalid_token"
- ✅ Expired token returns 401 "invalid_token"
- ✅ Non-refresh JWT returns 401 "invalid_token"
- ✅ New access token has same user ID as original
- ✅ Postman request succeeds using token from Task 9
- ✅ No new refresh token issued (only access token)

**Implementation Notes:**
- RefreshRequest struct with refresh_token
- Call ValidateRefreshToken() to validate and extract claims
- Call FindUserByEmail() to verify user still exists
- Call GenerateAccessToken() to create new access token
- Return only access token (not refresh token)
- Validate token has type=refresh before issuing

**Testing Commands:**
```bash
# Use refresh_token from Task 9 login response
curl -X POST http://localhost:8000/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"<your_refresh_token>"}'

# Test with invalid token
curl -X POST http://localhost:8000/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"invalid.token.here"}'
```

---

## Task 11: Logout Handler

**Goal:** Implement POST /logout endpoint (stateless acknowledgment, no revocation).

**Files to Create:**
- `pkg/auth/logout.go`

**Files to Update:**
- `main.go` (add logout route)

**Testable Outcome:**
- ✅ POST /logout accepts { refresh_token }
- ✅ Returns 200 with message "Logged out successfully"
- ✅ No database operation performed
- ✅ CORS headers present on response
- ✅ Works with any/invalid token (stateless)
- ✅ Postman request succeeds

**Implementation Notes:**
- LogoutRequest struct with refresh_token field
- Accept token but don't validate (stateless phase 1)
- Return LogoutResponse with success message
- No database changes (client responsible for discarding tokens)
- Respond to OPTIONS for CORS

**Testing Commands:**
```bash
curl -X POST http://localhost:8000/logout \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"any_token_here"}'

# Check CORS header
curl -X POST http://localhost:8000/logout -v \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"token"}' | grep "Access-Control-Allow-Origin"
```

---

# Phase 1D: Deployment & Polish

## Task 12: Dockerfile & Docker Compose Setup

**Goal:** Create Docker image and docker-compose for local deployment.

**Files to Create:**
- `Dockerfile` (multi-stage build)

**Files to Update:**
- `docker-compose.yml` (add auth-service, environment variables, dependencies)

**Testable Outcome:**
- ✅ `docker build -t auth-service:latest .` succeeds
- ✅ `docker-compose up` starts postgres and auth-service
- ✅ Both containers show healthy status
- ✅ Service accessible at http://localhost:8000/health
- ✅ Migrations can be run
- ✅ Can test endpoints via curl/Postman
- ✅ `docker-compose down` stops cleanly and preserves data

**Implementation Notes:**
- Dockerfile: Builder stage (golang 1.21) → compile → Runtime stage (alpine)
- Copy migrations folder into runtime image
- docker-compose: postgres (port 5432) + auth-service (port 8000)
- auth-service depends_on postgres with health check
- Environment variables for JWT_SECRET, DB config
- Postgres healthcheck ensures DB is ready before app starts
- Volume for postgres_data persistence

**Testing Commands:**
```bash
docker build -t auth-service:latest .
docker-compose up
# Wait for both services to be healthy
docker-compose ps

# In another terminal
curl http://localhost:8000/health

# Stop
docker-compose down
```

---

## Task 13: Makefile with Common Commands

**Goal:** Create Makefile with standard development commands.

**Files to Create:**
- `Makefile`

**Files to Update:**
- None

**Testable Outcome:**
- ✅ `make help` displays all available commands
- ✅ `make build` compiles binary successfully
- ✅ `make run` runs the service (requires DB)
- ✅ `make test` runs all tests
- ✅ `make lint` formats and checks code
- ✅ `make clean` removes build artifacts
- ✅ `make migrate` runs golang-migrate
- ✅ `make docker-build` builds Docker image
- ✅ `make docker-run` starts docker-compose
- ✅ `make docker-down` stops docker-compose

**Implementation Notes:**
- Targets: build, run, test, lint, clean, migrate, docker-build, docker-run, docker-down, help
- Each target has clear purpose
- test target runs: go test -v ./...
- lint target runs: go fmt and go vet
- migrate checks for golang-migrate install
- docker-run depends on docker-build

**Testing Commands:**
```bash
make help
make build
make test
make lint
make docker-run
make docker-down
```

---

## Task 14: Postman Collection with All Endpoints

**Goal:** Create Postman collection with all endpoints and test scenarios.

**Files to Create:**
- `postman/auth-service.postman_collection.json`

**Files to Update:**
- None

**Testable Outcome:**
- ✅ Postman imports collection successfully
- ✅ Collection has 10+ requests:
  - Health Check (GET /health)
  - Signup (POST /signup) - valid
  - Login (POST /login) - valid
  - Refresh Token (POST /refresh) - valid
  - Logout (POST /logout)
  - Error: Invalid Email
  - Error: Short Password
  - Error: Duplicate Email
  - Error: Wrong Password
  - Error: Invalid Token
- ✅ Can run happy path: Health → Signup → Login → Refresh → Logout
- ✅ Can test all error cases
- ✅ Uses {{refresh_token}} variable for token reuse

**Implementation Notes:**
- Export as Postman Collection v2.1.0 format
- Each request has name, method, URL, body, headers
- Happy path requests ordered for sequential testing
- Error cases clearly labeled
- Uses variables for token reuse (e.g., {{refresh_token}})
- All localhost:8000 URLs

**Testing Commands:**
```bash
# Open Postman
# File → Import → Select postman/auth-service.postman_collection.json
# Run Health Check → Signup → Login → Refresh → Logout in sequence
# Test error cases individually
```

---

## Task 15: Setup Documentation (Optional Polish)

**Goal:** Create README with setup instructions.

**Files to Create:**
- `README-SETUP.md` (or update README.md)

**Testable Outcome:**
- ✅ Clear instructions for local development
- ✅ Docker setup documented
- ✅ Manual setup documented
- ✅ Testing procedures documented
- ✅ Example curl commands

**Implementation Notes:**
- Setup via Docker Compose (recommended)
- Manual setup (golang-migrate, local postgres)
- Testing with Postman
- Testing with curl examples
- Cleanup instructions

---

# Summary

**You now have:**
- ✅ 15 independently testable tasks
- ✅ Each task produces working, testable code
- ✅ Clear "Testable Outcome" for each task
- ✅ Specific commands to verify each task works
- ✅ Organized into 4 phases (Infrastructure, Auth Logic, Handlers, Deployment)
- ✅ No code included in plan (just descriptions and test commands)
- ✅ Ready to implement one task at a time

---

# Next Steps

**When you're ready to start:**
1. Let me know which task you'd like to begin with (usually Task 1)
2. I'll provide the exact code for that task only
3. You implement, test with the commands listed above
4. When satisfied, you commit the code yourself
5. Let me know when done, and we move to the next task

**Ready to start Task 1?**
