package limiter

import "sync"

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