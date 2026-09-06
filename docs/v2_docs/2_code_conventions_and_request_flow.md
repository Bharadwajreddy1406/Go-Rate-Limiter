# Version 2 Code Conventions and Request Flow

## Request flow

The server still receives one outer `http.Handler`. The complete flow is:

```text
Client
    ↓
RateLimiterMiddleware
    ↓ allowed
PrintMiddleware
    ↓
ServeMux
    ↓
HelloHandler
```

The rate limiter is the outer middleware, so rejected requests stop before logging and route handling.

## Startup flow

`main` owns resources that live for the entire process:

```text
open SQLite
    ↓
defer database close
    ↓
register routes and middleware
    ↓
start HTTP server
```

Constructors return errors for startup failures. The application handles those errors before it starts accepting requests.

## Constructor conventions

I use `New<Type>` when constructing an application type:

```go
repository.NewBucketRepository(...)
limiter.NewRateLimiter(...)
limiter.NewRateLimiterMiddleware(...)
```

The SQLite package uses `Open` because opening a database connection matches the standard `database/sql` vocabulary.

Dependencies are passed in instead of hidden in package-level variables. For example, the limiter receives `*sql.DB`; it does not open its own database. This makes ownership clear and lets tests provide an isolated database.

## Context convention

The middleware passes the request context into the limiter:

```go
allowed, err := limiter.Allow(r.Context(), key)
```

The repository then uses `BeginTx`, `QueryRowContext`, and `ExecContext`. If the client disconnects or the request is cancelled, the database work can stop too.

## Error convention

Each layer adds useful context while preserving the original error with `%w`:

```go
return false, fmt.Errorf("read bucket: %w", err)
```

The middleware logs the internal error but sends a generic response to the client. Database details should not leak through the HTTP response.

## Input convention

`X-User-ID` is the trust boundary for the current demo. I trim surrounding whitespace and reject a missing or blank value with `400 Bad Request`.

```text
missing X-User-ID → 400
valid ID, empty bucket → 429
valid ID, database error → 500
valid ID, token available → next handler
```

This avoids putting every anonymous request into one bucket under an empty key.

## SQL conventions

- Use placeholders (`?`) for values.
- Keep schema creation next to database initialization.
- Keep SQL operations inside the repository package.
- Use constraints for invariants SQLite can enforce.
- Store time as Unix nanoseconds so precision is explicit and conversion is simple.
- Keep a refill and token consumption inside the same transaction.

## What I deliberately removed from version 1

The old `Bucket` struct, bucket map, `sync.RWMutex`, and per-bucket `sync.Mutex` no longer exist. Keeping them would create two competing sources of truth.

SQLite now stores the mutable state. The limiter only validates configuration and delegates bucket changes to the repository.

## Verification

The focused test uses the real in-memory SQLite database. It checks that:

1. A new bucket starts full.
2. Available tokens are consumed.
3. A request is denied when the bucket is empty.
4. The token count really exists in the database.
5. Moving the stored refill time backward allows the bucket to refill.

Run all checks with:

```bash
go test ./...
go vet ./...
```
