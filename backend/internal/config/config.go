package config

import (
	"fmt"
	"os"
)

type Config struct {
	DatabaseURL          string
	OIDCIssuer           string
	OIDCIssuerInternal   string
	OIDCClientID         string
	OIDCClientSecret     string
	OIDCRedirectURL      string
	KeycloakAdminUser    string
	KeycloakAdminPassword string
	SessionSecret        []byte
	CookieHashKey        []byte
	CookieEncryptKey     []byte
	FrontendURL          string
	Port                 string
	AppEnv               string
	IsProduction         bool
	CapAPIURL            string
	CapSecretKey         string
}

func Load() *Config {
	cfg := &Config{
		DatabaseURL:           requireEnv("DATABASE_URL"),
		OIDCIssuer:            requireEnv("OIDC_ISSUER"),
		OIDCClientID:          requireEnv("OIDC_CLIENT_ID"),
		OIDCClientSecret:      requireEnv("OIDC_CLIENT_SECRET"),
		OIDCRedirectURL:       requireEnv("OIDC_REDIRECT_URL"),
		KeycloakAdminUser:     requireEnv("KEYCLOAK_ADMIN"),
		KeycloakAdminPassword: requireEnv("KEYCLOAK_ADMIN_PASSWORD"),
		FrontendURL:           requireEnv("FRONTEND_URL"),
		Port:                  envOr("BACKEND_PORT", "8090"),
		AppEnv:                envOr("APP_ENV", "development"),
	}

	cfg.OIDCIssuerInternal = envOr("OIDC_ISSUER_INTERNAL", cfg.OIDCIssuer)
	cfg.IsProduction = cfg.AppEnv == "production"
	cfg.CapAPIURL    = envOr("CAP_API_URL", "")
	cfg.CapSecretKey = envOr("CAP_SECRET_KEY", "")

	sessionSecret := []byte(requireEnv("SESSION_SECRET"))
	if len(sessionSecret) < 32 {
		panic("SESSION_SECRET must be at least 32 bytes")
	}
	cfg.SessionSecret = sessionSecret

	cookieHashKey := []byte(requireEnv("COOKIE_HASH_KEY"))
	if len(cookieHashKey) < 32 {
		panic("COOKIE_HASH_KEY must be at least 32 bytes")
	}
	cfg.CookieHashKey = cookieHashKey

	cookieEncryptKey := []byte(requireEnv("COOKIE_ENCRYPT_KEY"))
	if len(cookieEncryptKey) != 32 {
		panic(fmt.Sprintf("COOKIE_ENCRYPT_KEY must be exactly 32 bytes (got %d)", len(cookieEncryptKey)))
	}
	cfg.CookieEncryptKey = cookieEncryptKey

	return cfg
}

func requireEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		panic(fmt.Sprintf("required environment variable %q is not set", key))
	}
	return val
}

func envOr(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
