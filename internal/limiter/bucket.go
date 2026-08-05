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


func (b *Bucket) refill() {
	now := time.Now()

	elapsed := now.Sub(b.LastRefillAt).Seconds()
	tokensToAdd := int(elapsed * b.TokensPerSecond)

	if tokensToAdd <= 0 {
		return
	}

	b.Tokens = min(b.Capacity, b.Tokens+tokensToAdd)


	refillDuration := time.Duration(
		float64(tokensToAdd)/b.TokensPerSecond * float64(time.Second),
	)

	b.LastRefillAt = b.LastRefillAt.Add(refillDuration)
}


func (b *Bucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.refill()

	if b.Tokens <= 0 {
		return false
	}

	b.Tokens--

	return true
}