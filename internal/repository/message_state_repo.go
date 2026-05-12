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
	rdb *redis.Client
}

const releaseSeqInitLockScript = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0
`

func NewMessageStateRepo(rdb *redis.Client) MessageStateRepo {
	return &messageStateRepo{rdb: rdb}
}

func (r *messageStateRepo) HasNextSeq(ctx context.Context, conversationID uint64) (bool, error) {
	count, err := r.rdb.Exists(ctx, nextSeqKey(conversationID)).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *messageStateRepo) InitNextSeqIfAbsent(ctx context.Context, conversationID, seq uint64) (bool, error) {
	result, err := r.rdb.SetArgs(ctx, nextSeqKey(conversationID), strconv.FormatUint(seq, 10), redis.SetArgs{
		Mode: "NX",
	}).Result()
	if err == redis.Nil {
		return false, nil
	}
	return result == "OK", err
}

func (r *messageStateRepo) NextSeq(ctx context.Context, conversationID uint64) (uint64, error) {
	return r.rdb.Incr(ctx, nextSeqKey(conversationID)).Uint64()
}

func (r *messageStateRepo) AcquireSeqInitLock(ctx context.Context, conversationID uint64, ttl time.Duration) (string, bool, error) {
	token := newLockToken()
	result, err := r.rdb.SetArgs(ctx, seqInitLockKey(conversationID), token, redis.SetArgs{
		Mode: "NX",
		TTL:  ttl,
	}).Result()

	if err == redis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return token, result == "OK", nil
}

func (r *messageStateRepo) ReleaseSeqInitLock(ctx context.Context, conversationID uint64, token string) error {
	if token == "" {
		return nil
	}
	return r.rdb.Eval(ctx, releaseSeqInitLockScript, []string{seqInitLockKey(conversationID)}, token).Err()
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
