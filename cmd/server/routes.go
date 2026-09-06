package main

import (
	"database/sql"
	"fmt"
	"net/http"

	"rate-limiter/internal/limiter"
	"rate-limiter/internal/logger"
)

func HelloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello World!")
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ponytail: demo-only wildcard; restrict allowed origins before adding authentication.
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

func registerMiddleware(db *sql.DB) (http.Handler, error) {
	// Correct Steps to build a Middleware

	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/hello", HelloHandler)

	config := limiter.Config{
		Capacity:        100,
		TokensPerSecond: 1.67,
	}

	// Wrap the router with middlewares

	printMiddleware := logger.PrintMiddlewareHandler(apiMux)
	rateLimiterMiddleware, err := limiter.NewRateLimiterMiddleware(printMiddleware, db, config)
	if err != nil {
		fmt.Println("Error creating rate limiter middleware:", err)
		return nil, err
	}

	mux := http.NewServeMux()
	mux.Handle("/hello", cors(rateLimiterMiddleware))
	mux.Handle("/", http.FileServer(http.Dir("web")))
	return mux, nil
}
