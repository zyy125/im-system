package config

import (
	"testing"
	"time"
)

func TestConfigValidate(t *testing.T) {
	base := &Config{
		App: App{
			Env:      "development",
			HTTPAddr: ":8080",
		},
		Mysql: Mysql{DSN: "root:pass@tcp(localhost:3306)/im"},
		Redis: Redis{Addr: "localhost:6379"},
		JWT: JWT{
			Secret: "development-secret",
			Expiry: 24,
		},
		Presence: Presence{
			TTL:               90 * time.Second,
			HeartbeatInterval: 30 * time.Second,
		},
	}

	if err := base.Validate(); err != nil {
		t.Fatalf("expected config to be valid, got %v", err)
	}

	production := *base
	production.App.Env = "production"
	production.JWT.Secret = "short"
	if err := production.Validate(); err == nil {
		t.Fatalf("expected production config to reject weak jwt secret")
	}

	production = *base
	production.App.Env = "production"
	production.JWT.Secret = "long-enough-production-secret"
	if err := production.Validate(); err == nil {
		t.Fatalf("expected production config to require ws origins")
	}
}
