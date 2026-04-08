package middleware

import (
	"log"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	errorSourceKey  = "error_source"
	errorMessageKey = "error_message"
	errorCodeKey    = "error_code"
)

type ErrorSource struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Function string `json:"function"`
}

func ErrorSourceLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if c.Writer.Status() < 400 {
			return
		}

		message := getString(c, errorMessageKey)
		code := getString(c, errorCodeKey)
		source, ok := getErrorSource(c)
		if !ok {
			return
		}
		log.Printf(
			"error source: status=%d code=%s method=%s path=%s file=%s line=%d func=%s message=%s",
			c.Writer.Status(),
			code,
			c.Request.Method,
			c.Request.URL.Path,
			source.File,
			source.Line,
			source.Function,
			message,
		)
	}
}

func getString(c *gin.Context, key string) string {
	value, ok := c.Get(key)
	if !ok {
		return ""
	}
	s, _ := value.(string)
	return s
}

func getErrorSource(c *gin.Context) (ErrorSource, bool) {
	value, ok := c.Get(errorSourceKey)
	if !ok {
		return ErrorSource{}, false
	}
	switch source := value.(type) {
	case ErrorSource:
		source.File = shortenFile(source.File)
		return source, true
	case map[string]any:
		return errorSourceFromMap(source)
	case gin.H:
		return errorSourceFromMap(map[string]any(source))
	default:
		return ErrorSource{}, false
	}
}

func shortenFile(file string) string {
	normalized := filepath.ToSlash(file)
	if idx := strings.Index(normalized, "/im-system/"); idx >= 0 {
		return normalized[idx+1:]
	}
	return normalized
}

func errorSourceFromMap(source map[string]any) (ErrorSource, bool) {
	file, _ := source["file"].(string)
	function, _ := source["function"].(string)

	line := 0
	switch v := source["line"].(type) {
	case int:
		line = v
	case int32:
		line = int(v)
	case int64:
		line = int(v)
	case float64:
		line = int(v)
	}

	if file == "" || line == 0 {
		return ErrorSource{}, false
	}

	return ErrorSource{
		File:     shortenFile(file),
		Line:     line,
		Function: function,
	}, true
}
