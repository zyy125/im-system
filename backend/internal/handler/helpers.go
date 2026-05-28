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

func currentUserPK(c *gin.Context) uint64 {
	return c.GetUint64("userID")
}

func currentSessionID(c *gin.Context) string {
	return c.GetString("sessionID")
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
		response.FailError(c, apperr.InvalidBody("parameter validation error"))
		return false
	}
	return true
}

func bindOptionalJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil && !errors.Is(err, io.EOF) {
		response.FailError(c, apperr.InvalidBody("parameter validation error"))
		return false
	}
	return true
}

func parseUintParam(c *gin.Context, name, invalidMessage string) (uint64, bool) {
	value, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || value == 0 {
		response.FailError(c, apperr.InvalidArgument(invalidMessage))
		return 0, false
	}
	return value, true
}

func parseUintQueryError(c *gin.Context, key string, err error) (uint64, bool) {
	value, parseErr := strconv.ParseUint(c.Query(key), 10, 64)
	if parseErr != nil || value == 0 {
		response.FailError(c, err)
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
		PublicID: user.PublicID,
		Username: user.Username,
		Online:   online,
	}
}

func buildFriendInfoResp(friend service.FriendInfo) dto.FriendInfoResp {
	return dto.FriendInfoResp{
		PublicID:       friend.UserID,
		Username:       friend.Username,
		Online:         friend.Online,
		ConversationID: friend.ConversationID,
	}
}

func buildFriendRequestUserResp(user service.FriendRequestUser) dto.FriendRequestUserResp {
	return dto.FriendRequestUserResp{
		PublicID: user.ID,
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
	}
	if conversation.Peer != nil {
		item.Peer = &dto.ConversationPeerResp{
			PublicID: conversation.Peer.ID,
			Username: conversation.Peer.Username,
			Online:   conversation.Peer.Online,
		}
	}
	return item
}

func buildMessageResp(msg model.Message, publicIDs map[uint64]uint64) dto.MessageResp {
	return dto.MessageResp{
		ID:             msg.ID,
		MsgID:          msg.MsgID,
		ConversationID: msg.ConversationID,
		Seq:            msg.Seq,
		Type:           msg.Type,
		Event:          msg.Event,
		FromPublicID:   publicIDForUser(msg.From, publicIDs),
		SendTime:       msg.SendTime,
		Content:        msg.Content,
		Extra:          msg.Extra,
	}
}

func buildMessageResps(msgs []model.Message, publicIDs map[uint64]uint64) []dto.MessageResp {
	items := make([]dto.MessageResp, 0, len(msgs))
	for _, msg := range msgs {
		items = append(items, buildMessageResp(msg, publicIDs))
	}
	return items
}

func collectMessageSenderIDs(msgs []model.Message) []uint64 {
	ids := make([]uint64, 0, len(msgs))
	seen := make(map[uint64]struct{}, len(msgs))
	for _, msg := range msgs {
		if msg.From == 0 {
			continue
		}
		if _, ok := seen[msg.From]; ok {
			continue
		}
		seen[msg.From] = struct{}{}
		ids = append(ids, msg.From)
	}
	return ids
}

func publicIDForUser(userID uint64, publicIDs map[uint64]uint64) uint64 {
	if publicID, ok := publicIDs[userID]; ok {
		return publicID
	}
	return 0
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
