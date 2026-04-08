package dto

import "github.com/zyy125/im-system/internal/model"

type MessageHistoryResp struct {
	Messages     []model.ChatMessage `json:"messages"`
	HasMore      bool                `json:"has_more"`
	NextBeforeID uint64              `json:"next_before_id,omitempty"`
}

type MarkReadReq struct {
	ConversationID string `json:"conversation_id" binding:"required"`
	MsgID          string `json:"msg_id" binding:"required"`
}

type ConversationPeerResp struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
	Online   bool   `json:"online"`
}

type ConversationItemResp struct {
	ID          uint64                 `json:"id"`
	Type        model.ConversationType `json:"type"`
	Name        string                 `json:"name"`
	UnreadCount int64                  `json:"unread_count"`
	Peer        *ConversationPeerResp  `json:"peer,omitempty"`
	LastMessage *model.ChatMessage     `json:"last_message,omitempty"`
}

type ConversationListResp struct {
	Conversations []ConversationItemResp `json:"conversations"`
}

type OpenConversationResp struct {
	Conversation ConversationItemResp `json:"conversation"`
}
