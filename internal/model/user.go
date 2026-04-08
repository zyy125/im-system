package model

import "time"

type User struct {
	ID        uint64    `gorm:"primaryKey"`
	Username  string    `gorm:"size:64;uniqueIndex;not null"`
	Password  string    `gorm:"size:255;not null" json:"-"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}
