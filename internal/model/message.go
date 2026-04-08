package model

import "time"

type ChatMessage struct {
	ID             uint64    `gorm:"primaryKey;index:idx_chat_message_conversation_seq,priority:2;index:idx_chat_message_conversation_to_seq,priority:3" json:"id"`
	MsgID          string    `gorm:"size:64;uniqueIndex;not null" json:"msg_id"`
	ConversationID string    `gorm:"size:32;not null;index:idx_chat_message_conversation_seq,priority:1;index:idx_chat_message_conversation_to_seq,priority:1" json:"conversation_id"`
	From           uint64    `gorm:"not null;index:idx_chat_message_pair_time,priority:1" json:"from"`
	To             uint64    `gorm:"not null;index:idx_chat_message_pair_time,priority:2;index:idx_chat_message_conversation_to_seq,priority:2" json:"to"`
	SendTime       int64     `gorm:"not null;index:idx_chat_message_pair_time,priority:3" json:"send_time"`
	Content        string    `gorm:"type:text;not null" json:"content"`
	CreatedAt      time.Time `json:"-"`
}

func (ChatMessage) TableName() string {
	return "chat_msgs"
}
