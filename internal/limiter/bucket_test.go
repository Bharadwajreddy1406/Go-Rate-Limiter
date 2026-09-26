package limiter

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func redisTestClient(t *testing.T) *redis.Client {
	t.Helper()

	address := os.Getenv("REDIS_TEST_ADDR")
	if address == "" {
		t.Skip("set REDIS_TEST_ADDR to run Redis integration tests")
	}

	client := redis.NewClient(&redis.Options{Addr: address, DB: 15})
	if err := client.Ping(context.Background()).Err(); err != nil {
		client.Close()
		t.Fatalf("connect to test Redis: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

func uniqueUser(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("test:%s:%d", t.Name(), time.Now().UnixNano())
}

func TestRateLimiterStoresAndRefillsBucketsInRedis(t *testing.T) {
	client := redisTestClient(t)
	user := uniqueUser(t)
	key := "rate-limiter:bucket:" + user
	t.Cleanup(func() { client.Del(context.Background(), key) })

	rateLimiter, err := NewRateLimiter(client, Config{Capacity: 2, TokensPerSecond: 10})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if tokens, err := rateLimiter.GetTokens(ctx, user); err != nil || tokens != 2 {
		t.Fatalf("new bucket tokens = %d, %v; want 2, nil", tokens, err)
	}

	for request, want := range []bool{true, true, false} {
		decision, err := rateLimiter.Allow(ctx, user)
		if err != nil {
			t.Fatal(err)
		}
		if decision.Allowed != want {
			t.Fatalf("request %d: got allowed=%v, want %v", request+1, decision.Allowed, want)
		}
	}

	storedTokens, err := client.HGet(ctx, key, "tokens").Int()
	if err != nil {
		t.Fatal(err)
	}
	if storedTokens != 0 {
		t.Fatalf("stored tokens = %d, want 0", storedTokens)
	}

	redisTime, err := client.Time(ctx).Result()
	if err != nil {
		t.Fatal(err)
	}
	if err := client.HSet(ctx, key, "last_refill_at", redisTime.Add(-time.Second).UnixMilli()).Err(); err != nil {
		t.Fatal(err)
	}

	decision, err := rateLimiter.Allow(ctx, user)
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Allowed || decision.TokensBefore != 2 || decision.TokensAfter != 1 {
		t.Fatalf("decision after refill = %+v, want allowed with tokens 2 -> 1", decision)
	}
}

func TestRateLimiterConsumesTokensAtomically(t *testing.T) {
	client := redisTestClient(t)
	user := uniqueUser(t)
	key := "rate-limiter:bucket:" + user
	t.Cleanup(func() { client.Del(context.Background(), key) })

	rateLimiter, err := NewRateLimiter(client, Config{Capacity: 10, TokensPerSecond: 0.001})
	if err != nil {
		t.Fatal(err)
	}

	var allowed atomic.Int64
	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			decision, err := rateLimiter.Allow(context.Background(), user)
			if err != nil {
				t.Errorf("allow request: %v", err)
				return
			}
			if decision.Allowed {
				allowed.Add(1)
			}
		}()
	}
	wg.Wait()

	if got := allowed.Load(); got != 10 {
		t.Fatalf("allowed requests = %d, want 10", got)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []Config{
		{Capacity: 0, TokensPerSecond: 1},
		{Capacity: 1, TokensPerSecond: 0},
	}

	for _, config := range tests {
		if err := config.Validate(); err == nil {
			t.Fatalf("Validate(%+v) returned nil", config)
		}
	}
}
