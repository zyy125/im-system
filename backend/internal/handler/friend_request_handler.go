package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/zyy125/im-system/internal/handler/dto"
	"github.com/zyy125/im-system/internal/service"
)

type FriendRequestHandler struct {
	friendRequestService service.FriendRequestService
	userService          service.UserService
}

func NewFriendRequestHandler(friendRequestService service.FriendRequestService, userService service.UserService) *FriendRequestHandler {
	return &FriendRequestHandler{
		friendRequestService: friendRequestService,
		userService:          userService,
	}
}

// Send 发送好友申请
// @Summary 发送好友申请
// @Description 向指定 user_id 用户发送好友申请；若存在反向待处理申请，则自动同意
// @Tags 好友申请
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "目标用户 user_id"
// @Param req body dto.SendFriendRequestReq false "附言"
// @Success 200 {object} response.Response "发送成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未认证"
// @Failure 404 {object} response.Response "目标用户不存在"
// @Failure 409 {object} response.Response "申请状态冲突"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/friend-requests/{id} [post]
func (h *FriendRequestHandler) Send(c *gin.Context) {
	userPK := currentUserPK(c)
	targetUserID, ok := parseUintParam(c, "id", "invalid target user id")
	if !ok {
		return
	}

	var req dto.SendFriendRequestReq
	if !bindOptionalJSON(c, &req) {
		return
	}

	result, err := h.friendRequestService.Send(requestContext(c), userPK, targetUserID, req.Message)
	if err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, gin.H{"result": result})
}

// Accept 同意好友申请
// @Summary 同意好友申请
// @Description 同意指定的好友申请，并建立好友关系与单聊会话
// @Tags 好友申请
// @Produce json
// @Security BearerAuth
// @Param id path int true "好友申请ID"
// @Success 200 {object} response.Response "处理成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未认证"
// @Failure 403 {object} response.Response "无权限"
// @Failure 404 {object} response.Response "好友申请不存在"
// @Failure 409 {object} response.Response "好友申请状态不合法"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/friend-requests/{id}/accept [post]
func (h *FriendRequestHandler) Accept(c *gin.Context) {
	userPK := currentUserPK(c)
	requestID, ok := parseUintParam(c, "id", "invalid request id")
	if !ok {
		return
	}

	if err := h.friendRequestService.Accept(requestContext(c), userPK, requestID); err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, nil)
}

// Reject 拒绝好友申请
// @Summary 拒绝好友申请
// @Description 拒绝指定的好友申请
// @Tags 好友申请
// @Produce json
// @Security BearerAuth
// @Param id path int true "好友申请ID"
// @Success 200 {object} response.Response "处理成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未认证"
// @Failure 403 {object} response.Response "无权限"
// @Failure 404 {object} response.Response "好友申请不存在"
// @Failure 409 {object} response.Response "好友申请状态不合法"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/friend-requests/{id}/reject [post]
func (h *FriendRequestHandler) Reject(c *gin.Context) {
	userPK := currentUserPK(c)
	requestID, ok := parseUintParam(c, "id", "invalid request id")
	if !ok {
		return
	}

	if err := h.friendRequestService.Reject(requestContext(c), userPK, requestID); err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, nil)
}

// Incoming 获取收到的好友申请
// @Summary 获取收到的好友申请
// @Description 返回当前用户收到的待处理好友申请列表
// @Tags 好友申请
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=dto.FriendRequestListResp} "查询成功"
// @Failure 401 {object} response.Response "未认证"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/friend-requests/incoming [get]
func (h *FriendRequestHandler) Incoming(c *gin.Context) {
	userPK := currentUserPK(c)
	reqs, err := h.friendRequestService.ListIncoming(requestContext(c), userPK)
	if err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, dto.FriendRequestListResp{
		Requests: buildFriendRequestItems(reqs),
	})
}

// Outgoing 获取发出的好友申请
// @Summary 获取发出的好友申请
// @Description 返回当前用户发出的待处理好友申请列表
// @Tags 好友申请
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=dto.FriendRequestListResp} "查询成功"
// @Failure 401 {object} response.Response "未认证"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/friend-requests/outgoing [get]
func (h *FriendRequestHandler) Outgoing(c *gin.Context) {
	userPK := currentUserPK(c)
	reqs, err := h.friendRequestService.ListOutgoing(requestContext(c), userPK)
	if err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, dto.FriendRequestListResp{
		Requests: buildFriendRequestItems(reqs),
	})
}

func buildFriendRequestItems(reqs []service.FriendRequestSummary) []dto.FriendRequestItemResp {
	res := make([]dto.FriendRequestItemResp, 0, len(reqs))
	for _, req := range reqs {
		res = append(res, dto.FriendRequestItemResp{
			ID:        req.ID,
			Status:    req.Status,
			Message:   req.Message,
			Requester: buildFriendRequestUserResp(req.Requester),
			Receiver:  buildFriendRequestUserResp(req.Receiver),
		})
	}
	return res
}
