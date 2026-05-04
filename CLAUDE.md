# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

All Docker operations use `make` from the repo root. The default env file is `.env.dev`.

```bash
make dev-build      # First-run: rebuild images then start (takes ~2 min)
make dev            # Start without rebuilding
make dev-down       # Stop, keep volumes
make dev-clean      # Stop + wipe all volumes (full reset)
make logs           # Tail all service logs

make migrate-status # Show migration status (inside running backend container)
make migrate-up     # Run pending migrations
make migrate-down   # Roll back latest migration

make kc-export      # Export Keycloak realm to keycloak/realm-export.json
```

**Services when running:**
- Frontend: http://localhost:5173
- Backend: http://localhost:8090
- Keycloak admin: http://localhost:8080 (admin / admin)
- App DB: localhost:5432 | Keycloak DB: localhost:5433

**Running backend code directly** (requires the Docker stack or a local DB):
```bash
cd backend
go build ./...
go test ./...
go run ./cmd/server     # start server
go run ./cmd/migrate up # run migrations
```

**Frontend** (standalone, proxies to backend via Vite config):
```bash
cd frontend
npm install
npm run dev    # Vite dev server
npm run build  # tsc + vite build
```

## Architecture

```
Browser (React SPA)  ─── /api/ ───►  Go BFF (Gin + SCS)  ───► PostgreSQL 16
       │                                       │
       └── OIDC redirect ──►  Keycloak 24 ◄───┘ (JWKS fetch via internal Docker hostname)
```

### Backend (`backend/`)

Module: `github.com/jharaxus/rosetta`

```
cmd/server/main.go       Entry point — wires all dependencies, registers routes
cmd/migrate/main.go      Migration CLI (up | down | status) using goose
internal/
  auth/
    handler.go           Login, Callback, Logout, Me handlers
    middleware.go        RequireAuth gin middleware
    oidc.go              OIDCProvider — PKCE, token exchange, JWKS verification
    cookie.go            Pre-auth (state+verifier) and id_hint cookie helpers
  config/config.go       Reads env vars; panics on missing required ones
  db/
    db.go                pgxpool setup
    queries.go           UpsertUser, InsertLoginRecord (hand-written SQL)
  model/                 User, SessionUser structs
  session/
    store.go             scs.Store backed by PostgreSQL with AES-256-GCM encryption
    session.go           NewManager — creates scs.SessionManager with pgStore
migrations/              Goose SQL files (timestamp-based naming)
```

**Critical wiring detail:** SCS session middleware wraps `gin.Engine` at the `http.Server` level, not as a Gin middleware:
```go
srv.Handler = sessionMgr.LoadAndSave(r)  // r is *gin.Engine
```
Handlers access the session via `c.Request.Context()`, not directly from `c`.

### Frontend (`frontend/`)

React 18 + Vite 5 + TypeScript + React Router v6 + TanStack Query v5.

```
src/
  main.tsx          QueryClientProvider + RouterProvider setup
  router.tsx        Route definitions — / (LandingPage), /dashboard (ProtectedRoute)
  api/auth.ts       fetch wrappers for /api/auth/* endpoints
  hooks/useAuth.ts  useQuery over /api/auth/me; staleTime 5 min, no retry on 401
  components/       ProtectedRoute — redirects to login if unauthenticated
  pages/            LandingPage, DashboardPage
  types/            TypeScript types (User, etc.)
```

## Authentication Flow (BFF Pattern)

Tokens **never reach the browser**. The Go backend drives the full OIDC Authorization Code + PKCE flow:

1. `GET /api/auth/login` — backend generates state + PKCE verifier, stores them in a signed+encrypted httpOnly cookie (`gorilla/securecookie`), redirects to Keycloak.
2. `GET /api/auth/callback` — backend validates state (constant-time compare), exchanges code for tokens with the PKCE verifier, verifies `id_token` via JWKS.
3. Backend upserts user in PostgreSQL, destroys old SCS session (session fixation protection), creates new session with `SessionUser` JSON.
4. `rosetta_session` httpOnly cookie → opaque SCS token (no JWT).
5. `rosetta_id_hint` httpOnly cookie → raw `id_token`, scoped only to the `/api/auth/logout` path, used for Keycloak end-session.
6. `POST /api/auth/logout` — destroys session, returns Keycloak end-session URL for the browser to navigate to.

### Session Encryption

`internal/session/store.go` implements `scs.Store`:
- Key: `SHA-256(SESSION_SECRET)` → 32-byte AES key
- Each session row: `nonce || AES-256-GCM(data)` — nonce prepended to ciphertext

### Keycloak OIDC URLs (dev)

Two separate issuer URLs are used because the backend runs inside Docker:
- `OIDC_ISSUER` — external URL (`http://localhost:8080/...`), must match what Keycloak puts in `iss` claim
- `OIDC_ISSUER_INTERNAL` — internal Docker hostname (`http://keycloak:8080/...`), used for JWKS fetch

`InsecureIssuerURLContext` is used in dev to allow this mismatch. In production both values are the same.

## Environment Variables

Required secrets (see `.env.example` for all vars):
- `SESSION_SECRET` — ≥32 bytes, used to derive AES-256 session encryption key
- `COOKIE_HASH_KEY` — ≥32 bytes, HMAC key for pre-auth cookie signing
- `COOKIE_ENCRYPT_KEY` — exactly 32 bytes, AES-256 key for pre-auth cookie encryption
- `OIDC_CLIENT_SECRET` — copy from Keycloak admin after realm import

Generate secrets: `openssl rand -hex 32` (for 32-byte keys) or `openssl rand -hex 16` (for exactly-32-char keys).

## Database

Three tables (see `backend/migrations/`):
- `users` — one row per Keycloak `sub`, upserted on every login
- `login_records` — one row per login event (IP, user agent, session ID)
- `sessions` — SCS store (opaque token → encrypted blob)

Migration files use timestamp-based naming: `20260503000001_<name>.sql`. The `migrate` init container runs before the backend starts in Docker Compose.
