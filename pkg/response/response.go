package response

import (
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/zyy125/im-system/internal/apperr"
)

type Response struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func Success(c *gin.Context, code int, data any) {
	c.JSON(code, Response{
		Code:    string(apperr.CodeOK),
		Message: "success",
		Data:    data,
	})
}

func FailError(c *gin.Context, err error) {
	appErr := apperr.From(err)
	setErrorMeta(c, appErr, 2)
	c.JSON(apperr.HTTPStatus(appErr.Code), Response{
		Code:    string(appErr.Code),
		Message: appErr.Message,
		Data:    nil,
	})
}

func setErrorMeta(c *gin.Context, appErr *apperr.Error, callerSkip int) {
	if pc, file, line, ok := runtime.Caller(callerSkip); ok {
		function := ""
		if fn := runtime.FuncForPC(pc); fn != nil {
			function = fn.Name()
		}
		c.Set("error_source", gin.H{
			"file":     file,
			"line":     line,
			"function": function,
		})
	}
	c.Set("error_message", appErr.Message)
	c.Set("error_code", string(appErr.Code))
}
