package repository

import (
	"context"
	"sync"
	"time"
)

type inMemoryRefreshSession struct {
	userID           uint64
	refreshTokenHash string
	expiresAt        time.Time
}

type inMemoryRefreshSessionRepo struct {
	mu       sync.Mutex
	sessions map[string]inMemoryRefreshSession
}

var _ RefreshSessionRepo = (*inMemoryRefreshSessionRepo)(nil)

func NewInMemoryRefreshSessionRepo() RefreshSessionRepo {
	return &inMemoryRefreshSessionRepo{
		sessions: make(map[string]inMemoryRefreshSession),
	}
}

func (r *inMemoryRefreshSessionRepo) Create(_ context.Context, sessionID string, userID uint64, refreshTokenHash string, ttl time.Duration) error {
	if sessionID == "" || userID == 0 || refreshTokenHash == "" || ttl <= 0 {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[sessionID] = inMemoryRefreshSession{
		userID:           userID,
		refreshTokenHash: refreshTokenHash,
		expiresAt:        time.Now().Add(ttl),
	}
	return nil
}

func (r *inMemoryRefreshSessionRepo) Rotate(_ context.Context, sessionID, oldRefreshTokenHash, newRefreshTokenHash string, ttl time.Duration) (uint64, bool, error) {
	if sessionID == "" || oldRefreshTokenHash == "" || newRefreshTokenHash == "" || ttl <= 0 {
		return 0, false, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessions[sessionID]
	if !ok || session.expiresAt.Before(time.Now()) {
		delete(r.sessions, sessionID)
		return 0, false, nil
	}
	if session.refreshTokenHash != oldRefreshTokenHash {
		return 0, false, nil
	}

	session.refreshTokenHash = newRefreshTokenHash
	session.expiresAt = time.Now().Add(ttl)
	r.sessions[sessionID] = session
	return session.userID, true, nil
}

func (r *inMemoryRefreshSessionRepo) Delete(_ context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, sessionID)
	return nil
}
