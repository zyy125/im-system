package dto

type CheckUserOnlineResp struct {
	UserID uint64 `json:"user_id"`
	Online bool   `json:"online"`
}

type UserInfoResp struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
	Online   bool   `json:"online"`
}

type FriendInfoResp struct {
	UserID   uint64 `json:"user_id"`
	Username string `json:"username"`
	Online   bool   `json:"online"`
}

type FriendListResp struct {
	Friends []FriendInfoResp `json:"friends"`
}
