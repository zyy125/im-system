package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/zyy125/im-system/internal/handler/dto"
	"github.com/zyy125/im-system/internal/service"
)

type ConversationHandler struct {
	conversationService service.ConversationService
}

func NewConversationHandler(conversationService service.ConversationService) *ConversationHandler {
	return &ConversationHandler{conversationService: conversationService}
}

// List 获取会话列表
// @Summary 获取会话列表
// @Description 返回当前用户可见的会话列表、未读数和最近一条消息
// @Tags 会话
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=dto.ConversationListResp} "查询成功"
// @Failure 401 {object} response.Response "未认证"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/conversations [get]
func (h *ConversationHandler) List(c *gin.Context) {
	userID := currentUserID(c)

	conversations, err := h.conversationService.ListConversations(requestContext(c), userID)
	if err != nil {
		respondError(c, err)
		return
	}

	res := make([]dto.ConversationItemResp, 0, len(conversations))
	for _, conversation := range conversations {
		res = append(res, buildConversationItemResp(conversation))
	}

	respondOK(c, dto.ConversationListResp{
		Conversations: res,
	})
}

// Hide 隐藏会话
// @Summary 隐藏会话
// @Description 将指定会话从当前用户的会话列表中隐藏
// @Tags 会话
// @Produce json
// @Security BearerAuth
// @Param id path int true "会话ID"
// @Success 200 {object} response.Response "隐藏成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未认证"
// @Failure 404 {object} response.Response "会话成员不存在"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/conversations/{id}/hide [post]
func (h *ConversationHandler) Hide(c *gin.Context) {
	userID := currentUserID(c)
	conversationID, ok := parseUintParam(c, "id", "invalid conversation id")
	if !ok {
		return
	}

	if err := h.conversationService.HideConversation(requestContext(c), userID, conversationID); err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, nil)
}

// OpenDirect 打开单聊会话
// @Summary 打开单聊会话
// @Description 与指定好友打开或恢复单聊会话
// @Tags 会话
// @Produce json
// @Security BearerAuth
// @Param id path int true "好友用户ID"
// @Success 200 {object} response.Response{data=dto.OpenConversationResp} "打开成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未认证"
// @Failure 403 {object} response.Response "非好友不可打开"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/conversations/direct/{id}/open [post]
func (h *ConversationHandler) OpenDirect(c *gin.Context) {
	userID := currentUserID(c)
	friendID, ok := parseUintParam(c, "id", "invalid friend id")
	if !ok {
		return
	}

	conversation, err := h.conversationService.OpenDirectConversation(requestContext(c), userID, friendID)
	if err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, dto.OpenConversationResp{
		Conversation: buildConversationItemResp(conversation),
	})
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
