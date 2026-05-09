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
		RETURNING id, subject, email, display_name, assimil_number, created_at, updated_at
	`
	var u model.User
	row := q.pool.QueryRow(ctx, query, subject, email, displayName)
	err := row.Scan(&u.ID, &u.Subject, &u.Email, &u.DisplayName, &u.AssimilNumber, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

func (q *Queries) GetUserByID(ctx context.Context, id uuid.UUID) (model.User, error) {
	const query = `
		SELECT id, subject, email, display_name, assimil_number, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	var u model.User
	row := q.pool.QueryRow(ctx, query, id)
	err := row.Scan(&u.ID, &u.Subject, &u.Email, &u.DisplayName, &u.AssimilNumber, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

func (q *Queries) UpdateAssimilNumber(ctx context.Context, id uuid.UUID, assimilNumber int) (model.User, error) {
	const query = `
		UPDATE users
		SET assimil_number = $2,
		    updated_at     = now()
		WHERE id = $1
		RETURNING id, subject, email, display_name, assimil_number, created_at, updated_at
	`
	var u model.User
	row := q.pool.QueryRow(ctx, query, id, assimilNumber)
	err := row.Scan(&u.ID, &u.Subject, &u.Email, &u.DisplayName, &u.AssimilNumber, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

func (q *Queries) CountWords(ctx context.Context) (int64, error) {
	var n int64
	err := q.pool.QueryRow(ctx, `SELECT COUNT(*) FROM words`).Scan(&n)
	return n, err
}

func (q *Queries) GetRandomWordForUser(ctx context.Context, assimilNumber int) (model.Word, error) {
	const query = `
		SELECT id, french, german, assimil_number, category, is_regular, created_at
		FROM words
		WHERE assimil_number <= $1
		ORDER BY random()
		LIMIT 1
	`
	var w model.Word
	row := q.pool.QueryRow(ctx, query, assimilNumber)
	err := row.Scan(&w.ID, &w.French, &w.German, &w.AssimilNumber, &w.Category, &w.IsRegular, &w.CreatedAt)
	return w, err
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
