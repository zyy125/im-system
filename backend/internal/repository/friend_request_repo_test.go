package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zyy125/im-system/internal/model"
)

func TestFriendRequestRepo_FindListAndResolve(t *testing.T) {
	db := newTestDB(t)
	repo := NewFriendRequestRepo(db)
	ctx := context.Background()

	requests := []model.FriendRequest{
		{RequesterID: 1, ReceiverID: 2, Status: model.FriendRequestPending, Message: "old"},
		{RequesterID: 1, ReceiverID: 2, Status: model.FriendRequestPending, Message: "new"},
		{RequesterID: 2, ReceiverID: 1, Status: model.FriendRequestPending, Message: "reverse"},
		{RequesterID: 3, ReceiverID: 2, Status: model.FriendRequestRejected, Message: "ignored"},
	}
	for i := range requests {
		assert.NoError(t, db.Create(&requests[i]).Error)
	}

	found, err := repo.FindPendingBetween(ctx, 1, 2)
	assert.NoError(t, err)
	assert.Equal(t, "new", found.Message)

	incoming, err := repo.ListIncomingPending(ctx, 2)
	assert.NoError(t, err)
	assert.Len(t, incoming, 2)

	outgoing, err := repo.ListOutgoingPending(ctx, 1)
	assert.NoError(t, err)
	assert.Len(t, outgoing, 2)

	assert.NoError(t, repo.ResolvePendingBetween(ctx, 1, 2, model.FriendRequestAccepted))

	var pendingCount int64
	assert.NoError(t, db.Model(&model.FriendRequest{}).Where("status = ?", model.FriendRequestPending).Count(&pendingCount).Error)
	assert.Equal(t, int64(0), pendingCount)

	var accepted []model.FriendRequest
	assert.NoError(t, db.Where("status = ?", model.FriendRequestAccepted).Find(&accepted).Error)
	assert.Len(t, accepted, 3)
	for _, req := range accepted {
		assert.NotNil(t, req.HandledAt)
	}
}
