package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryMessageStateRepo_InitNextSeqIfAbsentDoesNotOverwrite(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryMessageStateRepo()

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

func TestInMemoryMessageStateRepo_SeqInitLockTokenPreventsWrongRelease(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryMessageStateRepo()

	token, ok, err := repo.AcquireSeqInitLock(ctx, 1, time.Minute)
	require.NoError(t, err)
	require.True(t, ok)
	require.NotEmpty(t, token)

	err = repo.ReleaseSeqInitLock(ctx, 1, "wrong-token")
	require.NoError(t, err)

	_, ok, err = repo.AcquireSeqInitLock(ctx, 1, time.Minute)
	require.NoError(t, err)
	assert.False(t, ok)

	err = repo.ReleaseSeqInitLock(ctx, 1, token)
	require.NoError(t, err)

	_, ok, err = repo.AcquireSeqInitLock(ctx, 1, time.Minute)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestInMemoryMessageStateRepo_NextSeqMonotonic(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryMessageStateRepo()

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
