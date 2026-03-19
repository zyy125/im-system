package handler

import(
	"net/http"

	"github.com/zyy125/im-system/internal/service"
	"github.com/zyy125/im-system/pkg/response"
	"github.com/zyy125/im-system/internal/handler/dto"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userSvc *service.UserService
}

func NewUserHandler(userSvc *service.UserService) *UserHandler {
	return &UserHandler{userSvc: userSvc}
}

func (h *UserHandler)CheckUserOnline(c *gin.Context) {
	userID := c.GetUint64("userID")

	ctx := c.Request.Context()
	online, err := h.userSvc.CheckUserOnline(ctx, userID)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	res := dto.CheckUserOnlineRes{
		UserID: userID,
		Online: online,
	}
	response.Success(c, http.StatusOK, res)
}
