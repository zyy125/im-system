package repository

import (
	"context"
	"sync"
	"time"
)

type inMemoryMessageStateRepo struct {
	mu           sync.Mutex
	nextSeq      map[uint64]uint64
	seqInitLocks map[uint64]inMemoryLock
}

type inMemoryLock struct {
	token     string
	expiresAt time.Time
}

var _ MessageStateRepo = (*inMemoryMessageStateRepo)(nil)

func NewInMemoryMessageStateRepo() MessageStateRepo {
	return &inMemoryMessageStateRepo{
		nextSeq:      make(map[uint64]uint64),
		seqInitLocks: make(map[uint64]inMemoryLock),
	}
}

func (r *inMemoryMessageStateRepo) HasNextSeq(_ context.Context, conversationID uint64) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.nextSeq[conversationID]
	return ok, nil
}

func (r *inMemoryMessageStateRepo) InitNextSeqIfAbsent(_ context.Context, conversationID, seq uint64) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.nextSeq[conversationID]; ok {
		return false, nil
	}
	r.nextSeq[conversationID] = seq
	return true, nil
}

func (r *inMemoryMessageStateRepo) NextSeq(_ context.Context, conversationID uint64) (uint64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextSeq[conversationID]++
	return r.nextSeq[conversationID], nil
}

func (r *inMemoryMessageStateRepo) AcquireSeqInitLock(_ context.Context, conversationID uint64, ttl time.Duration) (string, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	if lock, ok := r.seqInitLocks[conversationID]; ok && lock.expiresAt.After(now) {
		return "", false, nil
	}
	token := newLockToken()
	r.seqInitLocks[conversationID] = inMemoryLock{token: token, expiresAt: now.Add(ttl)}
	return token, true, nil
}

func (r *inMemoryMessageStateRepo) ReleaseSeqInitLock(_ context.Context, conversationID uint64, token string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	lock, ok := r.seqInitLocks[conversationID]
	if !ok || lock.token != token {
		return nil
	}
	delete(r.seqInitLocks, conversationID)
	return nil
}
