package logger

import (
	"fmt"
	"net/http"
	"time"
)

type PrintMiddleware struct {
	next http.Handler
}

func PrintMiddlewareHandler(next http.Handler) *PrintMiddleware {
	return &PrintMiddleware{
		next: next,
	}
}

func (p *PrintMiddleware) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {

	start := time.Now()
	fmt.Println("========== Incoming Request ==========")
	fmt.Printf("%s %s\n", r.Method, r.URL.Path)
	fmt.Println("======================================")

	defer func() {
		fmt.Println("Request Duration:", time.Since(start))
	}()

	p.next.ServeHTTP(w, r)
}