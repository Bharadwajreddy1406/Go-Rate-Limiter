package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type BucketRepository struct {
	db              *sql.DB
	capacity        int
	tokensPerSecond float64
}

func NewBucketRepository(db *sql.DB, capacity int, tokensPerSecond float64) *BucketRepository {
	return &BucketRepository{
		db:              db,
		capacity:        capacity,
		tokensPerSecond: tokensPerSecond,
	}
}

func (r *BucketRepository) GetTokens(ctx context.Context, key string) (int, error) {
	var tokens int
	err := r.db.QueryRowContext(ctx, "SELECT tokens FROM buckets WHERE key = ?", key).Scan(&tokens)
	if errors.Is(err, sql.ErrNoRows) {
		return r.capacity, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read bucket tokens: %w", err)
	}
	return tokens, nil
}

func (r *BucketRepository) Allow(ctx context.Context, key string) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin bucket transaction: %w", err)
	}
	defer tx.Rollback()

	now := time.Now()
	var tokens int
	var lastRefillAt int64
	err = tx.QueryRowContext(ctx,
		"SELECT tokens, last_refill_at FROM buckets WHERE key = ?", key,
	).Scan(&tokens, &lastRefillAt)

	if errors.Is(err, sql.ErrNoRows) {
		_, err = tx.ExecContext(ctx,
			"INSERT INTO buckets (key, tokens, last_refill_at) VALUES (?, ?, ?)",
			key, r.capacity-1, now.UnixNano(),
		)
		if err != nil {
			return false, fmt.Errorf("insert bucket: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return false, fmt.Errorf("commit bucket: %w", err)
		}
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("read bucket: %w", err)
	}

	elapsed := now.Sub(time.Unix(0, lastRefillAt)).Seconds()
	tokensToAdd := int(elapsed * r.tokensPerSecond)
	if tokensToAdd > 0 {
		tokens = min(r.capacity, tokens+tokensToAdd)
		refillDuration := time.Duration(float64(tokensToAdd) / r.tokensPerSecond * float64(time.Second))
		lastRefillAt = time.Unix(0, lastRefillAt).Add(refillDuration).UnixNano()
	}

	allowed := tokens > 0
	if allowed {
		tokens--
	}

	_, err = tx.ExecContext(ctx,
		"UPDATE buckets SET tokens = ?, last_refill_at = ? WHERE key = ?",
		tokens, lastRefillAt, key,
	)
	if err != nil {
		return false, fmt.Errorf("update bucket: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit bucket: %w", err)
	}

	return allowed, nil
}
