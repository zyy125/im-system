package model

import (
	"encoding/json"
	"time"
)

// Message 是消息的持久化模型。
// MsgID 由客户端生成（幂等键），Seq 由服务端分配（会话内单调递增）。
type Message struct {
	ID             uint64          `gorm:"primaryKey" json:"id"`
	MsgID          string          `gorm:"size:64;uniqueIndex;not null" json:"msg_id"`                                                                                                                                                                    // 客户端生成的幂等 ID，用于去重
	ConversationID uint64          `gorm:"not null;index:idx_message_conversation_seq,priority:1;index:idx_message_conversation_time,priority:1" json:"conversation_id"`
	Seq            uint64          `gorm:"not null;uniqueIndex:idx_message_conversation_seq,priority:2;index:idx_message_conversation_time,priority:3" json:"seq"` // 会话内单调递增序列号，由 SeqAllocator 分配
	Type           MessageType     `gorm:"type:tinyint unsigned;not null;default:1;index" json:"type"`
	Event          MessageEvent    `gorm:"size:64;not null;default:'';index" json:"event"` // 系统消息事件类型，普通文本消息为空
	From           uint64          `gorm:"not null;index:idx_message_conversation_sender_time,priority:2" json:"from"`
	SendTime       int64           `gorm:"not null;index:idx_message_conversation_time,priority:2;index:idx_message_conversation_sender_time,priority:3" json:"send_time"` // Unix 毫秒时间戳
	Content        string          `gorm:"type:text;not null" json:"content"`
	Extra          json.RawMessage `gorm:"type:json" json:"extra,omitempty" swaggertype:"object"` // 系统消息携带的结构化附加数据
	CreatedAt      time.Time       `json:"-"`
}

// MessageType 区分普通文本消息和系统事件消息。
type MessageType uint8

const (
	MessageTypeText   MessageType = 1 // 用户发送的文本消息
	MessageTypeSystem MessageType = 2 // 系统自动生成的事件消息（如建群、改名）
)

// MessageEvent 标识系统消息的具体事件类型，普通文本消息保持空字符串。
type MessageEvent string

const (
	MessageEventNone               MessageEvent = ""
	MessageEventGroupCreated       MessageEvent = "group_created"
	MessageEventGroupRenamed       MessageEvent = "group_renamed"
	MessageEventGroupMembersJoined MessageEvent = "group_members_joined"
	MessageEventGroupMemberRemoved MessageEvent = "group_member_removed"
	MessageEventGroupMemberLeft    MessageEvent = "group_member_left"
	MessageEventGroupDismissed     MessageEvent = "group_dismissed"
)

func (Message) TableName() string {
	return "messages"
}
