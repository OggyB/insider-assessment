package redis

import (
	"context"
	"github.com/oggyb/insider-assessment/internal/cache"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client is a thin Redis-backed implementation of the cache interface.
type Client struct {
	rdb *redis.Client
}

// New creates a new Redis client with the given address, password and DB number.
func New(addr, password string, dbNumber int) *Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       dbNumber,
	})
	return &Client{rdb: rdb}
}

// Ping checks if Redis is reachable.
func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

// Set stores a value with the given TTL.
func (c *Client) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return c.rdb.Set(ctx, key, value, ttl).Err()
}

// Get retrieves a value by key.
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.rdb.Get(ctx, key).Result()
}

// Del deletes a key from Redis.
func (c *Client) Del(ctx context.Context, key string) error {
	return c.rdb.Del(ctx, key).Err()
}

// Incr atomically increments the numeric value at key.
func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	return c.rdb.Incr(ctx, key).Result()
}

// Decr atomically decrements the numeric value at key.
func (c *Client) Decr(ctx context.Context, key string) (int64, error) {
	return c.rdb.Decr(ctx, key).Result()
}

var _ cache.Cache = (*Client)(nil)
