package limiter

import (
	"sync"
)

type RateLimiter struct {
	config Config

	buckets map[string]*Bucket

	mu sync.RWMutex
}

func NewRateLimiter(config Config) (*RateLimiter, error) {

	if err := config.Validate(); err != nil {
		return nil, err
	}

	rl := &RateLimiter{
		config:  config,
		buckets: make(map[string]*Bucket),
	}

	return rl, nil
}

func (rl *RateLimiter) getOrCreateBucket(key string) *Bucket {
	rl.mu.RLock()
	bucket, exists := rl.buckets[key]
	rl.mu.RUnlock()

	if exists {
		return bucket
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, exists = rl.buckets[key]
	if exists {
		return bucket
	}

	bucket = NewBucket(rl.config)
	rl.buckets[key] = bucket

	return bucket
}

func (rl *RateLimiter) Allow(key string) bool {
	bucket := rl.getOrCreateBucket(key)
	return bucket.Allow()
}

func (rl *RateLimiter) GetTokens(key string) int {
	rl.mu.RLock()
	bucket, exists := rl.buckets[key]
	rl.mu.RUnlock()
	if !exists {
		return 0
	}

	return bucket.Tokens
}
