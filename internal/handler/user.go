package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/zyy125/im-system/internal/handler/dto"
	"github.com/zyy125/im-system/pkg/response"
	"github.com/zyy125/im-system/internal/service"
	"net/http"
)

type UserHandler struct {
	userSvc *service.UserService
}

func (h *UserHandler) Register(c *gin.Context) {
	var req dto.UserRegisterReq
	if err := c.ShouldBind(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "parameter validation error")
	}

	if err := h.userSvc.Register(c, req.Username, req.Password); err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
	}
	response.Success(c, nil)
}

func (h *UserHandler) Login(c *gin.Context) {
	var req dto.UserLoginReq
	if err := c.ShouldBind(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "parameter validation error")
	}
	
	if token, err := h.userSvc.Login(c, req.Username, req.Password); err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
	} else {
		response.Success(c, gin.H{"token": token})
	}
}
