package main

import (
	"fmt"
	"net/http"

	"github.com/redis/go-redis/v9"

	"rate-limiter/internal/limiter"
	"rate-limiter/internal/logger"
)

func HelloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello World!")
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "X-User-ID")
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", http.MethodGet)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func registerMiddleware(client *redis.Client) (http.Handler, error) {
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/hello", HelloHandler)

	config := limiter.Config{
		Capacity:        100,
		TokensPerSecond: 1.67,
	}

	printMiddleware := logger.PrintMiddlewareHandler(apiMux)
	rateLimiterMiddleware, err := limiter.NewRateLimiterMiddleware(printMiddleware, client, config)
	if err != nil {
		return nil, fmt.Errorf("create rate limiter middleware: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/hello", cors(rateLimiterMiddleware))
	mux.Handle("/", http.FileServer(http.Dir("web")))
	return mux, nil
}
