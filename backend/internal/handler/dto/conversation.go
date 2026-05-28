package dto

import "github.com/zyy125/im-system/internal/model"

type ConversationPeerResp struct {
	PublicID uint64 `json:"public_id"`
	Username string `json:"username"`
	Online   bool   `json:"online"`
}

type ConversationItemResp struct {
	ID          uint64                 `json:"id"`
	Type        model.ConversationType `json:"type"`
	Name        string                 `json:"name"`
	UnreadCount int64                  `json:"unread_count"`
	Peer        *ConversationPeerResp  `json:"peer,omitempty"`
	LastMessage *MessageResp           `json:"last_message,omitempty"`
}

type ConversationListResp struct {
	Conversations []ConversationItemResp `json:"conversations"`
}

type LatestReadStateResp struct {
	LatestSentSeq   uint64   `json:"latest_sent_seq"`
	ReadByPublicIDs []uint64 `json:"read_by_public_ids"`
	ReadCount       int      `json:"read_count"`
}

type OpenConversationResp struct {
	Conversation    ConversationItemResp `json:"conversation"`
	LatestReadState *LatestReadStateResp `json:"latest_read_state,omitempty"`
}

type CreateGroupReq struct {
	Name      string   `json:"name" binding:"required"`
	MemberIDs []uint64 `json:"member_ids"`
}

type GroupDetailResp struct {
	ID          uint64                       `json:"id"`
	Name        string                       `json:"name"`
	Avatar      string                       `json:"avatar"`
	OwnerID     uint64                       `json:"owner_id"`
	Status      model.ConversationStatus     `json:"status"`
	MyRole      model.ConversationMemberRole `json:"my_role"`
	MemberCount int64                        `json:"member_count"`
}

type GroupMemberResp struct {
	PublicID uint64                       `json:"public_id"`
	Username string                       `json:"username"`
	Role     model.ConversationMemberRole `json:"role"`
	Online   bool                         `json:"online"`
}

type GroupMemberListResp struct {
	Members []GroupMemberResp `json:"members"`
}

type GroupDetailEnvelopeResp struct {
	Group GroupDetailResp `json:"group"`
}

type GroupConversationResp struct {
	Conversation ConversationItemResp `json:"conversation"`
}

type UpdateGroupNameReq struct {
	Name string `json:"name" binding:"required"`
}

type InviteGroupMembersReq struct {
	MemberIDs []uint64 `json:"member_ids" binding:"required"`
}
