package model

import "time"

type Friend struct {
	ID        uint64    `gorm:"primaryKey"`
	UserID    uint64    `gorm:"not null;uniqueIndex:idx_user_friend;index"`
	FriendID  uint64    `gorm:"not null;uniqueIndex:idx_user_friend;index"`
	CreatedAt time.Time `json:"-"`
}
