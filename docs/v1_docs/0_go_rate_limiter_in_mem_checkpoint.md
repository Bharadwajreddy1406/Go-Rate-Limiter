# Go Rate Limiter - Branch Notes (Checkpoint 1)

> **Checkpoint Scope**
>
> These notes cover everything completed **before** learning `http.Handler`, `http.HandlerFunc`, and middleware.
>
> They intentionally stop at the point where the next topic is request handling.

---

# 1. Project Goal

The objective is **not** just to build a rate limiter.

The objective is to learn:

* Idiomatic Go
* Go project structure
* Packages
* Constructors
* Methods
* Mutexes
* `net/http`
* Server architecture
* Design thinking

The rate limiter is simply the project through which these concepts are learned.

---

# 2. Project Structure

Current structure

```text
rate-limiter/

│
├── cmd/
│   └── server/
│       └── main.go
│
├── internal/
│   └── limiter/
│       ├── bucket.go
│       ├── config.go
│       ├── limiter.go
│       └── bucket_test.go
│
├── go.mod
└── go.sum
```

---

## Why `cmd/`?

`cmd/` contains executables.

Example:

```text
cmd/
    server/
    worker/
    migrate/
```

Each folder produces one executable.

```
go build ./cmd/server
```

↓

```
server
```

```
go build ./cmd/worker
```

↓

```
worker
```

This keeps entry points separate and scales well.

---

## Why `internal/`?

`internal/` contains implementation details.

Packages inside `internal` **cannot be imported by other Go modules**.

This lets us hide implementation details.

Everything related to the token bucket lives inside

```
internal/limiter
```

---

# 3. Module vs Package

## Module

A module is the entire project.

```
go.mod
```

defines the module root.

Example

```
module rate-limiter
```

---

## Package

A package is a collection of Go files.

Example

```
internal/limiter/
```

contains

```
bucket.go

config.go

limiter.go
```

All belong to

```go
package limiter
```

Go compiles all files belonging to the same package together.

---

# 4. Go Constructors

Go has no constructors.

Instead, constructors are ordinary functions.

Example

```go
func NewBucket(config Config) *Bucket
```

Convention:

```
New<Type>()
```

Examples from Go itself

```
http.NewServeMux()

bufio.NewReader()

bytes.NewBuffer()
```

---

# 5. Keyed Struct Literals

Preferred

```go
return &Bucket{
    Tokens: config.Capacity,
    Capacity: config.Capacity,
}
```

Avoid

```go
return &Bucket{
    config.Capacity,
    config.Capacity,
}
```

Reasons

* easier to read
* compiler catches mistakes
* survives field reordering
* self-documenting

---

# 6. Bucket Design

## Responsibility

A bucket represents **one user's token bucket**.

It knows

* current tokens
* refill configuration
* refill time
* synchronization

It does **not** know

* HTTP
* maps
* users
* middleware

---

## Final Bucket

```go
type Bucket struct {
    Tokens          int
    Capacity        int
    TokensPerSecond float64
    LastRefillAt    time.Time

    mu sync.Mutex
}
```

---

## Fields

### Tokens

Current available tokens.

Example

```
67
```

---

### Capacity

Maximum bucket size.

Example

```
100
```

Tokens can never exceed capacity.

---

### TokensPerSecond

Refill rate.

Example

```
1.5 tokens/second
```

---

### LastRefillAt

Time when refill calculation last advanced.

Used to calculate elapsed time.

---

### mu

Protects bucket state.

Protects

* Tokens
* LastRefillAt

Only one goroutine may modify them at once.

---

# 7. Config

```go
type Config struct {
    Capacity int

    TokensPerSecond float64
}
```

Configuration is validated once.

```go
func (c Config) Validate() error
```

Validation happens inside

```
NewRateLimiter()
```

---

# 8. RateLimiter Design

## Responsibility

RateLimiter manages buckets.

It does **not** implement the token bucket algorithm.

It only

* stores buckets
* creates buckets
* returns buckets

---

## Final Structure

```go
type RateLimiter struct {
    config Config

    buckets map[string]*Bucket

    mu sync.RWMutex
}
```

---

## Fields

### config

Shared configuration.

Every bucket uses the same configuration.

---

### buckets

```
userID

↓

Bucket
```

Example

```
alice

↓

Bucket
```

---

### mu

Protects the buckets map.

Not the buckets themselves.

---

# 9. Why Two Mutexes?

This is one of the most important concepts.

We have two different shared resources.

---

## Resource 1

The users map

```go
buckets map[string]*Bucket
```

Needs protection.

Example

```
User A arrives

↓

Create Bucket
```

At the same time

```
User A arrives again

↓

Create Bucket
```

Without synchronization

Both goroutines create separate buckets.

The map becomes inconsistent.

Therefore

```
RateLimiter.mu
```

protects the map.

---

## Resource 2

The Bucket

Inside a bucket

```
Tokens

LastRefillAt
```

Multiple requests for the same user may arrive simultaneously.

Without synchronization

```
Request A

↓

Tokens--

Request B

↓

Tokens--
```

Both modify the same memory.

Therefore

```
Bucket.mu
```

protects bucket state.

---

## Summary

```
RateLimiter.mu

↓

Protects map


Bucket.mu

↓

Protects bucket
```

Different resources.

Different mutexes.

---

# 10. Mutex vs RWMutex

## Mutex

```
Lock()

Unlock()
```

Only one goroutine may enter.

Everyone else waits.

Useful when

* writing
* modifying

---

## RWMutex

Adds

```
RLock()

RUnlock()
```

Multiple readers may read simultaneously.

Only writers require exclusive access.

---

Example

Many requests

```
Find bucket
```

Reading only.

These can happen together.

Creating a bucket

```
Write
```

Needs exclusive access.

Hence

```
RWMutex
```

is ideal for the map.

---

# 11. Double-Checked Locking

Algorithm

```
RLock

↓

Bucket exists?

↓

Yes

↓

Return

------------------

No

↓

Unlock

↓

Lock

↓

Check again

↓

Still missing?

↓

Create
```

Why check twice?

Two goroutines may observe

```
Bucket missing
```

simultaneously.

The second check prevents duplicate creation.

---

# 12. Bucket API

Public

```go
Allow()
```

Private

```go
refill()
```

Reason

Outside packages should only ask

```
Can this request proceed?
```

They should never manually refill buckets.

---

# 13. Request Algorithm

```
Allow()

↓

Lock

↓

refill()

↓

Tokens > 0 ?

↓

Yes

↓

Consume

↓

Return true

------------------

No

↓

Return false

↓

Unlock
```

---

# 14. getOrCreateBucket()

Algorithm

```
Read Lock

↓

Exists?

↓

Return

------------------

No

↓

Write Lock

↓

Check Again

↓

Create

↓

Return
```

---

# 15. Public RateLimiter API

```go
Allow(key string)
```

Internally

```
bucket := getOrCreateBucket(key)

↓

bucket.Allow()
```

The outside world never knows buckets exist.

---

# 16. Default ServeMux

When writing

```go
http.HandleFunc("/hello", HelloHandler)
```

Go registers the route in

```
http.DefaultServeMux
```

This is a global router.

Later

```go
http.ListenAndServe(":8080", nil)
```

Passing

```go
nil
```

means

```
Use DefaultServeMux
```

Internally, it's conceptually similar to

```go
http.ListenAndServe(":8080", http.DefaultServeMux)
```

---

## Request Flow

```
Browser

↓

Operating System

↓

Port 8080

↓

DefaultServeMux

↓

Matching Handler
```

---

# 17. Custom ServeMux

Instead of relying on the global router

```go
mux := http.NewServeMux()
```

Routes become

```go
mux.HandleFunc("/hello", HelloHandler)
```

Server

```go
server := &http.Server{
    Addr: ":8080",
    Handler: mux,
}
```

Advantages

* explicit ownership
* no global state
* easier testing
* multiple routers possible
* better for middleware

This is the preferred production approach.

---

# 18. Server vs ServeMux

These are different concepts.

## http.Server

Responsible for

* listening on ports
* accepting TCP connections
* server configuration
* lifecycle

It does **not** decide which handler runs.

---

## ServeMux

Responsible for routing.

Receives

```
GET /hello
```

Chooses

```
HelloHandler
```

Think of it as a routing table.

---

# 19. Final Architecture

```
Internet
     │
     ▼
Operating System
     │
     ▼
http.Server
     │
     ▼
ServeMux
     │
     ▼
RateLimiter
     │
     ▼
Bucket
     │
     ▼
Token Algorithm
```

Each layer has one responsibility.

---

# 20. Next Topic

The next learning milestone is:

* `http.Handler`
* `http.HandlerFunc`
* Middleware
* Request pipeline
* Integrating the RateLimiter into the HTTP server
