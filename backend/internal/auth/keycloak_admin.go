package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/jharaxus/rosetta/internal/config"
)

// ErrEmailConflict is returned when Keycloak reports that the email is already registered.
var ErrEmailConflict = errors.New("email already registered")

type keycloakAdmin struct {
	baseURL   string
	realm     string
	adminUser string
	adminPass string
	http      *http.Client
}

// newKeycloakAdmin derives the Keycloak base URL and realm from cfg.OIDCIssuerInternal
// (expected format: "http://keycloak:8080/realms/rosetta").
func newKeycloakAdmin(cfg *config.Config) (*keycloakAdmin, error) {
	parts := strings.SplitN(cfg.OIDCIssuerInternal, "/realms/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("cannot parse realm from OIDC_ISSUER_INTERNAL %q", cfg.OIDCIssuerInternal)
	}
	return &keycloakAdmin{
		baseURL:   parts[0],
		realm:     parts[1],
		adminUser: cfg.KeycloakAdminUser,
		adminPass: cfg.KeycloakAdminPassword,
		http:      &http.Client{Timeout: 10 * time.Second},
	}, nil
}

type kcTokenResponse struct {
	AccessToken string `json:"access_token"`
}

// getAdminToken fetches a short-lived admin token from the master realm using
// Resource Owner Password Credentials against the built-in admin-cli client.
func (k *keycloakAdmin) getAdminToken(ctx context.Context) (string, error) {
	endpoint := k.baseURL + "/realms/master/protocol/openid-connect/token"

	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("client_id", "admin-cli")
	form.Set("username", k.adminUser)
	form.Set("password", k.adminPass)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := k.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("keycloak token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("keycloak token request returned %d: %s", resp.StatusCode, body)
	}

	var tr kcTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}
	return tr.AccessToken, nil
}

type kcCreateUserRequest struct {
	Username      string          `json:"username"`
	Email         string          `json:"email"`
	FirstName     string          `json:"firstName"`
	Enabled       bool            `json:"enabled"`
	EmailVerified bool            `json:"emailVerified"`
	Credentials   []kcCredential  `json:"credentials"`
}

type kcCredential struct {
	Type      string `json:"type"`
	Value     string `json:"value"`
	Temporary bool   `json:"temporary"`
}

// createUser creates the user in Keycloak and returns the new user's subject (UUID).
// Returns ErrEmailConflict if Keycloak responds 409.
func (k *keycloakAdmin) createUser(ctx context.Context, token, email, displayName, password string) (string, error) {
	endpoint := fmt.Sprintf("%s/admin/realms/%s/users", k.baseURL, k.realm)

	body := kcCreateUserRequest{
		Username:      email,
		Email:         email,
		FirstName:     displayName,
		Enabled:       true,
		EmailVerified: true,
		Credentials: []kcCredential{
			{Type: "password", Value: password, Temporary: false},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal create user request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("build create user request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := k.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("keycloak create user request: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	switch resp.StatusCode {
	case http.StatusCreated:
		location := resp.Header.Get("Location")
		if location == "" {
			return "", fmt.Errorf("keycloak did not return Location header after user creation")
		}
		return path.Base(location), nil
	case http.StatusConflict:
		return "", ErrEmailConflict
	default:
		return "", fmt.Errorf("keycloak create user returned unexpected status %d", resp.StatusCode)
	}
}
