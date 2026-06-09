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

make filter-verbs       # Extract verb rows from Deutch.csv → resources/Deutch_verbs.csv
make conjugate-verbs    # Fetch Wiktionary conjugations → resources/Deutch_verbs_and_conjugations.csv
make seed-conjugations  # Load conjugations into DB (requires running Docker stack)
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

## Development guidelines

### `backend/fsrs` — TDD required

The `fsrs` package is developed strictly with Test-Driven Development:
- Write or update tests **before** touching implementation code.
- Run the full test suite after every change: `cd backend && go test ./fsrs/... -v`
- **No modification to `fsrs/` is valid unless all tests pass.** Never adjust expected test values to fit a broken implementation — fix the math instead.

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
  main.tsx              QueryClientProvider + RouterProvider setup; imports global.css
  router.tsx            Routes: / · /register · /dashboard (protected) · /profile (protected)
  api/auth.ts           fetch wrappers for /api/auth/* and /api/user/* endpoints
  hooks/useAuth.ts      useQuery over /api/auth/me; staleTime 5 min, no retry on 401
  components/           ProtectedRoute, LogoutButton
  pages/                LandingPage, RegisterPage, DashboardPage, ProfilePage, NotFoundPage
  types/                User (id, sub, email, display_name, assimil_number)
  styles/
    tokens.css          All CSS custom properties (colors, fonts, radii, shadows, keyframes)
    global.css          Box-sizing reset + base html/body styles; imports tokens.css
```

#### Design System — Direction D · Atelier

All frontend UI follows the **Atelier** design language. Full reference: `/design-system` slash command (`.claude/commands/design-system.md`).

**Styling approach:** CSS Modules (`.module.css` co-located with each page). No Tailwind, no CSS-in-JS.

**Key tokens** (defined in `src/styles/tokens.css`):

| Token | Value | Role |
|-------|-------|------|
| `--color-bg` | `#fbf6ec` | Ivory page background |
| `--color-paper` | `#ffffff` | Card / input surfaces |
| `--color-ink` | `#1a2238` | Primary text + primary button bg |
| `--color-gold` | `#b88a3a` | Accent, active states, CTAs |
| `--color-gold-soft` | `#e8d4a4` | Tinted surfaces, hover |
| `--color-dim` | `#7a7466` | Muted / caption text |
| `--font-display` | Fraunces, serif | Headings, italic name accents |
| `--font-sans` | Manrope, sans-serif | Body, buttons, labels |
| `--font-mono` | JetBrains Mono | ALL-CAPS section labels |

**Rules:**
- All copy is in **French**.
- User names always render as italic gold Fraunces: `<em>Élise</em>`.
- Directional arrow is `→` inside an italic `<span>` using `var(--font-display)`.
- Buttons: pill shape (`border-radius: var(--radius-pill)`), height 48–52 px.
  - Primary = navy bg / ivory text · Gold = `--color-gold` bg / ink text · Ghost = transparent + 1 px border.
- Cards: white paper on ivory, `var(--shadow-card)`, `var(--radius-card)`.
- Section labels: `var(--font-mono)`, 9–11 px, ALL CAPS, `letter-spacing: 1.4px`, `color: var(--color-gold)`.
- Entrance animations: `animation: fadeUp 0.7s ease-out both` (stagger secondaries +0.1 s).

**Google Fonts** loaded in `frontend/index.html`: Fraunces (ital, opsz, wght 400/500/600), Manrope (wght 400–700), JetBrains Mono (wght 400/500).

### Keycloak Theme (`keycloak/themes/rosetta/`)

Custom login theme matching the Atelier design. Structure:

```
keycloak/themes/rosetta/
  login/
    theme.properties        parent=base, imports common/keycloak, loads css/login.css
    login.ftl               Standalone FreeMarker login template (no layout macro)
    resources/css/login.css Atelier CSS — same tokens as the React app, Google Fonts inlined
```

The theme is mounted into the container via `compose.dev.yml` and activated by `"loginTheme": "rosetta"` in `keycloak/realm-export.json`. After changing either, run `make dev-clean && make dev-build` to reimport the realm on a fresh DB.

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

Tables (see `backend/migrations/`):
- `users` — one row per Keycloak `sub`, upserted on every login
- `login_records` — one row per login event (IP, user agent, session ID)
- `sessions` — SCS store (opaque token → encrypted blob)
- `words` — vocabulary entries seeded from `resources/Deutch.csv`
- `conjugations` — verb conjugation forms; PK on `(word_id, tense, person)`; trigger enforces `word_id` must reference a Verb

Migration files use timestamp-based naming: `20260503000001_<name>.sql`. The `migrate` init container runs before the backend starts in Docker Compose.

## Vocabulary lexicon format (`resources/Deutch.csv`)

### Column structure

The CSV uses `;` as the delimiter and has 6 columns:

```
Français;Allemand;Leçon;Catégorie;Régularité;Annotation
```

### `Allemand` column — always `[alt1;alt2;...]`

Every German value is wrapped in brackets, even single words:

| Raw source | `Allemand` value | `Annotation` value |
|---|---|---|
| Single word | `[lernen]` | — |
| Multiple accepted translations | `[fahren;losfahren;weggehen]` | — |
| Case marker `ohne (+ acc.)` | `[ohne]` | `+ acc.` |
| Contraction `zu dem (= zum)` | `[zu dem;zum]` | — |
| Optional word `(genau) so` | `[so;genau so]` | — |
| Optional suffix `nie(mals)` | `[nie;niemals]` | — |
| Separable verb `auf/räumen` | `[aufräumen]` | `separation: auf/räumen` |
| Inflection paradigm `jeder/jede/jedes` | `[jeder]` | `inflection: jeder/jede/jedes` |
| Space-slash alternatives `rechts / links abbiegen` | `[rechts abbiegen;links abbiegen]` | — |

**Rules:**
- Fields containing `;` must be CSV-quoted (Python's `csv.writer` with `QUOTE_MINIMAL` handles this automatically).
- The `Annotation` column is free plain text, `NULL` when absent. It is **never** wrapped in `[...]`. The prefixes `separation:` and `inflection:` are used for cat-5 entries only; case markers and other notes are stored as-is.
- The `canonicalGerman()` function in `backend/internal/seed/seed_audio.go` strips `[...]` and returns the first `;`-delimited alternative — used for TTS generation and audio filename hashing.
- The frontend's `parseAlternatives(german)` in `frontend/src/utils/levenshtein.ts` does the same parsing at runtime for distance computation and display.

### Adding or editing entries

When adding a new word:
1. Write the `Allemand` value in `[alt1;alt2;...]` format — always bracketed, even for a single word.
2. Put case markers, separable-verb notation, or inflection paradigms in the `Annotation` column (plain text, no brackets).
3. Run the validator to check all rows conform:
   ```bash
   python3 -c "
   import csv, sys
   errors = []
   with open('resources/Deutch.csv') as f:
       r = csv.reader(f, delimiter=';')
       next(r)
       for i, row in enumerate(r, 2):
           if len(row) != 6: errors.append(f'L{i}: wrong field count {len(row)}')
           elif not (row[1].startswith('[') and row[1].endswith(']')): errors.append(f'L{i}: not bracketed: {row[1]}')
           elif '[' in row[5]: errors.append(f'L{i}: annotation has brackets: {row[5]}')
   print(errors or 'OK')
   "
   ```
4. Re-run `make filter-verbs && make conjugate-verbs` if any Verb entries changed, then `make seed-conjugations`.

## Conjugation pipeline

### Overview

German verb conjugations are stored in the `conjugations` table and seeded from
`resources/Deutch_verbs_and_conjugations.csv`. The pipeline has three stages:

1. **Filter**: extract verb rows from the full lexicon → `Deutch_verbs.csv`
2. **Conjugate**: fetch conjugations from de.wiktionary.org → `Deutch_verbs_and_conjugations.csv`
3. **Seed**: load the conjugation CSV into the DB via the Go seeder

### Generating the data files

Both CSV files are committed to the repo and should only be regenerated when the lexicon changes or conjugation data needs refreshing.

```bash
make filter-verbs    # Step 1: filter verb rows (fast, no network)
make conjugate-verbs # Step 2: fetch conjugations (~1 req/s, ~5 min for ~250 verbs)
make seed-conjugations # Step 3: seed the DB (requires running Docker stack)
```

To force re-seeding even if the table is already populated:
```bash
docker compose -f compose.dev.yml exec migrate \
  sh -c "FORCE_RESEED_CONJ=true go run ./cmd/migrate seed-conjugations"
```

### Python script (`resources/fetch_conjugations.py`)

Handles both `filter` and `conjugate` subcommands. Install dependencies:
```bash
pip install -r resources/requirements.txt
```

**TDD requirement**: `resources/fetch_conjugations_test.py` uses pytest with pre-fetched HTML fixtures in `resources/fixtures/`. Run tests with:
```bash
cd resources && pytest fetch_conjugations_test.py -v
```

Tests must pass before any change to `fetch_conjugations.py` is valid. Fixture JSON files (`resources/fixtures/flexion_*.json`) are committed and must not be auto-regenerated. The E2E reference is `resources/fixtures/expected_conjugations.csv`.

### Environment variables (conjugation seeder)

| Variable | Default | Description |
|---|---|---|
| `CONJ_SEED_FILE` | `/app/resources/Deutch_verbs_and_conjugations.csv` | Path to conjugation CSV |
| `FORCE_RESEED_CONJ` | `"false"` | Set to `"true"` to truncate and reload |

### Conjugation CSV format

`resources/Deutch_verbs_and_conjugations.csv` — 4 columns, `;` delimiter:

```
verb;tense;person;conjugation
[lernen];praesens_indikativ;1;[lerne]
[fahren;losfahren;weggehen];praesens_indikativ;1;"[fahre;fahre;gehe weg]"
```

- `verb` — the exact `Allemand` value from `Deutch.csv`, including `[...]` brackets. Multi-alternative verbs appear as one row with the merged forms from all alternatives.
- `conjugation` — always `[form1;form2;...]`. For a multi-alternative verb, forms from each individual alternative are concatenated in order. Single-form entries: `[fuhr]`.
- Fields containing `;` (multi-form conjugations) are CSV-quoted.

**Multi-alternative merging:** for a verb like `[sprechen;reden]`, `fetch_conjugations.py` fetches Wiktionary for each alternative separately, then merges `(tense, person)` form lists. The ich-Präsens row becomes `[spreche;rede]`.

### Schema

- `verb_tense` — PostgreSQL enum with 15 values (tense × mood: e.g. `praesens_indikativ`, `futur_1_konjunktiv_2`)
- `verb_person` — PostgreSQL enum: `p1_sg` (ich) through `p3_pl` (sie)
- `conjugations (word_id, tense, person, forms TEXT)` — `forms` is stored as `[form1;form2;...]` plain text; PK on the first three columns; trigger prevents inserting non-Verb word IDs
