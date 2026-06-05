package tts

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

const defaultCacheTTL = 24 * time.Hour

// ValkeyCache implements AudioCache backed by a Redis-compatible Valkey store.
type ValkeyCache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewValkeyCache wraps an existing redis.Client.
func NewValkeyCache(client *redis.Client) *ValkeyCache {
	return &ValkeyCache{client: client, ttl: defaultCacheTTL}
}

// Get retrieves cached audio bytes. Returns (nil, false, nil) on a miss.
func (v *ValkeyCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	b, err := v.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return b, true, nil
}

// Set stores audio bytes with the configured TTL.
func (v *ValkeyCache) Set(ctx context.Context, key string, value []byte) error {
	return v.client.Set(ctx, key, value, v.ttl).Err()
}
