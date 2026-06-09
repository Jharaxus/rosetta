package seed

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/text/unicode/norm"
)

const defaultAudioDir = "/app/resources/audio"

// canonicalGerman returns the first alternative from the [alt1;alt2;...] format
// used in words.german. This is the form used for TTS generation and filename hashing.
func canonicalGerman(german string) string {
	s := strings.TrimPrefix(strings.TrimSuffix(german, "]"), "[")
	first, _, _ := strings.Cut(s, ";")
	return first
}

// audioFilename is the cross-language contract function.
// It MUST produce identical output to resources/scripts/generate_audio.py::audio_filename().
// Algorithm: NFC-normalise → lowercase → trim → SHA256 → hex → + ".ogg"
func audioFilename(german string) string {
	canonical := canonicalGerman(german)
	normalised := norm.NFC.String(strings.ToLower(strings.TrimSpace(canonical)))
	h := sha256.Sum256([]byte(normalised))
	return hex.EncodeToString(h[:]) + ".ogg"
}

type audioUpdateRow struct {
	id        string
	audioFile string
}

// SeedAudio populates words.audio_file for words whose matching .ogg file exists in audioDir.
// It is idempotent: rows with audio_file already set are skipped unless FORCE_RESEED_AUDIO=true.
func SeedAudio(ctx context.Context, pool *pgxpool.Pool) error {
	audioDir := os.Getenv("AUDIO_DIR")
	if audioDir == "" {
		audioDir = defaultAudioDir
	}

	entries, err := os.ReadDir(audioDir)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Warn("seed_audio: audio directory does not exist, skipping", "dir", audioDir)
			return nil
		}
		return fmt.Errorf("seed_audio: read dir %s: %w", audioDir, err)
	}

	// Build a set of available filenames for O(1) lookup — avoid N filesystem calls.
	available := make(map[string]bool, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".ogg") {
			available[e.Name()] = true
		}
	}

	forceReseed := os.Getenv("FORCE_RESEED_AUDIO") == "true"

	var query string
	if forceReseed {
		query = "SELECT id, german FROM words"
	} else {
		query = "SELECT id, german FROM words WHERE audio_file IS NULL"
	}

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("seed_audio: query words: %w", err)
	}
	defer rows.Close()

	var updates []audioUpdateRow
	for rows.Next() {
		var id, german string
		if err := rows.Scan(&id, &german); err != nil {
			return fmt.Errorf("seed_audio: scan row: %w", err)
		}
		filename := audioFilename(german)
		if available[filename] {
			updates = append(updates, audioUpdateRow{id: id, audioFile: filename})
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("seed_audio: iterate rows: %w", err)
	}

	if len(updates) == 0 {
		slog.Info("seed_audio: no words to update", "audio_files_found", len(available))
		return nil
	}

	batch := &pgx.Batch{}
	for _, u := range updates {
		batch.Queue("UPDATE words SET audio_file = $1 WHERE id = $2", u.audioFile, u.id)
	}

	results := pool.SendBatch(ctx, batch)
	defer results.Close()

	var updateErrors int
	for range updates {
		if _, err := results.Exec(); err != nil {
			slog.Error("seed_audio: batch update error", "err", err)
			updateErrors++
		}
	}
	if err := results.Close(); err != nil {
		return fmt.Errorf("seed_audio: close batch: %w", err)
	}

	slog.Info("seed_audio complete",
		"audio_files_found", len(available),
		"words_updated", len(updates)-updateErrors,
		"errors", updateErrors,
	)

	if updateErrors > 0 {
		return fmt.Errorf("seed_audio: %d updates failed", updateErrors)
	}
	return nil
}
