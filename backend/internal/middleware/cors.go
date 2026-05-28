package middleware

import (
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
)

func CORS(appEnv string, allowedOrigins []string) gin.HandlerFunc {
	normalized := normalizeOrigins(allowedOrigins)
	allowAnyOrigin := strings.TrimSpace(appEnv) != "production" && len(normalized) == 0

	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.Request.Header.Get("Origin"))
		if origin == "" {
			c.Next()
			return
		}

		if allowAnyOrigin || slices.Contains(normalized, origin) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func normalizeOrigins(origins []string) []string {
	items := make([]string, 0, len(origins))
	for _, origin := range origins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		items = append(items, origin)
	}
	return items
}
