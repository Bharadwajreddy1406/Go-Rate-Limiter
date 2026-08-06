package limiter

import (
	"fmt"
	"net/http"
)

type RateLimiterMiddleware struct {
	limiter *RateLimiter
	next    http.Handler
}

func NewRateLimiterMiddleware(next http.Handler, config Config) (http.Handler, error) {
	limiter, err := NewRateLimiter(config)

	if err != nil {
		return nil, err
	}

	return &RateLimiterMiddleware{
		limiter: limiter,
		next:    next,
	}, nil
}

func (rtlm *RateLimiterMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	key := r.Header.Get("X-User-ID")
	fmt.Printf(" User Id %s\n and tokens before %d\n", key, rtlm.limiter.GetTokens(key))
	if !rtlm.limiter.Allow(key) {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}
	rtlm.next.ServeHTTP(w, r)

	fmt.Printf("User Id %s and Tokens After %d\n", key, rtlm.limiter.GetTokens(key))
}
