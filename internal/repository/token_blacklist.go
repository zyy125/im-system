package repository

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// internal/repository/token_blacklist.go
type TokenBlacklistRepo interface {
    IsBlacklisted(ctx context.Context, jti string) (bool, error)
    Blacklist(ctx context.Context, jti string) error
}

type RedisTokenBlacklist struct { Rdb *redis.Client }

func (r RedisTokenBlacklist) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
    return r.Rdb.SIsMember(ctx, "jwt:blacklist", jti).Result()
}
func (r RedisTokenBlacklist) Blacklist(ctx context.Context, jti string) error {
    return r.Rdb.SAdd(ctx, "jwt:blacklist", jti).Err()
}