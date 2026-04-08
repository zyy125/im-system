package model

import "time"

type ConversationType uint8

const (
	ConversationTypeSingle ConversationType = 1
	ConversationTypeGroup  ConversationType = 2
)

type ConversationMemberRole uint8

const (
	ConversationMemberRoleOwner  ConversationMemberRole = 1
	ConversationMemberRoleAdmin  ConversationMemberRole = 2
	ConversationMemberRoleMember ConversationMemberRole = 3
)

type Conversation struct {
	ID        uint64           `gorm:"primaryKey"`
	Type      ConversationType `gorm:"type:tinyint unsigned;not null;index;uniqueIndex:idx_conversation_type_single_key,priority:1"` // 1:单聊 2:群聊
	Name      string           `gorm:"size:128;not null;default:''"`                                                                 // 群名称（单聊可选）
	OwnerID   uint64           `gorm:"index;not null;default:0"`                                                                     // 群主ID（单聊可空）
	SingleKey *string          `gorm:"size:64;uniqueIndex:idx_conversation_type_single_key,priority:2"`                              // 仅单聊使用：min(a,b):max(a,b) 如 1:3；群聊保持 NULL
	CreatedAt time.Time        `json:"-"`
	UpdatedAt time.Time        `json:"-"`
}

type ConversationMember struct {
	ID                  uint64                 `gorm:"primaryKey"`
	ConversationID      uint64                 `gorm:"uniqueIndex:idx_conversation_user;index;not null;index:idx_conversation_member_user_visible,priority:3"`
	UserID              uint64                 `gorm:"uniqueIndex:idx_conversation_user;index;not null;index:idx_conversation_member_user_visible,priority:1"`
	Role                ConversationMemberRole `gorm:"type:tinyint unsigned;not null;default:3"` // 1:群主 2:管理员 3:普通成员
	Visible             bool                   `gorm:"not null;default:true;index:idx_conversation_member_user_visible,priority:2"`
	LastReadMsgSeq      uint64                 `gorm:"not null;default:0"`
	LastDeliveredMsgSeq uint64                 `gorm:"not null;default:0"`
	CreatedAt           time.Time              `json:"-"`
	UpdatedAt           time.Time              `json:"-"`
}

func (c Conversation) IsSingle() bool {
	return c.Type == ConversationTypeSingle
}

func (c Conversation) SingleKeyValue() string {
	if c.SingleKey == nil {
		return ""
	}
	return *c.SingleKey
}
