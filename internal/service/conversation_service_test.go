package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/model"
	"gorm.io/gorm"
)

func TestConversationService_OpenDirectConversationRequiresFriendship(t *testing.T) {
	service := NewConversationService(
		&stubConversationRepo{},
		&stubMessageRepo{},
		&stubUserRepo{},
		&stubPresenceRepo{},
		&stubFriendRepo{
			areFriendsFn: func(ctx context.Context, userID, friendID uint64) (bool, error) {
				return false, nil
			},
		},
	)

	_, err := service.OpenDirectConversation(context.Background(), 1, 2)
	assert.Error(t, err)
	assert.Equal(t, apperr.CodeFriendNotFriends, apperr.CodeOf(err))
}

func TestConversationService_ListOfflineMessagesSortsAcrossConversations(t *testing.T) {
	service := NewConversationService(
		&stubConversationRepo{
			listMembersByUserFn: func(ctx context.Context, userID uint64) ([]model.ConversationMember, error) {
				return []model.ConversationMember{
					{ConversationID: 1, UserID: userID, LastReadMsgSeq: 1, LastDeliveredMsgSeq: 3},
					{ConversationID: 2, UserID: userID, LastReadMsgSeq: 0, LastDeliveredMsgSeq: 2},
				}, nil
			},
		},
		&stubMessageRepo{
			listConversationPendingForUserFn: func(ctx context.Context, conversationID string, userID, afterSeq, untilSeq uint64) ([]model.ChatMessage, error) {
				assert.Equal(t, uint64(9), userID)
				if conversationID == "1" {
					return []model.ChatMessage{
						{ID: 3, MsgID: "m3", ConversationID: "1", To: userID, SendTime: 3000},
						{ID: 2, MsgID: "m2", ConversationID: "1", To: userID, SendTime: 2000},
					}, nil
				}
				return []model.ChatMessage{
					{ID: 4, MsgID: "m4", ConversationID: "2", To: userID, SendTime: 2000},
					{ID: 1, MsgID: "m1", ConversationID: "2", To: userID, SendTime: 1000},
				}, nil
			},
		},
		&stubUserRepo{},
		&stubPresenceRepo{},
		&stubFriendRepo{},
	)

	msgs, err := service.ListOfflineMessages(context.Background(), 9)
	assert.NoError(t, err)
	assert.Len(t, msgs, 4)
	assert.Equal(t, []string{"m1", "m2", "m4", "m3"}, []string{msgs[0].MsgID, msgs[1].MsgID, msgs[2].MsgID, msgs[3].MsgID})
}

func TestConversationService_MarkReadAndListConversations(t *testing.T) {
	ctx := context.Background()

	t.Run("mark read updates sequence", func(t *testing.T) {
		var updatedConversationID uint64
		var updatedUserID uint64
		var updatedSeq uint64

		service := NewConversationService(
			&stubConversationRepo{
				getMemberFn: func(ctx context.Context, conversationID, userID uint64) (model.ConversationMember, error) {
					return model.ConversationMember{ConversationID: conversationID, UserID: userID, LastDeliveredMsgSeq: 55}, nil
				},
				updateLastReadFn: func(ctx context.Context, conversationID, userID, msgSeq uint64) error {
					updatedConversationID = conversationID
					updatedUserID = userID
					updatedSeq = msgSeq
					return nil
				},
			},
			&stubMessageRepo{
				getByConversationMsgIDFn: func(ctx context.Context, conversationID, msgID string) (model.ChatMessage, error) {
					return model.ChatMessage{ID: 55, MsgID: msgID, ConversationID: conversationID, To: 9}, nil
				},
			},
			&stubUserRepo{},
			&stubPresenceRepo{},
			&stubFriendRepo{},
		)

		err := service.MarkRead(ctx, 9, "12", "mid-1")
		assert.NoError(t, err)
		assert.Equal(t, uint64(12), updatedConversationID)
		assert.Equal(t, uint64(9), updatedUserID)
		assert.Equal(t, uint64(55), updatedSeq)
	})

	t.Run("mark read rejects self-sent message", func(t *testing.T) {
		service := NewConversationService(
			&stubConversationRepo{
				getMemberFn: func(ctx context.Context, conversationID, userID uint64) (model.ConversationMember, error) {
					return model.ConversationMember{ConversationID: conversationID, UserID: userID, LastDeliveredMsgSeq: 55}, nil
				},
			},
			&stubMessageRepo{
				getByConversationMsgIDFn: func(ctx context.Context, conversationID, msgID string) (model.ChatMessage, error) {
					return model.ChatMessage{ID: 55, MsgID: msgID, ConversationID: conversationID, To: 7}, nil
				},
			},
			&stubUserRepo{},
			&stubPresenceRepo{},
			&stubFriendRepo{},
		)

		err := service.MarkRead(ctx, 9, "12", "mid-1")
		assert.Error(t, err)
		assert.Equal(t, apperr.CodeMessageNotReadable, apperr.CodeOf(err))
	})

	t.Run("mark read rejects messages beyond delivered seq", func(t *testing.T) {
		service := NewConversationService(
			&stubConversationRepo{
				getMemberFn: func(ctx context.Context, conversationID, userID uint64) (model.ConversationMember, error) {
					return model.ConversationMember{ConversationID: conversationID, UserID: userID, LastDeliveredMsgSeq: 54}, nil
				},
			},
			&stubMessageRepo{
				getByConversationMsgIDFn: func(ctx context.Context, conversationID, msgID string) (model.ChatMessage, error) {
					return model.ChatMessage{ID: 55, MsgID: msgID, ConversationID: conversationID, To: 9}, nil
				},
			},
			&stubUserRepo{},
			&stubPresenceRepo{},
			&stubFriendRepo{},
		)

		err := service.MarkRead(ctx, 9, "12", "mid-1")
		assert.Error(t, err)
		assert.Equal(t, apperr.CodeMessageNotDelivered, apperr.CodeOf(err))
	})

	t.Run("list conversations builds summary", func(t *testing.T) {
		service := NewConversationService(
			&stubConversationRepo{
				listMembersByUserFn: func(ctx context.Context, userID uint64) ([]model.ConversationMember, error) {
					return []model.ConversationMember{{ConversationID: 1, UserID: userID, LastReadMsgSeq: 10}}, nil
				},
				listConversationsByUserFn: func(ctx context.Context, userID uint64) ([]model.Conversation, error) {
					key := "1:2"
					return []model.Conversation{{ID: 1, Type: model.ConversationTypeSingle, SingleKey: &key}}, nil
				},
				getMemberFn: func(ctx context.Context, conversationID, userID uint64) (model.ConversationMember, error) {
					return model.ConversationMember{ConversationID: conversationID, UserID: userID, LastReadMsgSeq: 10}, nil
				},
			},
			&stubMessageRepo{
				getLatestByConversationFn: func(ctx context.Context, conversationID string) (model.ChatMessage, error) {
					return model.ChatMessage{ID: 11, MsgID: "m11", ConversationID: conversationID, SendTime: 12345, Content: "hello"}, nil
				},
				countUnreadFn: func(ctx context.Context, conversationID string, userID uint64, afterSeq uint64) (int64, error) {
					return 3, nil
				},
			},
			&stubUserRepo{
				getByIDFn: func(ctx context.Context, id uint64) (model.User, error) {
					return model.User{ID: id, Username: "peer-user"}, nil
				},
			},
			&stubPresenceRepo{
				isOnlineFn: func(ctx context.Context, userID uint64) (bool, error) {
					return true, nil
				},
			},
			&stubFriendRepo{},
		)

		items, err := service.ListConversations(ctx, 1)
		assert.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, "peer-user", items[0].Name)
		assert.Equal(t, int64(3), items[0].UnreadCount)
		assert.NotNil(t, items[0].Peer)
		assert.Equal(t, uint64(2), items[0].Peer.ID)
		assert.NotNil(t, items[0].LastMessage)
		assert.Equal(t, "m11", items[0].LastMessage.MsgID)
	})

	t.Run("list conversations ignores not found latest message", func(t *testing.T) {
		service := NewConversationService(
			&stubConversationRepo{
				listMembersByUserFn: func(ctx context.Context, userID uint64) ([]model.ConversationMember, error) {
					return []model.ConversationMember{{ConversationID: 2, UserID: userID}}, nil
				},
				listConversationsByUserFn: func(ctx context.Context, userID uint64) ([]model.Conversation, error) {
					return []model.Conversation{{ID: 2, Type: model.ConversationTypeGroup, Name: "group"}}, nil
				},
				getMemberFn: func(ctx context.Context, conversationID, userID uint64) (model.ConversationMember, error) {
					return model.ConversationMember{ConversationID: conversationID, UserID: userID}, nil
				},
			},
			&stubMessageRepo{
				getLatestByConversationFn: func(ctx context.Context, conversationID string) (model.ChatMessage, error) {
					return model.ChatMessage{}, gorm.ErrRecordNotFound
				},
				countUnreadFn: func(ctx context.Context, conversationID string, userID uint64, afterSeq uint64) (int64, error) {
					return 0, nil
				},
			},
			&stubUserRepo{},
			&stubPresenceRepo{},
			&stubFriendRepo{},
		)

		items, err := service.ListConversations(ctx, 1)
		assert.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Nil(t, items[0].LastMessage)
	})
}
