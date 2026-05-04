package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/jharaxus/rosetta/migrations"
)

func NewPool(cfg string) *pgxpool.Pool {
	ctx := context.Background()

	var pool *pgxpool.Pool
	var err error

	for i := 0; i < 10; i++ {
		pool, err = pgxpool.New(ctx, cfg)
		if err == nil {
			if pingErr := pool.Ping(ctx); pingErr == nil {
				return pool
			}
		}
		time.Sleep(time.Duration(i+1) * time.Second)
	}

	panic("failed to connect to database after retries: " + err.Error())
}

// NewSQLDB returns a *sql.DB wrapping the pool, for use with SCS session store.
// The caller is responsible for closing it alongside the pool.
func NewSQLDB(pool *pgxpool.Pool) *sql.DB {
	return stdlib.OpenDBFromPool(pool)
}

func RunMigrations(pool *pgxpool.Pool) {
	sqlDB := NewSQLDB(pool)
	defer sqlDB.Close()

	goose.SetBaseFS(migrations.FS)

	if err := goose.SetDialect("postgres"); err != nil {
		panic("goose set dialect: " + err.Error())
	}
	if err := goose.Up(sqlDB, "."); err != nil {
		panic("goose up: " + err.Error())
	}
}
