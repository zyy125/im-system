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

	first := &model.Message{
		MsgID:          "m1",
		ConversationID: 1,
		Seq:            1,
		From:           1,
		SendTime:       1000,
		Content:        "hello",
	}
	assert.NoError(t, repo.Create(ctx, first))
	assert.NotZero(t, first.ID)

	second := &model.Message{
		MsgID:          "m1",
		ConversationID: 1,
		Seq:            1,
		From:           1,
		SendTime:       2000,
		Content:        "changed",
	}
	assert.NoError(t, repo.Create(ctx, second))
	assert.Equal(t, first.ID, second.ID)
	assert.Equal(t, "hello", second.Content)

	var count int64
	assert.NoError(t, db.Model(&model.Message{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestMessageRepo_QueryMethods(t *testing.T) {
	db := newTestDB(t)
	repo := NewMessageRepo(db)
	ctx := context.Background()

	messages := []model.Message{
		{MsgID: "m1", ConversationID: 1, Seq: 1, From: 1, SendTime: 1000, Content: "a"},
		{MsgID: "m2", ConversationID: 1, Seq: 2, From: 2, SendTime: 2000, Content: "b"},
		{MsgID: "m3", ConversationID: 1, Seq: 3, From: 2, SendTime: 3000, Content: "c"},
	}
	for i := range messages {
		assert.NoError(t, db.Create(&messages[i]).Error)
	}

	rangeMessages, hasMore, err := repo.ListConversationRangeAfterSeq(ctx, 1, messages[0].Seq, messages[2].Seq, 1)
	assert.NoError(t, err)
	assert.True(t, hasMore)
	assert.Len(t, rangeMessages, 1)
	assert.Equal(t, "m2", rangeMessages[0].MsgID)

	rangeMessages, hasMore, err = repo.ListConversationRangeAfterSeq(ctx, 1, 0, messages[2].Seq, 10)
	assert.NoError(t, err)
	assert.False(t, hasMore)
	assert.Len(t, rangeMessages, 3)
	assert.Equal(t, []string{"m1", "m2", "m3"}, []string{rangeMessages[0].MsgID, rangeMessages[1].MsgID, rangeMessages[2].MsgID})

	history, hasMore, err := repo.ListConversationHistory(ctx, 1, 2, 0, 0)
	assert.NoError(t, err)
	assert.True(t, hasMore)
	assert.Equal(t, []string{"m2", "m3"}, []string{history[0].MsgID, history[1].MsgID})

	history, hasMore, err = repo.ListConversationHistory(ctx, 1, 2, messages[1].Seq, 0)
	assert.NoError(t, err)
	assert.False(t, hasMore)
	assert.Equal(t, []string{"m1"}, []string{history[0].MsgID})

	count, err := repo.CountUnreadByConversation(ctx, 1, 1, messages[1].Seq)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)

	latest, err := repo.GetLatestByConversation(ctx, 1)
	assert.NoError(t, err)
	assert.Equal(t, "m3", latest.MsgID)
}
