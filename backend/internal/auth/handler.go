package auth

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"

	"github.com/jharaxus/rosetta/internal/config"
	"github.com/jharaxus/rosetta/internal/db"
	"github.com/jharaxus/rosetta/internal/model"
)

const (
	sessionKeyUser = "user"
	contextKeyUser = "auth_user"
)

type Handler struct {
	queries  *db.Queries
	oidc     *OIDCProvider
	sessions *scs.SessionManager
	cfg      *config.Config
}

func NewHandler(queries *db.Queries, oidc *OIDCProvider, sessions *scs.SessionManager, cfg *config.Config) *Handler {
	return &Handler{queries: queries, oidc: oidc, sessions: sessions, cfg: cfg}
}

// Login starts the OIDC flow: generate state + PKCE verifier, store in signed cookie,
// redirect browser to Keycloak.
func (h *Handler) Login(c *gin.Context) {
	state, err := h.oidc.GenerateState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "auth_error"})
		return
	}
	verifier, err := h.oidc.GenerateVerifier()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "auth_error"})
		return
	}
	if err := setPreAuthCookie(c.Writer, c.Request, state, verifier, h.cfg.IsProduction); err != nil {
		slog.Error("set pre-auth cookie", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "auth_error"})
		return
	}
	c.Redirect(http.StatusFound, h.oidc.AuthCodeURL(state, verifier))
}

// Callback handles Keycloak's redirect after authentication.
func (h *Handler) Callback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	if code == "" || state == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "auth_error"})
		return
	}

	preAuth, err := readPreAuthCookie(c.Request)
	if err != nil {
		slog.Warn("invalid or missing pre-auth cookie", "err", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "auth_error"})
		return
	}
	deletePreAuthCookie(c.Writer, h.cfg.IsProduction)

	// Constant-time comparison prevents timing side-channels on the state value.
	if subtle.ConstantTimeCompare([]byte(state), []byte(preAuth.State)) != 1 {
		slog.Warn("state mismatch on OIDC callback")
		c.JSON(http.StatusBadRequest, gin.H{"error": "auth_error"})
		return
	}

	token, err := h.oidc.Exchange(c.Request.Context(), code, preAuth.Verifier)
	if err != nil {
		slog.Error("token exchange failed", "err", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "auth_error"})
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		slog.Error("id_token missing from Keycloak token response")
		c.JSON(http.StatusBadGateway, gin.H{"error": "auth_error"})
		return
	}

	idToken, err := h.oidc.VerifyIDToken(c.Request.Context(), rawIDToken)
	if err != nil {
		slog.Error("id_token verification failed", "err", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "auth_error"})
		return
	}

	claims, err := h.oidc.ExtractClaims(idToken)
	if err != nil {
		slog.Error("extract id_token claims", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "auth_error"})
		return
	}

	displayName := claims.Name
	if displayName == "" {
		displayName = claims.PreferredUsername
	}
	if displayName == "" {
		displayName = claims.Email
	}
	if displayName == "" {
		displayName = claims.Sub
	}

	user, err := h.queries.UpsertUser(c.Request.Context(), claims.Sub, claims.Email, displayName)
	if err != nil {
		slog.Error("upsert user", "sub", claims.Sub, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "session_error"})
		return
	}

	// Destroy old session before creating a new one — session fixation protection.
	// Soft-fail: the OIDC code exchange already succeeded; aborting here would lock
	// the user out. The old session will expire naturally if Destroy fails.
	if err := h.sessions.Destroy(c.Request.Context()); err != nil {
		slog.Warn("destroy old session", "err", err)
	}

	sessionUser := model.SessionUser{
		ID:          user.ID,
		Subject:     user.Subject,
		Email:       user.Email,
		DisplayName: user.DisplayName,
	}
	userJSON, err := json.Marshal(sessionUser)
	if err != nil {
		slog.Error("marshal session user", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "session_error"})
		return
	}
	h.sessions.Put(c.Request.Context(), sessionKeyUser, string(userJSON))

	// Insert login record after the new session is created so we capture the new session token.
	if err := h.queries.InsertLoginRecord(
		c.Request.Context(),
		user.ID,
		realIP(c.Request),
		c.Request.UserAgent(),
		h.sessions.Token(c.Request.Context()),
	); err != nil {
		slog.Warn("insert login record", "user_id", user.ID, "err", err)
	}

	// id_token stored in a separate httpOnly cookie scoped to /api/auth/logout only.
	// Never stored server-side; never in any API response.
	setIDHintCookie(c.Writer, rawIDToken, idToken.Expiry, h.cfg.IsProduction)

	c.Redirect(http.StatusFound, h.cfg.FrontendURL)
}

// Logout destroys the session and returns the Keycloak end-session URL for the
// client to navigate to. Uses POST so SameSite=Lax blocks cross-site logout CSRF.
func (h *Handler) Logout(c *gin.Context) {
	idTokenHint := readIDHintCookie(c.Request)
	deleteIDHintCookie(c.Writer, h.cfg.IsProduction)

	if err := h.sessions.Destroy(c.Request.Context()); err != nil {
		slog.Error("destroy session on logout", "err", err)
	}

	endSessionURL, err := h.oidc.EndSessionURL(idTokenHint, h.cfg.FrontendURL)
	if err != nil {
		slog.Error("build end_session URL", "err", err)
		c.JSON(http.StatusOK, gin.H{"redirect": h.cfg.FrontendURL})
		return
	}
	c.JSON(http.StatusOK, gin.H{"redirect": endSessionURL})
}

// Me returns the authenticated user's profile, always fetched fresh from the DB.
func (h *Handler) Me(c *gin.Context) {
	sessionUser, ok := c.Get(contextKeyUser)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "context_missing_user"})
		return
	}
	su := sessionUser.(*model.SessionUser)
	user, err := h.queries.GetUserByID(c.Request.Context(), su.ID)
	if err != nil {
		slog.Error("get user by id", "id", su.ID, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":             user.ID,
		"sub":            user.Subject,
		"email":          user.Email,
		"display_name":   user.DisplayName,
		"assimil_number": user.AssimilNumber,
	})
}

type updateProfileRequest struct {
	AssimilNumber int `json:"assimil_number" binding:"required,min=1,max=100"`
}

// UpdateProfile updates the authenticated user's profile fields.
func (h *Handler) UpdateProfile(c *gin.Context) {
	sessionUser, ok := c.Get(contextKeyUser)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "context_missing_user"})
		return
	}
	su := sessionUser.(*model.SessionUser)

	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_error", "detail": err.Error()})
		return
	}

	user, err := h.queries.UpdateAssimilNumber(c.Request.Context(), su.ID, req.AssimilNumber)
	if err != nil {
		slog.Error("update assimil number", "id", su.ID, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":             user.ID,
		"sub":            user.Subject,
		"email":          user.Email,
		"display_name":   user.DisplayName,
		"assimil_number": user.AssimilNumber,
	})
}

// HealthCheck responds 200 for liveness probes.
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type registerRequest struct {
	Email       string `json:"email"        binding:"required,email,max=254"`
	DisplayName string `json:"display_name" binding:"required,min=2,max=100"`
	Password    string `json:"password"     binding:"required,min=8"`
}

// Register creates a new user in Keycloak and upserts them into PostgreSQL.
func (h *Handler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_error", "detail": err.Error()})
		return
	}

	admin, err := newKeycloakAdmin(h.cfg)
	if err != nil {
		slog.Error("build keycloak admin client", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	token, err := admin.getAdminToken(c.Request.Context())
	if err != nil {
		slog.Error("get keycloak admin token", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	sub, err := admin.createUser(c.Request.Context(), token, req.Email, req.DisplayName, req.Password)
	if errors.Is(err, ErrEmailConflict) {
		c.JSON(http.StatusConflict, gin.H{"error": "email_taken"})
		return
	}
	if err != nil {
		slog.Error("create keycloak user", "email", req.Email, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	user, err := h.queries.UpsertUser(c.Request.Context(), sub, req.Email, req.DisplayName)
	if err != nil {
		slog.Error("upsert user after keycloak create", "sub", sub, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":           user.ID,
		"sub":          user.Subject,
		"email":        user.Email,
		"display_name": user.DisplayName,
	})
}

func realIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Nginx appends the client IP as the rightmost entry; leftmost entries
		// are forwarded by the client and cannot be trusted.
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[len(parts)-1])
	}
	// RemoteAddr is "ip:port"; strip the port so netip.ParseAddr succeeds.
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
