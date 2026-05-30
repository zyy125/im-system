package repository

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type MessageStateRepo interface {
	HasNextSeq(ctx context.Context, conversationID uint64) (bool, error)
	InitNextSeqIfAbsent(ctx context.Context, conversationID, seq uint64) (bool, error)
	NextSeq(ctx context.Context, conversationID uint64) (uint64, error)
	AcquireSeqInitLock(ctx context.Context, conversationID uint64, ttl time.Duration) (string, bool, error)
	ReleaseSeqInitLock(ctx context.Context, conversationID uint64, token string) error
}

type messageStateRepo struct {
	rdb     *redis.Client
	metrics RedisMetricsRecorder
}

const releaseSeqInitLockScript = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0
`

func NewMessageStateRepo(rdb *redis.Client) MessageStateRepo {
	return NewMessageStateRepoWithMetrics(rdb, nil)
}

func NewMessageStateRepoWithMetrics(rdb *redis.Client, metrics RedisMetricsRecorder) MessageStateRepo {
	return &messageStateRepo{rdb: rdb, metrics: withRedisMetrics(metrics)}
}

func (r *messageStateRepo) HasNextSeq(ctx context.Context, conversationID uint64) (bool, error) {
	start := time.Now()
	count, err := r.rdb.Exists(ctx, nextSeqKey(conversationID)).Result()
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

func (r *messageStateRepo) InitNextSeqIfAbsent(ctx context.Context, conversationID, seq uint64) (bool, error) {
	start := time.Now()
	result, err := r.rdb.SetArgs(ctx, nextSeqKey(conversationID), strconv.FormatUint(seq, 10), redis.SetArgs{
		Mode: "NX",
	}).Result()
	if err == redis.Nil {
		r.observe("set_nx", "conflict", time.Since(start))
		return false, nil
	}
	if err != nil {
		r.observe("set_nx", "error", time.Since(start))
		return result == "OK", err
	}
	status := "ok"
	if result != "OK" {
		status = "conflict"
	}
	r.observe("set_nx", status, time.Since(start))
	return result == "OK", err
}

func (r *messageStateRepo) NextSeq(ctx context.Context, conversationID uint64) (uint64, error) {
	start := time.Now()
	seq, err := r.rdb.Incr(ctx, nextSeqKey(conversationID)).Uint64()
	r.observe("incr", resultFromErr(err), time.Since(start))
	return seq, err
}

func (r *messageStateRepo) AcquireSeqInitLock(ctx context.Context, conversationID uint64, ttl time.Duration) (string, bool, error) {
	start := time.Now()
	token := newLockToken()
	result, err := r.rdb.SetArgs(ctx, seqInitLockKey(conversationID), token, redis.SetArgs{
		Mode: "NX",
		TTL:  ttl,
	}).Result()

	if err == redis.Nil {
		r.observe("set_nx", "conflict", time.Since(start))
		return "", false, nil
	}
	if err != nil {
		r.observe("set_nx", "error", time.Since(start))
		return "", false, err
	}
	status := "ok"
	if result != "OK" {
		status = "conflict"
	}
	r.observe("set_nx", status, time.Since(start))
	return token, result == "OK", nil
}

func (r *messageStateRepo) ReleaseSeqInitLock(ctx context.Context, conversationID uint64, token string) error {
	if token == "" {
		return nil
	}
	start := time.Now()
	err := r.rdb.Eval(ctx, releaseSeqInitLockScript, []string{seqInitLockKey(conversationID)}, token).Err()
	r.observe("eval", resultFromErr(err), time.Since(start))
	return err
}

func (r *messageStateRepo) observe(op, result string, duration time.Duration) {
	if r == nil || r.metrics == nil {
		return
	}
	r.metrics.ObserveOperation("message_state", op, result, duration)
}

func nextSeqKey(conversationID uint64) string {
	return fmt.Sprintf("im:conv:%d:next_seq", conversationID)
}

func seqInitLockKey(conversationID uint64) string {
	return fmt.Sprintf("im:conv:%d:seq_init_lock", conversationID)
}

func newLockToken() string {
	return fmt.Sprintf("%d:%d", time.Now().UnixNano(), rand.Uint64())
}
