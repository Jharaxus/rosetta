// Standalone migration runner — used as a Docker init container and for manual
// migration management via: go run ./cmd/migrate [up|down|status]
package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/pressly/goose/v3"

	"github.com/jharaxus/rosetta/internal/config"
	"github.com/jharaxus/rosetta/internal/db"
	"github.com/jharaxus/rosetta/migrations"
)

func main() {
	cfg := config.Load()

	pool := db.NewPool(cfg.DatabaseURL)
	defer pool.Close()

	sqlDB := db.NewSQLDB(pool)
	defer sqlDB.Close()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		slog.Error("goose set dialect", "err", err)
		os.Exit(1)
	}

	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	switch command {
	case "up":
		slog.Info("running migrations", "command", "up")
		if err := goose.Up(sqlDB, "."); err != nil {
			slog.Error("goose up", "err", err)
			os.Exit(1)
		}
	case "down":
		slog.Info("rolling back one migration")
		if err := goose.Down(sqlDB, "."); err != nil {
			slog.Error("goose down", "err", err)
			os.Exit(1)
		}
	case "status":
		if err := goose.Status(sqlDB, "."); err != nil {
			slog.Error("goose status", "err", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q — use: up | down | status\n", command)
		os.Exit(1)
	}

	slog.Info("done", "command", command)
}
