package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/model"
	"gorm.io/gorm"
)

func TestFriendRequestService_SendAutoAcceptedOnReversePending(t *testing.T) {
	ctx := context.Background()
	repo := &stubFriendRequestRepo{}
	friendRepo := &stubFriendRepo{}
	userRepo := &stubUserRepo{}
	conversationRepo := &stubConversationRepo{}

	var addPairCalled bool
	var resolveCalled bool

	userRepo.getByIDFn = func(ctx context.Context, id uint64) (model.User, error) {
		return model.User{ID: id, Username: "u"}, nil
	}
	friendRepo.areFriendsFn = func(ctx context.Context, userID, friendID uint64) (bool, error) {
		return false, nil
	}
	friendRepo.addPairFn = func(ctx context.Context, userID, friendID uint64) error {
		addPairCalled = true
		return nil
	}
	conversationRepo.getOrCreateSingleFn = func(ctx context.Context, userA, userB uint64) (model.Conversation, error) {
		key := "1:2"
		return model.Conversation{ID: 10, Type: model.ConversationTypeSingle, SingleKey: &key}, nil
	}
	conversationRepo.setVisibleFn = func(ctx context.Context, conversationID, userID uint64, visible bool) error {
		return nil
	}
	repo.findPendingBetweenFn = func(ctx context.Context, requesterID, receiverID uint64) (model.FriendRequest, error) {
		if requesterID == 2 && receiverID == 1 {
			return model.FriendRequest{ID: 1, RequesterID: 2, ReceiverID: 1, Status: model.FriendRequestPending}, nil
		}
		return model.FriendRequest{}, gorm.ErrRecordNotFound
	}
	repo.resolvePendingFn = func(ctx context.Context, userA, userB uint64, status model.FriendRequestStatus) error {
		resolveCalled = true
		assert.Equal(t, model.FriendRequestAccepted, status)
		return nil
	}

	friendService := NewFriendService(friendRepo, userRepo, &stubPresenceRepo{}, conversationRepo)
	service := NewFriendRequestService(repo, friendService, userRepo, &stubPresenceRepo{})

	result, err := service.Send(ctx, 1, 2, "hi")
	assert.NoError(t, err)
	assert.Equal(t, "auto_accepted", result)
	assert.True(t, addPairCalled)
	assert.True(t, resolveCalled)
}

func TestFriendRequestService_AcceptRejectPermissionAndStatus(t *testing.T) {
	ctx := context.Background()

	t.Run("accept forbidden", func(t *testing.T) {
		repo := &stubFriendRequestRepo{
			getByIDFn: func(ctx context.Context, id uint64) (model.FriendRequest, error) {
				return model.FriendRequest{ID: id, RequesterID: 1, ReceiverID: 2, Status: model.FriendRequestPending}, nil
			},
		}
		service := NewFriendRequestService(repo, NewFriendService(&stubFriendRepo{}, &stubUserRepo{}, &stubPresenceRepo{}, &stubConversationRepo{}), &stubUserRepo{}, &stubPresenceRepo{})

		err := service.Accept(ctx, 3, 1)
		assert.Error(t, err)
		assert.Equal(t, apperr.CodeFriendRequestNoPermission, apperr.CodeOf(err))
	})

	t.Run("reject not pending", func(t *testing.T) {
		repo := &stubFriendRequestRepo{
			getByIDFn: func(ctx context.Context, id uint64) (model.FriendRequest, error) {
				return model.FriendRequest{ID: id, RequesterID: 1, ReceiverID: 2, Status: model.FriendRequestAccepted}, nil
			},
		}
		service := NewFriendRequestService(repo, NewFriendService(&stubFriendRepo{}, &stubUserRepo{}, &stubPresenceRepo{}, &stubConversationRepo{}), &stubUserRepo{}, &stubPresenceRepo{})

		err := service.Reject(ctx, 2, 1)
		assert.Error(t, err)
		assert.Equal(t, apperr.CodeFriendRequestNotPending, apperr.CodeOf(err))
	})
}

func TestFriendRequestService_ListIncomingFiltersExistingFriends(t *testing.T) {
	ctx := context.Background()
	repo := &stubFriendRequestRepo{
		listIncomingPendingFn: func(ctx context.Context, userID uint64) ([]model.FriendRequest, error) {
			return []model.FriendRequest{
				{ID: 1, RequesterID: 10, ReceiverID: 20, Status: model.FriendRequestPending, Message: "keep"},
				{ID: 2, RequesterID: 30, ReceiverID: 20, Status: model.FriendRequestPending, Message: "drop"},
			}, nil
		},
	}
	userRepo := &stubUserRepo{
		getByIDFn: func(ctx context.Context, id uint64) (model.User, error) {
			return model.User{ID: id, Username: map[uint64]string{10: "alice", 20: "bob", 30: "charlie"}[id]}, nil
		},
	}
	presenceRepo := &stubPresenceRepo{
		isOnlineFn: func(ctx context.Context, userID uint64) (bool, error) {
			return userID == 10, nil
		},
	}
	friendRepo := &stubFriendRepo{
		areFriendsFn: func(ctx context.Context, userID, friendID uint64) (bool, error) {
			return userID == 30 || friendID == 30, nil
		},
	}

	friendService := NewFriendService(friendRepo, userRepo, presenceRepo, &stubConversationRepo{})
	service := NewFriendRequestService(repo, friendService, userRepo, presenceRepo)

	items, err := service.ListIncoming(ctx, 20)
	assert.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, uint64(1), items[0].ID)
	assert.Equal(t, "alice", items[0].Requester.Username)
	assert.True(t, items[0].Requester.Online)
	assert.Equal(t, "bob", items[0].Receiver.Username)
}
