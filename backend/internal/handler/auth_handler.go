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
// @Failure 409 {object} response.Response "注册冲突"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.UserRegisterReq
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.authService.Register(requestContext(c), req.Username, req.Password)
	if err != nil {
		respondError(c, err)
		return
	}
	respondCreated(c, dto.UserRegisterResp{
		PublicID: result.PublicID,
	})
}

// Login 用户登录
// @Summary 用户登录
// @Description 使用 public_id 和密码登录，返回 access token 和 refresh token。后续 HTTP 受保护接口必须通过 `Authorization: Bearer <access_token>` 传递。
// @Tags 用户
// @Accept json
// @Produce json
// @Param req body dto.UserLoginReq true "用户登录请求"
// @Success 200 {object} response.Response{data=dto.UserLoginResp} "登录成功"
// @Failure 400 {object} response.Response "参数校验错误"
// @Failure 401 {object} response.Response "账号号或密码错误"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.UserLoginReq
	if !bindJSON(c, &req) {
		return
	}

	tokens, err := h.authService.Login(requestContext(c), req.PublicID, req.Password)
	if err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, dto.UserLoginResp{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
	})
}

// Refresh 刷新 access token
// @Summary 刷新 access token
// @Description 使用 refresh token 换取新的一组 access/refresh token。
// @Tags 用户
// @Accept json
// @Produce json
// @Param req body dto.UserRefreshReq true "刷新 token 请求"
// @Success 200 {object} response.Response{data=dto.UserRefreshResp} "刷新成功"
// @Failure 400 {object} response.Response "参数校验错误"
// @Failure 401 {object} response.Response "refresh token 无效"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req dto.UserRefreshReq
	if !bindJSON(c, &req) {
		return
	}

	tokens, err := h.authService.Refresh(requestContext(c), req.RefreshToken)
	if err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, dto.UserRefreshResp{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
	})
}

// Logout 用户登出
// @Summary 用户登出
// @Description 用户登出，将当前 Bearer token 的 jti 加入黑名单；普通 HTTP 接口不接受 query token。
// @Tags 用户
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response "登出成功"
// @Failure 401 {object} response.Response "未认证"
// @Failure 500 {object} response.Response "内部服务器错误"
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	if err := h.authService.Logout(requestContext(c), c.GetString("jti"), currentSessionID(c), currentTokenExpiresAt(c)); err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, nil)
}
