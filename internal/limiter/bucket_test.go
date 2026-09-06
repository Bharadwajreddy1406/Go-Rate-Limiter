package limiter

import (
	"context"
	"testing"
	"time"

	sqlitedb "rate-limiter/internal/sqlite"
)

func TestRateLimiterStoresAndRefillsBucketsInSQLite(t *testing.T) {
	db, err := sqlitedb.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	limiter, err := NewRateLimiter(db, Config{Capacity: 2, TokensPerSecond: 10})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	for request, want := range []bool{true, true, false} {
		got, err := limiter.Allow(ctx, "user-1")
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("request %d: got allowed=%v, want %v", request+1, got, want)
		}
	}

	var tokens int
	if err := db.QueryRow("SELECT tokens FROM buckets WHERE key = ?", "user-1").Scan(&tokens); err != nil {
		t.Fatal(err)
	}
	if tokens != 0 {
		t.Fatalf("stored tokens = %d, want 0", tokens)
	}

	_, err = db.Exec("UPDATE buckets SET last_refill_at = ? WHERE key = ?", time.Now().Add(-time.Second).UnixNano(), "user-1")
	if err != nil {
		t.Fatal(err)
	}
	allowed, err := limiter.Allow(ctx, "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if !allowed {
		t.Fatal("request should be allowed after refill")
	}
}
