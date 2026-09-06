package limiter

import (
	"database/sql"
	"log"
	"net/http"
	"strings"
)

type RateLimiterMiddleware struct {
	limiter *RateLimiter
	next    http.Handler
}

func NewRateLimiterMiddleware(next http.Handler, db *sql.DB, config Config) (http.Handler, error) {
	limiter, err := NewRateLimiter(db, config)

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

	allowed, err := rtlm.limiter.Allow(r.Context(), key)
	if err != nil {
		log.Printf("rate limiter: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if !allowed {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}
	rtlm.next.ServeHTTP(w, r)
}
