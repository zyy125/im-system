package apperr

import "strings"

func InvalidArgument(message string) *Error {
	return New(CodeInvalidArgument, message)
}

func InvalidBody(message string) *Error {
	return New(CodeInvalidBody, message)
}

func Unauthorized(code Code, message string) *Error {
	return New(code, message)
}

func Forbidden(code Code, message string) *Error {
	return New(code, message)
}

func NotFound(code Code, message string) *Error {
	return New(code, message)
}

func Conflict(code Code, message string) *Error {
	return New(code, message)
}

func Internal(message string, cause error) *Error {
	return Wrap(CodeInternal, message, cause)
}

func Required(fields ...string) *Error {
	return InvalidArgument(joinFields(fields...) + " are required")
}

func RequiredOne(field string) *Error {
	return InvalidArgument(field + " is required")
}

func InvalidID(name string) *Error {
	return InvalidArgument("invalid " + name)
}

func CredentialsRequired() *Error {
	return New(CodeAuthCredentialsRequired, "username and password are required")
}

func InvalidCredentials() *Error {
	return Unauthorized(CodeAuthInvalidCredentials, "username or password is incorrect")
}

func TokenMissing() *Error {
	return Unauthorized(CodeAuthTokenMissing, "token is required")
}

func TokenInvalid() *Error {
	return Unauthorized(CodeAuthTokenInvalid, "token is invalid")
}

func TokenBlacklisted() *Error {
	return Unauthorized(CodeAuthTokenBlacklisted, "token is blacklisted")
}

func UserNotFound() *Error {
	return NotFound(CodeUserNotFound, "user not found")
}

func UserAlreadyExists() *Error {
	return Conflict(CodeUserAlreadyExist, "username already exists")
}

func FriendCannotAddSelf() *Error {
	return New(CodeFriendCannotAddSelf, "cannot add self")
}

func FriendNotFriends() *Error {
	return Forbidden(CodeFriendNotFriends, "not friends")
}

func FriendRequestNotPending() *Error {
	return Conflict(CodeFriendRequestNotPending, "friend request is not pending")
}

func FriendRequestNoPermission(action string) *Error {
	return Forbidden(CodeFriendRequestNoPermission, "no permission to "+action+" this request")
}

func FriendRequestNotFound() *Error {
	return NotFound(CodeFriendRequestNotFound, "friend request not found")
}

func ConversationInvalidSingleKey() *Error {
	return New(CodeConversationInvalidSingleKey, "conversation single key is invalid")
}

func ConversationMemberNotFound() *Error {
	return NotFound(CodeConversationMemberNotFound, "conversation member not found")
}

func ConversationMemberUpdateFailed() *Error {
	return New(CodeConversationMemberUpdateFail, "conversation member update failed")
}

func ConversationNotAccessible() *Error {
	return Forbidden(CodeConversationNotAccessible, "conversation is not accessible")
}

func MessageInvalidPeerID() *Error {
	return New(CodeMessageInvalidPeerID, "invalid peer_id")
}

func MessageInvalidPayload() *Error {
	return New(CodeMessageInvalidPayload, "invalid message")
}

func MessageIDRequired() *Error {
	return New(CodeMessageIDRequired, "msg_id is required")
}

func MessageConversationRequired() *Error {
	return New(CodeMessageConversationRequired, "conversation_id is required")
}

func MessageNotFound() *Error {
	return NotFound(CodeMessageNotFound, "message not found")
}

func MessageNotReadable() *Error {
	return Forbidden(CodeMessageNotReadable, "message is not readable by current user")
}

func MessageNotDelivered() *Error {
	return Conflict(CodeMessageNotDelivered, "message has not been delivered to current user yet")
}

func joinFields(fields ...string) string {
	filtered := make([]string, 0, len(fields))
	for _, field := range fields {
		if strings.TrimSpace(field) == "" {
			continue
		}
		filtered = append(filtered, field)
	}
	return strings.Join(filtered, " and ")
}
