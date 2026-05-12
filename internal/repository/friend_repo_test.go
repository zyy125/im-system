package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zyy125/im-system/internal/model"
)

func TestFriendRepo_AddPairAndRemovePair(t *testing.T) {
	db := newTestDB(t)
	repo := NewFriendRepo(db)
	ctx := context.Background()

	assert.NoError(t, db.Create(&model.User{ID: 1, Username: "alice", Password: "x"}).Error)
	assert.NoError(t, db.Create(&model.User{ID: 2, Username: "bob", Password: "x"}).Error)

	assert.NoError(t, repo.AddPair(ctx, 1, 2, 10))
	assert.NoError(t, repo.AddPair(ctx, 1, 2, 11))

	ok, err := repo.AreFriends(ctx, 1, 2)
	assert.NoError(t, err)
	assert.True(t, ok)

	items, err := repo.ListFriendProfiles(ctx, 1)
	assert.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, uint64(2), items[0].UserID)
	assert.Equal(t, "bob", items[0].Username)
	assert.Equal(t, uint64(11), items[0].ConversationID)

	ids, err := repo.ListFriendIDs(ctx, 1)
	assert.NoError(t, err)
	assert.Equal(t, []uint64{2}, ids)

	assert.NoError(t, repo.RemovePair(ctx, 1, 2))

	ok, err = repo.AreFriends(ctx, 1, 2)
	assert.NoError(t, err)
	assert.False(t, ok)
}
