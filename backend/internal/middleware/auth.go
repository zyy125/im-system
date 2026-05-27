package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/repository"
	"github.com/zyy125/im-system/pkg/jwt"
	"github.com/zyy125/im-system/pkg/response"
)

type AuthResult struct {
	UserID    uint64
	SessionID string
	JTI       string
	ExpiresAt time.Time
}

func AuthMiddleware(secret string, tokenBlacklistRepo repository.TokenBlacklistRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := AuthenticateHTTPRequest(c.Request, secret, tokenBlacklistRepo)
		if err != nil {
			abortUnauthorized(c, err)
			return
		}

		c.Set("userID", result.UserID)
		c.Set("sessionID", result.SessionID)
		c.Set("jti", result.JTI)
		c.Set("tokenExpiresAt", result.ExpiresAt)
		c.Next()
	}
}

func AuthenticateHTTPRequest(r *http.Request, secret string, tokenBlacklistRepo repository.TokenBlacklistRepo) (AuthResult, error) {
	token := parseBearerToken(r.Header.Get("Authorization"))
	if token == "" {
		return AuthResult{}, apperr.TokenMissing()
	}
	return authenticateToken(r.Context(), token, secret, tokenBlacklistRepo)
}

func AuthenticateWSRequest(r *http.Request, secret string, tokenBlacklistRepo repository.TokenBlacklistRepo) (AuthResult, error) {
	token := parseBearerToken(r.Header.Get("Authorization"))
	if token == "" {
		token = strings.TrimSpace(r.URL.Query().Get("token"))
	}
	if token == "" {
		return AuthResult{}, apperr.TokenMissing()
	}
	return authenticateToken(r.Context(), token, secret, tokenBlacklistRepo)
}

func authenticateToken(ctx context.Context, token, secret string, tokenBlacklistRepo repository.TokenBlacklistRepo) (AuthResult, error) {
	claims, err := jwt.ParseJWT(token, secret)
	if err != nil {
		return AuthResult{}, apperr.TokenInvalid()
	}
	if claims.ID == "" {
		return AuthResult{}, apperr.TokenInvalid()
	}
	if claims.ExpiresAt == nil {
		return AuthResult{}, apperr.TokenInvalid()
	}
	if strings.TrimSpace(claims.SessionID) == "" {
		return AuthResult{}, apperr.TokenInvalid()
	}

	blacklisted, err := tokenBlacklistRepo.IsBlacklisted(ctx, claims.ID)
	if err != nil {
		return AuthResult{}, err
	}
	if blacklisted {
		return AuthResult{}, apperr.TokenBlacklisted()
	}

	uid, err := strconv.ParseUint(claims.UserID, 10, 64)
	if err != nil || uid == 0 {
		return AuthResult{}, apperr.TokenInvalid()
	}

	return AuthResult{
		UserID:    uid,
		SessionID: claims.SessionID,
		JTI:       claims.ID,
		ExpiresAt: claims.ExpiresAt.Time,
	}, nil
}

func parseBearerToken(headerValue string) string {
	token := strings.TrimSpace(headerValue)
	if token == "" {
		return ""
	}

	lower := strings.ToLower(token)
	if !strings.HasPrefix(lower, "bearer ") {
		return ""
	}
	return strings.TrimSpace(token[7:])
}

func abortUnauthorized(c *gin.Context, err error) {
	response.FailError(c, err)
	c.Abort()
}
