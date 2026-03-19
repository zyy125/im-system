package repository

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type TokenBlacklistRepo interface {
    IsBlacklisted(ctx context.Context, jti string) (bool, error)
    Blacklist(ctx context.Context, jti string) error
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
    return r.Rdb.SIsMember(ctx, "jwt:blacklist", jti).Result()
}
func (r *tokenBlacklistRepo) Blacklist(ctx context.Context, jti string) error {
    return r.Rdb.SAdd(ctx, "jwt:blacklist", jti).Err()
}

