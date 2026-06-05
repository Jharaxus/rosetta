package tts

import (
	"crypto/sha256"
	"encoding/hex"
)

// CacheKey computes the Valkey cache key for a given text+voice pair.
// Exported so tests can pre-populate the cache with the expected key.
// The null-byte separator prevents collisions where text ends with voice prefix.
func CacheKey(text, voice string) string {
	h := sha256.Sum256([]byte(text + "\x00" + voice))
	return "tts_audio:" + hex.EncodeToString(h[:])
}
