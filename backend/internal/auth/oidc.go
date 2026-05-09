package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/jharaxus/rosetta/internal/config"
)

type OIDCProvider struct {
	provider    *gooidc.Provider
	oauthConfig oauth2.Config
	verifier    *gooidc.IDTokenVerifier
	cfg         *config.Config
}

// IDTokenClaims holds the standard claims we extract from the id_token.
type IDTokenClaims struct {
	Sub               string `json:"sub"`
	Email             string `json:"email"`
	Name              string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
}

func NewOIDCProvider(ctx context.Context, cfg *config.Config) (*OIDCProvider, error) {
	fetchCtx := ctx

	// In dev, Keycloak's issuer claim uses the external hostname (e.g. localhost:8080)
	// but the backend fetches discovery from the internal Docker hostname (keycloak:8080).
	// InsecureIssuerURLContext allows mismatched fetch URL vs issuer claim.
	// This is safe because token verification still validates `iss` against the discovery doc.
	if cfg.OIDCIssuerInternal != cfg.OIDCIssuer {
		fetchCtx = gooidc.InsecureIssuerURLContext(ctx, cfg.OIDCIssuer)
	}

	provider, err := gooidc.NewProvider(fetchCtx, cfg.OIDCIssuerInternal)
	if err != nil {
		return nil, fmt.Errorf("initializing OIDC provider from %s: %w", cfg.OIDCIssuerInternal, err)
	}

	oauthConfig := oauth2.Config{
		ClientID:     cfg.OIDCClientID,
		ClientSecret: cfg.OIDCClientSecret,
		RedirectURL:  cfg.OIDCRedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{gooidc.ScopeOpenID, "profile", "email"},
	}

	verifier := provider.Verifier(&gooidc.Config{ClientID: cfg.OIDCClientID})

	return &OIDCProvider{
		provider:    provider,
		oauthConfig: oauthConfig,
		verifier:    verifier,
		cfg:         cfg,
	}, nil
}

func generateRandomBase64URL(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (p *OIDCProvider) GenerateState() (string, error) {
	return generateRandomBase64URL(32)
}

func (p *OIDCProvider) GenerateVerifier() (string, error) {
	return generateRandomBase64URL(32)
}

// AuthCodeURL builds the Keycloak authorization URL with PKCE S256.
// oauth2.S256ChallengeOption takes the raw verifier and computes the challenge internally.
func (p *OIDCProvider) AuthCodeURL(state, verifier string) string {
	return p.oauthConfig.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))
}

func (p *OIDCProvider) Exchange(ctx context.Context, code, verifier string) (*oauth2.Token, error) {
	return p.oauthConfig.Exchange(ctx, code, oauth2.VerifierOption(verifier))
}

func (p *OIDCProvider) VerifyIDToken(ctx context.Context, rawIDToken string) (*gooidc.IDToken, error) {
	return p.verifier.Verify(ctx, rawIDToken)
}

func (p *OIDCProvider) ExtractClaims(idToken *gooidc.IDToken) (IDTokenClaims, error) {
	var claims IDTokenClaims
	err := idToken.Claims(&claims)
	return claims, err
}

// EndSessionURL builds the Keycloak end_session URL for front-channel logout.
func (p *OIDCProvider) EndSessionURL(idTokenHint, postLogoutRedirectURI string) (string, error) {
	var disc struct {
		EndSessionEndpoint string `json:"end_session_endpoint"`
	}
	if err := p.provider.Claims(&disc); err != nil || disc.EndSessionEndpoint == "" {
		// Fallback to well-known Keycloak path
		disc.EndSessionEndpoint = p.cfg.OIDCIssuer + "/protocol/openid-connect/logout"
	}

	u, err := url.Parse(disc.EndSessionEndpoint)
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Set("client_id", p.cfg.OIDCClientID)
	if idTokenHint != "" {
		q.Set("id_token_hint", idTokenHint)
	}
	if postLogoutRedirectURI != "" {
		q.Set("post_logout_redirect_uri", postLogoutRedirectURI)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}
