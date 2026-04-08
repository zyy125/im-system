package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/zyy125/im-system/internal/handler/dto"
	"github.com/zyy125/im-system/internal/service"
)

type UserHandler struct {
	userService service.UserService
}

func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// CheckUserOnline 查询当前用户在线状态
// @Summary 查询当前用户在线状态
// @Description 查询当前登录用户是否在线
// @Tags 用户
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=dto.CheckUserOnlineResp} "查询成功"
// @Failure 401 {object} response.Response "未认证"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/user/online [get]
func (h *UserHandler) CheckUserOnline(c *gin.Context) {
	userID := currentUserID(c)
	online, err := h.userService.IsOnline(requestContext(c), userID)
	if err != nil {
		respondError(c, err)
		return
	}

	res := dto.CheckUserOnlineResp{
		UserID: userID,
		Online: online,
	}
	respondOK(c, res)
}

// GetMe 获取当前用户信息
// @Summary 获取当前用户信息
// @Description 返回当前登录用户的基础信息与在线状态
// @Tags 用户
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=dto.UserInfoResp} "查询成功"
// @Failure 401 {object} response.Response "未认证"
// @Failure 404 {object} response.Response "用户不存在"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/users/me [get]
func (h *UserHandler) GetMe(c *gin.Context) {
	userID := currentUserID(c)
	user, online, err := h.userService.GetUserByID(requestContext(c), userID)
	if err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, buildUserInfoResp(user, online))
}

// GetUserInfo 获取指定用户信息
// @Summary 获取指定用户信息
// @Description 根据用户ID查询基础信息与在线状态
// @Tags 用户
// @Produce json
// @Security BearerAuth
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response{data=dto.UserInfoResp} "查询成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未认证"
// @Failure 404 {object} response.Response "用户不存在"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/users/{id} [get]
func (h *UserHandler) GetUserInfo(c *gin.Context) {
	userID, ok := parseUintParam(c, "id", "invalid user id")
	if !ok {
		return
	}
	user, online, err := h.userService.GetUserByID(requestContext(c), userID)
	if err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, buildUserInfoResp(user, online))
}
