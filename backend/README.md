# Backend

Go HTTP API server using the BFF (Backend For Frontend) pattern for OIDC authentication.

## Stack

- **Runtime**: Go 1.23
- **Router**: [Gin](https://github.com/gin-gonic/gin)
- **OIDC**: [coreos/go-oidc](https://github.com/coreos/go-oidc) + [golang.org/x/oauth2](https://pkg.go.dev/golang.org/x/oauth2)
- **Sessions**: [alexedwards/scs](https://github.com/alexedwards/scs) with PostgreSQL store
- **Database**: [pgx/v5](https://github.com/jackc/pgx) (pgxpool for queries, stdlib adapter for SCS)
- **Migrations**: [pressly/goose](https://github.com/pressly/goose) (embedded SQL, timestamp-based)
- **Cookie signing**: [gorilla/securecookie](https://github.com/gorilla/securecookie) (AES-256 + HMAC-SHA256)
- **Hot-reload**: [Air](https://github.com/air-verse/air)

## Directory Layout

```
backend/
├── cmd/
│   ├── migrate/main.go   Standalone migration runner (Docker init container)
│   └── server/main.go    HTTP server entrypoint
├── internal/
│   ├── auth/
│   │   ├── cookie.go     Pre-auth cookie (signed+encrypted) + id_hint cookie
│   │   ├── handler.go    Login, Callback, Logout, Me, HealthCheck
│   │   ├── middleware.go RequireAuth Gin middleware
│   │   └── oidc.go       OIDC provider: PKCE, token exchange, JWT verification
│   ├── config/
│   │   └── config.go     Environment variable loading with entropy validation
│   ├── db/
│   │   ├── db.go         pgxpool setup + goose migration runner
│   │   └── queries.go    UpsertUser, InsertLoginRecord
│   ├── model/
│   │   └── model.go      User, LoginRecord, SessionUser structs
│   └── session/
│       └── session.go    SCS session manager with postgresstore
└── migrations/
    ├── embed.go                          Embeds *.sql into the binary
    ├── 20260503000001_create_users.sql
    ├── 20260503000002_create_login_records.sql
    └── 20260503000003_create_sessions.sql
```

## API Endpoints

| Method | Path                | Auth | Description                               |
|--------|---------------------|------|-------------------------------------------|
| GET    | /healthz            | —    | Liveness probe                            |
| GET    | /api/auth/login     | —    | Start OIDC flow → redirect to Keycloak    |
| GET    | /api/auth/callback  | —    | OIDC callback — exchange code, set session|
| GET    | /api/auth/logout    | —    | Destroy session → redirect to KC logout   |
| GET    | /api/auth/me        | ✓    | Return current user from session          |

## Environment Variables

| Variable              | Required | Description                                              |
|-----------------------|----------|----------------------------------------------------------|
| `DATABASE_URL`        | ✓        | PostgreSQL connection string                             |
| `OIDC_ISSUER`         | ✓        | External Keycloak issuer (what tokens say)               |
| `OIDC_ISSUER_INTERNAL`| —        | Internal Docker URL for fetching JWKS (dev only)         |
| `OIDC_CLIENT_ID`      | ✓        | Keycloak client ID                                       |
| `OIDC_CLIENT_SECRET`  | ✓        | Keycloak client secret                                   |
| `OIDC_REDIRECT_URL`   | ✓        | Callback URL registered in Keycloak                      |
| `FRONTEND_URL`        | ✓        | Frontend base URL (for CORS and post-login redirect)     |
| `SESSION_SECRET`      | ✓        | ≥32 bytes — SCS session encryption                       |
| `COOKIE_HASH_KEY`     | ✓        | ≥32 bytes — HMAC key for pre-auth cookie                 |
| `COOKIE_ENCRYPT_KEY`  | ✓        | Exactly 32 bytes — AES-256 key for pre-auth cookie       |
| `BACKEND_PORT`        | —        | HTTP listen port (default: 8090)                         |
| `APP_ENV`             | —        | `development` or `production` (default: development)     |

## Adding a Migration

```bash
# Create a new timestamped migration file
touch backend/migrations/$(date +%Y%m%d%H%M%S)_your_name.sql
```

Add `-- +goose Up` / `-- +goose Down` / `-- +goose StatementBegin` markers.
Migrations run automatically via the `migrate` Docker service on stack start.

## Key Security Design Decisions

- **Pre-auth cookie**: signed with HMAC-SHA256 and encrypted with AES-256 via `gorilla/securecookie`
- **State comparison**: `crypto/subtle.ConstantTimeCompare` prevents timing attacks
- **Session fixation**: `scs.Destroy()` before creating the post-auth session
- **id_token**: never stored server-side; stored in a short-lived httpOnly cookie scoped to `/api/auth/logout` only
- **Token isolation**: `SessionUser` struct contains only identity claims, no tokens
