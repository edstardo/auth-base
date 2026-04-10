# Phase 1 Auth Service Design

**Date:** 2026-04-10  
**Project:** auth-base (Single-App Authentication Service)  
**Phase:** Phase 1  
**Status:** Design (awaiting implementation)

---

## Executive Summary

A standalone Go-based authentication/authorization service designed for Phase 1 simplicity with Phase 2 multi-tenancy in mind. This service handles user signup, login, and token issuance (access + refresh tokens) for a single application. Deployable locally via Docker Compose and testable via Postman.

---

## Architecture Overview

### System Components

```
┌─────────────────────────────────────────────────────────────┐
│                    Go Auth Service                          │
│  (HTTP handlers for signup, login, logout + JWT signing)    │
└────────────┬──────────────────────────────────────────┬─────┘
             │                                          │
       ┌─────▼─────┐                            ┌──────▼──────┐
       │ PostgreSQL │                            │   Postman    │
       │  (Users)   │                            │ (Testing)    │
       └────────────┘                            └──────────────┘
```

**Key Design Principle:** Stateless JWT tokens mean no session storage or Redis required in Phase 1. Service is horizontally scalable (multiple instances work fine). Refresh tokens are self-contained JWTs, not database-backed.

---

## API Endpoints (Phase 1)

### 1. POST /signup
**Purpose:** Register a new user

**Request:**
```json
{
  "email": "user@example.com",
  "password": "secure_password_123"
}
```

**Response (201 Created):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "created_at": "2026-04-10T10:00:00Z"
}
```

**Validation:**
- Email must match regex: `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
- Email must be unique (not already in database)
- Password must be 8-256 characters
- No character complexity requirements
- No character restrictions (all printable characters allowed)

**Error Responses:**
- 400 Bad Request — invalid email format, password too short/long, missing fields
- 409 Conflict — email already registered

**Implementation Details:**
- Hash password with bcrypt before storing
- Generate user ID as UUID v4
- Store: (id, email, password_hash, created_at, updated_at)
- Use parameterized queries to prevent SQL injection

---

### 2. POST /login
**Purpose:** Authenticate user and issue tokens

**Request:**
```json
{
  "email": "user@example.com",
  "password": "secure_password_123"
}
```

**Response (200 OK):**
```json
{
  "access_token": "eyJhbGc...",
  "refresh_token": "eyJhbGc...",
  "expires_in": 900,
  "token_type": "Bearer"
}
```

**Validation:**
- Email must exist in database
- Password must match bcrypt hash
- Both conditions required; generic error message on failure ("Invalid email or password")

**Error Responses:**
- 401 Unauthorized — invalid email or password (no specifics)
- 400 Bad Request — malformed request

**Token Contents:**

**Access Token (JWT):**
- `exp`: 15 minutes from now
- `sub`: user ID
- `email`: user email
- `iat`: issued at
- Algorithm: HS256

**Refresh Token (JWT):**
- `exp`: 7 days from now
- `sub`: user ID
- `type`: "refresh"
- `iat`: issued at
- Algorithm: HS256

---

### 3. POST /refresh
**Purpose:** Issue a new access token using a valid refresh token

**Request:**
```json
{
  "refresh_token": "eyJhbGc..."
}
```

**Response (200 OK):**
```json
{
  "access_token": "eyJhbGc...",
  "expires_in": 900,
  "token_type": "Bearer"
}
```

**Validation:**
- Refresh token must be valid JWT (correct signature, not expired)
- Refresh token must have `type: "refresh"` claim
- User ID from token must exist in database

**Error Responses:**
- 401 Unauthorized — invalid, expired, or malformed refresh token
- 400 Bad Request — missing refresh_token field

**Implementation Details:**
- Validate refresh token signature using JWT_SECRET
- Check expiration
- Extract user ID from `sub` claim
- Generate new access token with same user ID
- Return new access token (no new refresh token issued)

---

### 4. POST /logout
**Purpose:** Acknowledge logout (no backend revocation in Phase 1)

**Request:**
```json
{
  "refresh_token": "eyJhbGc..."
}
```

**Response (200 OK):**
```json
{
  "message": "Logged out successfully"
}
```

**Notes:**
- No database operation required (tokens are stateless)
- Endpoint exists for API completeness
- Client discards tokens locally
- Phase 2 can add token blacklist for instant revocation if needed

---

## Error Response Format

All error responses follow a consistent JSON format:

```json
{
  "error": "error_code",
  "message": "Human-readable error description"
}
```

**HTTP Status Codes:**
- `400 Bad Request` — Malformed input, validation failure, missing required fields
- `401 Unauthorized` — Invalid credentials, expired/invalid token
- `409 Conflict` — Resource already exists (e.g., email already registered)
- `500 Internal Server Error` — Unexpected server error

**Example Error Responses:**

Signup with duplicate email (409):
```json
{
  "error": "email_already_exists",
  "message": "Email is already registered"
}
```

Invalid login credentials (401):
```json
{
  "error": "invalid_credentials",
  "message": "Invalid email or password"
}
```

Missing refresh token (400):
```json
{
  "error": "missing_field",
  "message": "refresh_token is required"
}
```

---

## Database Schema

### Users Table

```sql
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email VARCHAR(255) UNIQUE NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);
```

**Notes:**
- `id` is UUID for Phase 2 flexibility (easier to merge/reshard databases)
- `password_hash` stores bcrypt output (60 characters)
- Both `created_at` and `updated_at` use database-side defaults
- Email index enables fast lookups during login

**No other tables needed in Phase 1.** (Refresh tokens don't need storage—they're self-contained JWTs.)

---

## Technology Stack

| Component | Choice | Why |
|-----------|--------|-----|
| Language | Go 1.21+ | Efficient, fast, built-in HTTP support |
| Database | PostgreSQL 14+ | Production-ready, robust, scales, Docker-friendly |
| Password Hashing | bcrypt | Industry standard, secure, simple |
| Token Format | JWT (HS256) | Stateless, scalable, testable |
| JWT Library | `github.com/golang-jwt/jwt/v5` | Popular, well-maintained, simple API |
| HTTP Framework | Standard `net/http` | Minimal dependencies, clear code |
| Migrations | golang-migrate | Simple CLI, supports version control, Go-based |
| Local Dev | Docker Compose | PostgreSQL + service in one command |
| Testing | Postman collection | User-friendly, no code needed |
| Build/Deploy | Makefile + Dockerfile | Simple, familiar, reproducible |

---

## Deployment & Local Development

### Docker Compose
**File: docker-compose.yml**

Runs:
1. PostgreSQL 14+ container (port 5432, local)
2. Go auth service container (port 8000)

One command: `docker-compose up` starts both.

### Dockerfile
**Multi-stage build:**
1. Build stage: Go 1.21, compile binary
2. Runtime stage: Minimal base image, copy binary
3. Expose port 8000

### Makefile
**Commands:**
- `make build` — compile Go binary
- `make run` — run service locally (requires `docker-compose up` for DB)
- `make test` — run unit tests
- `make lint` — run linter (golangci-lint or similar)
- `make docker-build` — build Docker image
- `make docker-run` — run via Docker Compose
- `make migrate` — run database migrations

### Database Migrations
**Tool:** golang-migrate (https://github.com/golang-migrate/migrate)  
**Location:** `migrations/` directory with naming: `000001_initial_schema.up.sql` and `.down.sql`  
**Execution:** Manual via `make migrate` (run before service starts)  
**Note:** Service does NOT auto-migrate; migrations must run explicitly

---

## Code Structure

```
auth-base/
├── main.go                 # Entry point
├── Makefile                # Build commands
├── Dockerfile              # Container definition
├── docker-compose.yml      # Local dev environment
├── .env.example            # Environment variables template
├── go.mod / go.sum         # Dependencies
├── pkg/
│   ├── auth/               # Auth business logic
│   │   ├── signup.go       # Signup handler
│   │   ├── login.go        # Login handler
│   │   ├── refresh.go      # Refresh token handler
│   │   ├── logout.go       # Logout handler
│   │   ├── tokens.go       # JWT generation/validation/parsing
│   │   └── password.go     # Password hashing/validation with bcrypt
│   ├── db/                 # Database interactions
│   │   ├── postgres.go     # Connection pool setup
│   │   └── users.go        # User queries (create, find by email)
│   └── config/             # Configuration
│       └── config.go       # Load env vars, secrets
├── migrations/             # SQL migration files
├── tests/
│   ├── integration/        # Integration tests
│   └── unit/               # Unit tests
├── postman/                # Postman collection
│   └── auth-service.postman_collection.json
└── docs/
    └── superpowers/specs/  # Design documentation
```

---

## Security Considerations

### Password Storage
- Bcrypt with auto-generated salt (cost factor: 12)
- Never log passwords or store in plain text
- Validate length before hashing: 8-256 characters (reject early)
- No character restrictions (all printable Unicode allowed)
- Bcrypt output is 60 characters; store in VARCHAR(60) minimum

### SQL Injection Prevention
- **All database queries use parameterized statements** (prepared statements)
- Never concatenate user input into SQL strings
- Use Go's `database/sql` with `?` placeholders or `pgx` with `$1`, `$2`, etc.

### JWT Secrets
- Access token and refresh token use the same signing key for Phase 1
- **TODO (Phase 2):** Different signing keys for access vs. refresh tokens
- Secret stored in environment variable (`JWT_SECRET`), never in code
- Minimum length: 32 characters
- Algorithm: HS256 (HMAC SHA-256)

### Email Validation
- Regex pattern: `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
- No DNS lookup (Phase 1 simplicity)
- Uniqueness enforced in database (unique constraint)

### Error Messages
- Login endpoint: generic "Invalid email or password" (prevents user enumeration)
- Refresh endpoint: generic "Invalid or expired token"
- Signup: specific errors (email taken, password too short) for UX

### CORS (Cross-Origin Resource Sharing)
- Phase 1: Return `Access-Control-Allow-Origin: *` on all responses (permissive)
- This allows Postman and browser clients to call the service
- **TODO (Phase 2):** Restrict to specific origins in production

### HTTPS
- Phase 1: HTTP only (localhost development)
- Phase 2: Enforce HTTPS in production, use secure cookies for tokens

---

## Testing Strategy

### Unit Tests
- Password hashing correctness
- JWT generation (token structure, claims)
- Email validation regex
- Token parsing and validation

### Integration Tests
- Full signup → login → logout flow
- Database interactions (user creation, retrieval)
- Concurrent signup attempts (unique email constraint)
- Invalid inputs (wrong password, missing fields)

### Postman Collection
**Scenarios:**
1. Happy path: signup → login → refresh → logout
2. Error cases: duplicate email, invalid password, wrong credentials, expired token
3. Token refresh: use refresh token to get new access token
4. Token validation: decode tokens, verify claims and expiration
5. CORS headers: verify `Access-Control-Allow-Origin` header present

---

## Configuration & Environment Variables

**.env file (example):**
```
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres_password
DB_NAME=auth_db

JWT_SECRET=your_super_secret_key_minimum_32_chars
JWT_ACCESS_EXPIRY=900        # 15 minutes in seconds
JWT_REFRESH_EXPIRY=604800    # 7 days in seconds

SERVICE_PORT=8000
LOG_LEVEL=info
```

**Docker Compose overrides** these for local development.

---

## Phase 1 Scope (In)

✅ User signup with email + password  
✅ User login with credential validation  
✅ JWT access + refresh token issuance  
✅ Token refresh endpoint (get new access token)  
✅ Logout endpoint (stateless acknowledgment)  
✅ PostgreSQL user storage (UUID-based users table)  
✅ Bcrypt password hashing (cost factor 12)  
✅ Parameterized SQL queries (SQL injection prevention)  
✅ CORS headers on all endpoints  
✅ Consistent error response format  
✅ Docker Compose local environment  
✅ golang-migrate database migrations  
✅ Makefile for common commands  
✅ Postman collection for testing  
✅ Unit + integration tests  

---

## Phase 1 Scope (Out)

❌ Email verification / confirmation  
❌ Password reset / recovery  
❌ Multi-tenancy (Phase 2)  
❌ MFA / 2FA  
❌ OAuth2 / SSO (Google, GitHub login)  
❌ Role-based access control (RBAC)  
❌ Redis caching  
❌ Token revocation / blacklist  
❌ API rate limiting  
❌ Audit logging  

---

## Success Criteria

Phase 1 is complete when:

1. ✅ All 4 endpoints (signup, login, refresh, logout) return correct responses
2. ✅ Passwords are properly hashed with bcrypt (cost factor 12)
3. ✅ JWTs are valid, signed with HS256, and contain correct claims
4. ✅ All database queries use parameterized statements (no SQL injection)
5. ✅ CORS headers present on all responses (`Access-Control-Allow-Origin: *`)
6. ✅ Error responses follow consistent JSON format with `error` and `message` fields
7. ✅ Postman collection exercises all happy-path and error scenarios
8. ✅ Unit + integration tests pass (including refresh token validation)
9. ✅ `docker-compose up` runs service locally without manual setup
10. ✅ Migrations run successfully with golang-migrate (`make migrate`)
11. ✅ All Makefile commands work (`build`, `test`, `lint`, `docker-run`, `migrate`)
12. ✅ Code follows Go conventions (gofmt, goimports)

---

## Timeline Estimate

Not provided (per project guidance—focus on scope, not time).

---

## Next Steps (Phase 2 Preview)

- Multi-tenancy: Add `app_id` concept to support multiple SaaS apps
- API key authentication for cross-app calls
- Token revocation / blacklist (Redis or DB-backed)
- Password reset flow
- MFA support
- OAuth2 / SSO integrations
