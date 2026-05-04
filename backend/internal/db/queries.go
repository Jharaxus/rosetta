package db

import (
	"context"
	"net/netip"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jharaxus/rosetta/internal/model"
)

type Queries struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Queries {
	return &Queries{pool: pool}
}

func (q *Queries) UpsertUser(ctx context.Context, subject, email, displayName string) (model.User, error) {
	const query = `
		INSERT INTO users (subject, email, display_name)
		VALUES ($1, $2, $3)
		ON CONFLICT (subject) DO UPDATE
			SET email        = EXCLUDED.email,
			    display_name = EXCLUDED.display_name,
			    updated_at   = now()
		RETURNING id, subject, email, display_name, created_at, updated_at
	`
	var u model.User
	row := q.pool.QueryRow(ctx, query, subject, email, displayName)
	err := row.Scan(&u.ID, &u.Subject, &u.Email, &u.DisplayName, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

func (q *Queries) InsertLoginRecord(ctx context.Context, userID uuid.UUID, ip, userAgent, sessionID string) error {
	const query = `
		INSERT INTO login_records (user_id, ip_address, user_agent, session_id)
		VALUES ($1, $2, $3, $4)
	`
	// pgx v5 encodes netip.Addr as INET natively; nil interface = NULL
	addr, err := netip.ParseAddr(ip)
	var ipArg any
	if err == nil {
		ipArg = addr
	}
	_, err = q.pool.Exec(ctx, query, userID, ipArg, userAgent, sessionID)
	return err
}
