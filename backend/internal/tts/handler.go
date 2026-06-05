package tts

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/jharaxus/rosetta/internal/auth"
)

// Handler handles dynamic TTS synthesis requests.
// All dependencies are injected as interfaces to enable unit testing without
// real GCP or Valkey connections.
type Handler struct {
	synth   SpeechSynthesizer
	cache   AudioCache
	limiter RateLimiter
	voice   string
}

// NewHandler constructs a Handler with the given dependencies.
func NewHandler(synth SpeechSynthesizer, cache AudioCache, limiter RateLimiter, voice string) *Handler {
	return &Handler{synth: synth, cache: cache, limiter: limiter, voice: voice}
}

// Synthesize handles POST /api/tts/synthesize.
// Expects JSON body: {"text": "..."}
// Returns audio/ogg bytes.
func (h *Handler) Synthesize(c *gin.Context) {
	var req struct {
		Text string `json:"text"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "corps JSON invalide"})
		return
	}

	text := req.Text
	if text == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "text requis"})
		return
	}
	if len(text) > 500 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "text trop long (maximum 500 caractères)"})
		return
	}
	if strings.TrimSpace(text)[0] == '<' {
		c.JSON(http.StatusBadRequest, gin.H{"error": "SSML non autorisé"})
		return
	}

	su, ok := auth.GetSessionUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "non authentifié"})
		return
	}

	ctx := c.Request.Context()

	allowed, err := h.limiter.Allow(ctx, su.ID.String())
	if err != nil {
		slog.Error("tts rate limiter error", "err", err)
		// Fail open on limiter errors to avoid blocking users due to Valkey hiccups.
	} else if !allowed {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "trop de requêtes"})
		return
	}

	key := CacheKey(text, h.voice)

	cached, hit, err := h.cache.Get(ctx, key)
	if err != nil {
		slog.Warn("tts cache get error", "err", err)
	}
	if hit {
		slog.Info("tts synthesis", "user_id", su.ID, "text_len", len(text), "cache_hit", true)
		c.Header("Cache-Control", "public, max-age=86400")
		c.Data(http.StatusOK, "audio/ogg", cached)
		return
	}

	start := time.Now()
	audio, err := h.synth.Synthesize(ctx, text, h.voice)
	elapsed := time.Since(start)
	if err != nil {
		slog.Error("tts synthesis failed", "err", err, "text_len", len(text))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur de synthèse"})
		return
	}

	// Fire-and-forget cache write — a failure must not fail the request.
	if setErr := h.cache.Set(ctx, key, audio); setErr != nil {
		slog.Warn("tts cache set error", "err", setErr)
	}

	slog.Info("tts synthesis", "user_id", su.ID, "text_len", len(text), "cache_hit", false, "duration_ms", elapsed.Milliseconds())

	c.Header("Cache-Control", "public, max-age=86400")
	c.Data(http.StatusOK, "audio/ogg", audio)
}
