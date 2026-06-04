package db

import (
	"context"
	"net/netip"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

// UpdateAssimilNumberTx updates the user's assimil_number and inserts card rows
// for all newly unlocked words in a single transaction.
func (q *Queries) UpdateAssimilNumberTx(ctx context.Context, id uuid.UUID, assimilNumber int) (model.User, error) {
	tx, err := q.pool.Begin(ctx)
	if err != nil {
		return model.User{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var u model.User
	row := tx.QueryRow(ctx, `
		UPDATE users
		SET assimil_number = $2,
		    updated_at     = now()
		WHERE id = $1
		RETURNING id, subject, email, display_name, assimil_number, created_at, updated_at
	`, id, assimilNumber)
	if err := row.Scan(&u.ID, &u.Subject, &u.Email, &u.DisplayName, &u.AssimilNumber, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return model.User{}, err
	}

	if err := insertMissingCards(ctx, tx, id, assimilNumber); err != nil {
		return model.User{}, err
	}
	if err := insertMissingWritingCards(ctx, tx, id, assimilNumber); err != nil {
		return model.User{}, err
	}

	// Equalize all unseen cards so new and old-but-unreviewed cards form one
	// random pool rather than always appearing in lesson order.
	if _, err := tx.Exec(ctx,
		`UPDATE cards SET due = now() WHERE user_id = $1 AND reps = 0`,
		id,
	); err != nil {
		return model.User{}, err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE writing_cards SET due = now() WHERE user_id = $1 AND reps = 0`,
		id,
	); err != nil {
		return model.User{}, err
	}

	return u, tx.Commit(ctx)
}

// insertMissingCards inserts card rows for words newly unlocked by assimilNumber.
// ON CONFLICT DO NOTHING is a belt-and-suspenders guard for concurrent calls.
func insertMissingCards(ctx context.Context, tx pgx.Tx, userID uuid.UUID, assimilNumber int) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO cards (user_id, word_id, due)
		SELECT $1, w.id, now()
		FROM words w
		WHERE w.assimil_number <= $2
		  AND NOT EXISTS (
		      SELECT 1 FROM cards c
		      WHERE c.user_id = $1 AND c.word_id = w.id
		  )
		ON CONFLICT (user_id, word_id) DO NOTHING
	`, userID, assimilNumber)
	return err
}

// insertMissingWritingCards inserts writing_cards rows for words newly unlocked by assimilNumber.
func insertMissingWritingCards(ctx context.Context, tx pgx.Tx, userID uuid.UUID, assimilNumber int) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO writing_cards (user_id, word_id, due)
		SELECT $1, w.id, now()
		FROM words w
		WHERE w.assimil_number <= $2
		  AND NOT EXISTS (
		      SELECT 1 FROM writing_cards wc
		      WHERE wc.user_id = $1 AND wc.word_id = w.id
		  )
		ON CONFLICT (user_id, word_id) DO NOTHING
	`, userID, assimilNumber)
	return err
}

func (q *Queries) CountWords(ctx context.Context) (int64, error) {
	var n int64
	err := q.pool.QueryRow(ctx, `SELECT COUNT(*) FROM words`).Scan(&n)
	return n, err
}

// GetNextDueCard returns the oldest due card for the user joined with its word.
// Returns pgx.ErrNoRows when no card has due <= now().
func (q *Queries) GetNextDueCard(ctx context.Context, userID uuid.UUID) (model.CardWithWord, error) {
	const query = `
		SELECT
			c.user_id, c.word_id,
			c.stability, c.difficulty, c.state, c.step,
			c.due, c.last_review, c.reps, c.lapses,
			w.french, w.german, w.assimil_number, w.category, w.is_regular
		FROM cards c
		JOIN words w ON w.id = c.word_id
		WHERE c.user_id = $1
		  AND c.due <= now()
		ORDER BY c.due ASC, random()
		LIMIT 1
	`
	var cw model.CardWithWord
	row := q.pool.QueryRow(ctx, query, userID)
	err := row.Scan(
		&cw.Card.UserID, &cw.Card.WordID,
		&cw.Card.Stability, &cw.Card.Difficulty, &cw.Card.State, &cw.Card.Step,
		&cw.Card.Due, &cw.Card.LastReview, &cw.Card.Reps, &cw.Card.Lapses,
		&cw.French, &cw.German, &cw.AssimilNumber, &cw.Category, &cw.IsRegular,
	)
	return cw, err
}

// GetCard returns the FSRS state for a single (user, word) pair.
// Returns pgx.ErrNoRows when the card does not exist (word not yet unlocked).
func (q *Queries) GetCard(ctx context.Context, userID, wordID uuid.UUID) (model.Card, error) {
	const query = `
		SELECT user_id, word_id,
		       stability, difficulty, state, step,
		       due, last_review, reps, lapses
		FROM cards
		WHERE user_id = $1 AND word_id = $2
	`
	var c model.Card
	row := q.pool.QueryRow(ctx, query, userID, wordID)
	err := row.Scan(
		&c.UserID, &c.WordID,
		&c.Stability, &c.Difficulty, &c.State, &c.Step,
		&c.Due, &c.LastReview, &c.Reps, &c.Lapses,
	)
	return c, err
}

// UpdateCard persists the FSRS state for a (user, word) pair after a review.
func (q *Queries) UpdateCard(ctx context.Context, card model.Card) error {
	const query = `
		UPDATE cards
		SET stability   = $3,
		    difficulty  = $4,
		    state       = $5,
		    step        = $6,
		    due         = $7,
		    last_review = $8,
		    reps        = $9,
		    lapses      = $10
		WHERE user_id = $1 AND word_id = $2
	`
	_, err := q.pool.Exec(ctx, query,
		card.UserID, card.WordID,
		card.Stability, card.Difficulty, card.State, card.Step,
		card.Due, card.LastReview, card.Reps, card.Lapses,
	)
	return err
}

// ResetProgressionTx resets the user to lesson 1 in a single transaction:
// clears all card rows, resets assimil_number to 1, then re-seeds lesson-1 cards.
func (q *Queries) ResetProgressionTx(ctx context.Context, userID uuid.UUID) (model.User, error) {
	tx, err := q.pool.Begin(ctx)
	if err != nil {
		return model.User{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `DELETE FROM cards WHERE user_id = $1`, userID); err != nil {
		return model.User{}, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM writing_cards WHERE user_id = $1`, userID); err != nil {
		return model.User{}, err
	}

	var u model.User
	row := tx.QueryRow(ctx, `
		UPDATE users SET assimil_number = 1, updated_at = now()
		WHERE id = $1
		RETURNING id, subject, email, display_name, assimil_number, created_at, updated_at
	`, userID)
	if err := row.Scan(&u.ID, &u.Subject, &u.Email, &u.DisplayName, &u.AssimilNumber, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return model.User{}, err
	}

	if err := insertMissingCards(ctx, tx, userID, 1); err != nil {
		return model.User{}, err
	}
	if err := insertMissingWritingCards(ctx, tx, userID, 1); err != nil {
		return model.User{}, err
	}

	return u, tx.Commit(ctx)
}

// GetNextDueWritingCard returns the oldest due writing card for the user joined with its word.
// Returns pgx.ErrNoRows when no writing card has due <= now().
func (q *Queries) GetNextDueWritingCard(ctx context.Context, userID uuid.UUID) (model.CardWithWord, error) {
	const query = `
		SELECT
			wc.user_id, wc.word_id,
			wc.stability, wc.difficulty, wc.state, wc.step,
			wc.due, wc.last_review, wc.reps, wc.lapses,
			w.french, w.german, w.assimil_number, w.category, w.is_regular
		FROM writing_cards wc
		JOIN words w ON w.id = wc.word_id
		WHERE wc.user_id = $1
		  AND wc.due <= now()
		ORDER BY wc.due ASC, random()
		LIMIT 1
	`
	var cw model.CardWithWord
	row := q.pool.QueryRow(ctx, query, userID)
	err := row.Scan(
		&cw.Card.UserID, &cw.Card.WordID,
		&cw.Card.Stability, &cw.Card.Difficulty, &cw.Card.State, &cw.Card.Step,
		&cw.Card.Due, &cw.Card.LastReview, &cw.Card.Reps, &cw.Card.Lapses,
		&cw.French, &cw.German, &cw.AssimilNumber, &cw.Category, &cw.IsRegular,
	)
	return cw, err
}

// GetWritingCard returns the FSRS state for a single (user, word) writing card.
// Returns pgx.ErrNoRows when the writing card does not exist.
func (q *Queries) GetWritingCard(ctx context.Context, userID, wordID uuid.UUID) (model.Card, error) {
	const query = `
		SELECT user_id, word_id,
		       stability, difficulty, state, step,
		       due, last_review, reps, lapses
		FROM writing_cards
		WHERE user_id = $1 AND word_id = $2
	`
	var c model.Card
	row := q.pool.QueryRow(ctx, query, userID, wordID)
	err := row.Scan(
		&c.UserID, &c.WordID,
		&c.Stability, &c.Difficulty, &c.State, &c.Step,
		&c.Due, &c.LastReview, &c.Reps, &c.Lapses,
	)
	return c, err
}

// UpdateWritingCard persists the FSRS state for a (user, word) writing card after a review.
func (q *Queries) UpdateWritingCard(ctx context.Context, card model.Card) error {
	const query = `
		UPDATE writing_cards
		SET stability   = $3,
		    difficulty  = $4,
		    state       = $5,
		    step        = $6,
		    due         = $7,
		    last_review = $8,
		    reps        = $9,
		    lapses      = $10
		WHERE user_id = $1 AND word_id = $2
	`
	_, err := q.pool.Exec(ctx, query,
		card.UserID, card.WordID,
		card.Stability, card.Difficulty, card.State, card.Step,
		card.Due, card.LastReview, card.Reps, card.Lapses,
	)
	return err
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

// GetOrCreateUserSettings returns the settings row for the user, inserting defaults if absent.
func (q *Queries) GetOrCreateUserSettings(ctx context.Context, userID uuid.UUID) (model.UserSettings, error) {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO user_settings (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`,
		userID,
	)
	if err != nil {
		return model.UserSettings{}, err
	}
	var s model.UserSettings
	row := q.pool.QueryRow(ctx,
		`SELECT user_id, number_digit_size, updated_at FROM user_settings WHERE user_id = $1`,
		userID,
	)
	err = row.Scan(&s.UserID, &s.NumberDigitSize, &s.UpdatedAt)
	return s, err
}

// UpdateNumberDigitSize saves the user's preferred digit count, upserting the settings row.
func (q *Queries) UpdateNumberDigitSize(ctx context.Context, userID uuid.UUID, size int) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO user_settings (user_id, number_digit_size, updated_at)
		VALUES ($1, $2, now())
		ON CONFLICT (user_id) DO UPDATE
			SET number_digit_size = $2,
			    updated_at        = now()
	`, userID, size)
	return err
}

// GetDigitStats ensures all 10 digit rows exist for the user and returns them ordered by digit.
func (q *Queries) GetDigitStats(ctx context.Context, userID uuid.UUID) ([]model.DigitStat, error) {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO user_digit_stats (user_id, digit)
		SELECT $1, d FROM generate_series(0, 9) AS d
		ON CONFLICT DO NOTHING
	`, userID)
	if err != nil {
		return nil, err
	}
	rows, err := q.pool.Query(ctx,
		`SELECT user_id, digit, successes FROM user_digit_stats WHERE user_id = $1 ORDER BY digit`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make([]model.DigitStat, 0, 10)
	for rows.Next() {
		var s model.DigitStat
		if err := rows.Scan(&s.UserID, &s.Digit, &s.Successes); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// IncrementDigitSuccesses increments the success counter for every digit that appears in number.
// For example, 113 increments digit 1 by 2 and digit 3 by 1.
func (q *Queries) IncrementDigitSuccesses(ctx context.Context, userID uuid.UUID, number int) error {
	counts := make([]int, 10)
	n := number
	if n == 0 {
		counts[0]++
	}
	for n > 0 {
		counts[n%10]++
		n /= 10
	}

	digits := make([]int16, 0, 10)
	increments := make([]int32, 0, 10)
	for d, c := range counts {
		if c > 0 {
			digits = append(digits, int16(d))
			increments = append(increments, int32(c))
		}
	}

	_, err := q.pool.Exec(ctx, `
		INSERT INTO user_digit_stats (user_id, digit, successes)
		SELECT $1, unnest($2::smallint[]), unnest($3::int[])
		ON CONFLICT (user_id, digit)
		DO UPDATE SET successes = user_digit_stats.successes + EXCLUDED.successes
	`, userID, digits, increments)
	return err
}

// AddNumberFailure records a number the user failed to spell correctly.
// Duplicate entries are silently ignored.
func (q *Queries) AddNumberFailure(ctx context.Context, userID uuid.UUID, number int) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO number_failures (user_id, number) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, number,
	)
	return err
}

// RemoveNumberFailure removes a number from the user's failure list.
func (q *Queries) RemoveNumberFailure(ctx context.Context, userID uuid.UUID, number int) error {
	_, err := q.pool.Exec(ctx,
		`DELETE FROM number_failures WHERE user_id = $1 AND number = $2`,
		userID, number,
	)
	return err
}

// GetRandomNumberFailure returns a random number from the user's failure list.
// Returns pgx.ErrNoRows when the list is empty.
func (q *Queries) GetRandomNumberFailure(ctx context.Context, userID uuid.UUID) (int, error) {
	var number int
	err := q.pool.QueryRow(ctx,
		`SELECT number FROM number_failures WHERE user_id = $1 ORDER BY random() LIMIT 1`,
		userID,
	).Scan(&number)
	return number, err
}
