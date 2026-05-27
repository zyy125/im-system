package model

import "time"

// Friend 表示两个用户之间的好友关系（双向存储：A->B 和 B->A 各一条）。
// ConversationID 关联该好友对应的单聊会话。
type Friend struct {
	ID             uint64    `gorm:"primaryKey"`
	UserID         uint64    `gorm:"not null;uniqueIndex:idx_user_friend;index"`
	FriendID       uint64    `gorm:"not null;uniqueIndex:idx_user_friend;index"`
	ConversationID uint64    `gorm:"not null;index"` // 关联的单聊会话 ID
	CreatedAt      time.Time `json:"-"`
}
