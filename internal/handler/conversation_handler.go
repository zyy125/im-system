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
	return &ConversationHandler{
		conversationService: conversationService,
	}
}

// List 获取会话列表
// @Summary 获取会话列表
// @Description 获取当前用户可见会话、未读数和最后一条消息摘要
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

	respondOK(c, dto.ConversationListResp{Conversations: res})
}

// Hide 隐藏会话
// @Summary 隐藏会话
// @Description 将指定会话从当前用户会话列表中隐藏
// @Tags 会话
// @Produce json
// @Security BearerAuth
// @Param id path int true "会话ID"
// @Success 200 {object} response.Response "操作成功"
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

// Open 打开会话
// @Summary 打开会话
// @Description 按会话ID打开会话，不可见会话会自动恢复显示
// @Tags 会话
// @Produce json
// @Security BearerAuth
// @Param id path int true "会话ID"
// @Success 200 {object} response.Response{data=dto.OpenConversationResp} "打开成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未认证"
// @Failure 403 {object} response.Response "无权访问该会话"
// @Failure 404 {object} response.Response "会话或成员不存在"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/conversations/{id}/open [post]
func (h *ConversationHandler) Open(c *gin.Context) {
	userID := currentUserID(c)
	conversationID, ok := parseUintParam(c, "id", "invalid conversation id")
	if !ok {
		return
	}

	conversation, err := h.conversationService.OpenConversation(requestContext(c), userID, conversationID)
	if err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, dto.OpenConversationResp{Conversation: buildConversationItemResp(conversation)})
}

// ListGroups 获取群聊列表
// @Summary 获取群聊列表
// @Description 返回当前用户仍为活跃成员的全部群聊，不受消息栏 visible 状态影响
// @Tags 会话
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=dto.ConversationListResp} "查询成功"
// @Failure 401 {object} response.Response "未认证"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/conversations/groups [get]
func (h *ConversationHandler) ListGroups(c *gin.Context) {
	userID := currentUserID(c)

	conversations, err := h.conversationService.ListGroups(requestContext(c), userID)
	if err != nil {
		respondError(c, err)
		return
	}

	res := make([]dto.ConversationItemResp, 0, len(conversations))
	for _, conversation := range conversations {
		res = append(res, buildConversationItemResp(conversation))
	}

	respondOK(c, dto.ConversationListResp{Conversations: res})
}

// CreateGroup 创建群聊
// @Summary 创建群聊
// @Description 创建一个群聊并可携带初始成员
// @Tags 会话
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param req body dto.CreateGroupReq true "创建群聊请求"
// @Success 201 {object} response.Response{data=dto.GroupConversationResp} "创建成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未认证"
// @Failure 409 {object} response.Response "群人数超限或状态冲突"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/conversations/groups [post]
func (h *ConversationHandler) CreateGroup(c *gin.Context) {
	userID := currentUserID(c)
	var req dto.CreateGroupReq
	if !bindJSON(c, &req) {
		return
	}

	conversation, err := h.conversationService.CreateGroup(requestContext(c), userID, req.Name, req.MemberIDs)
	if err != nil {
		respondError(c, err)
		return
	}
	respondCreated(c, dto.GroupConversationResp{Conversation: buildConversationItemResp(conversation)})
}

// GetGroupDetail 获取群详情
// @Summary 获取群详情
// @Description 获取指定群聊的基础信息和当前用户角色
// @Tags 会话
// @Produce json
// @Security BearerAuth
// @Param id path int true "群会话ID"
// @Success 200 {object} response.Response{data=dto.GroupDetailEnvelopeResp} "查询成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未认证"
// @Failure 403 {object} response.Response "无权访问该群"
// @Failure 404 {object} response.Response "群会话不存在"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/conversations/groups/{id} [get]
func (h *ConversationHandler) GetGroupDetail(c *gin.Context) {
	userID := currentUserID(c)
	conversationID, ok := parseUintParam(c, "id", "invalid conversation id")
	if !ok {
		return
	}

	group, err := h.conversationService.GetGroupDetail(requestContext(c), userID, conversationID)
	if err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, dto.GroupDetailEnvelopeResp{Group: buildGroupDetailResp(group)})
}

// ListGroupMembers 获取群成员列表
// @Summary 获取群成员列表
// @Description 获取指定群聊的当前活跃成员列表
// @Tags 会话
// @Produce json
// @Security BearerAuth
// @Param id path int true "群会话ID"
// @Success 200 {object} response.Response{data=dto.GroupMemberListResp} "查询成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未认证"
// @Failure 403 {object} response.Response "无权访问该群"
// @Failure 404 {object} response.Response "群会话不存在"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/conversations/groups/{id}/members [get]
func (h *ConversationHandler) ListGroupMembers(c *gin.Context) {
	userID := currentUserID(c)
	conversationID, ok := parseUintParam(c, "id", "invalid conversation id")
	if !ok {
		return
	}

	members, err := h.conversationService.ListGroupMembers(requestContext(c), userID, conversationID)
	if err != nil {
		respondError(c, err)
		return
	}
	items := make([]dto.GroupMemberResp, 0, len(members))
	for _, member := range members {
		items = append(items, dto.GroupMemberResp{
			UserID:   member.UserID,
			Username: member.Username,
			Role:     member.Role,
			Online:   member.Online,
		})
	}
	respondOK(c, dto.GroupMemberListResp{Members: items})
}

// UpdateGroupName 修改群名称
// @Summary 修改群名称
// @Description 修改群聊名称并写入系统消息
// @Tags 会话
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "群会话ID"
// @Param req body dto.UpdateGroupNameReq true "修改群名请求"
// @Success 200 {object} response.Response "操作成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未认证"
// @Failure 403 {object} response.Response "无权限修改"
// @Failure 404 {object} response.Response "群会话不存在"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/conversations/groups/{id}/name [post]
func (h *ConversationHandler) UpdateGroupName(c *gin.Context) {
	userID := currentUserID(c)
	conversationID, ok := parseUintParam(c, "id", "invalid conversation id")
	if !ok {
		return
	}
	var req dto.UpdateGroupNameReq
	if !bindJSON(c, &req) {
		return
	}

	if err := h.conversationService.UpdateGroupName(requestContext(c), userID, conversationID, req.Name); err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, nil)
}

// InviteGroupMembers 邀请成员入群
// @Summary 邀请成员入群
// @Description 邀请用户加入指定群聊
// @Tags 会话
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "群会话ID"
// @Param req body dto.InviteGroupMembersReq true "邀请成员请求"
// @Success 200 {object} response.Response "操作成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未认证"
// @Failure 403 {object} response.Response "无权限邀请"
// @Failure 404 {object} response.Response "群会话不存在"
// @Failure 409 {object} response.Response "群人数超限或状态冲突"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/conversations/groups/{id}/invite [post]
func (h *ConversationHandler) InviteGroupMembers(c *gin.Context) {
	userID := currentUserID(c)
	conversationID, ok := parseUintParam(c, "id", "invalid conversation id")
	if !ok {
		return
	}
	var req dto.InviteGroupMembersReq
	if !bindJSON(c, &req) {
		return
	}

	if err := h.conversationService.InviteGroupMembers(requestContext(c), userID, conversationID, req.MemberIDs); err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, nil)
}

// RemoveGroupMember 移除群成员
// @Summary 移除群成员
// @Description 将指定成员从群聊中移除
// @Tags 会话
// @Produce json
// @Security BearerAuth
// @Param id path int true "群会话ID"
// @Param user_id path int true "成员用户ID"
// @Success 200 {object} response.Response "操作成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未认证"
// @Failure 403 {object} response.Response "无权限移除"
// @Failure 404 {object} response.Response "群会话或成员不存在"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/conversations/groups/{id}/members/{user_id}/remove [post]
func (h *ConversationHandler) RemoveGroupMember(c *gin.Context) {
	userID := currentUserID(c)
	conversationID, ok := parseUintParam(c, "id", "invalid conversation id")
	if !ok {
		return
	}
	memberID, ok := parseUintParam(c, "user_id", "invalid member id")
	if !ok {
		return
	}

	if err := h.conversationService.RemoveGroupMember(requestContext(c), userID, conversationID, memberID); err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, nil)
}

// LeaveGroup 退出群聊
// @Summary 退出群聊
// @Description 当前用户主动退出指定群聊
// @Tags 会话
// @Produce json
// @Security BearerAuth
// @Param id path int true "群会话ID"
// @Success 200 {object} response.Response "操作成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未认证"
// @Failure 403 {object} response.Response "无权限退出"
// @Failure 404 {object} response.Response "群会话不存在"
// @Failure 409 {object} response.Response "群主不可直接退群"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/conversations/groups/{id}/leave [post]
func (h *ConversationHandler) LeaveGroup(c *gin.Context) {
	userID := currentUserID(c)
	conversationID, ok := parseUintParam(c, "id", "invalid conversation id")
	if !ok {
		return
	}

	if err := h.conversationService.LeaveGroup(requestContext(c), userID, conversationID); err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, nil)
}

// DismissGroup 解散群聊
// @Summary 解散群聊
// @Description 由群主解散指定群聊
// @Tags 会话
// @Produce json
// @Security BearerAuth
// @Param id path int true "群会话ID"
// @Success 200 {object} response.Response "操作成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未认证"
// @Failure 403 {object} response.Response "无权限解散"
// @Failure 404 {object} response.Response "群会话不存在"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/conversations/groups/{id}/dismiss [post]
func (h *ConversationHandler) DismissGroup(c *gin.Context) {
	userID := currentUserID(c)
	conversationID, ok := parseUintParam(c, "id", "invalid conversation id")
	if !ok {
		return
	}

	if err := h.conversationService.DismissGroup(requestContext(c), userID, conversationID); err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, nil)
}
