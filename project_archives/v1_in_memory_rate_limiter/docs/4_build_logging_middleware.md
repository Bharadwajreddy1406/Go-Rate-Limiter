read [these](./write_these_before_you_read_this.md) instructions before you read this document.

I called it  PrintMiddleware 

---

# Request Pipeline Setup

## Overview

The HTTP server should not know about routing, middleware, or business logic. It only needs a single `http.Handler`.

Our job is to build a **handler chain** and give the outermost handler to the server.

---

## Request Flow

```text
Client Request
      │
      ▼
http.Server
      │
      ▼
Middleware(s)
      │
      ▼
ServeMux (Router)
      │
      ▼
Matched Route Handler
      │
      ▼
Response
```

---

# Step 1 — Create the Router

Create a `ServeMux` that will hold all application routes.

```go
mux := http.NewServeMux()
```

At this point, the router is empty.

---

# Step 2 — Register Routes

Register all endpoints on the router.

```go
mux.HandleFunc("/hello", HelloHandler)
mux.HandleFunc("/health", HealthHandler)
```

Conceptually, the router now maintains a routing table:

```text
/hello   -> HelloHandler
/health  -> HealthHandler
```

---

# Step 3 — Wrap the Router with Middleware

Middleware should wrap the router instead of modifying it.

```go
logging := logger.NewPrintMiddleware(mux)
```

The middleware stores the router as its `next` handler.

```text
Logging Middleware
        │
        ▼
     ServeMux
```

Because `ServeMux` implements `http.Handler`, it can be wrapped by any middleware that accepts an `http.Handler`.

---

# Step 4 — Give the Final Handler to the Server

The server receives only the outermost handler.

```go
server := &http.Server{
    Addr:    ":9090",
    Handler: logging,
}
```

The server does **not** know whether it is talking to:

* a router
* a middleware
* a custom handler

It simply invokes:

```go
handler.ServeHTTP(w, r)
```

---

# Complete Flow

```go
func buildHandler() http.Handler {
    mux := http.NewServeMux()

    // Register routes
    mux.HandleFunc("/hello", HelloHandler)
    mux.HandleFunc("/health", HealthHandler)

    // Wrap with middleware
    logging := logger.NewPrintMiddleware(mux)

    return logging
}
```

```go
func main() {
    handler := buildHandler()

    server := &http.Server{
        Addr:    ":9090",
        Handler: handler,
    }

    server.ListenAndServe()
}
```

---

# Why This Design?

* The router is responsible only for route matching.
* Middleware is responsible only for cross-cutting concerns (logging, authentication, rate limiting, recovery, etc.).
* The server is responsible only for accepting HTTP requests and invoking the supplied `http.Handler`.
* Each layer has a single responsibility, making the application modular and easy to extend.

---

# Scaling the Middleware Chain

As new middleware is added, each one wraps the previous handler.

```text
http.Server
      │
      ▼
Rate Limiter
      │
      ▼
Authentication
      │
      ▼
Logging
      │
      ▼
ServeMux
      │
      ▼
Route Handler
```

Every middleware implements the same `http.Handler` interface, allowing them to be composed into a pipeline without changing the server or the route handlers.
