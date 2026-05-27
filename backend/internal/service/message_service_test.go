package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/model"
)

func TestMessageService_ListHistoryDelegates(t *testing.T) {
	ctx := context.Background()
	expected := []model.Message{{MsgID: "m1"}, {MsgID: "m2"}}
	service := NewMessageService(
		&stubMessageRepo{
			listConversationHistoryFn: func(ctx context.Context, conversationID uint64, limit int, beforeSeq, afterSeq uint64) ([]model.Message, bool, error) {
				assert.Equal(t, uint64(2), conversationID)
				assert.Equal(t, 30, limit)
				assert.Equal(t, uint64(123), beforeSeq)
				assert.Equal(t, uint64(8), afterSeq)
				return expected, true, nil
			},
		},
		&stubConversationRepo{
			getByIDFn: func(ctx context.Context, conversationID uint64) (model.Conversation, error) {
				return model.Conversation{ID: conversationID, Status: model.ConversationStatusActive}, nil
			},
			getMemberFn: func(ctx context.Context, conversationID, userID uint64) (model.ConversationMember, error) {
				return model.ConversationMember{
					ConversationID: conversationID,
					UserID:         userID,
					Status:         model.ConversationMemberStatusActive,
					JoinedMsgSeq:   8,
				}, nil
			},
		},
	)

	msgs, hasMore, err := service.ListConversationHistory(ctx, 1, 2, 30, 123)
	assert.NoError(t, err)
	assert.Equal(t, expected, msgs)
	assert.True(t, hasMore)
}

func TestMessageService_MarkDeliveredRejectsUndeliveredSeq(t *testing.T) {
	ctx := context.Background()
	service := NewMessageService(
		&stubMessageRepo{
			getMaxSeqByConversationFn: func(ctx context.Context, conversationID uint64) (uint64, error) {
				return 5, nil
			},
		},
		&stubConversationRepo{
			getByIDFn: func(ctx context.Context, conversationID uint64) (model.Conversation, error) {
				return model.Conversation{ID: conversationID, Status: model.ConversationStatusActive}, nil
			},
			getMemberFn: func(ctx context.Context, conversationID, userID uint64) (model.ConversationMember, error) {
				return model.ConversationMember{
					ConversationID:  conversationID,
					UserID:          userID,
					Status:          model.ConversationMemberStatusActive,
					LastAckedMsgSeq: 3,
				}, nil
			},
		},
	)

	_, err := service.MarkDelivered(ctx, 9, 12, 6)
	assert.Error(t, err)
	assert.Equal(t, apperr.CodeMessageNotDelivered, apperr.CodeOf(err))
}

func TestMessageService_MarkDeliveredUpdatesSeq(t *testing.T) {
	ctx := context.Background()
	var deliveredSeq uint64
	service := NewMessageService(
		&stubMessageRepo{
			getMaxSeqByConversationFn: func(ctx context.Context, conversationID uint64) (uint64, error) {
				return 8, nil
			},
		},
		&stubConversationRepo{
			getByIDFn: func(ctx context.Context, conversationID uint64) (model.Conversation, error) {
				return model.Conversation{ID: conversationID, Status: model.ConversationStatusActive}, nil
			},
			getMemberFn: func(ctx context.Context, conversationID, userID uint64) (model.ConversationMember, error) {
				return model.ConversationMember{
					ConversationID:  conversationID,
					UserID:          userID,
					Status:          model.ConversationMemberStatusActive,
					LastAckedMsgSeq: 3,
				}, nil
			},
			updateLastAckedFn: func(ctx context.Context, conversationID, userID, msgSeq uint64) error {
				deliveredSeq = msgSeq
				return nil
			},
			listActiveMembersFn: func(ctx context.Context, conversationID uint64) ([]model.ConversationMember, error) {
				return []model.ConversationMember{
					{ConversationID: conversationID, UserID: 9, Status: model.ConversationMemberStatusActive},
					{ConversationID: conversationID, UserID: 10, Status: model.ConversationMemberStatusActive},
				}, nil
			},
		},
	)

	recipients, err := service.MarkDelivered(ctx, 9, 12, 7)
	assert.NoError(t, err)
	assert.Equal(t, uint64(7), deliveredSeq)
	assert.Equal(t, []uint64{9, 10}, recipients)
}

func TestMessageService_MarkDeliveredIgnoresDuplicateOrLowerSeq(t *testing.T) {
	ctx := context.Background()
	var updateCalled bool
	service := NewMessageService(
		&stubMessageRepo{},
		&stubConversationRepo{
			getByIDFn: func(ctx context.Context, conversationID uint64) (model.Conversation, error) {
				return model.Conversation{ID: conversationID, Status: model.ConversationStatusActive}, nil
			},
			getMemberFn: func(ctx context.Context, conversationID, userID uint64) (model.ConversationMember, error) {
				return model.ConversationMember{
					ConversationID:  conversationID,
					UserID:          userID,
					Status:          model.ConversationMemberStatusActive,
					LastAckedMsgSeq: 7,
				}, nil
			},
			updateLastAckedFn: func(ctx context.Context, conversationID, userID, msgSeq uint64) error {
				updateCalled = true
				return nil
			},
			listActiveMembersFn: func(ctx context.Context, conversationID uint64) ([]model.ConversationMember, error) {
				return []model.ConversationMember{{ConversationID: conversationID, UserID: 9, Status: model.ConversationMemberStatusActive}}, nil
			},
		},
	)

	recipients, err := service.MarkDelivered(ctx, 9, 12, 6)
	assert.NoError(t, err)
	assert.False(t, updateCalled)
	assert.Equal(t, []uint64{9}, recipients)
}
