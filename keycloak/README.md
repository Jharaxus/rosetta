# Keycloak

OIDC identity provider configuration for Rosetta.

## Realm: `rosetta`

The realm export at `realm-export.json` configures:

- **Client**: `rosetta-bff` — confidential client with PKCE (S256) enforced
  - Standard flow only (no implicit, no direct grants)
  - Exact redirect URIs (no wildcards)
  - Explicit token lifetimes (access: 5min, refresh: 30min, SSO: 8h)
- **Test user**: `testuser` / `rosetta-dev-password` (dev only — remove before production)

## Dev Access

| URL                              | Credentials                   |
|----------------------------------|-------------------------------|
| http://localhost:8080            | Keycloak admin console        |
| http://localhost:8080/realms/rosetta/.well-known/openid-configuration | OIDC discovery |

Admin login: `admin` / `rosetta-dev-admin` (from `.env.dev`)

## Updating the Realm

1. Make changes in the Keycloak admin console
2. Export the realm:
   ```bash
   make kc-export
   ```
3. Review `keycloak/realm-export.json` — remove any sensitive data before committing
4. Commit the updated export

## Client Secret

The dev client secret is hardcoded in `realm-export.json` as `rosetta-dev-bff-secret-000000000000` and matches `OIDC_CLIENT_SECRET` in `.env.dev`. For production:

1. Generate a new secret in the Keycloak admin UI (Clients → rosetta-bff → Credentials → Regenerate)
2. Set `OIDC_CLIENT_SECRET=<new_secret>` in your production `.env`

## Keycloak Docker Networking

In dev, Keycloak runs at `localhost:8080` from the host and `keycloak:8080` from within Docker. Both the browser and the backend's OIDC library must agree on the issuer URL.

`KC_HOSTNAME_URL=http://localhost:8080` makes Keycloak advertise `localhost:8080` as its issuer in all tokens. The backend uses `OIDC_ISSUER_INTERNAL=http://keycloak:8080/realms/rosetta` to fetch JWKS from the internal hostname while validating token `iss` claims against `OIDC_ISSUER=http://localhost:8080/realms/rosetta`.

In production, both URLs are the same (`https://auth.yourdomain.com`) and `OIDC_ISSUER_INTERNAL` can be omitted.
