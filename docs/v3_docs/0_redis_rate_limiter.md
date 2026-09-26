# Redis Rate Limiter — Version 3 Architecture

> **Current version:** Redis-backed token bucket  
> **Previous version:** SQLite-backed token bucket  
> **Main goal:** let several Go processes safely share the same rate-limit state

## Documentation map

| Guide | What it explains |
| --- | --- |
| **1. Architecture — this page** | Why Redis was introduced, package boundaries, stored data, and startup lifecycle |
| [2. Atomic token-bucket algorithm](1_atomic_token_bucket_algorithm.md) | The refill maths, Lua script, race prevention, expiry, and worked examples |
| [3. Running, testing, and observing](2_request_flow_running_and_testing.md) | HTTP flow, Docker commands, Redis inspection, tests, troubleshooting, and production notes |

## The learning path

The rate-limit rule stays the same in every version. What changes is where the bucket state lives and how concurrent requests update it safely.

```mermaid
flowchart LR
    V1["Version 1<br/>Go map + mutexes"]
    V2["Version 2<br/>SQLite + transactions"]
    V3["Version 3<br/>Redis + atomic Lua script"]
    V4["Possible next step<br/>multiple API instances"]

    V1 -->|move state outside a Go struct| V2
    V2 -->|move state outside one machine| V3
    V3 -->|share one Redis data set| V4

    classDef current fill:#dbeafe,stroke:#2563eb,stroke-width:2px,color:#172554;
    class V3 current;
```

| Version | Source of truth | Concurrency protection | Can separate servers share limits? |
| --- | --- | --- | --- |
| In memory | A Go map in one process | Go mutexes | No |
| SQLite | A local `.db` file | SQL transaction | Normally no |
| Redis | Keys in a Redis server | One atomic Lua script | Yes, when they use the same Redis instance |

The completed older implementations are preserved under `project_archives/`. The repository root contains the current version.

## What problem Redis solves

With in-memory storage, restarting the Go process removes every bucket. A second server also gets a completely different map.

SQLite makes bucket data durable on one machine, but the database file is still local. If two API servers use two different files, a user effectively gets two separate limits.

Redis gives all application instances one shared location:

```mermaid
flowchart TB
    Client[Client]
    LB[Load balancer]
    API1[Go server A]
    API2[Go server B]
    API3[Go server C]
    Redis[(Shared Redis)]

    Client --> LB
    LB --> API1
    LB --> API2
    LB --> API3
    API1 --> Redis
    API2 --> Redis
    API3 --> Redis

    classDef datastore fill:#fee2e2,stroke:#dc2626,stroke-width:2px,color:#450a0a;
    class Redis datastore;
```

This repository currently starts one Go server, but its bucket operation is designed so additional servers can use the same Redis keys safely.

## Current application architecture

The HTTP layer does not know how Redis fields are calculated. The Redis package does not know anything about HTTP headers. Each layer has one main responsibility.

```mermaid
flowchart LR
    Browser[Browser or API client]
    Mux[HTTP ServeMux]
    CORS[CORS wrapper]
    Middleware[RateLimiterMiddleware]
    Limiter[RateLimiter]
    Repository[BucketRepository]
    Redis[(Redis)]
    Logger[PrintMiddleware]
    Handler[HelloHandler]
    Static[web/index.html]

    Browser --> Mux
    Mux -->|GET /hello| CORS
    CORS --> Middleware
    Middleware --> Limiter
    Limiter --> Repository
    Repository -->|atomic Lua script| Redis
    Middleware -->|allowed| Logger
    Logger --> Handler
    Mux -->|GET /| Static

    classDef storage fill:#fef3c7,stroke:#d97706,stroke-width:2px,color:#451a03;
    class Redis storage;
```

### Package responsibilities

| Location | Responsibility | Deliberately does not do |
| --- | --- | --- |
| `cmd/server/main.go` | Opens Redis, builds the handler, and starts the HTTP server | Calculate tokens |
| `cmd/server/routes.go` | Defines routes, middleware order, capacity, and refill rate | Execute Redis commands |
| `internal/redis/redis.go` | Reads connection settings, creates the client, and checks `PING` | Know about buckets or HTTP |
| `internal/limiter/config.go` | Validates capacity and refill rate | Store mutable state |
| `internal/limiter/middleware.go` | Reads `X-User-ID` and converts decisions into HTTP responses | Implement the refill maths |
| `internal/limiter/limiter.go` | Provides the rate-limiter application boundary | Know the Redis key format |
| `internal/repository/bucket.go` | Owns the key format and atomic bucket operation | Know about status codes |

This separation makes future changes smaller. For example, adding response headers belongs in the middleware, while changing the stored bucket format belongs in the repository.

## Startup lifecycle

Redis is an application dependency, so the server checks it before accepting HTTP traffic.

```mermaid
sequenceDiagram
    participant Main as cmd/server
    participant Config as Environment
    participant Client as go-redis client
    participant Redis
    participant HTTP as HTTP server

    Main->>Config: Read REDIS_ADDR, username, password, DB
    Main->>Client: Create client
    Main->>Redis: PING with 5-second startup context
    alt Redis responds
        Redis-->>Main: PONG
        Main->>HTTP: Build middleware and ListenAndServe(:9090)
    else Redis is unavailable
        Redis--xMain: Connection error
        Main->>Main: Print contextual error and exit
    end
```

Opening the client once is important. The same concurrency-safe client and its connection pool are reused for all requests. A new Redis client is not created inside `ServeHTTP`.

## How one user is stored

The value of `X-User-ID` becomes the final part of a namespaced Redis key.

```text
Header: X-User-ID: ramu

Redis key: rate-limiter:bucket:ramu
Redis type: hash
```

The hash has two fields:

| Field | Example | Meaning |
| --- | ---: | --- |
| `tokens` | `73` | Whole tokens available after the latest decision |
| `last_refill_at` | `1790407907362.42` | Redis time in milliseconds, including saved partial refill progress |

An example from `redis-cli` looks like this:

```text
127.0.0.1:6379> HGETALL rate-limiter:bucket:ramu
1) "tokens"
2) "73"
3) "last_refill_at"
4) "1790407907362.42"
```

Why a Redis hash instead of two unrelated keys?

- Both fields describe one bucket.
- The script can read them together with `HMGET`.
- Redis can expire the whole bucket with one TTL.
- The key namespace is easy to inspect without mixing it with other application data.

## Bucket lifecycle

```mermaid
stateDiagram-v2
    [*] --> Missing
    Missing --> Active: first request creates bucket and consumes one token
    Active --> Active: refill and allowed request
    Active --> Empty: last token consumed
    Empty --> Empty: request denied before a full token is earned
    Empty --> Active: enough time passes to earn a token
    Active --> Missing: no request until TTL expires
    Empty --> Missing: no request until full-refill TTL expires
```

A missing bucket is treated as full. Therefore, deleting an inactive key after it has had enough time to refill does not change the rate-limit result.

## Configuration used by this project

The current values live in `cmd/server/routes.go`:

| Setting | Value | Effect |
| --- | ---: | --- |
| Capacity | `100` tokens | A new or fully refilled user may burst up to 100 requests |
| Refill rate | `1.67` tokens/second | Roughly 100 tokens are restored per minute |
| Request cost | `1` token | Every allowed `/hello` request consumes one token |

The approximate time to refill an empty bucket is:

```text
100 / 1.67 = 59.88 seconds
```

That same duration is used as the bucket TTL. The exact calculation and its reason are explained in the [algorithm guide](1_atomic_token_bucket_algorithm.md).

## What “distributed” means here

The bucket update is ready to be shared by multiple Go processes because:

1. Bucket state is outside every Go process.
2. Every process uses the same key naming rule.
3. Redis serializes the Lua operation.
4. Refill time comes from Redis rather than each application's clock.

It does **not** automatically provide a full production deployment. The current Compose file starts one Redis container without authentication or TLS, and the application uses a standard single-node Redis client. Those choices are suitable for local learning, not an internet-facing Redis service.

## Continue reading

Next: [Atomic token-bucket algorithm](1_atomic_token_bucket_algorithm.md)
