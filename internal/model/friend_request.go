package model

import "time"

type FriendRequestStatus uint8

const (
	FriendRequestPending  FriendRequestStatus = 1
	FriendRequestAccepted FriendRequestStatus = 2
	FriendRequestRejected FriendRequestStatus = 3
)

type FriendRequest struct {
	ID          uint64              `gorm:"primaryKey"`
	RequesterID uint64              `gorm:"not null;index:idx_friend_request_requester_status,priority:1;index:idx_friend_request_pair_status,priority:1"`
	ReceiverID  uint64              `gorm:"not null;index:idx_friend_request_receiver_status,priority:1;index:idx_friend_request_pair_status,priority:2"`
	Status      FriendRequestStatus `gorm:"type:tinyint unsigned;not null;default:1;index:idx_friend_request_requester_status,priority:2;index:idx_friend_request_receiver_status,priority:2;index:idx_friend_request_pair_status,priority:3"`
	Message     string              `gorm:"size:255;not null;default:''"`
	HandledAt   *time.Time          `json:"-"`
	CreatedAt   time.Time           `json:"-"`
	UpdatedAt   time.Time           `json:"-"`
}

func (r FriendRequest) IsPending() bool {
	return r.Status == FriendRequestPending
}
