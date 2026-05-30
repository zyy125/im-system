package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type TokenBlacklistRepo interface {
	IsBlacklisted(ctx context.Context, jti string) (bool, error)
	Blacklist(ctx context.Context, jti string, ttl time.Duration) error
}

var _ TokenBlacklistRepo = (*tokenBlacklistRepo)(nil)

// internal/repository/token_blacklist.go
type tokenBlacklistRepo struct {
	Rdb     *redis.Client
	metrics RedisMetricsRecorder
}

func NewTokenBlacklistRepo(rdb *redis.Client) *tokenBlacklistRepo {
	return NewTokenBlacklistRepoWithMetrics(rdb, nil)
}

func NewTokenBlacklistRepoWithMetrics(rdb *redis.Client, metrics RedisMetricsRecorder) *tokenBlacklistRepo {
	return &tokenBlacklistRepo{Rdb: rdb, metrics: withRedisMetrics(metrics)}
}

func (r *tokenBlacklistRepo) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	if jti == "" {
		return false, nil
	}
	start := time.Now()
	count, err := r.Rdb.Exists(ctx, blacklistKey(jti)).Result()
	if err != nil {
		r.observe("exists", "error", time.Since(start))
		return false, err
	}
	result := "ok"
	if count == 0 {
		result = "miss"
	}
	r.observe("exists", result, time.Since(start))
	return count > 0, nil
}

func (r *tokenBlacklistRepo) Blacklist(ctx context.Context, jti string, ttl time.Duration) error {
	if jti == "" || ttl <= 0 {
		return nil
	}
	start := time.Now()
	err := r.Rdb.Set(ctx, blacklistKey(jti), "1", ttl).Err()
	r.observe("set", resultFromErr(err), time.Since(start))
	return err
}

func blacklistKey(jti string) string {
	return fmt.Sprintf("jwt:blacklist:%s", jti)
}

func (r *tokenBlacklistRepo) observe(op, result string, duration time.Duration) {
	if r == nil || r.metrics == nil {
		return
	}
	r.metrics.ObserveOperation("token_blacklist", op, result, duration)
}
