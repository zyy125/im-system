package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zyy125/im-system/internal/handler/dto"
	"github.com/zyy125/im-system/internal/service"
	"github.com/zyy125/im-system/pkg/response"
)

type AuthHandler struct {
	authSvc *service.AuthService
}

func NewAuthHandler(authSvc *service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
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
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.UserRegisterReq
	if err := c.ShouldBind(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "parameter validation error")
		return
	}
	ctx := c.Request.Context()

	if err := h.authSvc.Register(ctx, req.Username, req.Password); err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	} else {
		response.Success(c, http.StatusCreated, nil)
	}
	
}

// Login 用户登录
// @Summary 用户登录
// @Description 用户登录，返回JWT token
// @Tags 用户
// @Accept json
// @Produce json
// @Param req body dto.UserLoginReq true "用户登录请求"
// @Success 200 {object} response.Response "登录成功"
// @Failure 400 {object} response.Response "参数校验错误"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.UserLoginReq
	if err := c.ShouldBind(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "parameter validation error")
		return
	}

	ctx := c.Request.Context()
	if token, err := h.authSvc.Login(ctx, req.Username, req.Password); err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
	} else {
		response.Success(c, http.StatusOK, dto.UserLoginResp{Token: token})
	}
}

// Logout 用户登出
// @Summary 用户登出
// @Description 用户登出，将JWT token加入黑名单
// @Tags 用户
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response "登出成功"
// @Failure 400 {object} response.Response "参数校验错误"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	ctx := c.Request.Context()
	jti := c.GetString("jti")
	if err := h.authSvc.Logout(ctx, jti); err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
	} else {
		response.Success(c, http.StatusOK, nil)
	}
}


