package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zyy125/im-system/internal/model"
)

func TestMessageRepo_CreateIsIdempotent(t *testing.T) {
	db := newTestDB(t)
	repo := NewMessageRepo(db)
	ctx := context.Background()

	first := &model.ChatMessage{
		MsgID:          "m1",
		ConversationID: "1",
		From:           1,
		To:             2,
		SendTime:       1000,
		Content:        "hello",
	}
	assert.NoError(t, repo.Create(ctx, first))
	assert.NotZero(t, first.ID)

	second := &model.ChatMessage{
		MsgID:          "m1",
		ConversationID: "1",
		From:           1,
		To:             2,
		SendTime:       2000,
		Content:        "changed",
	}
	assert.NoError(t, repo.Create(ctx, second))
	assert.Equal(t, first.ID, second.ID)
	assert.Equal(t, "hello", second.Content)

	var count int64
	assert.NoError(t, db.Model(&model.ChatMessage{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestMessageRepo_QueryMethods(t *testing.T) {
	db := newTestDB(t)
	repo := NewMessageRepo(db)
	ctx := context.Background()

	messages := []model.ChatMessage{
		{MsgID: "m1", ConversationID: "1", From: 1, To: 2, SendTime: 1000, Content: "a"},
		{MsgID: "m2", ConversationID: "1", From: 2, To: 1, SendTime: 2000, Content: "b"},
		{MsgID: "m3", ConversationID: "1", From: 2, To: 1, SendTime: 3000, Content: "c"},
	}
	for i := range messages {
		assert.NoError(t, db.Create(&messages[i]).Error)
	}

	pending, err := repo.ListConversationPending(ctx, "1", messages[0].ID, messages[2].ID)
	assert.NoError(t, err)
	assert.Len(t, pending, 2)
	assert.Equal(t, []string{"m2", "m3"}, []string{pending[0].MsgID, pending[1].MsgID})

	pendingForUser, err := repo.ListConversationPendingForUser(ctx, "1", 1, messages[0].ID, messages[2].ID)
	assert.NoError(t, err)
	assert.Len(t, pendingForUser, 2)
	assert.Equal(t, []string{"m2", "m3"}, []string{pendingForUser[0].MsgID, pendingForUser[1].MsgID})

	pendingForUser, err = repo.ListConversationPendingForUser(ctx, "1", 2, 0, messages[2].ID)
	assert.NoError(t, err)
	assert.Len(t, pendingForUser, 1)
	assert.Equal(t, "m1", pendingForUser[0].MsgID)

	history, hasMore, err := repo.ListBetween(ctx, 1, 2, 2, 0)
	assert.NoError(t, err)
	assert.True(t, hasMore)
	assert.Equal(t, []string{"m2", "m3"}, []string{history[0].MsgID, history[1].MsgID})

	history, hasMore, err = repo.ListBetween(ctx, 1, 2, 2, messages[1].ID)
	assert.NoError(t, err)
	assert.False(t, hasMore)
	assert.Equal(t, []string{"m1"}, []string{history[0].MsgID})

	count, err := repo.CountUnreadByConversation(ctx, "1", 1, messages[1].ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)

	latest, err := repo.GetLatestByConversation(ctx, "1")
	assert.NoError(t, err)
	assert.Equal(t, "m3", latest.MsgID)
}
