// Package cache wraps Redis for caching and distributed rate limiting.
package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	rdb *redis.Client
}

func New(addr, password string, db int) (*Cache, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return &Cache{rdb: rdb}, nil
}

func (c *Cache) Close() error { return c.rdb.Close() }

func (c *Cache) Get(ctx context.Context, key string) (string, bool, error) {
	v, err := c.rdb.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return v, true, nil
}

func (c *Cache) Set(ctx context.Context, key, val string, ttl time.Duration) error {
	return c.rdb.Set(ctx, key, val, ttl).Err()
}

func (c *Cache) Del(ctx context.Context, keys ...string) error {
	return c.rdb.Del(ctx, keys...).Err()
}

// SetNX stores a value only if the key does not exist; used for one-time tokens
// such as OAuth state and CSRF nonces.
func (c *Cache) SetNX(ctx context.Context, key, val string, ttl time.Duration) (bool, error) {
	return c.rdb.SetNX(ctx, key, val, ttl).Result()
}

// GetDel atomically reads and deletes a key (single-use tokens).
func (c *Cache) GetDel(ctx context.Context, key string) (string, bool, error) {
	v, err := c.rdb.GetDel(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return v, true, nil
}

// tokenBucket is an atomic Lua token-bucket rate limiter.
// KEYS[1] = bucket key
// ARGV[1] = capacity, ARGV[2] = refill tokens/sec, ARGV[3] = now (unix sec), ARGV[4] = requested
// Returns {allowed(0/1), remaining, retry_after_seconds}
var tokenBucket = redis.NewScript(`
local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local rate = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local requested = tonumber(ARGV[4])

local data = redis.call('HMGET', key, 'tokens', 'ts')
local tokens = tonumber(data[1])
local ts = tonumber(data[2])
if tokens == nil then
  tokens = capacity
  ts = now
end

local delta = math.max(0, now - ts)
tokens = math.min(capacity, tokens + delta * rate)

local allowed = 0
local retry = 0
if tokens >= requested then
  allowed = 1
  tokens = tokens - requested
else
  retry = math.ceil((requested - tokens) / rate)
end

redis.call('HMSET', key, 'tokens', tokens, 'ts', now)
redis.call('EXPIRE', key, math.ceil(capacity / rate) + 1)
return {allowed, math.floor(tokens), retry}
`)

type RateResult struct {
	Allowed    bool
	Remaining  int
	RetryAfter int
}

// Allow consumes one token from the bucket identified by key.
func (c *Cache) Allow(ctx context.Context, key string, capacity, refillPerSec int) (RateResult, error) {
	now := time.Now().Unix()
	res, err := tokenBucket.Run(ctx, c.rdb, []string{key}, capacity, refillPerSec, now, 1).Result()
	if err != nil {
		return RateResult{}, err
	}
	vals, ok := res.([]any)
	if !ok || len(vals) != 3 {
		return RateResult{Allowed: true}, nil
	}
	allowed, _ := vals[0].(int64)
	remaining, _ := vals[1].(int64)
	retry, _ := vals[2].(int64)
	return RateResult{Allowed: allowed == 1, Remaining: int(remaining), RetryAfter: int(retry)}, nil
}
