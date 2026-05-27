package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type PresenceRepo interface {
	SetOnline(ctx context.Context, userID uint64) error
	RefreshOnline(ctx context.Context, userID uint64) error
	SetOffline(ctx context.Context, userID uint64) error
	IsOnline(ctx context.Context, userID uint64) (bool, error)
	BatchGetOnline(ctx context.Context, userIDs []uint64) (map[uint64]bool, error)
}

var _ PresenceRepo = (*presenceRepo)(nil)

type presenceRepo struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewPresenceRepo(rdb *redis.Client, ttl time.Duration) *presenceRepo {
	return &presenceRepo{rdb: rdb, ttl: ttl}
}

func (r *presenceRepo) SetOnline(ctx context.Context, userID uint64) error {
	return r.rdb.Set(ctx, presenceKey(userID), "1", r.ttl).Err()
}

func (r *presenceRepo) RefreshOnline(ctx context.Context, userID uint64) error {
	return r.rdb.Expire(ctx, presenceKey(userID), r.ttl).Err()
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

func (r *presenceRepo) BatchGetOnline(ctx context.Context, userIDs []uint64) (map[uint64]bool, error) {
	status := make(map[uint64]bool, len(userIDs))
	if len(userIDs) == 0 {
		return status, nil
	}

	keys := make([]string, 0, len(userIDs))
	for _, userID := range userIDs {
		keys = append(keys, presenceKey(userID))
	}

	pipe := r.rdb.Pipeline()
	cmds := make([]*redis.IntCmd, 0, len(keys))
	for _, key := range keys {
		cmds = append(cmds, pipe.Exists(ctx, key))
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, err
	}

	for i, cmd := range cmds {
		exists, err := cmd.Result()
		if err != nil {
			return nil, err
		}
		status[userIDs[i]] = exists > 0
	}
	return status, nil
}

func presenceKey(userID uint64) string {
	return fmt.Sprintf("im:user:online:%d", userID)
}
