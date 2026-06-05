package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"

	"github.com/jharaxus/rosetta/internal/auth"
	"github.com/jharaxus/rosetta/internal/config"
	"github.com/jharaxus/rosetta/internal/db"
	"github.com/jharaxus/rosetta/internal/numbers"
	"github.com/jharaxus/rosetta/internal/session"
	"github.com/jharaxus/rosetta/internal/tts"
	"github.com/jharaxus/rosetta/internal/words"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()
	auth.InitCookies(cfg)

	if cfg.IsProduction {
		gin.SetMode(gin.ReleaseMode)
	}

	pool := db.NewPool(cfg.DatabaseURL)
	defer pool.Close()

	queries := db.New(pool)
	sessionMgr := session.NewManager(pool, cfg)

	oidcProvider, err := auth.NewOIDCProvider(ctx, cfg)
	if err != nil {
		slog.Error("failed to initialize OIDC provider", "err", err)
		os.Exit(1)
	}

	// Shared Valkey client — single connection pool for the process.
	valkeyClient := redis.NewClient(&redis.Options{Addr: cfg.ValkeyAddr})
	defer valkeyClient.Close()

	// TTS synthesis handler — skipped gracefully when credentials are absent.
	var ttsHandler *tts.Handler
	synth, synthErr := tts.NewGoogleSynthesizer(ctx)
	if synthErr != nil {
		slog.Warn("TTS synthesis unavailable — GOOGLE_APPLICATION_CREDENTIALS missing or invalid", "err", synthErr)
	} else {
		defer synth.Close()
		ttsCache := tts.NewValkeyCache(valkeyClient)
		ttsLimiter := tts.NewValkeyRateLimiter(valkeyClient, 20)
		ttsHandler = tts.NewHandler(synth, ttsCache, ttsLimiter, cfg.GoogleTTSVoice)
	}

	h := auth.NewHandler(queries, oidcProvider, sessionMgr, cfg)
	wh := words.NewHandler(queries, cfg.AudioBaseURL)
	nh := numbers.NewHandler(queries)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(slogMiddleware())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.FrontendURL},
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Rate limiter: 20 req/min per process (defense-in-depth; Nginx handles per-IP limiting)
	limiter := rate.NewLimiter(rate.Every(time.Minute/20), 20)
	authRateLimiter := func(c *gin.Context) {
		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate_limited"})
			return
		}
		c.Next()
	}

	r.GET("/healthz", h.HealthCheck)

	authGroup := r.Group("/api/auth")
	authGroup.Use(authRateLimiter)
	{
		authGroup.GET("/login", h.Login)
		authGroup.GET("/callback", h.Callback)
		authGroup.POST("/logout", auth.RequireAuth(sessionMgr), h.Logout)
		authGroup.GET("/me", auth.RequireAuth(sessionMgr), h.Me)
		authGroup.POST("/register", h.Register)
	}

	userGroup := r.Group("/api/user")
	userGroup.Use(auth.RequireAuth(sessionMgr))
	{
		userGroup.PATCH("/profile", h.UpdateProfile)
		userGroup.POST("/reset-progression", h.ResetProgression)
	}

	// Static audio files: public, immutable cache headers.
	// Named :filename (not *filepath) so the handler can use c.Param directly.
	r.GET("/static/audio/:filename", func(c *gin.Context) {
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.File(filepath.Join(cfg.AudioDir, c.Param("filename")))
	})

	// TTS dynamic synthesis: authenticated, Valkey-cached, rate-limited.
	if ttsHandler != nil {
		ttsGroup := r.Group("/api/tts")
		ttsGroup.Use(auth.RequireAuth(sessionMgr))
		ttsGroup.POST("/synthesize", ttsHandler.Synthesize)
	}

	wordsGroup := r.Group("/api/words")
	wordsGroup.Use(auth.RequireAuth(sessionMgr))
	{
		wordsGroup.GET("/flashcard", wh.GetFlashCard)
		wordsGroup.POST("/:word_id/review", wh.PostReview)
		wordsGroup.GET("/writing-flashcard", wh.GetWritingFlashCard)
		wordsGroup.POST("/:word_id/writing-review", wh.PostWritingReview)
	}

	userGroup.GET("/settings", nh.GetSettings)
	userGroup.PATCH("/settings", nh.UpdateSettings)

	numbersGroup := r.Group("/api/numbers")
	numbersGroup.Use(auth.RequireAuth(sessionMgr))
	{
		numbersGroup.GET("/practice", nh.GetPracticeNumber)
		numbersGroup.POST("/digit-success", nh.PostDigitSuccess)
		numbersGroup.GET("/failures/next", nh.GetNextFailure)
		numbersGroup.POST("/failures", nh.PostFailure)
		numbersGroup.DELETE("/failures/:number", nh.DeleteFailure)
	}

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      sessionMgr.LoadAndSave(r),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("server listening", "addr", srv.Addr, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal (SIGINT / SIGTERM via signal.NotifyContext at top of main).
	<-ctx.Done()
	stop() // release signal resources

	slog.Info("shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("forced shutdown", "err", err)
	}
	slog.Info("server stopped")
}

func slogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		slog.Info("request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency", time.Since(start),
			"ip", c.ClientIP(),
		)
	}
}
