package cache

import (
	"context"
	"time"
)

// Cache is a minimal key/value cache interface (e.g. Redis).
type Cache interface {
	// Ping checks if the cache is reachable.
	Ping(ctx context.Context) error

	// Set stores a value with the given TTL.
	Set(ctx context.Context, key string, value string, ttl time.Duration) error

	// Get retrieves a value by key.
	// Implementations should return a clear "not found" error if missing.
	Get(ctx context.Context, key string) (string, error)

	// Del removes a key. No-op if the key does not exist.
	Del(ctx context.Context, key string) error

	// Incr atomically increments a numeric value and returns the new value.
	Incr(ctx context.Context, key string) (int64, error)

	// Decr atomically decrements a numeric value and returns the new value.
	Decr(ctx context.Context, key string) (int64, error)
}
