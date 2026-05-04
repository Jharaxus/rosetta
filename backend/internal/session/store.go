package session

// Encrypted PostgreSQL session store implementing the scs.Store interface.
//
// Session data is encrypted with AES-256-GCM before writing to the DB.
// If the session table is ever read by an attacker, the data is ciphertext —
// they would need the SESSION_SECRET to decrypt it.
//
// Key derivation: SHA-256(sessionSecret) → 32-byte AES key.
// Each row uses a unique random nonce prepended to the ciphertext (standard GCM practice).

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgStore struct {
	pool   *pgxpool.Pool
	encKey [32]byte
}

func newPGStore(pool *pgxpool.Pool, sessionSecret []byte) *pgStore {
	return &pgStore{
		pool:   pool,
		encKey: sha256.Sum256(sessionSecret), // derive AES-256 key
	}
}

func (s *pgStore) Find(token string) ([]byte, bool, error) {
	const q = `SELECT data FROM sessions WHERE token = $1 AND expiry > now()`
	var encrypted []byte
	err := s.pool.QueryRow(context.Background(), q, token).Scan(&encrypted)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		slog.Error("session store find failed", "err", err)
		return nil, false, err
	}
	data, err := s.decrypt(encrypted)
	if err != nil {
		return nil, false, err
	}
	return data, true, nil
}

func (s *pgStore) Commit(token string, data []byte, expiry time.Time) error {
	encrypted, err := s.encrypt(data)
	if err != nil {
		return err
	}
	const q = `
		INSERT INTO sessions (token, data, expiry)
		VALUES ($1, $2, $3)
		ON CONFLICT (token) DO UPDATE
			SET data = EXCLUDED.data, expiry = EXCLUDED.expiry
	`
	_, err = s.pool.Exec(context.Background(), q, token, encrypted, expiry)
	return err
}

func (s *pgStore) Delete(token string) error {
	_, err := s.pool.Exec(context.Background(), `DELETE FROM sessions WHERE token = $1`, token)
	return err
}

func (s *pgStore) encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.encKey[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	// Prepend nonce to ciphertext so Decrypt can extract it.
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func (s *pgStore) decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.encKey[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("session: ciphertext too short")
	}
	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ct, nil)
}
