package model

import "time"

// FriendRequestStatus 标识好友申请的当前状态。
type FriendRequestStatus uint8

const (
	FriendRequestPending  FriendRequestStatus = 1 // 待处理
	FriendRequestAccepted FriendRequestStatus = 2 // 已同意
	FriendRequestRejected FriendRequestStatus = 3 // 已拒绝
)

// FriendRequest 记录一条好友申请。
// 同一对用户之间同方向只允许存在一条 Pending 记录。
type FriendRequest struct {
	ID          uint64              `gorm:"primaryKey"`
	RequesterID uint64              `gorm:"not null;index:idx_friend_request_requester_status,priority:1;index:idx_friend_request_pair_status,priority:1"`
	ReceiverID  uint64              `gorm:"not null;index:idx_friend_request_receiver_status,priority:1;index:idx_friend_request_pair_status,priority:2"`
	Status      FriendRequestStatus `gorm:"type:tinyint unsigned;not null;default:1;index:idx_friend_request_requester_status,priority:2;index:idx_friend_request_receiver_status,priority:2;index:idx_friend_request_pair_status,priority:3"`
	Message     string              `gorm:"size:255;not null;default:''"` // 申请附言
	HandledAt   *time.Time          `json:"-"` // 处理时间，Pending 时为 nil
	CreatedAt   time.Time           `json:"-"`
	UpdatedAt   time.Time           `json:"-"`
}

func (r FriendRequest) IsPending() bool {
	return r.Status == FriendRequestPending
}
