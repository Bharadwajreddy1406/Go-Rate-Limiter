# Go Rate Limiter

I am building this project while learning Go. The changes are intentionally incremental: each version keeps the token-bucket behavior while replacing the storage and deployment model underneath it. The current version limits requests to `/hello` independently by the `X-User-ID` header.

## Learning path

1. **Done — in-memory limiter:** one Go process held user buckets in a map, protected with mutexes. The original implementation is kept in `project_archives/v1_in_memory_rate_limiter`.
2. **Current — SQLite limiter:** bucket state is persisted in `rate-limiter.db`, and each refill/consume operation is a SQLite transaction.
3. **Next — Redis limiter:** move bucket state from a local database to Redis.
4. **Later — distributed limiter:** share the Redis-backed limit across multiple server instances instead of having a separate limiter per server.

## Run it

Requires the Go version declared in `go.mod`.

```bash
go run ./cmd/server
```

Open <http://localhost:9090> for the browser demo. It picks a user from a fixed list and calls `/hello` with that user's `X-User-ID` header.

You can also test the API directly:

```bash
curl -i -H 'X-User-ID: ramu' http://localhost:9090/hello
```

## API

### `GET /hello`

| Header | Required | Description |
| --- | --- | --- |
| `X-User-ID` | Yes | The key used for this user's rate-limit bucket. |

| Result | Status | Response |
| --- | --- | --- |
| Request allowed | `200 OK` | `Hello World!` |
| Header missing or blank | `400 Bad Request` | `X-User-ID header is required` |
| User has no tokens | `429 Too Many Requests` | `Rate limit exceeded` |
| Database failure | `500 Internal Server Error` | `Internal server error` |

The page shows a clear “user is rate limited” message when it receives `429`.

## Rate-limit behavior

The current configuration is in `cmd/server/routes.go`:

- Capacity: 100 tokens per user
- Refill rate: 1.67 tokens per second
- Each allowed request consumes one token

Every token update uses a SQLite transaction, so the bucket update is stored atomically. The server logs the user ID and token count before and after allowed requests.

## Database

Starting the server creates `rate-limiter.db` in the repository root. It stores a `buckets` table containing each user key, remaining tokens, and refill timestamp.

The file is ignored by Git and survives server restarts. Delete `rate-limiter.db` while the server is stopped to reset all buckets. Tests use an in-memory database and do not touch this file.

## Browser demo and Live Server

The Go server serves `web/index.html` at `/`, so no separate frontend server is required.

If you use VS Code Live Server instead, keep the Go server running too. The page calls `http://localhost:9090/hello` directly and the API permits the `X-User-ID` browser header with demo-only CORS settings.

## Test

```bash
go test ./...
```

## Project layout

```text
cmd/server/          HTTP server and route wiring
internal/limiter/    Rate-limiter middleware and configuration
internal/repository/ SQLite token-bucket transaction
internal/sqlite/     Database setup
web/                 Browser demo
docs/                Learning notes for the in-memory and SQLite versions
```
