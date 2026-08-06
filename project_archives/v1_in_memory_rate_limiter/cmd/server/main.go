package main

import (
	"fmt"
	"net/http"
)

func main() {
	fmt.Println("Server is running on port", Port)

	handler, err := registerMiddleware()
	if err != nil {
		fmt.Println("Error registering middleware:", err)
		return
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", Port),
		Handler: handler,
	}

	if err := server.ListenAndServe(); err != nil {
		fmt.Println("Error starting server:", err)
	}
}
