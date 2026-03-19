package model

type ChatMsg struct {
	ID       uint64 `json:"id" gorm:"primaryKey"`
	MsgID    string `json:"msg_id" gorm:"unique;not null"`
	From     uint64 `json:"from" gorm:"not null"`
	To       uint64 `json:"to" gorm:"not null"`
	SendTime int64  `json:"send_time" gorm:"index;not null"`
	Content  string `json:"content" gorm:"not null"`
}
