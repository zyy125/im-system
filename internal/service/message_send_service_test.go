package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zyy125/im-system/internal/model"
)

type fixedSeqAllocator uint64

func (a fixedSeqAllocator) Allocate(context.Context, uint64) (uint64, error) {
	return uint64(a), nil
}

func TestMessageSendService_SendTextMessagePersistsMessage(t *testing.T) {
	ctx := context.Background()
	var savedMsg model.Message
	var visibleUsers []uint64
	var lastSentConversationID uint64
	var lastSentUserID uint64
	var lastSentSeq uint64

	service := NewMessageSendService(
		&stubMessageTxManager{
			messageRepo: &stubMessageRepo{
				createFn: func(ctx context.Context, msg *model.Message) error {
					msg.ID = 101
					savedMsg = *msg
					return nil
				},
			},
			conversationRepo: &stubConversationRepo{
				getByIDFn: func(ctx context.Context, conversationID uint64) (model.Conversation, error) {
					return model.Conversation{ID: conversationID, Status: model.ConversationStatusActive}, nil
				},
				getMemberFn: func(ctx context.Context, conversationID, userID uint64) (model.ConversationMember, error) {
					return model.ConversationMember{ConversationID: conversationID, UserID: userID, Status: model.ConversationMemberStatusActive}, nil
				},
				listActiveMembersFn: func(ctx context.Context, conversationID uint64) ([]model.ConversationMember, error) {
					return []model.ConversationMember{
						{ConversationID: conversationID, UserID: 9, Status: model.ConversationMemberStatusActive},
						{ConversationID: conversationID, UserID: 10, Status: model.ConversationMemberStatusActive},
					}, nil
				},
				setVisibleForUsersFn: func(ctx context.Context, conversationID uint64, userIDs []uint64, visible bool) error {
					if visible {
						visibleUsers = append(visibleUsers, userIDs...)
					}
					return nil
				},
				updateLastSentFn: func(ctx context.Context, conversationID, userID, msgSeq uint64) error {
					lastSentConversationID = conversationID
					lastSentUserID = userID
					lastSentSeq = msgSeq
					return nil
				},
			},
		},
		fixedSeqAllocator(7),
	)

	msg, recipients, err := service.SendTextMessage(ctx, 9, 12, "m1", "hello")
	require.NoError(t, err)
	assert.Equal(t, uint64(101), msg.ID)
	assert.Equal(t, uint64(7), msg.Seq)
	assert.Equal(t, uint64(7), savedMsg.Seq)
	assert.Equal(t, []uint64{9, 10}, recipients)
	assert.Equal(t, []uint64{9, 10}, visibleUsers)
	assert.Equal(t, uint64(12), lastSentConversationID)
	assert.Equal(t, uint64(9), lastSentUserID)
	assert.Equal(t, uint64(7), lastSentSeq)
}
