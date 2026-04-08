package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFriendRepo_AddPairAndRemovePair(t *testing.T) {
	db := newTestDB(t)
	repo := NewFriendRepo(db)
	ctx := context.Background()

	assert.NoError(t, repo.AddPair(ctx, 1, 2))
	assert.NoError(t, repo.AddPair(ctx, 1, 2))

	ok, err := repo.AreFriends(ctx, 1, 2)
	assert.NoError(t, err)
	assert.True(t, ok)

	ids, err := repo.ListFriendIDs(ctx, 1)
	assert.NoError(t, err)
	assert.Equal(t, []uint64{2}, ids)

	assert.NoError(t, repo.RemovePair(ctx, 1, 2))

	ok, err = repo.AreFriends(ctx, 1, 2)
	assert.NoError(t, err)
	assert.False(t, ok)
}
