package handler

import (
	"context"
	"errors"
	"io"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/handler/dto"
	"github.com/zyy125/im-system/internal/model"
	"github.com/zyy125/im-system/internal/service"
	"github.com/zyy125/im-system/pkg/response"
)

func requestContext(c *gin.Context) context.Context {
	return c.Request.Context()
}

func currentUserID(c *gin.Context) uint64 {
	return c.GetUint64("userID")
}

func currentTokenExpiresAt(c *gin.Context) time.Time {
	value, ok := c.Get("tokenExpiresAt")
	if !ok {
		return time.Time{}
	}
	expiresAt, _ := value.(time.Time)
	return expiresAt
}

func respondOK(c *gin.Context, data any) {
	response.Success(c, 200, data)
}

func respondCreated(c *gin.Context, data any) {
	response.Success(c, 201, data)
}

func respondError(c *gin.Context, err error) {
	response.FailError(c, err)
}

func bindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		respondError(c, apperr.InvalidBody("parameter validation error"))
		return false
	}
	return true
}

func bindOptionalJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil && !errors.Is(err, io.EOF) {
		respondError(c, apperr.InvalidBody("parameter validation error"))
		return false
	}
	return true
}

func parseUintParam(c *gin.Context, name, invalidMessage string) (uint64, bool) {
	value, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || value == 0 {
		respondError(c, apperr.InvalidArgument(invalidMessage))
		return 0, false
	}
	return value, true
}

func parseUintQueryError(c *gin.Context, key string, err error) (uint64, bool) {
	value, parseErr := strconv.ParseUint(c.Query(key), 10, 64)
	if parseErr != nil || value == 0 {
		respondError(c, err)
		return 0, false
	}
	return value, true
}

func queryInt(c *gin.Context, key string, defaultValue int) int {
	value := c.Query(key)
	if value == "" {
		return defaultValue
	}

	n, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return n
}

func buildUserInfoResp(user model.User, online bool) dto.UserInfoResp {
	return dto.UserInfoResp{
		ID:       user.ID,
		Username: user.Username,
		Online:   online,
	}
}

func buildFriendInfoResp(friend service.FriendInfo) dto.FriendInfoResp {
	return dto.FriendInfoResp{
		UserID:         friend.UserID,
		Username:       friend.Username,
		Online:         friend.Online,
		ConversationID: friend.ConversationID,
	}
}

func buildFriendRequestUserResp(user service.FriendRequestUser) dto.FriendRequestUserResp {
	return dto.FriendRequestUserResp{
		ID:       user.ID,
		Username: user.Username,
		Online:   user.Online,
	}
}

func buildConversationItemResp(conversation service.ConversationSummary) dto.ConversationItemResp {
	item := dto.ConversationItemResp{
		ID:          conversation.ID,
		Type:        conversation.Type,
		Name:        conversation.Name,
		UnreadCount: conversation.UnreadCount,
		LastMessage: conversation.LastMessage,
	}
	if conversation.Peer != nil {
		item.Peer = &dto.ConversationPeerResp{
			ID:       conversation.Peer.ID,
			Username: conversation.Peer.Username,
			Online:   conversation.Peer.Online,
		}
	}
	return item
}

func buildGroupDetailResp(group service.GroupDetail) dto.GroupDetailResp {
	return dto.GroupDetailResp{
		ID:          group.ID,
		Name:        group.Name,
		Avatar:      group.Avatar,
		OwnerID:     group.OwnerID,
		Status:      group.Status,
		MyRole:      group.MyRole,
		MemberCount: group.MemberCount,
	}
}
