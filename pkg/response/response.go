package response

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type Response struct {
	Code    int `json:"code"`
	Message string 	`json:"message"`
	Data    any `json:"data"`
}

func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "success",
		Data:    data,
	})
}

func Fail(c *gin.Context, code int, message string) {
	c.JSON(code, Response{
		Code: code,
		Message: message,
		Data: nil,
	})
}