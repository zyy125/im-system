package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/zyy125/im-system/internal/handler/dto"
	"github.com/zyy125/im-system/internal/service"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Register 用户注册
// @Summary 用户注册
// @Description 用户注册
// @Tags 用户
// @Accept json
// @Produce json
// @Param req body dto.UserRegisterReq true "用户注册请求"
// @Success 201 {object} response.Response "注册成功"
// @Failure 400 {object} response.Response "参数校验错误"
// @Failure 409 {object} response.Response "用户名已存在"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.UserRegisterReq
	if !bindJSON(c, &req) {
		return
	}
	if err := h.authService.Register(requestContext(c), req.Username, req.Password); err != nil {
		respondError(c, err)
		return
	}
	respondCreated(c, nil)
}

// Login 用户登录
// @Summary 用户登录
// @Description 用户登录，返回JWT token
// @Tags 用户
// @Accept json
// @Produce json
// @Param req body dto.UserLoginReq true "用户登录请求"
// @Success 200 {object} response.Response{data=dto.UserLoginResp} "登录成功"
// @Failure 400 {object} response.Response "参数校验错误"
// @Failure 401 {object} response.Response "用户名或密码错误"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.UserLoginReq
	if !bindJSON(c, &req) {
		return
	}

	token, err := h.authService.Login(requestContext(c), req.Username, req.Password)
	if err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, dto.UserLoginResp{Token: token})
}

// Logout 用户登出
// @Summary 用户登出
// @Description 用户登出，将JWT token加入黑名单
// @Tags 用户
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response "登出成功"
// @Failure 401 {object} response.Response "未认证"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	if err := h.authService.Logout(requestContext(c), c.GetString("jti")); err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, nil)
}
