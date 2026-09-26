package limiter

import (
	"log"
	"net/http"
	"strings"

	"github.com/redis/go-redis/v9"
)

type RateLimiterMiddleware struct {
	limiter *RateLimiter
	next    http.Handler
}

func NewRateLimiterMiddleware(next http.Handler, client *redis.Client, config Config) (http.Handler, error) {
	limiter, err := NewRateLimiter(client, config)
	if err != nil {
		return nil, err
	}

	return &RateLimiterMiddleware{
		limiter: limiter,
		next:    next,
	}, nil
}

func (rtlm *RateLimiterMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimSpace(r.Header.Get("X-User-ID"))
	if key == "" {
		http.Error(w, "X-User-ID header is required", http.StatusBadRequest)
		return
	}

	decision, err := rtlm.limiter.Allow(r.Context(), key)
	if err != nil {
		log.Printf("rate limiter: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf(
		"rate limiter: user_id=%q tokens_before=%d tokens_after=%d allowed=%t",
		key,
		decision.TokensBefore,
		decision.TokensAfter,
		decision.Allowed,
	)

	if !decision.Allowed {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	rtlm.next.ServeHTTP(w, r)
}
