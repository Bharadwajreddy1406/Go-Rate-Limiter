package main

import (
	"fmt"
	"net/http"

	sqlitedb "rate-limiter/internal/sqlite"
)

func main() {
	fmt.Println("Server is running on port", Port)

	db, err := sqlitedb.Open()
	if err != nil {
		fmt.Println("Error opening database:", err)
		return
	}
	defer db.Close()

	handler, err := registerMiddleware(db)
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
