package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/handler/dto"
	"github.com/zyy125/im-system/internal/service"
)

type MessageHandler struct {
	messageService      service.MessageService
	friendService       service.FriendService
	conversationService service.ConversationService
}

func NewMessageHandler(
	messageService service.MessageService,
	friendService service.FriendService,
	conversationService service.ConversationService,
) *MessageHandler {
	return &MessageHandler{
		messageService:      messageService,
		friendService:       friendService,
		conversationService: conversationService,
	}
}

// History 获取消息历史
// @Summary 获取消息历史
// @Description 查询当前用户与指定好友之间的历史消息，支持按消息ID向更早消息分页
// @Tags 消息
// @Produce json
// @Security BearerAuth
// @Param peer_id query int true "好友用户ID"
// @Param limit query int false "返回条数上限，默认20，最大100"
// @Param before_id query int false "查询该消息ID之前的更早消息"
// @Success 200 {object} response.Response{data=dto.MessageHistoryResp} "查询成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未认证"
// @Failure 403 {object} response.Response "非好友不可查询"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/messages/history [get]
func (h *MessageHandler) History(c *gin.Context) {
	userID := currentUserID(c)
	peerID, ok := parseUintQueryError(c, "peer_id", apperr.MessageInvalidPeerID())
	if !ok {
		return
	}

	limit := queryInt(c, "limit", 20)
	var beforeID uint64
	if raw := c.Query("before_id"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || parsed == 0 {
			respondError(c, apperr.InvalidArgument("invalid before_id"))
			return
		}
		beforeID = parsed
	}

	ok, err := h.friendService.AreFriends(requestContext(c), userID, peerID)
	if err != nil {
		respondError(c, err)
		return
	}
	if !ok {
		respondError(c, apperr.FriendNotFriends())
		return
	}

	msgs, hasMore, err := h.messageService.ListHistory(requestContext(c), userID, peerID, limit, beforeID)
	if err != nil {
		respondError(c, err)
		return
	}

	var nextBeforeID uint64
	if hasMore && len(msgs) > 0 {
		nextBeforeID = msgs[0].ID
	}

	respondOK(c, dto.MessageHistoryResp{
		Messages:     msgs,
		HasMore:      hasMore,
		NextBeforeID: nextBeforeID,
	})
}

// MarkRead 标记消息已读
// @Summary 标记消息已读
// @Description 根据会话ID和消息ID推进当前用户的已读游标
// @Tags 消息
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param req body dto.MarkReadReq true "已读请求"
// @Success 200 {object} response.Response "标记成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未认证"
// @Failure 404 {object} response.Response "消息不存在"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/messages/read [post]
func (h *MessageHandler) MarkRead(c *gin.Context) {
	userID := currentUserID(c)

	var req dto.MarkReadReq
	if !bindJSON(c, &req) {
		return
	}
	if req.ConversationID == "" || req.MsgID == "" {
		respondError(c, apperr.Required("conversation_id", "msg_id"))
		return
	}

	if err := h.conversationService.MarkRead(requestContext(c), userID, req.ConversationID, req.MsgID); err != nil {
		respondError(c, err)
		return
	}

	respondOK(c, nil)
}
