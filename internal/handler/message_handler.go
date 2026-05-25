package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/handler/dto"
	"github.com/zyy125/im-system/internal/service"
)

type MessageHandler struct {
	messageQueryService     service.MessageService
	conversationSyncService service.ConversationService
}

func NewMessageHandler(
	messageQueryService service.MessageService,
	conversationSyncService service.ConversationService,
) *MessageHandler {
	return &MessageHandler{
		messageQueryService:     messageQueryService,
		conversationSyncService: conversationSyncService,
	}
}

// History 获取消息历史
// @Summary 获取消息历史
// @Description 按会话 ID 和 before_seq 查询历史消息，返回顺序为 seq 从小到大
// @Tags 消息
// @Produce json
// @Security BearerAuth
// @Param conversation_id query int true "会话ID"
// @Param limit query int false "返回条数上限，默认20，最大100"
// @Param before_seq query int false "查询该seq之前的更早消息"
// @Success 200 {object} response.Response{data=dto.MessageHistoryResp} "查询成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未认证"
// @Failure 403 {object} response.Response "无权访问该会话"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/messages/history [get]
func (h *MessageHandler) History(c *gin.Context) {
	userID := currentUserID(c)
	conversationID, ok := parseUintQueryError(c, "conversation_id", apperr.InvalidID("conversation_id"))
	if !ok {
		return
	}

	limit := queryInt(c, "limit", 20)
	var beforeSeq uint64
	if raw := c.Query("before_seq"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || parsed == 0 {
			respondError(c, apperr.InvalidArgument("invalid before_seq"))
			return
		}
		beforeSeq = parsed
	}

	msgs, hasMore, err := h.messageQueryService.ListConversationHistory(requestContext(c), userID, conversationID, limit, beforeSeq)
	if err != nil {
		respondError(c, err)
		return
	}

	var nextBeforeSeq uint64
	if hasMore && len(msgs) > 0 {
		nextBeforeSeq = msgs[0].Seq
	}

	respondOK(c, dto.MessageHistoryResp{
		Messages:      msgs,
		HasMore:       hasMore,
		NextBeforeSeq: nextBeforeSeq,
	})
}

// Sync 同步消息
// @Summary 同步消息
// @Description 按会话 ID 和 after_seq 补拉后续消息，返回顺序为 seq 从小到大
// @Tags 消息
// @Produce json
// @Security BearerAuth
// @Param conversation_id query int true "会话ID"
// @Param after_seq query int false "起始seq，不返回该seq本身"
// @Param limit query int false "返回条数上限，默认100，最大200"
// @Success 200 {object} response.Response{data=dto.MessageSyncResp} "同步成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未认证"
// @Failure 403 {object} response.Response "无权访问该会话"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/messages/sync [get]
func (h *MessageHandler) Sync(c *gin.Context) {
	userID := currentUserID(c)
	conversationID, ok := parseUintQueryError(c, "conversation_id", apperr.InvalidID("conversation_id"))
	if !ok {
		return
	}

	limit := queryInt(c, "limit", 100)
	var afterSeq uint64
	if raw := c.Query("after_seq"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			respondError(c, apperr.InvalidArgument("invalid after_seq"))
			return
		}
		afterSeq = parsed
	}

	msgs, hasMore, err := h.messageQueryService.SyncConversation(requestContext(c), userID, conversationID, afterSeq, limit)
	if err != nil {
		respondError(c, err)
		return
	}

	var maxReturnedSeq uint64
	for _, msg := range msgs {
		if msg.Seq > maxReturnedSeq {
			maxReturnedSeq = msg.Seq
		}
	}

	respondOK(c, dto.MessageSyncResp{
		Messages:       msgs,
		HasMore:        hasMore,
		MaxReturnedSeq: maxReturnedSeq,
	})
}
