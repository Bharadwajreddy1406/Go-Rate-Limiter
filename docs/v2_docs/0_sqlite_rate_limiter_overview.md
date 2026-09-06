# SQLite Rate Limiter - Version 2 Overview

## What I changed in version 2

In version 1, every user's token bucket lived in a Go map:

```text
map[userID]*Bucket
```

That taught me how mutexes protect shared in-memory data. In version 2, I moved the bucket state into an in-memory SQLite database.

```text
HTTP request
    ↓
Rate limiter middleware
    ↓
RateLimiter
    ↓
BucketRepository
    ↓
SQLite :memory: database
```

The database is still temporary. It disappears when the process stops, but SQLite now owns the shared bucket state instead of a Go map.

## Why use SQLite in memory?

This version is a bridge between a local in-memory implementation and a rate limiter backed by an external database.

It lets me learn:

- `database/sql`
- SQL tables and constraints
- repository boundaries
- transactions
- database error handling
- passing request contexts into database calls

It does not make buckets durable or share them between server processes. A file-backed or remote database would be needed for that.

## Project responsibilities

```text
cmd/server/
    creates the database, builds the handler chain, starts the server

internal/sqlite/
    opens SQLite and creates the schema

internal/repository/
    performs atomic bucket reads and writes

internal/limiter/
    validates configuration and applies rate limiting to HTTP requests
```

Each package has one reason to change. HTTP code does not contain SQL, and the repository does not know anything about headers or status codes.

## The buckets table

```sql
CREATE TABLE buckets (
    key            TEXT PRIMARY KEY,
    tokens         INTEGER NOT NULL CHECK (tokens >= 0),
    last_refill_at INTEGER NOT NULL
);
```

The columns have direct jobs:

- `key` identifies the caller. The middleware currently gets it from `X-User-ID`.
- `tokens` stores the number of whole tokens available.
- `last_refill_at` stores a Unix timestamp in nanoseconds.

The primary key guarantees one bucket per user. The `CHECK` constraint prevents negative token counts even if application code has a bug.

## One important SQLite detail

SQLite creates a separate `:memory:` database for each connection. `database/sql` normally manages a pool of connections, so unrestricted pooling could create multiple independent databases.

I keep exactly one connection open:

```go
db.SetMaxOpenConns(1)
db.SetMaxIdleConns(1)
```

This also serializes bucket transactions, which is correct for this local learning version. If throughput becomes important, the next step is not more in-process locking; it is choosing a storage design intended for concurrent or distributed rate limiting.

## Lifetime and ownership

The server opens the database once during startup:

```go
db, err := sqlitedb.Open()
```

It passes the same `*sql.DB` through the application and closes it when `main` returns.

The database must not be created inside `ServeHTTP`. Doing that would create a fresh empty database for every request, so no user would ever reach the limit.
