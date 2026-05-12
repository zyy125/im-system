package model

import "time"

// ConversationType 区分单聊和群聊。
type ConversationType uint8

const (
	ConversationTypeSingle ConversationType = 1 // 单聊
	ConversationTypeGroup  ConversationType = 2 // 群聊
)

// ConversationStatus 标识会话是否仍然有效。
type ConversationStatus uint8

const (
	ConversationStatusActive    ConversationStatus = 1 // 正常
	ConversationStatusDismissed ConversationStatus = 2 // 已解散（群聊专用）
)

// ConversationMemberRole 标识成员在群内的角色。
type ConversationMemberRole uint8

const (
	ConversationMemberRoleOwner  ConversationMemberRole = 1 // 群主
	ConversationMemberRoleAdmin  ConversationMemberRole = 2 // 管理员
	ConversationMemberRoleMember ConversationMemberRole = 3 // 普通成员
)

// ConversationMemberStatus 标识成员当前是否还在群内。
type ConversationMemberStatus uint8

const (
	ConversationMemberStatusActive  ConversationMemberStatus = 1 // 活跃成员
	ConversationMemberStatusLeft    ConversationMemberStatus = 2 // 主动退出
	ConversationMemberStatusRemoved ConversationMemberStatus = 3 // 被移出
)

// Conversation 是会话的持久化模型，单聊和群聊共用同一张表。
// 单聊通过 SingleKey（格式 "min_id:max_id"）唯一索引去重，群聊 SingleKey 为 NULL。
type Conversation struct {
	ID        uint64             `gorm:"primaryKey"`
	Type      ConversationType   `gorm:"type:tinyint unsigned;not null;index;uniqueIndex:idx_conversation_type_single_key,priority:1"` // 1:单聊 2:群聊
	Name      string             `gorm:"size:128;not null;default:''"` // 群名称（单聊可选）
	Avatar    string             `gorm:"size:255;not null;default:''"`
	OwnerID   uint64             `gorm:"index;not null;default:0"` // 群主 ID，单聊为 0
	Status    ConversationStatus `gorm:"type:tinyint unsigned;not null;default:1;index"`
	SingleKey *string            `gorm:"size:64;uniqueIndex:idx_conversation_type_single_key,priority:2"` // 仅单聊使用：min(a,b):max(a,b)，如 "1:3"；群聊为 NULL
	CreatedAt time.Time          `json:"-"`
	UpdatedAt time.Time          `json:"-"`
}

// ConversationMember 记录某个用户在某个会话中的状态和游标信息。
type ConversationMember struct {
	ID               uint64                   `gorm:"primaryKey"`
	ConversationID   uint64                   `gorm:"uniqueIndex:idx_conversation_user;index;not null;index:idx_conversation_member_user_visible,priority:3"`
	UserID           uint64                   `gorm:"uniqueIndex:idx_conversation_user;index;not null;index:idx_conversation_member_user_visible,priority:1"`
	Role             ConversationMemberRole   `gorm:"type:tinyint unsigned;not null;default:3"` // 1:群主 2:管理员 3:普通成员
	Status           ConversationMemberStatus `gorm:"type:tinyint unsigned;not null;default:1;index"`
	Visible          bool                     `gorm:"not null;default:true;index:idx_conversation_member_user_visible,priority:2"` // false 时会话从列表隐藏，收到新消息后自动恢复
	InvitedBy        uint64                   `gorm:"not null;default:0"` // 邀请人 ID，0 表示自己加入或创建
	JoinedMsgSeq     uint64                   `gorm:"not null;default:0"` // 入群时的消息 seq，只能看到此 seq 之后的历史
	LastAckedMsgSeq  uint64                   `gorm:"not null;default:0"` // 客户端已确认收到的最大连续 seq（ACK 游标）
	LastReadMsgSeq   uint64                   `gorm:"not null;default:0"` // 用户已读的最大 seq（已读游标）
	CreatedAt        time.Time                `json:"-"`
	UpdatedAt        time.Time                `json:"-"`
}

func (c Conversation) IsSingle() bool {
	return c.Type == ConversationTypeSingle
}

func (c Conversation) IsGroup() bool {
	return c.Type == ConversationTypeGroup
}

func (c Conversation) IsActive() bool {
	return c.Status == ConversationStatusActive
}

func (c Conversation) SingleKeyValue() string {
	if c.SingleKey == nil {
		return ""
	}
	return *c.SingleKey
}

func (m ConversationMember) IsActive() bool {
	return m.Status == ConversationMemberStatusActive
}
