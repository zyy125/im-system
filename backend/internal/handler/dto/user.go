package dto

type CheckUserOnlineResp struct {
	PublicID uint64 `json:"public_id"`
	Online   bool   `json:"online"`
}

type UserInfoResp struct {
	PublicID uint64 `json:"public_id"`
	Username string `json:"username"`
	Online   bool   `json:"online"`
}

type FriendInfoResp struct {
	PublicID       uint64 `json:"public_id"`
	Username       string `json:"username"`
	Online         bool   `json:"online"`
	ConversationID uint64 `json:"conversation_id"`
}

type FriendListResp struct {
	Friends []FriendInfoResp `json:"friends"`
}
