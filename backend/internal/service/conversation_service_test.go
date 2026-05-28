package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/model"
	"gorm.io/gorm"
)

func TestConversationService_ListOfflineMessagesSortsAcrossConversations(t *testing.T) {
	service := NewConversationService(
		&stubConversationRepo{
			listMembersByUserFn: func(ctx context.Context, userID uint64) ([]model.ConversationMember, error) {
				return []model.ConversationMember{
					{ConversationID: 1, UserID: userID, Status: model.ConversationMemberStatusActive, LastReadMsgSeq: 1, LastAckedMsgSeq: 1},
					{ConversationID: 2, UserID: userID, Status: model.ConversationMemberStatusActive, LastReadMsgSeq: 0, LastAckedMsgSeq: 0},
				}, nil
			},
		},
		&stubMessageRepo{
			listConversationRangeFn: func(ctx context.Context, conversationID, afterSeq, untilSeq uint64, limit int) ([]model.Message, bool, error) {
				assert.Equal(t, uint64(200), uint64(limit))
				if conversationID == 1 {
					assert.Equal(t, uint64(1), afterSeq)
					assert.Equal(t, uint64(3), untilSeq)
					return []model.Message{
						{ID: 3, MsgID: "m3", ConversationID: 1, Seq: 3, From: 2, SendTime: 3000},
						{ID: 2, MsgID: "m2", ConversationID: 1, Seq: 2, From: 2, SendTime: 2000},
					}, false, nil
				}
				assert.Equal(t, uint64(0), afterSeq)
				assert.Equal(t, uint64(2), untilSeq)
				return []model.Message{
					{ID: 4, MsgID: "m4", ConversationID: 2, Seq: 2, From: 3, SendTime: 2000},
					{ID: 1, MsgID: "m1", ConversationID: 2, Seq: 1, From: 3, SendTime: 1000},
				}, false, nil
			},
			getMaxSeqByConversationFn: func(ctx context.Context, conversationID uint64) (uint64, error) {
				if conversationID == 1 {
					return 3, nil
				}
				return 2, nil
			},
		},
		&stubUserRepo{},
		&stubPresenceRepo{},
		nil,
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
					return model.ConversationMember{ConversationID: conversationID, UserID: userID, Status: model.ConversationMemberStatusActive, LastAckedMsgSeq: 55}, nil
				},
				getByIDFn: func(ctx context.Context, conversationID uint64) (model.Conversation, error) {
					return model.Conversation{ID: conversationID, Type: model.ConversationTypeSingle, Status: model.ConversationStatusActive}, nil
				},
				updateLastReadFn: func(ctx context.Context, conversationID, userID, msgSeq uint64) error {
					updatedConversationID = conversationID
					updatedUserID = userID
					updatedSeq = msgSeq
					return nil
				},
				listGroupReadTargetsFn: func(ctx context.Context, conversationID, readerID, fromExclusive, toInclusive uint64) ([]uint64, error) {
					t.Fatal("single conversation should not query group read receipt targets")
					return nil, nil
				},
				listActiveMembersFn: func(ctx context.Context, conversationID uint64) ([]model.ConversationMember, error) {
					return []model.ConversationMember{
						{ConversationID: conversationID, UserID: 9, Status: model.ConversationMemberStatusActive},
						{ConversationID: conversationID, UserID: 10, Status: model.ConversationMemberStatusActive},
					}, nil
				},
			},
			&stubMessageRepo{},
			&stubUserRepo{},
			&stubPresenceRepo{},
			nil,
		)

		recipients, err := service.MarkRead(ctx, 9, 12, 55)
		assert.NoError(t, err)
		assert.Equal(t, uint64(12), updatedConversationID)
		assert.Equal(t, uint64(9), updatedUserID)
		assert.Equal(t, uint64(55), updatedSeq)
		assert.Equal(t, []uint64{9, 10}, recipients)
	})

	t.Run("mark read on group only targets senders whose latest message enters interval", func(t *testing.T) {
		var updatedConversationID uint64
		var updatedUserID uint64
		var updatedSeq uint64

		service := NewConversationService(
			&stubConversationRepo{
				getMemberFn: func(ctx context.Context, conversationID, userID uint64) (model.ConversationMember, error) {
					return model.ConversationMember{
						ConversationID:  conversationID,
						UserID:          userID,
						Status:          model.ConversationMemberStatusActive,
						LastReadMsgSeq:  40,
						LastAckedMsgSeq: 55,
					}, nil
				},
				getByIDFn: func(ctx context.Context, conversationID uint64) (model.Conversation, error) {
					return model.Conversation{ID: conversationID, Type: model.ConversationTypeGroup, Status: model.ConversationStatusActive}, nil
				},
				updateLastReadFn: func(ctx context.Context, conversationID, userID, msgSeq uint64) error {
					updatedConversationID = conversationID
					updatedUserID = userID
					updatedSeq = msgSeq
					return nil
				},
				listActiveMembersFn: func(ctx context.Context, conversationID uint64) ([]model.ConversationMember, error) {
					t.Fatal("group conversation should not broadcast read receipts to all active members")
					return nil, nil
				},
				listGroupReadTargetsFn: func(ctx context.Context, conversationID, readerID, fromExclusive, toInclusive uint64) ([]uint64, error) {
					assert.Equal(t, uint64(12), conversationID)
					assert.Equal(t, uint64(9), readerID)
					assert.Equal(t, uint64(40), fromExclusive)
					assert.Equal(t, uint64(55), toInclusive)
					return []uint64{10, 12}, nil
				},
			},
			&stubMessageRepo{},
			&stubUserRepo{},
			&stubPresenceRepo{},
			nil,
		)

		recipients, err := service.MarkRead(ctx, 9, 12, 55)
		assert.NoError(t, err)
		assert.Equal(t, uint64(12), updatedConversationID)
		assert.Equal(t, uint64(9), updatedUserID)
		assert.Equal(t, uint64(55), updatedSeq)
		assert.Equal(t, []uint64{10, 12}, recipients)
	})

	t.Run("mark read rejects messages before joined seq", func(t *testing.T) {
		service := NewConversationService(
			&stubConversationRepo{
				getMemberFn: func(ctx context.Context, conversationID, userID uint64) (model.ConversationMember, error) {
					return model.ConversationMember{ConversationID: conversationID, UserID: userID, Status: model.ConversationMemberStatusActive, JoinedMsgSeq: 55, LastAckedMsgSeq: 55}, nil
				},
				getByIDFn: func(ctx context.Context, conversationID uint64) (model.Conversation, error) {
					return model.Conversation{ID: conversationID, Status: model.ConversationStatusActive}, nil
				},
			},
			&stubMessageRepo{},
			&stubUserRepo{},
			&stubPresenceRepo{},
			nil,
		)

		_, err := service.MarkRead(ctx, 9, 12, 55)
		assert.Error(t, err)
		assert.Equal(t, apperr.CodeMessageNotReadable, apperr.CodeOf(err))
	})

	t.Run("mark read rejects messages beyond acked seq", func(t *testing.T) {
		service := NewConversationService(
			&stubConversationRepo{
				getMemberFn: func(ctx context.Context, conversationID, userID uint64) (model.ConversationMember, error) {
					return model.ConversationMember{ConversationID: conversationID, UserID: userID, Status: model.ConversationMemberStatusActive, LastAckedMsgSeq: 54}, nil
				},
				getByIDFn: func(ctx context.Context, conversationID uint64) (model.Conversation, error) {
					return model.Conversation{ID: conversationID, Status: model.ConversationStatusActive}, nil
				},
			},
			&stubMessageRepo{},
			&stubUserRepo{},
			&stubPresenceRepo{},
			nil,
		)

		_, err := service.MarkRead(ctx, 9, 12, 55)
		assert.Error(t, err)
		assert.Equal(t, apperr.CodeMessageNotDelivered, apperr.CodeOf(err))
	})

	t.Run("open conversation returns latest read state for single conversation", func(t *testing.T) {
		service := NewConversationService(
			&stubConversationRepo{
				getByIDFn: func(ctx context.Context, conversationID uint64) (model.Conversation, error) {
					key := "1:2"
					return model.Conversation{ID: conversationID, Type: model.ConversationTypeSingle, Status: model.ConversationStatusActive, SingleKey: &key}, nil
				},
				getMemberFn: func(ctx context.Context, conversationID, userID uint64) (model.ConversationMember, error) {
					return model.ConversationMember{
						ConversationID:  conversationID,
						UserID:          userID,
						Status:          model.ConversationMemberStatusActive,
						Visible:         true,
						LastSentMsgSeq:  21,
						LastReadMsgSeq:  10,
						LastAckedMsgSeq: 21,
					}, nil
				},
				listReadReceiptUsersFn: func(ctx context.Context, conversationID, senderID, sentSeq uint64) ([]uint64, error) {
					assert.Equal(t, uint64(12), conversationID)
					assert.Equal(t, uint64(1), senderID)
					assert.Equal(t, uint64(21), sentSeq)
					return []uint64{2}, nil
				},
			},
			&stubMessageRepo{
				getLatestByConversationFn: func(ctx context.Context, conversationID uint64) (model.Message, error) {
					return model.Message{ID: 11, MsgID: "m11", ConversationID: conversationID, Seq: 21, SendTime: 12345, Content: "hello"}, nil
				},
				countUnreadFn: func(ctx context.Context, conversationID, userID, afterSeq uint64) (int64, error) {
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
			nil,
		)

		result, err := service.OpenConversation(ctx, 1, 12)
		assert.NoError(t, err)
		assert.Equal(t, uint64(12), result.Conversation.ID)
		assert.NotNil(t, result.LatestReadState)
		assert.Equal(t, uint64(21), result.LatestReadState.LatestSentSeq)
		assert.Equal(t, []uint64{2}, result.LatestReadState.ReadByUserIDs)
	})

	t.Run("open single conversation rejects users who are no longer friends", func(t *testing.T) {
		key := "1:2"
		service := NewConversationServiceWithRuntime(
			&stubConversationRepo{
				getByIDFn: func(ctx context.Context, conversationID uint64) (model.Conversation, error) {
					return model.Conversation{ID: conversationID, Type: model.ConversationTypeSingle, Status: model.ConversationStatusActive, SingleKey: &key}, nil
				},
				getMemberFn: func(ctx context.Context, conversationID, userID uint64) (model.ConversationMember, error) {
					return model.ConversationMember{
						ConversationID: conversationID,
						UserID:         userID,
						Status:         model.ConversationMemberStatusActive,
						Visible:        false,
					}, nil
				},
				setVisibleFn: func(ctx context.Context, conversationID, userID uint64, visible bool) error {
					t.Fatal("conversation should not be reopened when users are not friends")
					return nil
				},
			},
			&stubMessageRepo{},
			&stubUserRepo{},
			&stubPresenceRepo{},
			&stubFriendRepo{
				areFriendsFn: func(ctx context.Context, userID, friendID uint64) (bool, error) {
					assert.Equal(t, uint64(1), userID)
					assert.Equal(t, uint64(2), friendID)
					return false, nil
				},
			},
			nil,
			nil,
		)

		_, err := service.OpenConversation(ctx, 1, 12)
		assert.Error(t, err)
		assert.Equal(t, apperr.CodeConversationNotAccessible, apperr.CodeOf(err))
	})

	t.Run("open conversation returns latest read state for group conversation", func(t *testing.T) {
		service := NewConversationService(
			&stubConversationRepo{
				getByIDFn: func(ctx context.Context, conversationID uint64) (model.Conversation, error) {
					return model.Conversation{ID: conversationID, Type: model.ConversationTypeGroup, Status: model.ConversationStatusActive, Name: "group"}, nil
				},
				getMemberFn: func(ctx context.Context, conversationID, userID uint64) (model.ConversationMember, error) {
					return model.ConversationMember{
						ConversationID:  conversationID,
						UserID:          userID,
						Status:          model.ConversationMemberStatusActive,
						Visible:         true,
						LastSentMsgSeq:  33,
						LastReadMsgSeq:  20,
						LastAckedMsgSeq: 33,
					}, nil
				},
				listReadReceiptUsersFn: func(ctx context.Context, conversationID, senderID, sentSeq uint64) ([]uint64, error) {
					assert.Equal(t, uint64(18), conversationID)
					assert.Equal(t, uint64(9), senderID)
					assert.Equal(t, uint64(33), sentSeq)
					return []uint64{10, 11}, nil
				},
			},
			&stubMessageRepo{
				getLatestByConversationFn: func(ctx context.Context, conversationID uint64) (model.Message, error) {
					return model.Message{ID: 30, MsgID: "m30", ConversationID: conversationID, Seq: 33, Content: "hello"}, nil
				},
				countUnreadFn: func(ctx context.Context, conversationID, userID, afterSeq uint64) (int64, error) {
					return 2, nil
				},
			},
			&stubUserRepo{
				listByIDsFn: func(ctx context.Context, ids []uint64) ([]model.User, error) {
					users := make([]model.User, 0, len(ids))
					for _, id := range ids {
						users = append(users, model.User{ID: id, Username: "u"})
					}
					return users, nil
				},
			},
			&stubPresenceRepo{},
			nil,
		)

		result, err := service.OpenConversation(ctx, 9, 18)
		assert.NoError(t, err)
		assert.Equal(t, uint64(18), result.Conversation.ID)
		assert.NotNil(t, result.LatestReadState)
		assert.Equal(t, uint64(33), result.LatestReadState.LatestSentSeq)
		assert.Equal(t, []uint64{10, 11}, result.LatestReadState.ReadByUserIDs)
	})

	t.Run("open conversation returns nil latest read state when last sent seq is zero", func(t *testing.T) {
		service := NewConversationService(
			&stubConversationRepo{
				getByIDFn: func(ctx context.Context, conversationID uint64) (model.Conversation, error) {
					return model.Conversation{ID: conversationID, Type: model.ConversationTypeGroup, Status: model.ConversationStatusActive, Name: "group"}, nil
				},
				getMemberFn: func(ctx context.Context, conversationID, userID uint64) (model.ConversationMember, error) {
					return model.ConversationMember{
						ConversationID:  conversationID,
						UserID:          userID,
						Status:          model.ConversationMemberStatusActive,
						Visible:         true,
						LastSentMsgSeq:  0,
						LastReadMsgSeq:  20,
						LastAckedMsgSeq: 20,
					}, nil
				},
				listReadReceiptUsersFn: func(ctx context.Context, conversationID, senderID, sentSeq uint64) ([]uint64, error) {
					t.Fatal("should not query read receipt snapshot when last sent seq is zero")
					return nil, nil
				},
			},
			&stubMessageRepo{
				getLatestByConversationFn: func(ctx context.Context, conversationID uint64) (model.Message, error) {
					return model.Message{ID: 31, MsgID: "m31", ConversationID: conversationID, Seq: 20, Content: "old"}, nil
				},
				countUnreadFn: func(ctx context.Context, conversationID, userID, afterSeq uint64) (int64, error) {
					return 0, nil
				},
			},
			&stubUserRepo{},
			&stubPresenceRepo{},
			nil,
		)

		result, err := service.OpenConversation(ctx, 9, 19)
		assert.NoError(t, err)
		assert.Equal(t, uint64(19), result.Conversation.ID)
		assert.Nil(t, result.LatestReadState)
	})

	t.Run("list conversations builds summary", func(t *testing.T) {
		service := NewConversationService(
			&stubConversationRepo{
				listMembersByUserFn: func(ctx context.Context, userID uint64) ([]model.ConversationMember, error) {
					return []model.ConversationMember{{ConversationID: 1, UserID: userID, Status: model.ConversationMemberStatusActive, LastReadMsgSeq: 10}}, nil
				},
				listConversationsByUserFn: func(ctx context.Context, userID uint64) ([]model.Conversation, error) {
					key := "1:2"
					return []model.Conversation{{ID: 1, Type: model.ConversationTypeSingle, Status: model.ConversationStatusActive, SingleKey: &key}}, nil
				},
				getMemberFn: func(ctx context.Context, conversationID, userID uint64) (model.ConversationMember, error) {
					return model.ConversationMember{ConversationID: conversationID, UserID: userID, Status: model.ConversationMemberStatusActive, LastReadMsgSeq: 10}, nil
				},
			},
			&stubMessageRepo{
				getLatestByConversationFn: func(ctx context.Context, conversationID uint64) (model.Message, error) {
					return model.Message{ID: 11, MsgID: "m11", ConversationID: conversationID, SendTime: 12345, Content: "hello"}, nil
				},
				countUnreadFn: func(ctx context.Context, conversationID, userID, afterSeq uint64) (int64, error) {
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
			nil,
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
					return []model.ConversationMember{{ConversationID: 2, UserID: userID, Status: model.ConversationMemberStatusActive}}, nil
				},
				listConversationsByUserFn: func(ctx context.Context, userID uint64) ([]model.Conversation, error) {
					return []model.Conversation{{ID: 2, Type: model.ConversationTypeGroup, Name: "group", Status: model.ConversationStatusActive}}, nil
				},
				getMemberFn: func(ctx context.Context, conversationID, userID uint64) (model.ConversationMember, error) {
					return model.ConversationMember{ConversationID: conversationID, UserID: userID, Status: model.ConversationMemberStatusActive}, nil
				},
			},
			&stubMessageRepo{
				getLatestByConversationFn: func(ctx context.Context, conversationID uint64) (model.Message, error) {
					return model.Message{}, gorm.ErrRecordNotFound
				},
				countUnreadFn: func(ctx context.Context, conversationID, userID, afterSeq uint64) (int64, error) {
					return 0, nil
				},
			},
			&stubUserRepo{},
			&stubPresenceRepo{},
			nil,
		)

		items, err := service.ListConversations(ctx, 1)
		assert.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Nil(t, items[0].LastMessage)
	})

	t.Run("list groups returns active groups regardless of visible", func(t *testing.T) {
		service := NewConversationService(
			&stubConversationRepo{
				listActiveGroupsByUserFn: func(ctx context.Context, userID uint64) ([]model.Conversation, error) {
					return []model.Conversation{
						{ID: 21, Type: model.ConversationTypeGroup, Name: "project", Status: model.ConversationStatusActive},
					}, nil
				},
				getMemberFn: func(ctx context.Context, conversationID, userID uint64) (model.ConversationMember, error) {
					return model.ConversationMember{
						ConversationID: conversationID,
						UserID:         userID,
						Status:         model.ConversationMemberStatusActive,
						Visible:        false,
					}, nil
				},
			},
			&stubMessageRepo{
				getLatestByConversationFn: func(ctx context.Context, conversationID uint64) (model.Message, error) {
					return model.Message{ID: 30, MsgID: "m30", ConversationID: conversationID, Content: "hello"}, nil
				},
				countUnreadFn: func(ctx context.Context, conversationID, userID, afterSeq uint64) (int64, error) {
					return 2, nil
				},
			},
			&stubUserRepo{},
			&stubPresenceRepo{},
			nil,
		)

		items, err := service.ListGroups(ctx, 1)
		assert.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, uint64(21), items[0].ID)
		assert.Equal(t, "project", items[0].Name)
		assert.Equal(t, int64(2), items[0].UnreadCount)
		assert.NotNil(t, items[0].LastMessage)
		assert.Nil(t, items[0].Peer)
	})
}

func TestConversationService_CreateGroupAndListMembers(t *testing.T) {
	ctx := context.Background()

	var createdConversation model.Conversation
	var upserted []model.ConversationMember
	var createdMessages []model.Message

	service := NewConversationService(
		&stubConversationRepo{
			createFn: func(ctx context.Context, conversation *model.Conversation) error {
				conversation.ID = 21
				createdConversation = *conversation
				return nil
			},
			upsertMemberFn: func(ctx context.Context, member *model.ConversationMember) error {
				upserted = append(upserted, *member)
				return nil
			},
			listActiveMembersFn: func(ctx context.Context, conversationID uint64) ([]model.ConversationMember, error) {
				return []model.ConversationMember{
					{ConversationID: conversationID, UserID: 1, Status: model.ConversationMemberStatusActive, Role: model.ConversationMemberRoleOwner},
					{ConversationID: conversationID, UserID: 2, Status: model.ConversationMemberStatusActive, Role: model.ConversationMemberRoleMember},
					{ConversationID: conversationID, UserID: 3, Status: model.ConversationMemberStatusActive, Role: model.ConversationMemberRoleMember},
				}, nil
			},
			updateLastAckedFn: func(ctx context.Context, conversationID, userID, msgSeq uint64) error {
				return nil
			},
			updateLastReadFn: func(ctx context.Context, conversationID, userID, msgSeq uint64) error {
				return nil
			},
			setVisibleForUsersFn: func(ctx context.Context, conversationID uint64, userIDs []uint64, visible bool) error {
				return nil
			},
			getMemberFn: func(ctx context.Context, conversationID, userID uint64) (model.ConversationMember, error) {
				return model.ConversationMember{
					ConversationID: conversationID,
					UserID:         userID,
					Status:         model.ConversationMemberStatusActive,
					Role:           model.ConversationMemberRoleOwner,
				}, nil
			},
			getByIDFn: func(ctx context.Context, conversationID uint64) (model.Conversation, error) {
				return model.Conversation{
					ID:      conversationID,
					Type:    model.ConversationTypeGroup,
					Status:  model.ConversationStatusActive,
					Name:    "project",
					OwnerID: 1,
				}, nil
			},
			countActiveMembersFn: func(ctx context.Context, conversationID uint64) (int64, error) {
				return 3, nil
			},
		},
		&stubMessageRepo{
			createFn: func(ctx context.Context, msg *model.Message) error {
				msg.ID = 101
				createdMessages = append(createdMessages, *msg)
				return nil
			},
			getLatestByConversationFn: func(ctx context.Context, conversationID uint64) (model.Message, error) {
				return model.Message{}, gorm.ErrRecordNotFound
			},
			countUnreadFn: func(ctx context.Context, conversationID, userID, afterSeq uint64) (int64, error) {
				return 0, nil
			},
		},
		&stubUserRepo{
			listByIDsFn: func(ctx context.Context, ids []uint64) ([]model.User, error) {
				users := make([]model.User, 0, len(ids))
				for _, id := range ids {
					users = append(users, model.User{ID: id, Username: "u"})
				}
				return users, nil
			},
		},
		&stubPresenceRepo{
			isOnlineFn: func(ctx context.Context, userID uint64) (bool, error) {
				return userID != 3, nil
			},
		},
		nil,
	)

	summary, err := service.CreateGroup(ctx, 1, " project ", []uint64{2, 3, 2})
	assert.NoError(t, err)
	assert.Equal(t, uint64(21), summary.ID)
	assert.Equal(t, model.ConversationTypeGroup, createdConversation.Type)
	assert.Equal(t, "project", createdConversation.Name)
	assert.Len(t, upserted, 3)
	assert.Len(t, createdMessages, 1)
	assert.Equal(t, model.MessageTypeSystem, createdMessages[0].Type)
	assert.Equal(t, model.MessageEventGroupCreated, createdMessages[0].Event)

	members, err := service.ListGroupMembers(ctx, 1, 21)
	assert.NoError(t, err)
	assert.Len(t, members, 3)
	assert.Equal(t, uint64(1), members[0].UserID)
	assert.Equal(t, model.ConversationMemberRoleOwner, members[0].Role)
}

func TestConversationService_SystemMessageExtraUsesUserIDs(t *testing.T) {
	ctx := context.Background()

	t.Run("invite group members stores user ids", func(t *testing.T) {
		var createdMessages []model.Message

		service := NewConversationService(
			&stubConversationRepo{
				getByIDFn: func(ctx context.Context, conversationID uint64) (model.Conversation, error) {
					return model.Conversation{
						ID:      conversationID,
						Type:    model.ConversationTypeGroup,
						Status:  model.ConversationStatusActive,
						Name:    "project",
						OwnerID: 1,
					}, nil
				},
				getMemberFn: func(ctx context.Context, conversationID, userID uint64) (model.ConversationMember, error) {
					return model.ConversationMember{
						ConversationID: conversationID,
						UserID:         userID,
						Status:         model.ConversationMemberStatusActive,
						Role:           model.ConversationMemberRoleOwner,
					}, nil
				},
				listActiveMembersFn: func(ctx context.Context, conversationID uint64) ([]model.ConversationMember, error) {
					return []model.ConversationMember{
						{ConversationID: conversationID, UserID: 1, Status: model.ConversationMemberStatusActive, Role: model.ConversationMemberRoleOwner},
						{ConversationID: conversationID, UserID: 2, Status: model.ConversationMemberStatusActive, Role: model.ConversationMemberRoleMember},
					}, nil
				},
				upsertMemberFn: func(ctx context.Context, member *model.ConversationMember) error {
					return nil
				},
				setVisibleForUsersFn: func(ctx context.Context, conversationID uint64, userIDs []uint64, visible bool) error {
					return nil
				},
				updateLastAckedFn: func(ctx context.Context, conversationID, userID, msgSeq uint64) error {
					return nil
				},
				updateLastReadFn: func(ctx context.Context, conversationID, userID, msgSeq uint64) error {
					return nil
				},
			},
			&stubMessageRepo{
				createFn: func(ctx context.Context, msg *model.Message) error {
					createdMessages = append(createdMessages, *msg)
					return nil
				},
				getMaxSeqByConversationFn: func(ctx context.Context, conversationID uint64) (uint64, error) {
					return 10, nil
				},
			},
			&stubUserRepo{
				listByIDsFn: func(ctx context.Context, ids []uint64) ([]model.User, error) {
					users := make([]model.User, 0, len(ids))
					for _, id := range ids {
						users = append(users, model.User{ID: id, Username: "u"})
					}
					return users, nil
				},
			},
			&stubPresenceRepo{},
			nil,
		)

		err := service.InviteGroupMembers(ctx, 1, 20, []uint64{2, 3})
		assert.NoError(t, err)
		require.Len(t, createdMessages, 1)
		assert.Equal(t, model.MessageTypeSystem, createdMessages[0].Type)
		assert.Equal(t, model.MessageEventGroupMembersJoined, createdMessages[0].Event)

		var extra struct {
			MemberIDs []uint64 `json:"member_ids"`
			Count     int      `json:"count"`
		}
		require.NoError(t, json.Unmarshal(createdMessages[0].Extra, &extra))
		assert.Equal(t, []uint64{3}, extra.MemberIDs)
		assert.Equal(t, 1, extra.Count)
	})

	t.Run("remove group member stores target user id", func(t *testing.T) {
		var createdMessages []model.Message

		service := NewConversationService(
			&stubConversationRepo{
				getByIDFn: func(ctx context.Context, conversationID uint64) (model.Conversation, error) {
					return model.Conversation{
						ID:      conversationID,
						Type:    model.ConversationTypeGroup,
						Status:  model.ConversationStatusActive,
						Name:    "project",
						OwnerID: 1,
					}, nil
				},
				getMemberFn: func(ctx context.Context, conversationID, userID uint64) (model.ConversationMember, error) {
					role := model.ConversationMemberRoleMember
					if userID == 1 {
						role = model.ConversationMemberRoleOwner
					}
					return model.ConversationMember{
						ConversationID: conversationID,
						UserID:         userID,
						Status:         model.ConversationMemberStatusActive,
						Role:           role,
					}, nil
				},
				listActiveMembersFn: func(ctx context.Context, conversationID uint64) ([]model.ConversationMember, error) {
					return []model.ConversationMember{
						{ConversationID: conversationID, UserID: 1, Status: model.ConversationMemberStatusActive, Role: model.ConversationMemberRoleOwner},
						{ConversationID: conversationID, UserID: 2, Status: model.ConversationMemberStatusActive, Role: model.ConversationMemberRoleMember},
					}, nil
				},
				updateMemberStatusFn: func(ctx context.Context, conversationID, userID uint64, status model.ConversationMemberStatus, visible bool) error {
					return nil
				},
				setVisibleForUsersFn: func(ctx context.Context, conversationID uint64, userIDs []uint64, visible bool) error {
					return nil
				},
				updateLastAckedFn: func(ctx context.Context, conversationID, userID, msgSeq uint64) error {
					return nil
				},
				updateLastReadFn: func(ctx context.Context, conversationID, userID, msgSeq uint64) error {
					return nil
				},
			},
			&stubMessageRepo{
				createFn: func(ctx context.Context, msg *model.Message) error {
					createdMessages = append(createdMessages, *msg)
					return nil
				},
				getMaxSeqByConversationFn: func(ctx context.Context, conversationID uint64) (uint64, error) {
					return 10, nil
				},
			},
			&stubUserRepo{
				listByIDsFn: func(ctx context.Context, ids []uint64) ([]model.User, error) {
					users := make([]model.User, 0, len(ids))
					for _, id := range ids {
						users = append(users, model.User{ID: id, Username: "u"})
					}
					return users, nil
				},
			},
			&stubPresenceRepo{},
			nil,
		)

		err := service.RemoveGroupMember(ctx, 1, 20, 2)
		assert.NoError(t, err)
		require.Len(t, createdMessages, 1)
		assert.Equal(t, model.MessageTypeSystem, createdMessages[0].Type)
		assert.Equal(t, model.MessageEventGroupMemberRemoved, createdMessages[0].Event)

		var extra struct {
			TargetUserID uint64 `json:"target_user_id"`
		}
		require.NoError(t, json.Unmarshal(createdMessages[0].Extra, &extra))
		assert.Equal(t, uint64(2), extra.TargetUserID)
	})

	t.Run("leave group stores user id", func(t *testing.T) {
		var createdMessages []model.Message

		service := NewConversationService(
			&stubConversationRepo{
				getByIDFn: func(ctx context.Context, conversationID uint64) (model.Conversation, error) {
					return model.Conversation{
						ID:      conversationID,
						Type:    model.ConversationTypeGroup,
						Status:  model.ConversationStatusActive,
						Name:    "project",
						OwnerID: 1,
					}, nil
				},
				getMemberFn: func(ctx context.Context, conversationID, userID uint64) (model.ConversationMember, error) {
					return model.ConversationMember{
						ConversationID: conversationID,
						UserID:         userID,
						Status:         model.ConversationMemberStatusActive,
						Role:           model.ConversationMemberRoleMember,
					}, nil
				},
				listActiveMembersFn: func(ctx context.Context, conversationID uint64) ([]model.ConversationMember, error) {
					return []model.ConversationMember{
						{ConversationID: conversationID, UserID: 1, Status: model.ConversationMemberStatusActive, Role: model.ConversationMemberRoleOwner},
						{ConversationID: conversationID, UserID: 2, Status: model.ConversationMemberStatusActive, Role: model.ConversationMemberRoleMember},
					}, nil
				},
				updateMemberStatusFn: func(ctx context.Context, conversationID, userID uint64, status model.ConversationMemberStatus, visible bool) error {
					return nil
				},
				setVisibleForUsersFn: func(ctx context.Context, conversationID uint64, userIDs []uint64, visible bool) error {
					return nil
				},
				updateLastAckedFn: func(ctx context.Context, conversationID, userID, msgSeq uint64) error {
					return nil
				},
				updateLastReadFn: func(ctx context.Context, conversationID, userID, msgSeq uint64) error {
					return nil
				},
			},
			&stubMessageRepo{
				createFn: func(ctx context.Context, msg *model.Message) error {
					createdMessages = append(createdMessages, *msg)
					return nil
				},
				getMaxSeqByConversationFn: func(ctx context.Context, conversationID uint64) (uint64, error) {
					return 10, nil
				},
			},
			&stubUserRepo{
				listByIDsFn: func(ctx context.Context, ids []uint64) ([]model.User, error) {
					users := make([]model.User, 0, len(ids))
					for _, id := range ids {
						users = append(users, model.User{ID: id, Username: "u"})
					}
					return users, nil
				},
			},
			&stubPresenceRepo{},
			nil,
		)

		err := service.LeaveGroup(ctx, 2, 20)
		assert.NoError(t, err)
		require.Len(t, createdMessages, 1)
		assert.Equal(t, model.MessageTypeSystem, createdMessages[0].Type)
		assert.Equal(t, model.MessageEventGroupMemberLeft, createdMessages[0].Event)

		var extra struct {
			UserID uint64 `json:"user_id"`
		}
		require.NoError(t, json.Unmarshal(createdMessages[0].Extra, &extra))
		assert.Equal(t, uint64(2), extra.UserID)
	})
}
