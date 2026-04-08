package dto

import "github.com/zyy125/im-system/internal/model"

type SendFriendRequestReq struct {
	Message string `json:"message"`
}

type FriendRequestUserResp struct {
	ID       uint64 `json:"id"`
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
