package session

import (
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jharaxus/rosetta/internal/config"
)

func NewManager(pool *pgxpool.Pool, cfg *config.Config) *scs.SessionManager {
	mgr := scs.New()
	mgr.Store = newPGStore(pool, cfg.SessionSecret)
	mgr.Lifetime = 24 * time.Hour
	mgr.Cookie.Name = "rosetta_session"
	mgr.Cookie.HttpOnly = true
	mgr.Cookie.SameSite = http.SameSiteLaxMode
	mgr.Cookie.Secure = cfg.IsProduction
	return mgr
}
