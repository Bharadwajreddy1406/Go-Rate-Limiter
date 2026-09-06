package main

import (
	"database/sql"
	"fmt"
	"net/http"

	"rate-limiter/internal/limiter"
	"rate-limiter/internal/logger"
)

func HelloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Inside")

	fmt.Fprintln(w, "Hello World!")
}

func registerMiddleware(db *sql.DB) (http.Handler, error) {
	// Correct Steps to build a Middleware

	// Create the router
	mux := http.NewServeMux()

	// Register all routes
	mux.HandleFunc("/hello", HelloHandler)

	config := limiter.Config{
		Capacity:        100,
		TokensPerSecond: 1.67,
	}

	// Wrap the router with middlewares

	printMiddleware := logger.PrintMiddlewareHandler(mux)
	rateLimiterMiddleware, err := limiter.NewRateLimiterMiddleware(printMiddleware, db, config)
	if err != nil {
		fmt.Println("Error creating rate limiter middleware:", err)
		return nil, err
	}

	// Return the complete handler chain
	return rateLimiterMiddleware, nil
}
