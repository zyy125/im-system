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
	rdb     *redis.Client
	ttl     time.Duration
	metrics RedisMetricsRecorder
}

func NewPresenceRepo(rdb *redis.Client, ttl time.Duration) *presenceRepo {
	return NewPresenceRepoWithMetrics(rdb, ttl, nil)
}

func NewPresenceRepoWithMetrics(rdb *redis.Client, ttl time.Duration, metrics RedisMetricsRecorder) *presenceRepo {
	return &presenceRepo{rdb: rdb, ttl: ttl, metrics: withRedisMetrics(metrics)}
}

func (r *presenceRepo) SetOnline(ctx context.Context, userID uint64) error {
	start := time.Now()
	err := r.rdb.Set(ctx, presenceKey(userID), "1", r.ttl).Err()
	r.observe("set", resultFromErr(err), time.Since(start))
	return err
}

func (r *presenceRepo) RefreshOnline(ctx context.Context, userID uint64) error {
	start := time.Now()
	err := r.rdb.Expire(ctx, presenceKey(userID), r.ttl).Err()
	r.observe("expire", resultFromErr(err), time.Since(start))
	return err
}

func (r *presenceRepo) SetOffline(ctx context.Context, userID uint64) error {
	start := time.Now()
	err := r.rdb.Del(ctx, presenceKey(userID)).Err()
	r.observe("del", resultFromErr(err), time.Since(start))
	return err
}

func (r *presenceRepo) IsOnline(ctx context.Context, userID uint64) (bool, error) {
	start := time.Now()
	res, err := r.rdb.Exists(ctx, presenceKey(userID)).Result()
	if err != nil {
		r.observe("exists", "error", time.Since(start))
		return false, err
	}
	result := "ok"
	if res == 0 {
		result = "miss"
	}
	r.observe("exists", result, time.Since(start))
	return res > 0, nil
}

func (r *presenceRepo) BatchGetOnline(ctx context.Context, userIDs []uint64) (map[uint64]bool, error) {
	start := time.Now()
	status := make(map[uint64]bool, len(userIDs))
	if len(userIDs) == 0 {
		r.observe("pipeline_exists", "ok", time.Since(start))
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
		r.observe("pipeline_exists", "error", time.Since(start))
		return nil, err
	}

	for i, cmd := range cmds {
		exists, err := cmd.Result()
		if err != nil {
			r.observe("pipeline_exists", "error", time.Since(start))
			return nil, err
		}
		status[userIDs[i]] = exists > 0
	}
	r.observe("pipeline_exists", "ok", time.Since(start))
	return status, nil
}

func (r *presenceRepo) observe(op, result string, duration time.Duration) {
	if r == nil || r.metrics == nil {
		return
	}
	r.metrics.ObserveOperation("presence", op, result, duration)
}

func resultFromErr(err error) string {
	if err != nil {
		return "error"
	}
	return "ok"
}

func presenceKey(userID uint64) string {
	return fmt.Sprintf("im:user:online:%d", userID)
}
