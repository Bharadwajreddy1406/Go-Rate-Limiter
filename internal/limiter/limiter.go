package limiter

import (
	"context"
	"database/sql"

	"rate-limiter/internal/repository"
)

type RateLimiter struct {
	buckets *repository.BucketRepository
}

func NewRateLimiter(db *sql.DB, config Config) (*RateLimiter, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &RateLimiter{
		buckets: repository.NewBucketRepository(db, config.Capacity, config.TokensPerSecond),
	}, nil
}

func (rl *RateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	return rl.buckets.Allow(ctx, key)
}
