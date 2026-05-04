.DEFAULT_GOAL := help
ENV_FILE      ?= .env.dev

.PHONY: help dev dev-build dev-down dev-clean \
        prod-build prod-up prod-down \
        migrate-status migrate-up migrate-down \
        kc-export logs

# ── Help ───────────────────────────────────────────────────────────────────────
help:
	@echo ""
	@echo "Rosetta — available targets:"
	@echo ""
	@echo "  dev           Start the dev stack (postgres, keycloak, backend, frontend)"
	@echo "  dev-build     Rebuild images then start the dev stack"
	@echo "  dev-down      Stop the dev stack (keep volumes)"
	@echo "  dev-clean     Stop the dev stack AND delete all volumes (full reset)"
	@echo ""
	@echo "  prod-build    Build production images"
	@echo "  prod-up       Start the production stack (detached)"
	@echo "  prod-down     Stop the production stack"
	@echo ""
	@echo "  migrate-status  Show current migration status"
	@echo "  migrate-up      Run pending migrations"
	@echo "  migrate-down    Roll back the latest migration"
	@echo ""
	@echo "  kc-export     Export the Keycloak realm to keycloak/realm-export.json"
	@echo "  logs          Tail logs from all dev services"
	@echo ""
	@echo "  ENV_FILE=<path>  Override the env file (default: .env.dev)"
	@echo ""

# ── Dev ────────────────────────────────────────────────────────────────────────
dev:
	docker compose -f compose.dev.yml --env-file $(ENV_FILE) up

dev-build:
	docker compose -f compose.dev.yml --env-file $(ENV_FILE) up --build

dev-down:
	docker compose -f compose.dev.yml --env-file $(ENV_FILE) down

dev-clean:
	docker compose -f compose.dev.yml --env-file $(ENV_FILE) down -v

logs:
	docker compose -f compose.dev.yml --env-file $(ENV_FILE) logs -f

# ── Production ─────────────────────────────────────────────────────────────────
prod-build:
	docker compose -f compose.prod.yml --env-file $(ENV_FILE) build

prod-up:
	docker compose -f compose.prod.yml --env-file $(ENV_FILE) up -d

prod-down:
	docker compose -f compose.prod.yml --env-file $(ENV_FILE) down

# ── Migrations (run inside running backend container) ──────────────────────────
migrate-status:
	docker compose -f compose.dev.yml --env-file $(ENV_FILE) exec backend \
	  go run ./cmd/migrate status

migrate-up:
	docker compose -f compose.dev.yml --env-file $(ENV_FILE) exec backend \
	  go run ./cmd/migrate up

migrate-down:
	docker compose -f compose.dev.yml --env-file $(ENV_FILE) exec backend \
	  go run ./cmd/migrate down

# ── Keycloak ───────────────────────────────────────────────────────────────────
kc-export:
	@echo "Exporting Keycloak realm to keycloak/realm-export.json ..."
	docker compose -f compose.dev.yml --env-file $(ENV_FILE) exec keycloak \
	  /opt/keycloak/bin/kc.sh export \
	    --realm rosetta \
	    --file /tmp/realm-export.json \
	    --users realm_file
	docker compose -f compose.dev.yml --env-file $(ENV_FILE) \
	  cp keycloak:/tmp/realm-export.json ./keycloak/realm-export.json
	@echo "Done. Review keycloak/realm-export.json before committing."
