# Go Rate Limiter

This project is a token-bucket rate limiter built in small versions so each storage change is easy to understand.

1. Version 1 stored buckets in a Go map.
2. Version 2 stored buckets in SQLite.
3. Version 3, the current version, stores buckets in Redis.

The earlier implementations are kept in `project_archives/`. The root of the repository always contains the current version.

## How version 3 works

Every value from the `X-User-ID` header gets its own token bucket. Redis stores that bucket as a hash:

```text
rate-limiter:bucket:<user-id>
  tokens
  last_refill_at
```

One Lua script performs the refill, availability check, token consumption, update, and expiry. Redis runs the script atomically, so concurrent requests and multiple Go server instances cannot consume the same token.

Inactive bucket keys expire after the time required to refill to full capacity. Removing such a key is safe because the next request should receive a full bucket anyway.

## Run it

Start Redis in Docker Desktop:

```bash
docker compose up -d
```

Create your local environment file if it does not already exist:

```powershell
Copy-Item .env.example .env
```

Start the Go server:

```bash
go run ./cmd/server
```

Open <http://localhost:9090> for the browser demo, or call the API directly:

```bash
curl -i -H "X-User-ID: ramu" http://localhost:9090/hello
```

The server loads `.env` automatically. `.env.example` contains safe defaults for the local Docker Redis service:

```dotenv
REDIS_ADDR=localhost:6379
REDIS_USERNAME=
REDIS_PASSWORD=
REDIS_DB=0
```

The real `.env` file is ignored by Git so credentials are not committed. Existing operating-system environment variables take priority over values in `.env`.

## API behavior

`GET /hello` requires an `X-User-ID` header.

| Result | Status | Response |
| --- | --- | --- |
| Token available | `200 OK` | `Hello World!` |
| Header missing or blank | `400 Bad Request` | `X-User-ID header is required` |
| Bucket empty | `429 Too Many Requests` | `Rate limit exceeded` |
| Redis unavailable | `500 Internal Server Error` | `Internal server error` |

The limiter currently gives each user 100 tokens and refills 1.67 tokens per second. Change these values in `cmd/server/routes.go`.

## Test it

Unit tests run without Redis. Integration tests run when `REDIS_TEST_ADDR` is set.

PowerShell:

```powershell
$env:REDIS_TEST_ADDR = "localhost:6379"
go test ./...
go vet ./...
```

The integration tests use Redis database 15 and delete only the unique keys they create.

## Detailed documentation

The Version 3 guide contains diagrams, worked examples, and operational notes:

1. [Redis architecture](docs/v3_docs/0_redis_rate_limiter.md)
2. [Atomic token-bucket algorithm](docs/v3_docs/1_atomic_token_bucket_algorithm.md)
3. [Request flow, running, testing, and observing](docs/v3_docs/2_request_flow_running_and_testing.md)

## Useful commands

Inspect rate-limit keys:

```bash
docker compose exec redis redis-cli --scan --pattern "rate-limiter:bucket:*"
```

Stop Redis while keeping its data:

```bash
docker compose down
```

Stop Redis and reset its data:

```bash
docker compose down -v
```

## Project layout

```text
cmd/server/           HTTP server and route wiring
internal/limiter/     Configuration and HTTP middleware
internal/repository/  Atomic Redis token-bucket operation
internal/redis/       Redis connection setup
web/                  Browser demo
docs/                 Notes for each learning version
project_archives/     Complete older implementations
```
