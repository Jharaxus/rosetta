// Package seed loads the initial German vocabulary from a CSV file into the
// words table. It is designed to be called once at boot, before the backend
// starts. The load is skipped if the table is already populated.
package seed

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultSeedFile = "/app/resources/Lexique Allemand - export.csv"

// row mirrors one CSV record after parsing.
type row struct {
	french        string
	german        string
	assimilNumber int
	category      string
	isRegular     *bool
}

// Seed loads vocabulary data from the CSV pointed to by the SEED_FILE env var
// (defaulting to defaultSeedFile). It is idempotent: if words already exist
// and FORCE_RESEED is not set, it exits immediately without touching the DB.
func Seed(ctx context.Context, pool *pgxpool.Pool) error {
	count, err := countWords(ctx, pool)
	if err != nil {
		return fmt.Errorf("seed: count words: %w", err)
	}

	force := os.Getenv("FORCE_RESEED") == "true"
	if count > 0 && !force {
		slog.Info("seed: words table already populated, skipping", "count", count)
		return nil
	}
	if count > 0 && force {
		slog.Info("seed: FORCE_RESEED=true, truncating words table")
		if _, err := pool.Exec(ctx, `TRUNCATE TABLE words`); err != nil {
			return fmt.Errorf("seed: truncate words: %w", err)
		}
	}

	path := os.Getenv("SEED_FILE")
	if path == "" {
		path = defaultSeedFile
	}

	rows, err := parseCSV(path)
	if err != nil {
		return fmt.Errorf("seed: parse csv: %w", err)
	}

	if err := bulkInsert(ctx, pool, rows); err != nil {
		return fmt.Errorf("seed: bulk insert: %w", err)
	}

	slog.Info("seed: words loaded", "count", len(rows))
	return nil
}

func countWords(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM words`).Scan(&n)
	return n, err
}

func parseCSV(path string) ([]row, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.Comma = ';'
	r.FieldsPerRecord = 5
	r.LazyQuotes = true

	// skip header
	if _, err := r.Read(); err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	var rows []row
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read record: %w", err)
		}

		assimil, err := strconv.Atoi(strings.TrimSpace(rec[2]))
		if err != nil {
			return nil, fmt.Errorf("parse assimil_number %q: %w", rec[2], err)
		}

		var isRegular *bool
		if reg := strings.TrimSpace(rec[4]); reg != "" {
			b := reg == "regular"
			isRegular = &b
		}

		rows = append(rows, row{
			french:        strings.TrimSpace(rec[0]),
			german:        strings.TrimSpace(rec[1]),
			assimilNumber: assimil,
			category:      strings.TrimSpace(rec[3]),
			isRegular:     isRegular,
		})
	}

	return rows, nil
}

func bulkInsert(ctx context.Context, pool *pgxpool.Pool, rows []row) error {
	cols := []string{"french", "german", "assimil_number", "category", "is_regular"}

	_, err := pool.CopyFrom(
		ctx,
		pgx.Identifier{"words"},
		cols,
		pgx.CopyFromSlice(len(rows), func(i int) ([]any, error) {
			r := rows[i]
			return []any{r.french, r.german, r.assimilNumber, r.category, r.isRegular}, nil
		}),
	)
	return err
}
