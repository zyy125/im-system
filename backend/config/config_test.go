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
		Mysql:   Mysql{DSN: "root:pass@tcp(localhost:3306)/im"},
		Redis:   Redis{Addr: "localhost:6379"},
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
	if base.Mysql.MaxOpenConns != 50 {
		t.Fatalf("expected mysql max open conns default to be set, got %d", base.Mysql.MaxOpenConns)
	}
	if base.Mysql.MaxIdleConns != 10 {
		t.Fatalf("expected mysql max idle conns default to be set, got %d", base.Mysql.MaxIdleConns)
	}
	if base.Mysql.ConnMaxLifetime != time.Hour {
		t.Fatalf("expected mysql conn max lifetime default to be set, got %v", base.Mysql.ConnMaxLifetime)
	}
	if base.Redis.PoolSize != 10 {
		t.Fatalf("expected redis pool size default to be set, got %d", base.Redis.PoolSize)
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

	invalidOpen := *base
	invalidOpen.Mysql.MaxOpenConns = -1
	if err := invalidOpen.Validate(); err == nil {
		t.Fatalf("expected negative mysql max open conns to be rejected")
	}

	invalidIdle := *base
	invalidIdle.Mysql.MaxOpenConns = 5
	invalidIdle.Mysql.MaxIdleConns = 6
	if err := invalidIdle.Validate(); err == nil {
		t.Fatalf("expected mysql max idle conns greater than max open conns to be rejected")
	}

	invalidLifetime := *base
	invalidLifetime.Mysql.ConnMaxLifetime = -time.Second
	if err := invalidLifetime.Validate(); err == nil {
		t.Fatalf("expected negative mysql conn max lifetime to be rejected")
	}

	invalidPool := *base
	invalidPool.Redis.PoolSize = -1
	if err := invalidPool.Validate(); err == nil {
		t.Fatalf("expected negative redis pool size to be rejected")
	}
}
