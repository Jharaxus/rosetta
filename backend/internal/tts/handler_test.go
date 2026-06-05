package tts_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/jharaxus/rosetta/internal/model"
	"github.com/jharaxus/rosetta/internal/tts"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ── stub implementations ──────────────────────────────────────────────────────

type stubSynth struct {
	called int
	audio  []byte
	err    error
}

func (s *stubSynth) Synthesize(_ context.Context, _, _ string) ([]byte, error) {
	s.called++
	return s.audio, s.err
}

type stubCache struct {
	store   map[string][]byte
	setErr  error
}

func newStubCache() *stubCache { return &stubCache{store: map[string][]byte{}} }

func (c *stubCache) Get(_ context.Context, key string) ([]byte, bool, error) {
	v, ok := c.store[key]
	return v, ok, nil
}

func (c *stubCache) Set(_ context.Context, key string, value []byte) error {
	if c.setErr != nil {
		return c.setErr
	}
	c.store[key] = value
	return nil
}

type stubLimiter struct{ deny bool; err error }

func (l *stubLimiter) Allow(_ context.Context, _ string) (bool, error) {
	return !l.deny, l.err
}

// ── test helpers ──────────────────────────────────────────────────────────────

func newEngine(h *tts.Handler) *gin.Engine {
	r := gin.New()
	// Inject a test session user — matches the key set by auth.RequireAuth.
	r.Use(func(c *gin.Context) {
		c.Set("auth_user", &model.SessionUser{ID: uuid.New(), Email: "test@example.com"})
		c.Next()
	})
	r.POST("/synthesize", h.Synthesize)
	return r
}

func postJSON(r *gin.Engine, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/synthesize", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ── validation tests ──────────────────────────────────────────────────────────

func TestSynthesize_EmptyText(t *testing.T) {
	h := tts.NewHandler(&stubSynth{}, newStubCache(), &stubLimiter{}, "de-DE-Neural2-F")
	w := postJSON(newEngine(h), map[string]string{"text": ""})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSynthesize_TextTooLong(t *testing.T) {
	h := tts.NewHandler(&stubSynth{}, newStubCache(), &stubLimiter{}, "de-DE-Neural2-F")
	long := make([]byte, 501)
	for i := range long {
		long[i] = 'a'
	}
	w := postJSON(newEngine(h), map[string]string{"text": string(long)})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSynthesize_SSMLRejected(t *testing.T) {
	h := tts.NewHandler(&stubSynth{}, newStubCache(), &stubLimiter{}, "de-DE-Neural2-F")
	w := postJSON(newEngine(h), map[string]string{"text": "<speak>lernen</speak>"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["error"] == "" {
		t.Error("expected non-empty error field in response body")
	}
}

func TestSynthesize_RateLimited(t *testing.T) {
	h := tts.NewHandler(&stubSynth{audio: []byte("ogg")}, newStubCache(), &stubLimiter{deny: true}, "de-DE-Neural2-F")
	w := postJSON(newEngine(h), map[string]string{"text": "lernen"})
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}
}

// ── cache miss ────────────────────────────────────────────────────────────────

func TestSynthesize_CacheMiss_CallsSynth(t *testing.T) {
	synth := &stubSynth{audio: []byte("fake-ogg")}
	cache := newStubCache()
	h := tts.NewHandler(synth, cache, &stubLimiter{}, "de-DE-Neural2-F")

	w := postJSON(newEngine(h), map[string]string{"text": "lernen"})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if synth.called != 1 {
		t.Errorf("expected SpeechSynthesizer called once, got %d", synth.called)
	}
	if !bytes.Equal(w.Body.Bytes(), []byte("fake-ogg")) {
		t.Error("response body does not match synthesized audio")
	}
	if w.Header().Get("Content-Type") != "audio/ogg" {
		t.Errorf("expected Content-Type audio/ogg, got %s", w.Header().Get("Content-Type"))
	}
}

func TestSynthesize_CacheMiss_StoresInCache(t *testing.T) {
	synth := &stubSynth{audio: []byte("fake-ogg")}
	cache := newStubCache()
	h := tts.NewHandler(synth, cache, &stubLimiter{}, "de-DE-Neural2-F")

	postJSON(newEngine(h), map[string]string{"text": "lernen"})

	// The cache must contain exactly one entry after a miss.
	if len(cache.store) != 1 {
		t.Errorf("expected 1 cache entry, got %d", len(cache.store))
	}
}

func TestSynthesize_CacheMiss_CacheWriteFailureDoesNotFailRequest(t *testing.T) {
	synth := &stubSynth{audio: []byte("fake-ogg")}
	cache := &stubCache{store: map[string][]byte{}, setErr: errors.New("valkey down")}
	h := tts.NewHandler(synth, cache, &stubLimiter{}, "de-DE-Neural2-F")

	// Cache write failure must not propagate to the HTTP response.
	w := postJSON(newEngine(h), map[string]string{"text": "lernen"})
	if w.Code != http.StatusOK {
		t.Fatalf("cache write failure should not fail request; got %d", w.Code)
	}
}

// ── cache hit ─────────────────────────────────────────────────────────────────

func TestSynthesize_CacheHit_DoesNotCallSynth(t *testing.T) {
	synth := &stubSynth{}
	cache := newStubCache()
	h := tts.NewHandler(synth, cache, &stubLimiter{}, "de-DE-Neural2-F")

	// Pre-populate cache with the expected key.
	key := tts.CacheKey("lernen", "de-DE-Neural2-F")
	cache.store[key] = []byte("cached-ogg")

	w := postJSON(newEngine(h), map[string]string{"text": "lernen"})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if synth.called != 0 {
		t.Errorf("SpeechSynthesizer must not be called on cache hit, called %d times", synth.called)
	}
	if !bytes.Equal(w.Body.Bytes(), []byte("cached-ogg")) {
		t.Error("response body does not match cached audio")
	}
}

// ── synthesis error ───────────────────────────────────────────────────────────

func TestSynthesize_SynthError_Returns500(t *testing.T) {
	synth := &stubSynth{err: errors.New("google API down")}
	h := tts.NewHandler(synth, newStubCache(), &stubLimiter{}, "de-DE-Neural2-F")

	w := postJSON(newEngine(h), map[string]string{"text": "lernen"})
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on synth error, got %d", w.Code)
	}
}
