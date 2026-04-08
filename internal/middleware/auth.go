package middleware

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/repository"
	"github.com/zyy125/im-system/pkg/jwt"
	"github.com/zyy125/im-system/pkg/response"
)

func AuthMiddleware(secret string, tokenBlacklistRepo repository.TokenBlacklistRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Request.Header.Get("Authorization")
		if token == "" {
			token = c.Query("token")
		}
		token = strings.TrimSpace(token)
		lower := strings.ToLower(token)
		if strings.HasPrefix(lower, "bearer ") {
			token = token[7:]
		}

		if token == "" {
			abortUnauthorized(c, apperr.TokenMissing())
			return
		}

		claims, err := jwt.ParseJWT(token, secret)
		if err != nil {
			abortUnauthorized(c, apperr.TokenInvalid())
			return
		}

		jti := claims.ID
		if jti == "" {
			abortUnauthorized(c, apperr.TokenInvalid())
			return
		}

		bl, err := tokenBlacklistRepo.IsBlacklisted(c.Request.Context(), jti)
		if err != nil {
			response.FailError(c, err)
			c.Abort()
			return
		}
		if bl {
			abortUnauthorized(c, apperr.TokenBlacklisted())
			return
		}

		uid, _ := strconv.ParseUint(claims.UserID, 10, 64)
		if uid <= 0 {
			abortUnauthorized(c, apperr.TokenInvalid())
			return
		}

		c.Set("userID", uid)
		c.Set("jti", jti)

		c.Next()
	}
}

func abortUnauthorized(c *gin.Context, err error) {
	response.FailError(c, err)
	c.Abort()
}
