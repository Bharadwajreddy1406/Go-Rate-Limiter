# Token Bucket Logic with SQLite Transactions

## The rule I need to preserve

A token bucket has a capacity and a refill rate.

For each request:

1. Find or create the user's bucket.
2. Add the whole tokens earned since the last refill.
3. Cap the result at the configured capacity.
4. Consume one token when one is available.
5. Reject the request when no token is available.

Moving data into SQLite changes storage, not the token bucket algorithm.

## Creating a bucket

If the user has no row, the repository inserts one. The first request is allowed immediately, so the stored count is `capacity - 1`.

```sql
INSERT INTO buckets (key, tokens, last_refill_at)
VALUES (?, ?, ?);
```

I use SQL parameters instead of building SQL strings. This avoids quoting bugs and SQL injection.

## Refilling an existing bucket

The elapsed time is calculated from the stored timestamp:

```go
elapsed := now.Sub(time.Unix(0, lastRefillAt)).Seconds()
tokensToAdd := int(elapsed * tokensPerSecond)
```

Converting to `int` means only complete tokens are added. Any partial progress must not be lost, so I advance the refill timestamp only by the time represented by the whole tokens that were added:

```go
refillDuration := time.Duration(
    float64(tokensToAdd) / tokensPerSecond * float64(time.Second),
)
```

Example with a rate of 2 tokens per second:

```text
750 ms elapsed
    ↓
1 complete token is added
    ↓
500 ms is applied to last_refill_at
    ↓
250 ms remains for the next request
```

If I replaced `last_refill_at` with the current time after every refill, that remaining 250 ms would be discarded.

## Why the operation uses a transaction

Reading the token count and updating it are one logical operation.

Without a transaction, two requests could both read one remaining token and both decide they are allowed:

```text
Request A reads 1
Request B reads 1
Request A writes 0
Request B writes 0
```

That spends one stored token twice.

The repository starts a transaction before reading and commits only after updating:

```text
BEGIN
    SELECT bucket
    calculate refill
    consume token when available
    UPDATE bucket
COMMIT
```

The in-memory database uses one connection, so concurrent calls wait for that connection and execute these transactions one at a time.

## Rollback convention

Immediately after beginning a transaction, I defer a rollback:

```go
defer tx.Rollback()
```

This is safe even after a successful commit. It guarantees that every early error path releases the transaction without repeating rollback code after every database operation.

## Errors are different from denied requests

The repository returns two values:

```go
(allowed bool, err error)
```

These cases mean different things:

- `false, nil`: the bucket is empty, so the client receives `429 Too Many Requests`.
- `false, err`: SQLite failed, so the client receives `500 Internal Server Error`.
- `true, nil`: a token was consumed and the next handler runs.

A storage failure must not look like a normal rate-limit rejection.
