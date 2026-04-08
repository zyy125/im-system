package apperr

import "net/http"

type Code string

const (
	CodeOK Code = "ok"

	CodeInvalidArgument    Code = "common.invalid_argument"
	CodeInvalidBody        Code = "common.invalid_body"
	CodeUnauthorized       Code = "common.unauthorized"
	CodeForbidden          Code = "common.forbidden"
	CodeNotFound           Code = "common.not_found"
	CodeConflict           Code = "common.conflict"
	CodeInternal           Code = "common.internal"
	CodeRateLimited        Code = "common.rate_limited"
	CodeTimeout            Code = "common.timeout"
	CodeServiceUnavailable Code = "common.service_unavailable"

	CodeAuthCredentialsRequired Code = "auth.credentials_required"
	CodeAuthInvalidCredentials  Code = "auth.invalid_credentials"
	CodeAuthTokenMissing        Code = "auth.token_missing"
	CodeAuthTokenInvalid        Code = "auth.token_invalid"
	CodeAuthTokenExpired        Code = "auth.token_expired"
	CodeAuthTokenBlacklisted    Code = "auth.token_blacklisted"

	CodeUserNotFound     Code = "user.not_found"
	CodeUserAlreadyExist Code = "user.already_exists"

	CodeFriendCannotAddSelf Code = "friend.cannot_add_self"
	CodeFriendNotFriends    Code = "friend.not_friends"
	CodeFriendAlreadyExist  Code = "friend.already_exists"

	CodeFriendRequestAlreadyPending Code = "friend_request.already_pending"
	CodeFriendRequestAlreadyFriends Code = "friend_request.already_friends"
	CodeFriendRequestNotPending     Code = "friend_request.not_pending"
	CodeFriendRequestNoPermission   Code = "friend_request.no_permission"
	CodeFriendRequestNotFound       Code = "friend_request.not_found"

	CodeConversationNotFound         Code = "conversation.not_found"
	CodeConversationInvalidSingleKey Code = "conversation.invalid_single_key"
	CodeConversationMemberNotFound   Code = "conversation.member_not_found"
	CodeConversationMemberUpdateFail Code = "conversation.member_update_failed"
	CodeConversationNotAccessible    Code = "conversation.not_accessible"

	CodeMessageInvalidPeerID        Code = "message.invalid_peer_id"
	CodeMessageInvalidPayload       Code = "message.invalid_payload"
	CodeMessageIDRequired           Code = "message.msg_id_required"
	CodeMessageConversationRequired Code = "message.conversation_required"
	CodeMessageNotFound             Code = "message.not_found"
	CodeMessageNotReadable          Code = "message.not_readable"
	CodeMessageNotDelivered         Code = "message.not_delivered"
)

var httpStatusByCode = map[Code]int{
	CodeOK:                 http.StatusOK,
	CodeInvalidArgument:    http.StatusBadRequest,
	CodeInvalidBody:        http.StatusBadRequest,
	CodeUnauthorized:       http.StatusUnauthorized,
	CodeForbidden:          http.StatusForbidden,
	CodeNotFound:           http.StatusNotFound,
	CodeConflict:           http.StatusConflict,
	CodeInternal:           http.StatusInternalServerError,
	CodeRateLimited:        http.StatusTooManyRequests,
	CodeTimeout:            http.StatusGatewayTimeout,
	CodeServiceUnavailable: http.StatusServiceUnavailable,

	CodeAuthCredentialsRequired: http.StatusBadRequest,
	CodeAuthInvalidCredentials:  http.StatusUnauthorized,
	CodeAuthTokenMissing:        http.StatusUnauthorized,
	CodeAuthTokenInvalid:        http.StatusUnauthorized,
	CodeAuthTokenExpired:        http.StatusUnauthorized,
	CodeAuthTokenBlacklisted:    http.StatusUnauthorized,

	CodeUserNotFound:     http.StatusNotFound,
	CodeUserAlreadyExist: http.StatusConflict,

	CodeFriendCannotAddSelf: http.StatusBadRequest,
	CodeFriendNotFriends:    http.StatusForbidden,
	CodeFriendAlreadyExist:  http.StatusConflict,

	CodeFriendRequestAlreadyPending: http.StatusConflict,
	CodeFriendRequestAlreadyFriends: http.StatusConflict,
	CodeFriendRequestNotPending:     http.StatusConflict,
	CodeFriendRequestNoPermission:   http.StatusForbidden,
	CodeFriendRequestNotFound:       http.StatusNotFound,

	CodeConversationNotFound:         http.StatusNotFound,
	CodeConversationInvalidSingleKey: http.StatusInternalServerError,
	CodeConversationMemberNotFound:   http.StatusNotFound,
	CodeConversationMemberUpdateFail: http.StatusInternalServerError,
	CodeConversationNotAccessible:    http.StatusForbidden,

	CodeMessageInvalidPeerID:        http.StatusBadRequest,
	CodeMessageInvalidPayload:       http.StatusBadRequest,
	CodeMessageIDRequired:           http.StatusBadRequest,
	CodeMessageConversationRequired: http.StatusBadRequest,
	CodeMessageNotFound:             http.StatusNotFound,
	CodeMessageNotReadable:          http.StatusForbidden,
	CodeMessageNotDelivered:         http.StatusConflict,
}

func HTTPStatus(code Code) int {
	if status, ok := httpStatusByCode[code]; ok {
		return status
	}
	return http.StatusInternalServerError
}
