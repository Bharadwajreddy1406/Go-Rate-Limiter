package repository

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/redis/go-redis/v9"
)

const bucketKeyPrefix = "rate-limiter:bucket:"

var allowScript = redis.NewScript(`
local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local tokens_per_second = tonumber(ARGV[2])
local ttl_ms = tonumber(ARGV[3])

local redis_time = redis.call("TIME")
local now_ms = tonumber(redis_time[1]) * 1000 + math.floor(tonumber(redis_time[2]) / 1000)
local bucket = redis.call("HMGET", key, "tokens", "last_refill_at")

local tokens = capacity
local last_refill_at = now_ms

if bucket[1] then
    tokens = tonumber(bucket[1])
    last_refill_at = tonumber(bucket[2])

    local elapsed_ms = math.max(0, now_ms - last_refill_at)
    local tokens_to_add = math.floor(elapsed_ms * tokens_per_second / 1000)

    if tokens_to_add > 0 then
        tokens = math.min(capacity, tokens + tokens_to_add)
        last_refill_at = last_refill_at + (tokens_to_add / tokens_per_second * 1000)
    end
end

local tokens_before = tokens
local allowed = 0

if tokens > 0 then
    tokens = tokens - 1
    allowed = 1
end

redis.call("HSET", key, "tokens", tokens, "last_refill_at", last_refill_at)
redis.call("PEXPIRE", key, ttl_ms)

return {allowed, tokens_before, tokens}
`)

type Decision struct {
	Allowed      bool
	TokensBefore int
	TokensAfter  int
}

type BucketRepository struct {
	client          *redis.Client
	capacity        int
	tokensPerSecond float64
	ttlMilliseconds int64
}

func NewBucketRepository(client *redis.Client, capacity int, tokensPerSecond float64) *BucketRepository {
	return &BucketRepository{
		client:          client,
		capacity:        capacity,
		tokensPerSecond: tokensPerSecond,
		ttlMilliseconds: max(1, int64(math.Ceil(float64(capacity)/tokensPerSecond*1000))),
	}
}

func (r *BucketRepository) GetTokens(ctx context.Context, key string) (int, error) {
	tokens, err := r.client.HGet(ctx, bucketKeyPrefix+key, "tokens").Int()
	if errors.Is(err, redis.Nil) {
		return r.capacity, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read bucket tokens: %w", err)
	}
	return tokens, nil
}

func (r *BucketRepository) Allow(ctx context.Context, key string) (Decision, error) {
	values, err := allowScript.Run(
		ctx,
		r.client,
		[]string{bucketKeyPrefix + key},
		r.capacity,
		r.tokensPerSecond,
		r.ttlMilliseconds,
	).Int64Slice()
	if err != nil {
		return Decision{}, fmt.Errorf("run bucket script: %w", err)
	}
	if len(values) != 3 {
		return Decision{}, fmt.Errorf("run bucket script: expected 3 values, got %d", len(values))
	}

	return Decision{
		Allowed:      values[0] == 1,
		TokensBefore: int(values[1]),
		TokensAfter:  int(values[2]),
	}, nil
}
