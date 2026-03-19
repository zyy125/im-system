package dto

type CheckUserOnlineRes struct {
	UserID uint64 `json:"userID"`
	Online bool   `json:"online"`
}
