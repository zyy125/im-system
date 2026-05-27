package repository

import (
	"context"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRefreshSessionRepo(t *testing.T) RefreshSessionRepo {
	t.Helper()

	server, err := miniredis.Run()
	if err != nil {
		t.Skipf("miniredis unavailable in current sandbox: %v", err)
	}
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
		server.Close()
	})
	return NewRefreshSessionRepo(client)
}

func TestRefreshSessionRepoRedis_CreateRotateDelete(t *testing.T) {
	ctx := context.Background()
	repo := newTestRefreshSessionRepo(t)

	require.NoError(t, repo.Create(ctx, "session-1", 9, "old-hash", time.Hour))

	userID, ok, err := repo.Rotate(ctx, "session-1", "old-hash", "new-hash", 2*time.Hour)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, uint64(9), userID)

	_, ok, err = repo.Rotate(ctx, "session-1", "old-hash", "another-hash", 2*time.Hour)
	require.NoError(t, err)
	assert.False(t, ok)

	require.NoError(t, repo.Delete(ctx, "session-1"))

	_, ok, err = repo.Rotate(ctx, "session-1", "new-hash", "after-delete", 2*time.Hour)
	require.NoError(t, err)
	assert.False(t, ok)
}
