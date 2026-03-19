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
	redisKey := fmt.Sprintf("im:user:online:%d", userID)
	return r.rdb.Set(ctx, redisKey, "1", 0).Err()
}

func (r *presenceRepo) SetOffline(ctx context.Context, userID uint64) error {
	redisKey := fmt.Sprintf("im:user:online:%d", userID)
	return r.rdb.Del(ctx, redisKey).Err()
}

func (r *presenceRepo) IsOnline(ctx context.Context, userID uint64) (bool, error) {
	redisKey := fmt.Sprintf("im:user:online:%d", userID)
	res, err := r.rdb.Exists(ctx, redisKey).Result() 
	if err != nil {
		return false, err
	}
	return res > 0, nil
}

