package middleware

import (
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zyy125/im-system/pkg/response"
)

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		args := []any{
			"event_type", "http_request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", status,
			"latency_ms", time.Since(start).Milliseconds(),
			"client_ip", c.ClientIP(),
			"user_agent", c.Request.UserAgent(),
		}

		if userID := c.GetUint64("userID"); userID != 0 {
			args = append(args, "user_id", userID)
		}

		if value, ok := c.Get(response.RequestErrorContextKey); ok {
			requestErr, ok := value.(response.RequestError)
			if ok {
				if requestErr.Code != "" {
					args = append(args, "error_code", requestErr.Code)
				}
				if requestErr.Message != "" {
					args = append(args, "error_message", requestErr.Message)
				}
				if requestErr.File != "" {
					normalized := filepath.ToSlash(requestErr.File)
					if idx := strings.Index(normalized, "/im-system/"); idx >= 0 {
						normalized = normalized[idx+1:]
					}
					args = append(args, "file", normalized)
				}
				if requestErr.Line != 0 {
					args = append(args, "line", requestErr.Line)
				}
				if requestErr.Function != "" {
					args = append(args, "function", requestErr.Function)
				}
			}
		}

		logger := slog.Default()
		switch {
		case status >= 500:
			logger.Error("http request completed", args...)
		case status >= 400:
			logger.Warn("http request completed", args...)
		default:
			logger.Info("http request completed", args...)
		}
	}
}
