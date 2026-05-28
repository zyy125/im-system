package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/handler/dto"
	"github.com/zyy125/im-system/internal/service"
)

type AvatarHandler struct {
	userService service.UserService
}

func NewAvatarHandler(userService service.UserService) *AvatarHandler {
	return &AvatarHandler{userService: userService}
}

// UploadAvatar 上传当前用户头像
// @Summary 上传当前用户头像
// @Description 上传当前登录用户头像，成功后返回新的头像访问路径。
// @Tags 用户
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "头像文件"
// @Success 200 {object} response.Response{data=dto.UserAvatarResp} "上传成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未认证"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/users/avatar [post]
func (h *AvatarHandler) UploadAvatar(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		respondError(c, apperr.InvalidArgument("file is required"))
		return
	}

	avatar, err := h.userService.UpdateAvatar(requestContext(c), currentUserPK(c), fileHeader)
	if err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, dto.UserAvatarResp{Avatar: avatar})
}

// ClearAvatar 清空当前用户头像
// @Summary 清空当前用户头像
// @Description 清空当前登录用户头像。
// @Tags 用户
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response "清空成功"
// @Failure 401 {object} response.Response "未认证"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/users/avatar [delete]
func (h *AvatarHandler) ClearAvatar(c *gin.Context) {
	if err := h.userService.ClearAvatar(requestContext(c), currentUserPK(c)); err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, nil)
}
