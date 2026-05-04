# Nginx

Production reverse proxy configuration.

## What It Does

- TLS termination (TLS 1.2/1.3, OCSP stapling, HSTS)
- Routes `/auth/` → Keycloak, `/api/` → Go backend, `/` → React SPA
- Blocks Keycloak admin console from public access
- Per-IP rate limiting on auth endpoints (10 req/min) and all API endpoints (60 req/min)
- Security headers: HSTS, CSP, X-Frame-Options, X-Content-Type-Options, Referrer-Policy

## TLS Setup

Place certificates in `nginx/certs/` (gitignored):

```
nginx/certs/
├── fullchain.pem   # Certificate chain (cert + intermediates)
└── privkey.pem     # Private key
```

Recommended: use [Certbot](https://certbot.eff.org) or [acme.sh](https://acme.sh) to obtain Let's Encrypt certificates.

## Configuration

`nginx.prod.conf` is mounted read-only into the Nginx container. Edit it directly; changes take effect after `make prod-down && make prod-up`.

## Not Used in Dev

In development, the Vite dev server proxies `/api/*` directly to the Go backend. Nginx is only present in `compose.prod.yml`.

## Keycloak Admin Console

The Nginx config explicitly returns 403 for:
- `/auth/admin/**`
- `/auth/master/**`
- `/auth/realms/master/**`

The Keycloak admin console must only be accessed via a direct connection to port 8080 (not exposed externally in prod) or via a VPN/bastion host.
