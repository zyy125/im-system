package model

import "time"

// User 是用户的持久化模型，Password 字段在序列化时隐藏。
type User struct {
	ID        uint64    `gorm:"primaryKey"`
	PublicID  uint64    `gorm:"uniqueIndex;not null"`
	Username  string    `gorm:"size:64;not null;index"`
	Avatar    string    `gorm:"size:255;not null;default:''"`
	Password  string    `gorm:"size:255;not null" json:"-"` // bcrypt 哈希，不对外暴露
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}
