package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"github.com/jharaxus/rosetta/internal/auth"
	"github.com/jharaxus/rosetta/internal/config"
	"github.com/jharaxus/rosetta/internal/db"
	"github.com/jharaxus/rosetta/internal/session"
	"github.com/jharaxus/rosetta/internal/words"
)

func main() {
	cfg := config.Load()
	auth.InitCookies(cfg)

	if cfg.IsProduction {
		gin.SetMode(gin.ReleaseMode)
	}

	pool := db.NewPool(cfg.DatabaseURL)
	defer pool.Close()

	queries := db.New(pool)
	sessionMgr := session.NewManager(pool, cfg)

	oidcProvider, err := auth.NewOIDCProvider(context.Background(), cfg)
	if err != nil {
		slog.Error("failed to initialize OIDC provider", "err", err)
		os.Exit(1)
	}

	h := auth.NewHandler(queries, oidcProvider, sessionMgr, cfg)
	wh := words.NewHandler(queries)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(slogMiddleware())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.FrontendURL},
		AllowMethods:     []string{"GET", "POST", "PATCH", "OPTIONS"},
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
		userGroup.DELETE("/cards", h.DeleteCards)
	}

	wordsGroup := r.Group("/api/words")
	wordsGroup.Use(auth.RequireAuth(sessionMgr))
	{
		wordsGroup.GET("/flashcard", wh.GetFlashCard)
		wordsGroup.POST("/:word_id/review", wh.PostReview)
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

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
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
