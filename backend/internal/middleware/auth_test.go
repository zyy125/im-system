package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zyy125/im-system/pkg/jwt"
)

type authTestBlacklistRepo struct{}

func (authTestBlacklistRepo) IsBlacklisted(context.Context, string) (bool, error) { return false, nil }
func (authTestBlacklistRepo) Blacklist(context.Context, string, time.Duration) error {
	return nil
}

func TestAuthenticateHTTPRequestRequiresBearerHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me?token=abc", nil)
	if _, err := AuthenticateHTTPRequest(req, "secret", authTestBlacklistRepo{}); err == nil {
		t.Fatalf("expected http auth to reject query token")
	}
}

func TestAuthenticateWSRequestAllowsQueryToken(t *testing.T) {
	token, _, err := jwt.GenerateJWT("1", "session-1", "secret", time.Hour)
	if err != nil {
		t.Fatalf("generate jwt: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ws/?token="+token, nil)
	result, err := AuthenticateWSRequest(req, "secret", authTestBlacklistRepo{})
	if err != nil {
		t.Fatalf("expected ws auth to accept query token, got %v", err)
	}
	if result.UserID != 1 {
		t.Fatalf("expected user id 1, got %d", result.UserID)
	}
	if result.SessionID != "session-1" {
		t.Fatalf("expected session id session-1, got %q", result.SessionID)
	}
	if result.ExpiresAt.IsZero() {
		t.Fatal("expected token expiry to be present")
	}
}
