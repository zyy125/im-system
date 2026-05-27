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
	rdb *redis.Client
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
	return &refreshSessionRepo{rdb: rdb}
}

func (r *refreshSessionRepo) Create(ctx context.Context, sessionID string, userID uint64, refreshTokenHash string, ttl time.Duration) error {
	if sessionID == "" || userID == 0 || refreshTokenHash == "" || ttl <= 0 {
		return nil
	}

	pipe := r.rdb.TxPipeline()
	pipe.HSet(ctx, refreshSessionKey(sessionID), map[string]any{
		"user_id":            strconv.FormatUint(userID, 10),
		"refresh_token_hash": refreshTokenHash,
	})
	pipe.Expire(ctx, refreshSessionKey(sessionID), ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func (r *refreshSessionRepo) Rotate(ctx context.Context, sessionID, oldRefreshTokenHash, newRefreshTokenHash string, ttl time.Duration) (uint64, bool, error) {
	if sessionID == "" || oldRefreshTokenHash == "" || newRefreshTokenHash == "" || ttl <= 0 {
		return 0, false, nil
	}

	result, err := r.rdb.Eval(
		ctx,
		rotateRefreshSessionScript,
		[]string{refreshSessionKey(sessionID)},
		oldRefreshTokenHash,
		newRefreshTokenHash,
		strconv.FormatInt(int64(ttl/time.Second), 10),
	).Result()
	if err != nil {
		return 0, false, err
	}

	rows, ok := result.([]interface{})
	if !ok || len(rows) != 2 {
		return 0, false, nil
	}

	status, ok := rows[0].(int64)
	if !ok || status != 1 {
		return 0, false, nil
	}
	userIDRaw, ok := rows[1].(string)
	if !ok || userIDRaw == "" {
		return 0, false, nil
	}
	userID, err := strconv.ParseUint(userIDRaw, 10, 64)
	if err != nil {
		return 0, false, err
	}
	if userID == 0 {
		return 0, false, nil
	}
	return userID, true, nil
}

func (r *refreshSessionRepo) Delete(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	return r.rdb.Del(ctx, refreshSessionKey(sessionID)).Err()
}

func refreshSessionKey(sessionID string) string {
	return fmt.Sprintf("auth:session:%s", sessionID)
}
