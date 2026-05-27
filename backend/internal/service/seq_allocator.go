package service

import (
	"context"
	"time"

	"github.com/zyy125/im-system/internal/repository"
)

const seqInitLockTTL = 2 * time.Second

type SeqAllocator interface {
	// Allocate 为某个会话分配下一个单调递增的业务 seq。
	Allocate(ctx context.Context, conversationID uint64) (uint64, error)
}

type redisSeqAllocator struct {
	messageRepo repository.MessageRepo
	stateRepo   repository.MessageStateRepo
}

func NewSeqAllocator(messageRepo repository.MessageRepo, stateRepo repository.MessageStateRepo) SeqAllocator {
	return &redisSeqAllocator{
		messageRepo: messageRepo,
		stateRepo:   stateRepo,
	}
}

func (a *redisSeqAllocator) Allocate(ctx context.Context, conversationID uint64) (uint64, error) {
	hasKey, err := a.stateRepo.HasNextSeq(ctx, conversationID)
	if err != nil {
		return 0, err
	}
	if !hasKey {
		if err := a.initializeNextSeq(ctx, conversationID); err != nil {
			return 0, err
		}
	}
	return a.stateRepo.NextSeq(ctx, conversationID)
}

func (a *redisSeqAllocator) initializeNextSeq(ctx context.Context, conversationID uint64) error {
	for {
		hasKey, err := a.stateRepo.HasNextSeq(ctx, conversationID)
		if err != nil {
			return err
		}
		if hasKey {
			return nil
		}
		token, locked, err := a.stateRepo.AcquireSeqInitLock(ctx, conversationID, seqInitLockTTL)
		if err != nil {
			return err
		}
		if !locked {
			// 只有极少数“next_seq 缺失”的恢复场景才会走到这里，短暂自旋即可。
			time.Sleep(10 * time.Millisecond)
			continue
		}
		defer func() {
			_ = a.stateRepo.ReleaseSeqInitLock(context.Background(), conversationID, token)
		}()

		// 新链路下消息先提交 DB 再对外可见，回源 DB 即可恢复当前会话的最大已分配 seq。
		maxSeq, err := a.messageRepo.GetMaxSeqByConversation(ctx, conversationID)
		if err != nil {
			return err
		}
		// 这里保存的是“当前已分配到哪里”，下一次真正发号时还会再执行一次 Incr。
		_, err = a.stateRepo.InitNextSeqIfAbsent(ctx, conversationID, maxSeq)
		return err
	}
}
