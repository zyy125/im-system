package repository

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type PresenceRepo interface {
	SetOnline(ctx context.Context, userID uint64) error
	SetOffline(ctx context.Context, userID uint64) error
	IsOnline(ctx context.Context, userID uint64) (bool, error)
}

var _ PresenceRepo = (*presenceRepo)(nil)

type presenceRepo struct {
	rdb *redis.Client
}

func NewPresenceRepo(rdb *redis.Client) *presenceRepo {
	return &presenceRepo{rdb: rdb}
}

func (r *presenceRepo) SetOnline(ctx context.Context, userID uint64) error {
	return r.rdb.Set(ctx, presenceKey(userID), "1", 0).Err()
}

func (r *presenceRepo) SetOffline(ctx context.Context, userID uint64) error {
	return r.rdb.Del(ctx, presenceKey(userID)).Err()
}

func (r *presenceRepo) IsOnline(ctx context.Context, userID uint64) (bool, error) {
	res, err := r.rdb.Exists(ctx, presenceKey(userID)).Result()
	if err != nil {
		return false, err
	}
	return res > 0, nil
}

func presenceKey(userID uint64) string {
	return fmt.Sprintf("im:user:online:%d", userID)
}
