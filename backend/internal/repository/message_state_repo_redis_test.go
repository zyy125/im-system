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

func newTestRedisStateRepo(t *testing.T) MessageStateRepo {
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
	return NewMessageStateRepo(client)
}

func TestMessageStateRepoRedis_InitNextSeqIfAbsentDoesNotOverwrite(t *testing.T) {
	ctx := context.Background()
	repo := newTestRedisStateRepo(t)

	ok, err := repo.InitNextSeqIfAbsent(ctx, 1, 10)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = repo.InitNextSeqIfAbsent(ctx, 1, 3)
	require.NoError(t, err)
	assert.False(t, ok)

	next, err := repo.NextSeq(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, uint64(11), next)
}

func TestMessageStateRepoRedis_SeqInitLockTokenPreventsWrongRelease(t *testing.T) {
	ctx := context.Background()
	repo := newTestRedisStateRepo(t)

	token, ok, err := repo.AcquireSeqInitLock(ctx, 1, time.Minute)
	require.NoError(t, err)
	require.True(t, ok)
	require.NotEmpty(t, token)

	require.NoError(t, repo.ReleaseSeqInitLock(ctx, 1, "wrong-token"))

	_, ok, err = repo.AcquireSeqInitLock(ctx, 1, time.Minute)
	require.NoError(t, err)
	assert.False(t, ok)

	require.NoError(t, repo.ReleaseSeqInitLock(ctx, 1, token))

	_, ok, err = repo.AcquireSeqInitLock(ctx, 1, time.Minute)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestMessageStateRepoRedis_NextSeqMonotonic(t *testing.T) {
	ctx := context.Background()
	repo := newTestRedisStateRepo(t)

	ok, err := repo.InitNextSeqIfAbsent(ctx, 1, 5)
	require.NoError(t, err)
	require.True(t, ok)

	first, err := repo.NextSeq(ctx, 1)
	require.NoError(t, err)
	second, err := repo.NextSeq(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, uint64(6), first)
	assert.Equal(t, uint64(7), second)
}
