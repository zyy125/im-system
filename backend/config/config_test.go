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
		Storage: Storage{},
		Mysql: Mysql{DSN: "root:pass@tcp(localhost:3306)/im"},
		Redis: Redis{Addr: "localhost:6379"},
		JWT: JWT{
			Secret:        "development-secret",
			AccessExpiry:  24,
			RefreshExpiry: 720,
		},
		Presence: Presence{
			TTL:               90 * time.Second,
			HeartbeatInterval: 30 * time.Second,
		},
	}

	if err := base.Validate(); err != nil {
		t.Fatalf("expected config to be valid, got %v", err)
	}
	if base.Storage.AvatarDir == "" {
		t.Fatalf("expected avatar dir default to be set")
	}
	if base.Storage.AvatarPublicBase == "" {
		t.Fatalf("expected avatar public base default to be set")
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
