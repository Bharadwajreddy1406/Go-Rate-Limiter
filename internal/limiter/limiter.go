package limiter

import (
	"context"

	"github.com/redis/go-redis/v9"

	"rate-limiter/internal/repository"
)

type Decision struct {
	Allowed      bool
	TokensBefore int
	TokensAfter  int
}

type RateLimiter struct {
	buckets *repository.BucketRepository
}

func NewRateLimiter(client *redis.Client, config Config) (*RateLimiter, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &RateLimiter{
		buckets: repository.NewBucketRepository(client, config.Capacity, config.TokensPerSecond),
	}, nil
}

func (rl *RateLimiter) Allow(ctx context.Context, key string) (Decision, error) {
	decision, err := rl.buckets.Allow(ctx, key)
	if err != nil {
		return Decision{}, err
	}

	return Decision{
		Allowed:      decision.Allowed,
		TokensBefore: decision.TokensBefore,
		TokensAfter:  decision.TokensAfter,
	}, nil
}

func (rl *RateLimiter) GetTokens(ctx context.Context, key string) (int, error) {
	return rl.buckets.GetTokens(ctx, key)
}
