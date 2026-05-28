package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zyy125/im-system/internal/model"
	"github.com/zyy125/im-system/internal/repository"
)

func TestFriendService_ListFriendsIncludesConversationID(t *testing.T) {
	service := NewFriendService(
		&stubFriendRepo{
			listFriendProfilesFn: func(ctx context.Context, userID uint64) ([]repository.FriendProfile, error) {
				return []repository.FriendProfile{{UserID: 2, Username: "bob", ConversationID: 12}}, nil
			},
		},
		&stubUserRepo{
			getByIDFn: func(ctx context.Context, id uint64) (model.User, error) {
				return model.User{ID: id, Username: "bob"}, nil
			},
		},
		&stubPresenceRepo{
			isOnlineFn: func(ctx context.Context, userID uint64) (bool, error) {
				return true, nil
			},
		},
		&stubConversationRepo{},
	)

	items, err := service.ListFriends(context.Background(), 1)
	assert.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, uint64(2), items[0].UserID)
	assert.Equal(t, "bob", items[0].Username)
	assert.True(t, items[0].Online)
	assert.Equal(t, uint64(12), items[0].ConversationID)
}

func TestFriendService_AddFriendStoresConversationID(t *testing.T) {
	var savedConversationID uint64

	service := NewFriendService(
		&stubFriendRepo{
			areFriendsFn: func(ctx context.Context, userID, friendID uint64) (bool, error) {
				return false, nil
			},
			addPairFn: func(ctx context.Context, userID, friendID, conversationID uint64) error {
				savedConversationID = conversationID
				return nil
			},
		},
		&stubUserRepo{
			getByIDFn: func(ctx context.Context, id uint64) (model.User, error) {
				return model.User{ID: id, Username: "bob"}, nil
			},
		},
		&stubPresenceRepo{},
		&stubConversationRepo{
			getOrCreateSingleFn: func(ctx context.Context, userA, userB uint64) (model.Conversation, error) {
				return model.Conversation{ID: 77, Type: model.ConversationTypeSingle}, nil
			},
			setVisibleFn: func(ctx context.Context, conversationID, userID uint64, visible bool) error {
				return nil
			},
		},
	)

	err := service.AddFriend(context.Background(), 1, 2)
	assert.NoError(t, err)
	assert.Equal(t, uint64(77), savedConversationID)
}

func TestFriendService_ListFriendsRepairsMissingConversationID(t *testing.T) {
	var repairedConversationID uint64

	service := NewFriendService(
		&stubFriendRepo{
			listFriendProfilesFn: func(ctx context.Context, userID uint64) ([]repository.FriendProfile, error) {
				return []repository.FriendProfile{{UserID: 2, Username: "bob", ConversationID: 0}}, nil
			},
			addPairFn: func(ctx context.Context, userID, friendID, conversationID uint64) error {
				repairedConversationID = conversationID
				return nil
			},
		},
		&stubUserRepo{
			getByIDFn: func(ctx context.Context, id uint64) (model.User, error) {
				return model.User{ID: id, Username: "bob"}, nil
			},
		},
		&stubPresenceRepo{
			isOnlineFn: func(ctx context.Context, userID uint64) (bool, error) {
				return false, nil
			},
		},
		&stubConversationRepo{
			getOrCreateSingleFn: func(ctx context.Context, userA, userB uint64) (model.Conversation, error) {
				return model.Conversation{ID: 34, Type: model.ConversationTypeSingle}, nil
			},
		},
	)

	items, err := service.ListFriends(context.Background(), 1)
	assert.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, uint64(34), repairedConversationID)
	assert.Equal(t, uint64(34), items[0].ConversationID)
}

func TestFriendService_ListFriendsEmpty(t *testing.T) {

	service := NewFriendService(
		&stubFriendRepo{
			listFriendProfilesFn: func(ctx context.Context, userID uint64) ([]repository.FriendProfile, error) {
				return []repository.FriendProfile{}, nil
			},
		},
		&stubUserRepo{},
		&stubPresenceRepo{},
		&stubConversationRepo{},
	)

	items, err := service.ListFriends(context.Background(), 1)
	assert.NoError(t, err)
	assert.Empty(t, items)
}
