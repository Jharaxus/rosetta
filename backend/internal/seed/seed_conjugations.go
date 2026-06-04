// Package seed — conjugation seeder.
// Loads verb conjugations from resources/Deutch_verbs_and_conjugations.csv
// into the conjugations table. Designed to run after the words seeder.
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
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultConjSeedFile = "/app/resources/Deutch_verbs_and_conjugations.csv"

// conjRow is one parsed CSV record ready for bulk insertion.
type conjRow struct {
	wordID pgtype.UUID
	tense  string   // verb_tense enum value (e.g. "praesens_indikativ")
	person string   // verb_person enum value (e.g. "p1_sg")
	forms  []string // one or more alternate conjugated forms
}

// personIntToEnum maps the CSV person column (1–6) to the DB enum string.
var personIntToEnum = map[int]string{
	1: "p1_sg",
	2: "p2_sg",
	3: "p3_sg",
	4: "p1_pl",
	5: "p2_pl",
	6: "p3_pl",
}

// validTenses is the set of accepted verb_tense enum values.
var validTenses = map[string]struct{}{
	"praesens_indikativ": {}, "praesens_konjunktiv_1": {},
	"praeteritum_indikativ": {}, "praeteritum_konjunktiv_2": {},
	"perfekt_indikativ": {}, "perfekt_konjunktiv_1": {},
	"plusquamperfekt_indikativ": {}, "plusquamperfekt_konjunktiv_2": {},
	"futur_1_indikativ": {}, "futur_1_konjunktiv_1": {}, "futur_1_konjunktiv_2": {},
	"futur_2_indikativ": {}, "futur_2_konjunktiv_1": {}, "futur_2_konjunktiv_2": {},
	"imperativ": {},
}

// SeedConjugations loads conjugation data from the CSV pointed to by the
// CONJ_SEED_FILE env var (defaulting to defaultConjSeedFile). It is
// idempotent: if conjugations already exist and FORCE_RESEED_CONJ is not
// set to "true", it exits without touching the DB.
func SeedConjugations(ctx context.Context, pool *pgxpool.Pool) error {
	count, err := countConjugations(ctx, pool)
	if err != nil {
		return fmt.Errorf("seed conjugations: count: %w", err)
	}

	force := os.Getenv("FORCE_RESEED_CONJ") == "true"
	if count > 0 && !force {
		slog.Info("seed conjugations: table already populated, skipping", "count", count)
		return nil
	}
	if count > 0 && force {
		slog.Info("seed conjugations: FORCE_RESEED_CONJ=true, truncating conjugations table")
		if _, err := pool.Exec(ctx, `TRUNCATE TABLE conjugations`); err != nil {
			return fmt.Errorf("seed conjugations: truncate: %w", err)
		}
	}

	path := os.Getenv("CONJ_SEED_FILE")
	if path == "" {
		path = defaultConjSeedFile
	}

	wordIDs, err := loadVerbWordIDMap(ctx, pool)
	if err != nil {
		return fmt.Errorf("seed conjugations: load word IDs: %w", err)
	}

	rows, err := parseConjugationsCSV(path, wordIDs)
	if err != nil {
		return fmt.Errorf("seed conjugations: parse csv: %w", err)
	}

	if err := bulkInsertConjugations(ctx, pool, rows); err != nil {
		return fmt.Errorf("seed conjugations: bulk insert: %w", err)
	}

	slog.Info("seed conjugations: loaded", "count", len(rows))
	return nil
}

func countConjugations(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM conjugations`).Scan(&n)
	return n, err
}

// loadVerbWordIDMap builds a map[trimmed_german]UUID for all Verb-category words.
func loadVerbWordIDMap(ctx context.Context, pool *pgxpool.Pool) (map[string]pgtype.UUID, error) {
	rows, err := pool.Query(ctx, `SELECT id, german FROM words WHERE category = 'Verb'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := make(map[string]pgtype.UUID)
	for rows.Next() {
		var id pgtype.UUID
		var german string
		if err := rows.Scan(&id, &german); err != nil {
			return nil, err
		}
		m[strings.TrimSpace(german)] = id
	}
	return m, rows.Err()
}

// parseConjugationsCSV reads the 4-column conjugation CSV and returns rows
// ready for bulk insertion. Rows whose verb is not found in wordIDMap are
// skipped with a warning.
func parseConjugationsCSV(path string, wordIDMap map[string]pgtype.UUID) ([]conjRow, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.Comma = ';'
	r.FieldsPerRecord = 4
	r.LazyQuotes = true

	// skip header
	if _, err := r.Read(); err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	type dedupeKey struct {
		wordID pgtype.UUID
		tense  string
		person string
	}
	seen := make(map[dedupeKey]struct{})

	var rows []conjRow
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read record: %w", err)
		}

		verb := strings.TrimSpace(rec[0])
		tense := strings.TrimSpace(rec[1])
		personStr := strings.TrimSpace(rec[2])
		conjugation := strings.TrimSpace(rec[3])

		// Validate tense
		if _, ok := validTenses[tense]; !ok {
			slog.Warn("seed conjugations: unknown tense, skipping", "tense", tense, "verb", verb)
			continue
		}

		// Map person integer to enum
		personInt, err := strconv.Atoi(personStr)
		if err != nil {
			slog.Warn("seed conjugations: invalid person, skipping", "person", personStr, "verb", verb)
			continue
		}
		personEnum, ok := personIntToEnum[personInt]
		if !ok {
			slog.Warn("seed conjugations: person out of range, skipping", "person", personInt, "verb", verb)
			continue
		}

		// Look up word_id
		wordID, ok := wordIDMap[verb]
		if !ok {
			slog.Warn("seed conjugations: verb not found in words table, skipping", "verb", verb)
			continue
		}

		// Split comma-separated alternate forms
		var forms []string
		for _, f := range strings.Split(conjugation, ", ") {
			if s := strings.TrimSpace(f); s != "" {
				forms = append(forms, s)
			}
		}
		if len(forms) == 0 {
			continue
		}

		key := dedupeKey{wordID: wordID, tense: tense, person: personEnum}
		if _, dup := seen[key]; dup {
			slog.Warn("seed conjugations: duplicate row, skipping",
				"verb", verb, "tense", tense, "person", personEnum)
			continue
		}
		seen[key] = struct{}{}

		rows = append(rows, conjRow{
			wordID: wordID,
			tense:  tense,
			person: personEnum,
			forms:  forms,
		})
	}

	return rows, nil
}

func bulkInsertConjugations(ctx context.Context, pool *pgxpool.Pool, rows []conjRow) error {
	cols := []string{"word_id", "tense", "person", "forms"}

	_, err := pool.CopyFrom(
		ctx,
		pgx.Identifier{"conjugations"},
		cols,
		pgx.CopyFromSlice(len(rows), func(i int) ([]any, error) {
			r := rows[i]
			return []any{r.wordID, r.tense, r.person, r.forms}, nil
		}),
	)
	return err
}
