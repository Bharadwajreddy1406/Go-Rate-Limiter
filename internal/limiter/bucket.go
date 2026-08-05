package limiter


import (
	"sync"
	"time"
)

type Bucket struct {
	Tokens          int
	Capacity        int
	TokensPerSecond float64
	LastRefillAt    time.Time

	mu sync.Mutex
}

func NewBucket(config Config) *Bucket {
	return &Bucket{
		Tokens:          config.Capacity,
		Capacity:        config.Capacity,
		TokensPerSecond: config.TokensPerSecond,
		LastRefillAt:    time.Now(),
	}
}


func (bucket *Bucket) refill() {
	
	now := time.Now()
	elapsed := now.Sub(bucket.LastRefillAt).Seconds()
	tokensToAdd := int(elapsed * bucket.TokensPerSecond)
	if tokensToAdd > 0 {
		bucket.Tokens = min(bucket.Capacity, bucket.Tokens+tokensToAdd)
		bucket.LastRefillAt = now
	}
}

func (bucket *Bucket) Allow() bool {
	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	bucket.refill()

	if bucket.Tokens > 0 {
		bucket.Tokens--
		return true
	}
	return false
}