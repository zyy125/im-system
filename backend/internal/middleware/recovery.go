package middleware

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/pkg/response"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}

			panicErr, ok := recovered.(error)
			if !ok {
				panicErr = fmt.Errorf("%v", recovered)
			}

			requestErr := response.RequestError{
				Code:    string(apperr.CodeInternal),
				Message: "internal server error",
			}

			pcs := make([]uintptr, 32)
			n := runtime.Callers(3, pcs)
			frames := runtime.CallersFrames(pcs[:n])
			for {
				frame, more := frames.Next()
				if !shouldSkipRecoveryFrame(frame.Function) {
					requestErr.File = frame.File
					requestErr.Line = frame.Line
					requestErr.Function = frame.Function
					break
				}
				if !more {
					break
				}
			}

			response.SetRequestError(c, requestErr)
			if !c.Writer.Written() {
				response.FailError(c, apperr.Internal("internal server error", panicErr))
			}
			c.Abort()
		}()

		c.Next()
	}
}

func shouldSkipRecoveryFrame(function string) bool {
	if function == "" {
		return true
	}
	return strings.HasPrefix(function, "runtime.") ||
		strings.Contains(function, "github.com/gin-gonic/gin") ||
		strings.Contains(function, "internal/middleware.Recovery") ||
		strings.HasSuffix(function, ".Recovery.func1")
}
