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
	Rdb *redis.Client
}

func NewTokenBlacklistRepo(rdb *redis.Client) *tokenBlacklistRepo {
	return &tokenBlacklistRepo{Rdb: rdb}
}

func (r *tokenBlacklistRepo) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	if jti == "" {
		return false, nil
	}
	count, err := r.Rdb.Exists(ctx, blacklistKey(jti)).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *tokenBlacklistRepo) Blacklist(ctx context.Context, jti string, ttl time.Duration) error {
	if jti == "" || ttl <= 0 {
		return nil
	}
	return r.Rdb.Set(ctx, blacklistKey(jti), "1", ttl).Err()
}

func blacklistKey(jti string) string {
	return fmt.Sprintf("jwt:blacklist:%s", jti)
}
