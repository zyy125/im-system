package dto

import "github.com/zyy125/im-system/internal/model"

type SendFriendRequestReq struct {
	Username string `json:"username" binding:"required"`
	Message  string `json:"message"`
}

type FriendRequestUserResp struct {
	UserID   uint64 `json:"user_id"`
	Avatar   string `json:"avatar"`
	Username string `json:"username"`
	Online   bool   `json:"online"`
}

type FriendRequestItemResp struct {
	ID        uint64                    `json:"id"`
	Status    model.FriendRequestStatus `json:"status"`
	Message   string                    `json:"message"`
	Requester FriendRequestUserResp     `json:"requester"`
	Receiver  FriendRequestUserResp     `json:"receiver"`
}

type FriendRequestListResp struct {
	Requests []FriendRequestItemResp `json:"requests"`
}
