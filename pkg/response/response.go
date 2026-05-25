package response

import (
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zyy125/im-system/internal/apperr"
)

const RequestErrorContextKey = "request_error"

type Response struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

type RequestError struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Function string `json:"function"`
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
	if appErr == nil {
		appErr = apperr.Internal("internal server error", nil)
	}
	if _, ok := c.Get(RequestErrorContextKey); !ok {
		c.Set(RequestErrorContextKey, buildRequestError(appErr))
	}
	c.JSON(apperr.HTTPStatus(appErr.Code), Response{
		Code:    string(appErr.Code),
		Message: appErr.Message,
		Data:    nil,
	})
}

func SetRequestError(c *gin.Context, requestErr RequestError) {
	c.Set(RequestErrorContextKey, requestErr)
}

func buildRequestError(appErr *apperr.Error) RequestError {
	requestErr := RequestError{
		Code:    string(appErr.Code),
		Message: appErr.Message,
	}

	pcs := make([]uintptr, 32)
	n := runtime.Callers(2, pcs)
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		if !shouldSkipRequestErrorFrame(frame.Function) {
			requestErr.File = frame.File
			requestErr.Line = frame.Line
			requestErr.Function = frame.Function
			break
		}
		if !more {
			break
		}
	}

	return requestErr
}

func shouldSkipRequestErrorFrame(function string) bool {
	if function == "" {
		return true
	}

	if strings.HasPrefix(function, "runtime.") ||
		strings.Contains(function, "github.com/gin-gonic/gin") ||
		strings.HasPrefix(function, "github.com/zyy125/im-system/pkg/response.") {
		return true
	}

	return strings.HasSuffix(function, ".respondError") ||
		strings.HasSuffix(function, ".bindJSON") ||
		strings.HasSuffix(function, ".bindOptionalJSON") ||
		strings.HasSuffix(function, ".parseUintParam") ||
		strings.HasSuffix(function, ".parseUintQueryError") ||
		strings.HasSuffix(function, ".abortUnauthorized")
}
