package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/model"
)

func TestMessageService_SaveMessageValidation(t *testing.T) {
	service := NewMessageService(&stubMessageRepo{}, &stubConversationRepo{}, nil)
	ctx := context.Background()

	_, err := service.SaveMessage(ctx, &model.ChatMessage{ConversationID: "1"})
	assert.Error(t, err)
	assert.Equal(t, apperr.CodeMessageIDRequired, apperr.CodeOf(err))

	_, err = service.SaveMessage(ctx, &model.ChatMessage{MsgID: "m1"})
	assert.Error(t, err)
	assert.Equal(t, apperr.CodeMessageConversationRequired, apperr.CodeOf(err))
}

func TestMessageService_SaveMessageUpdatesConversationState(t *testing.T) {
	ctx := context.Background()
	msgRepo := &stubMessageRepo{}
	conversationRepo := &stubConversationRepo{}
	service := NewMessageService(msgRepo, conversationRepo, &stubMessageTxManager{
		messageRepo:      msgRepo,
		conversationRepo: conversationRepo,
	})

	var ensured [][2]uint64
	var deliveredConversationID uint64
	var deliveredUserID uint64
	var deliveredSeq uint64
	var visibleOps [][3]uint64

	msgRepo.createFn = func(ctx context.Context, msg *model.ChatMessage) error {
		msg.ID = 101
		return nil
	}
	conversationRepo.ensureMemberFn = func(ctx context.Context, conversationID, userID uint64) error {
		ensured = append(ensured, [2]uint64{conversationID, userID})
		return nil
	}
	conversationRepo.updateLastDeliveredFn = func(ctx context.Context, conversationID, userID, msgSeq uint64) error {
		deliveredConversationID = conversationID
		deliveredUserID = userID
		deliveredSeq = msgSeq
		return nil
	}
	conversationRepo.setVisibleFn = func(ctx context.Context, conversationID, userID uint64, visible bool) error {
		var flag uint64
		if visible {
			flag = 1
		}
		visibleOps = append(visibleOps, [3]uint64{conversationID, userID, flag})
		return nil
	}

	msg := &model.ChatMessage{
		MsgID:          "m1",
		ConversationID: "12",
		From:           9,
		To:             10,
		Content:        "hello",
	}

	saved, err := service.SaveMessage(ctx, msg)
	assert.NoError(t, err)
	assert.NotZero(t, msg.SendTime)
	assert.Equal(t, uint64(101), saved.ID)
	assert.Equal(t, [][2]uint64{{12, 9}, {12, 10}}, ensured)
	assert.Equal(t, uint64(12), deliveredConversationID)
	assert.Equal(t, uint64(10), deliveredUserID)
	assert.Equal(t, uint64(101), deliveredSeq)
	assert.Len(t, visibleOps, 2)
}

func TestMessageService_ListHistoryDelegates(t *testing.T) {
	ctx := context.Background()
	expected := []model.ChatMessage{{MsgID: "m1"}, {MsgID: "m2"}}
	service := NewMessageService(
		&stubMessageRepo{
			listBetweenFn: func(ctx context.Context, userID, peerID uint64, limit int, beforeID uint64) ([]model.ChatMessage, bool, error) {
				assert.Equal(t, uint64(1), userID)
				assert.Equal(t, uint64(2), peerID)
				assert.Equal(t, 30, limit)
				assert.Equal(t, uint64(123), beforeID)
				return expected, true, nil
			},
		},
		&stubConversationRepo{},
		nil,
	)

	msgs, hasMore, err := service.ListHistory(ctx, 1, 2, 30, 123)
	assert.NoError(t, err)
	assert.Equal(t, expected, msgs)
	assert.True(t, hasMore)
}
