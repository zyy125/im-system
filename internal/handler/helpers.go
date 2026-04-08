package handler

import (
	"context"
	"errors"
	"io"
	"strconv"

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

func queryInt64(c *gin.Context, key string, defaultValue int64) int64 {
	value := c.Query(key)
	if value == "" {
		return defaultValue
	}

	n, err := strconv.ParseInt(value, 10, 64)
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
		UserID:   friend.UserID,
		Username: friend.Username,
		Online:   friend.Online,
	}
}

func buildFriendRequestUserResp(user service.FriendRequestUser) dto.FriendRequestUserResp {
	return dto.FriendRequestUserResp{
		ID:       user.ID,
		Username: user.Username,
		Online:   user.Online,
	}
}
