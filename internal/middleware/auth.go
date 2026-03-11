package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zyy125/im-system/internal/repository"
	"github.com/zyy125/im-system/pkg/jwt"
	"github.com/zyy125/im-system/pkg/response"
)

func AuthMiddleware(secret string, tokenBlacklistRepo repository.TokenBlacklistRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Request.Header.Get("Authorization")
		token = strings.TrimSpace(token)
		lower := strings.ToLower(token)
		if strings.HasPrefix(lower, "bearer ") {
			token = token[7:]
		}

		if token == "" {
			response.Fail(c, http.StatusUnauthorized, "token is empty")
			c.Abort()
			return
		}

		claims, err := jwt.ParseJWT(token, secret)
		if err != nil {
			response.Fail(c, http.StatusUnauthorized, "token is invalid")
			c.Abort()
			return
		}

		jti := claims.ID
		if jti == "" {
			response.Fail(c, http.StatusUnauthorized, "token is invalid")
			c.Abort()
			return
		}

		bl, err := tokenBlacklistRepo.IsBlacklisted(c.Request.Context(), jti)
		if err != nil || bl {
			response.Fail(c, http.StatusUnauthorized, "token is invalid")
			c.Abort()
			return
		}

		uid, _ := strconv.ParseInt(claims.UserID, 10, 64)
		if uid <= 0 {
			response.Fail(c, http.StatusUnauthorized, "token is invalid")
			c.Abort()
			return
		}

		c.Set("userID", uid)
		c.Set("jti", jti)

		c.Next()
	}
}
