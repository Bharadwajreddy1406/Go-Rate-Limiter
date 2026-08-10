read [logging_middleware_setup](./build_logging_middleware.md) to understand how to set up logging middleware.


# Code Walkthrough - Rate Limiter Middleware Integration

This section shows how the rate limiter was integrated into the HTTP request pipeline.

---

# 1. Create the Router

The first step is to create the application's router.

```go
mux := http.NewServeMux()
```

At this point, the router contains no routes and no middleware.

---

# 2. Register Application Routes

All endpoints are registered on the router before any middleware is applied.

```go
mux.HandleFunc("/hello", HelloHandler)
```

Conceptually, the router now looks like:

```text
/hello  ---> HelloHandler
```

---

# 3. Configure the Rate Limiter

The middleware accepts a configuration object instead of hardcoding values.

```go
config := limiter.Config{
    Capacity:        100,
    TokensPerSecond: 1.67,
}
```

This keeps the middleware reusable while allowing different applications to configure different limits.

---

# 4. Wrap Existing Handlers

The router is first wrapped by the logging middleware.

```go
loggingMiddleware := logger.PrintMiddlewareHandler(mux)
```

This produces the following handler chain:

```text
Logging Middleware
        │
        ▼
     ServeMux
```

Next, the logging middleware is wrapped by the rate limiter.

```go
rateLimiterMiddleware := limiter.RateLimiterMiddlewareHandler(
    loggingMiddleware,
    config,
)
```

The final request pipeline becomes:

```text
RateLimiter Middleware
          │
          ▼
Logging Middleware
          │
          ▼
      ServeMux
```

Notice that middleware **wraps** an existing handler instead of modifying it.

Each middleware receives another `http.Handler` and decides whether to continue the request.

---

# 5. Return the Outermost Handler

The middleware registration function returns the outermost handler.

```go
func registerMiddleware() http.Handler {
    mux := http.NewServeMux()

    mux.HandleFunc("/hello", HelloHandler)

    config := limiter.Config{
        Capacity:        100,
        TokensPerSecond: 1.67,
    }

    loggingMiddleware := logger.PrintMiddlewareHandler(mux)

    rateLimiterMiddleware := limiter.RateLimiterMiddlewareHandler(
        loggingMiddleware,
        config,
    )

    return rateLimiterMiddleware
}
```

The caller does not know how many middleware layers exist.

It simply receives an `http.Handler`.

---

# 6. Start the Server

The server only needs a single handler.

```go
func main() {
    server := &http.Server{
        Addr:    ":9090",
        Handler: registerMiddleware(),
    }

    server.ListenAndServe()
}
```

The HTTP server is completely unaware of:

* routing
* logging
* rate limiting
* token buckets

It simply invokes:

```go
handler.ServeHTTP(w, r)
```

---

# 7. Middleware Construction

The middleware owns a single `RateLimiter` instance.

```go
type RateLimiterMiddleware struct {
    limiter *RateLimiter
    next    http.Handler
}
```

During application startup, the middleware creates the limiter exactly once.

```go
func RateLimiterMiddlewareHandler(
    next http.Handler,
    config Config,
) http.Handler {

    limiter, err := NewRateLimiter(config)
    if err != nil {
        panic(err)
    }

    return &RateLimiterMiddleware{
        limiter: limiter,
        next:    next,
    }
}
```

This ensures that:

* one shared bucket map exists,
* all requests reuse the same limiter,
* buckets persist across requests.

If the limiter were created inside `ServeHTTP()`, every request would receive a fresh bucket map and rate limiting would never work.

---

# 8. Request Processing

Every incoming request first passes through the middleware.

```go
func (m *RateLimiterMiddleware) ServeHTTP(
    w http.ResponseWriter,
    r *http.Request,
) {
    key := r.RemoteAddr

    if !m.limiter.Allow(key) {
        http.Error(
            w,
            "Rate limit exceeded",
            http.StatusTooManyRequests,
        )
        return
    }

    m.next.ServeHTTP(w, r)
}
```

The middleware itself contains almost no business logic.

Its responsibilities are limited to:

1. Extract the request key.
2. Ask the limiter whether the request is allowed.
3. Reject or forward the request.

The token bucket algorithm remains completely encapsulated inside the `RateLimiter` and `Bucket` types.

---

# Final Request Lifecycle

```text
Client
   │
   ▼
http.Server
   │
   ▼
RateLimiterMiddleware
   │
   ├── limiter.Allow(key)
   │
   ├── Allowed?
   │      │
   │      ├── No  ─────► HTTP 429
   │      │
   │      └── Yes
   │
   ▼
LoggerMiddleware
   │
   ▼
ServeMux
   │
   ▼
HelloHandler
```

This design keeps each component focused on a single responsibility while allowing middleware to be composed simply by wrapping one `http.Handler` around another.
