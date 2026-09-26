package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	redisdb "rate-limiter/internal/redis"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := redisdb.Open(ctx)
	if err != nil {
		fmt.Println("Error opening Redis:", err)
		return
	}
	defer client.Close()

	handler, err := registerMiddleware(client)
	if err != nil {
		fmt.Println("Error registering middleware:", err)
		return
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", Port),
		Handler: handler,
	}

	fmt.Println("Server is running on port", Port)
	if err := server.ListenAndServe(); err != nil {
		fmt.Println("Error starting server:", err)
	}
}
