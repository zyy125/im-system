package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RefreshSessionRepo interface {
	Create(ctx context.Context, sessionID string, userID uint64, refreshTokenHash string, ttl time.Duration) error
	Rotate(ctx context.Context, sessionID, oldRefreshTokenHash, newRefreshTokenHash string, ttl time.Duration) (uint64, bool, error)
	Delete(ctx context.Context, sessionID string) error
}

type refreshSessionRepo struct {
	rdb     *redis.Client
	metrics RedisMetricsRecorder
}

const rotateRefreshSessionScript = `
local current = redis.call("HGET", KEYS[1], "refresh_token_hash")
if not current then
	return {0, ""}
end
if current ~= ARGV[1] then
	return {-1, ""}
end
local user_id = redis.call("HGET", KEYS[1], "user_id")
if not user_id then
	return {0, ""}
end
redis.call("HSET", KEYS[1], "refresh_token_hash", ARGV[2])
redis.call("EXPIRE", KEYS[1], tonumber(ARGV[3]))
return {1, user_id}
`

var _ RefreshSessionRepo = (*refreshSessionRepo)(nil)

func NewRefreshSessionRepo(rdb *redis.Client) RefreshSessionRepo {
	return NewRefreshSessionRepoWithMetrics(rdb, nil)
}

func NewRefreshSessionRepoWithMetrics(rdb *redis.Client, metrics RedisMetricsRecorder) RefreshSessionRepo {
	return &refreshSessionRepo{rdb: rdb, metrics: withRedisMetrics(metrics)}
}

func (r *refreshSessionRepo) Create(ctx context.Context, sessionID string, userID uint64, refreshTokenHash string, ttl time.Duration) error {
	if sessionID == "" || userID == 0 || refreshTokenHash == "" || ttl <= 0 {
		return nil
	}

	start := time.Now()
	pipe := r.rdb.TxPipeline()
	pipe.HSet(ctx, refreshSessionKey(sessionID), map[string]any{
		"user_id":            strconv.FormatUint(userID, 10),
		"refresh_token_hash": refreshTokenHash,
	})
	pipe.Expire(ctx, refreshSessionKey(sessionID), ttl)
	_, err := pipe.Exec(ctx)
	r.observe("tx_pipeline_hset_expire", resultFromErr(err), time.Since(start))
	return err
}

func (r *refreshSessionRepo) Rotate(ctx context.Context, sessionID, oldRefreshTokenHash, newRefreshTokenHash string, ttl time.Duration) (uint64, bool, error) {
	if sessionID == "" || oldRefreshTokenHash == "" || newRefreshTokenHash == "" || ttl <= 0 {
		return 0, false, nil
	}

	start := time.Now()
	result, err := r.rdb.Eval(
		ctx,
		rotateRefreshSessionScript,
		[]string{refreshSessionKey(sessionID)},
		oldRefreshTokenHash,
		newRefreshTokenHash,
		strconv.FormatInt(int64(ttl/time.Second), 10),
	).Result()
	if err != nil {
		r.observe("eval", "error", time.Since(start))
		return 0, false, err
	}

	rows, ok := result.([]interface{})
	if !ok || len(rows) != 2 {
		r.observe("eval", "error", time.Since(start))
		return 0, false, nil
	}

	status, ok := rows[0].(int64)
	if !ok {
		r.observe("eval", "error", time.Since(start))
		return 0, false, nil
	}
	if status == -1 {
		r.observe("eval", "conflict", time.Since(start))
		return 0, false, nil
	}
	if status != 1 {
		r.observe("eval", "miss", time.Since(start))
		return 0, false, nil
	}
	userIDRaw, ok := rows[1].(string)
	if !ok || userIDRaw == "" {
		r.observe("eval", "error", time.Since(start))
		return 0, false, nil
	}
	userID, err := strconv.ParseUint(userIDRaw, 10, 64)
	if err != nil {
		r.observe("eval", "error", time.Since(start))
		return 0, false, err
	}
	if userID == 0 {
		r.observe("eval", "miss", time.Since(start))
		return 0, false, nil
	}
	r.observe("eval", "ok", time.Since(start))
	return userID, true, nil
}

func (r *refreshSessionRepo) Delete(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	start := time.Now()
	err := r.rdb.Del(ctx, refreshSessionKey(sessionID)).Err()
	r.observe("del", resultFromErr(err), time.Since(start))
	return err
}

func refreshSessionKey(sessionID string) string {
	return fmt.Sprintf("auth:session:%s", sessionID)
}

func (r *refreshSessionRepo) observe(op, result string, duration time.Duration) {
	if r == nil || r.metrics == nil {
		return
	}
	r.metrics.ObserveOperation("refresh_session", op, result, duration)
}
