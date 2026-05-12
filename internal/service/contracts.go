package service

import (
	"context"

	"github.com/zyy125/im-system/internal/model"
)

type MessageQueryService interface {
	ListConversationHistory(ctx context.Context, userID, conversationID uint64, limit int, beforeSeq uint64) ([]model.Message, bool, error)
	SyncConversation(ctx context.Context, userID, conversationID, afterSeq uint64, limit int) ([]model.Message, bool, error)
}

type MessageReceiptService interface {
	MarkDelivered(ctx context.Context, userID, conversationID, deliveredSeq uint64) ([]uint64, error)
}

type MessageCommandService interface {
	SendTextMessage(ctx context.Context, senderID, conversationID uint64, msgID, content string) (model.Message, []uint64, error)
}

type ConversationQueryService interface {
	OpenConversation(ctx context.Context, userID, conversationID uint64) (ConversationSummary, error)
	ListConversations(ctx context.Context, userID uint64) ([]ConversationSummary, error)
	ListGroups(ctx context.Context, userID uint64) ([]ConversationSummary, error)
	GetGroupDetail(ctx context.Context, userID, conversationID uint64) (GroupDetail, error)
	ListGroupMembers(ctx context.Context, userID, conversationID uint64) ([]GroupMember, error)
}

type ConversationCommandService interface {
	HideConversation(ctx context.Context, userID, conversationID uint64) error
	CreateGroup(ctx context.Context, ownerID uint64, name string, memberIDs []uint64) (ConversationSummary, error)
	UpdateGroupName(ctx context.Context, userID, conversationID uint64, name string) error
	InviteGroupMembers(ctx context.Context, userID, conversationID uint64, memberIDs []uint64) error
	RemoveGroupMember(ctx context.Context, userID, conversationID, memberID uint64) error
	LeaveGroup(ctx context.Context, userID, conversationID uint64) error
	DismissGroup(ctx context.Context, userID, conversationID uint64) error
}

type ConversationSyncService interface {
	ListOfflineMessages(ctx context.Context, userID uint64) ([]model.Message, error)
	MarkRead(ctx context.Context, userID, conversationID, readSeq uint64) ([]uint64, error)
}
