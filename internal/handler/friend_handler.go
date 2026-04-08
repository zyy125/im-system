package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/zyy125/im-system/internal/handler/dto"
	"github.com/zyy125/im-system/internal/service"
)

type FriendHandler struct {
	friendService service.FriendService
}

func NewFriendHandler(friendService service.FriendService) *FriendHandler {
	return &FriendHandler{friendService: friendService}
}

// RemoveFriend 删除好友
// @Summary 删除好友
// @Description 删除当前用户与指定用户的好友关系
// @Tags 好友
// @Produce json
// @Security BearerAuth
// @Param id path int true "好友用户ID"
// @Success 200 {object} response.Response "删除成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未认证"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/friends/{id} [delete]
func (h *FriendHandler) RemoveFriend(c *gin.Context) {
	userID := currentUserID(c)
	friendID, ok := parseUintParam(c, "id", "invalid friend id")
	if !ok {
		return
	}

	if err := h.friendService.RemoveFriend(requestContext(c), userID, friendID); err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, nil)
}

// ListFriends 获取好友列表
// @Summary 获取好友列表
// @Description 返回当前用户的好友列表及在线状态
// @Tags 好友
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=dto.FriendListResp} "查询成功"
// @Failure 401 {object} response.Response "未认证"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/friends [get]
func (h *FriendHandler) ListFriends(c *gin.Context) {
	userID := currentUserID(c)
	friends, err := h.friendService.ListFriends(requestContext(c), userID)
	if err != nil {
		respondError(c, err)
		return
	}

	res := dto.FriendListResp{Friends: make([]dto.FriendInfoResp, 0, len(friends))}
	for _, f := range friends {
		res.Friends = append(res.Friends, buildFriendInfoResp(f))
	}
	respondOK(c, res)
}
