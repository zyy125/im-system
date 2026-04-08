package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/model"
)

func TestConversationRepo_GetOrCreateSingle(t *testing.T) {
	db := newTestDB(t)
	repo := NewConversationRepo(db)
	ctx := context.Background()

	conv, err := repo.GetOrCreateSingle(ctx, 2, 1)
	assert.NoError(t, err)
	assert.Equal(t, model.ConversationTypeSingle, conv.Type)
	assert.Equal(t, "1:2", conv.SingleKeyValue())

	again, err := repo.GetOrCreateSingle(ctx, 1, 2)
	assert.NoError(t, err)
	assert.Equal(t, conv.ID, again.ID)

	var members []model.ConversationMember
	err = db.Where("conversation_id = ?", conv.ID).Order("user_id ASC").Find(&members).Error
	assert.NoError(t, err)
	assert.Len(t, members, 2)
	assert.Equal(t, uint64(1), members[0].UserID)
	assert.Equal(t, uint64(2), members[1].UserID)
}

func TestConversationRepo_ListConversationsByUserAndUpdateSeq(t *testing.T) {
	db := newTestDB(t)
	repo := NewConversationRepo(db)
	ctx := context.Background()

	key := "1:2"
	conv1 := model.Conversation{Type: model.ConversationTypeSingle, SingleKey: &key}
	conv2 := model.Conversation{Type: model.ConversationTypeGroup, Name: "hidden"}
	assert.NoError(t, db.Create(&conv1).Error)
	assert.NoError(t, db.Create(&conv2).Error)
	assert.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv1.ID, UserID: 1, Visible: true}).Error)
	assert.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv2.ID, UserID: 1, LastDeliveredMsgSeq: 5}).Error)
	assert.NoError(t, repo.SetVisible(ctx, conv2.ID, 1, false))

	items, err := repo.ListConversationsByUser(ctx, 1)
	assert.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, conv1.ID, items[0].ID)

	assert.NoError(t, repo.UpdateLastDeliveredMsgSeq(ctx, conv2.ID, 1, 10))
	assert.NoError(t, repo.UpdateLastDeliveredMsgSeq(ctx, conv2.ID, 1, 8))

	member, err := repo.GetMember(ctx, conv2.ID, 1)
	assert.NoError(t, err)
	assert.Equal(t, uint64(10), member.LastDeliveredMsgSeq)

	err = repo.SetVisible(ctx, 999, 1, true)
	assert.Error(t, err)
	assert.Equal(t, apperr.CodeConversationMemberNotFound, apperr.CodeOf(err))
}

func TestConversationRepo_GroupConversationAllowsMultipleNullSingleKeys(t *testing.T) {
	db := newTestDB(t)

	conv1 := model.Conversation{Type: model.ConversationTypeGroup, Name: "g1"}
	conv2 := model.Conversation{Type: model.ConversationTypeGroup, Name: "g2"}

	assert.NoError(t, db.Create(&conv1).Error)
	assert.NoError(t, db.Create(&conv2).Error)
	assert.NotZero(t, conv1.ID)
	assert.NotZero(t, conv2.ID)
}
