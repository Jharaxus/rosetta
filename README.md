# Rosetta

Full-stack web application groundwork: React + Go + Keycloak + PostgreSQL.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│  Browser                                                │
│  React SPA (Vite)  ──/api/──►  Go BFF (Gin + SCS)      │
│  localhost:5173               localhost:8090            │
└──────────────┬──────────────────────────┬───────────────┘
               │ OIDC redirect            │ JDBC / pgx
               ▼                          ▼
         Keycloak 24                 PostgreSQL 16
         localhost:8080              localhost:5432
                  │
                  └── own DB (kc-db, port 5433)
```

### Services

| Service    | Dev URL                  | Description                            |
|------------|--------------------------|----------------------------------------|
| `frontend` | http://localhost:5173    | React SPA — Vite dev server            |
| `backend`  | http://localhost:8090    | Go API — Air hot-reload                |
| `keycloak` | http://localhost:8080    | OIDC identity provider                 |
| `app-db`   | localhost:5432           | Application PostgreSQL database        |
| `kc-db`    | localhost:5433           | Keycloak PostgreSQL database           |

### Authentication (BFF Pattern)

The Go backend drives the full OIDC Authorization Code + PKCE flow. Tokens **never reach the browser**:

1. Browser clicks "Sign in" → `GET /api/auth/login`
2. Backend generates state + PKCE verifier, stores in signed httpOnly cookie, redirects to Keycloak
3. Keycloak authenticates the user, redirects to `GET /api/auth/callback`
4. Backend validates state, exchanges code for tokens (PKCE), verifies `id_token` via JWKS
5. Backend upserts user and login record in PostgreSQL, creates SCS session
6. `id_token` stored in a separate short-lived httpOnly cookie scoped only to `/api/auth/logout`
7. Browser receives an opaque `rosetta_session` httpOnly cookie — **no JWT ever in JS**

## Quick Start (Dev)

### Prerequisites

- Docker + Docker Compose v2
- `make`

### Steps

```bash
# Clone and enter the repository
git clone <repo-url>
cd rosetta

# Start the full dev stack (first run downloads images and builds containers)
make dev-build

# Wait for all services to be healthy (≈60–90 seconds on first run)
# Keycloak takes the longest — watch logs with:
make logs
```

Once all services are healthy:

| URL | What you see |
|-----|--------------|
| http://localhost:5173 | Welcome page |
| http://localhost:8080 | Keycloak admin console (admin / admin) |

**Test credentials:** `admin` / `admin`

### Common Commands

```bash
make dev           # Start without rebuilding
make dev-down      # Stop (keep data)
make dev-clean     # Stop + wipe all volumes (full reset)
make logs          # Tail all service logs
make kc-export     # Export Keycloak realm config to keycloak/realm-export.json
```

## Production

```bash
# Create a .env file with production secrets (see .env.example)
cp .env.example .env
# Edit .env with real values

# Place TLS certificates in nginx/certs/
# nginx/certs/fullchain.pem
# nginx/certs/privkey.pem

make prod-build
make prod-up
```

## Repository Structure

```
rosetta/
├── backend/        Go API server (Gin + SCS + go-oidc + pgx)
├── frontend/       React SPA (Vite + React Router + TanStack Query)
├── keycloak/       Realm configuration and export
├── nginx/          Production reverse proxy config
├── compose.dev.yml Development Docker Compose (hot-reload, exposed ports)
├── compose.prod.yml Production Docker Compose (distroless, TLS, rate limiting)
├── .env.dev        Dev-safe default values (committed)
└── .env.example    Template for production secrets (committed, no real values)
```

See each service's `README.md` for details.

## Database Schema

Three tables in the application database:

- `users` — one row per Keycloak subject (`sub` claim), upserted on every login
- `login_records` — one row per login event (timestamp, IP, user agent, session ID)
- `sessions` — SCS session store (opaque token → encrypted session data)
