package tts

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// ValkeyRateLimiter implements RateLimiter using Valkey INCR + EXPIRE.
// Each user is limited to `limit` requests per 60-second sliding window.
type ValkeyRateLimiter struct {
	client *redis.Client
	limit  int64
}

// NewValkeyRateLimiter constructs a rate limiter with the given per-user request limit per minute.
func NewValkeyRateLimiter(client *redis.Client, limit int64) *ValkeyRateLimiter {
	return &ValkeyRateLimiter{client: client, limit: limit}
}

// Allow returns true if userID is within quota, false if rate-limited.
// Uses a pipeline to minimise round-trips: INCR then EXPIRE (only on first request in window).
func (v *ValkeyRateLimiter) Allow(ctx context.Context, userID string) (bool, error) {
	key := "tts_rl:" + userID

	pipe := v.client.Pipeline()
	incrCmd := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, 60*time.Second)
	if _, err := pipe.Exec(ctx); err != nil {
		return false, err
	}

	count := incrCmd.Val()
	return count <= v.limit, nil
}
