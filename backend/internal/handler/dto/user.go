package dto

type CheckUserOnlineResp struct {
	UserID uint64 `json:"user_id"`
	Avatar string `json:"avatar"`
	Online bool   `json:"online"`
}

type UserInfoResp struct {
	UserID   uint64 `json:"user_id"`
	Avatar   string `json:"avatar"`
	Username string `json:"username"`
	Online   bool   `json:"online"`
}

type FriendInfoResp struct {
	UserID         uint64 `json:"user_id"`
	Avatar         string `json:"avatar"`
	Username       string `json:"username"`
	Online         bool   `json:"online"`
	ConversationID uint64 `json:"conversation_id"`
}

type FriendListResp struct {
	Friends []FriendInfoResp `json:"friends"`
}

type UserAvatarResp struct {
	Avatar string `json:"avatar"`
}
