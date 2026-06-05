package tts

import "context"

// SpeechSynthesizer is the boundary between the handler and the Google Cloud TTS API.
// Inject into Handler so unit tests can provide a mock without a real GCP connection.
type SpeechSynthesizer interface {
	Synthesize(ctx context.Context, text, voice string) ([]byte, error)
}

// AudioCache is the boundary between the handler and Valkey.
// Get returns (nil, false, nil) on a cache miss and (bytes, true, nil) on a hit.
type AudioCache interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte) error
}

// RateLimiter enforces per-user request quotas.
// Allow returns true if the request is within quota, false if the user is rate-limited.
type RateLimiter interface {
	Allow(ctx context.Context, userID string) (bool, error)
}
