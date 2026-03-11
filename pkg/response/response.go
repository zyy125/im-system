package response

import (
	"github.com/gin-gonic/gin"
)

type Response struct {
	Code    int `json:"code"`
	Message string 	`json:"message"`
	Data    any `json:"data"`
}

func Success(c *gin.Context,code int, data any) {
	c.JSON(code, Response{
		Code:    code,
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