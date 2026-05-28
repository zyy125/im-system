package config

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 是整个应用的顶层配置结构，由 config.yaml 映射而来。
type Config struct {
	App      App      `mapstructure:"app"`
	HTTP     HTTP     `mapstructure:"http"`
	Storage  Storage  `mapstructure:"storage"`
	Mysql    Mysql    `mapstructure:"mysql"`
	Redis    Redis    `mapstructure:"redis"`
	JWT      JWT      `mapstructure:"jwt"`
	WS       WS       `mapstructure:"ws"`
	Presence Presence `mapstructure:"presence"`
}

// App 保存应用级配置。
type App struct {
	Env      string `mapstructure:"env"`
	HTTPAddr string `mapstructure:"http_addr"`
}

type HTTP struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

type Storage struct {
	AvatarDir        string `mapstructure:"avatar_dir"`
	AvatarPublicBase string `mapstructure:"avatar_public_base"`
}

// Mysql 保存 MySQL 连接串。
type Mysql struct {
	DSN string `mapstructure:"dsn"`
}

// Redis 保存 Redis 连接信息，用于在线状态、序列号、消息缓存等。
type Redis struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// JWT 保存签名密钥和 access/refresh token 的有效期（小时）。
type JWT struct {
	Secret        string `mapstructure:"secret"`
	AccessExpiry  int64  `mapstructure:"access_expiry"`
	RefreshExpiry int64  `mapstructure:"refresh_expiry"`
}

type WS struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

type Presence struct {
	TTL               time.Duration `mapstructure:"ttl"`
	HeartbeatInterval time.Duration `mapstructure:"heartbeat_interval"`
}

// LoadConfig 从 ./config/config.yaml 或当前目录读取配置，
// 并允许通过 IM_ 前缀的环境变量（点号替换为下划线）覆盖任意配置项，
// 例如 IM_MYSQL_DSN 覆盖 mysql.dsn。
func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")

	viper.SetEnvPrefix("IM")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if c == nil {
		return errors.New("config is nil")
	}

	c.App.Env = normalizeEnv(c.App.Env)
	if !slices.Contains([]string{"development", "test", "production"}, c.App.Env) {
		return fmt.Errorf("invalid app.env: %q", c.App.Env)
	}
	if strings.TrimSpace(c.App.HTTPAddr) == "" {
		return errors.New("app.http_addr is required")
	}
	if strings.TrimSpace(c.Mysql.DSN) == "" {
		return errors.New("mysql.dsn is required")
	}
	if strings.TrimSpace(c.Redis.Addr) == "" {
		return errors.New("redis.addr is required")
	}
	if strings.TrimSpace(c.JWT.Secret) == "" {
		return errors.New("jwt.secret is required")
	}
	if c.App.Env == "production" && len(strings.TrimSpace(c.JWT.Secret)) < 16 {
		return errors.New("jwt.secret must be at least 16 characters in production")
	}
	if c.JWT.AccessExpiry <= 0 {
		return errors.New("jwt.access_expiry must be greater than 0")
	}
	if c.JWT.RefreshExpiry <= 0 {
		return errors.New("jwt.refresh_expiry must be greater than 0")
	}
	if c.JWT.RefreshExpiry <= c.JWT.AccessExpiry {
		return errors.New("jwt.refresh_expiry must be greater than jwt.access_expiry")
	}
	if c.Presence.TTL <= 0 {
		return errors.New("presence.ttl must be greater than 0")
	}
	if c.Presence.HeartbeatInterval <= 0 {
		return errors.New("presence.heartbeat_interval must be greater than 0")
	}
	if c.Presence.HeartbeatInterval >= c.Presence.TTL {
		return errors.New("presence.heartbeat_interval must be smaller than presence.ttl")
	}
	if strings.TrimSpace(c.Storage.AvatarDir) == "" {
		c.Storage.AvatarDir = "./storage/avatars"
	}
	c.Storage.AvatarDir = strings.TrimSpace(c.Storage.AvatarDir)
	if strings.TrimSpace(c.Storage.AvatarPublicBase) == "" {
		c.Storage.AvatarPublicBase = "/uploads/avatars"
	}
	c.Storage.AvatarPublicBase = normalizePublicBase(c.Storage.AvatarPublicBase)
	if c.App.Env == "production" && len(c.HTTP.AllowedOrigins) == 0 {
		return errors.New("http.allowed_origins is required in production")
	}
	for i, origin := range c.HTTP.AllowedOrigins {
		c.HTTP.AllowedOrigins[i] = strings.TrimSpace(origin)
	}
	if c.App.Env == "production" && len(c.WS.AllowedOrigins) == 0 {
		return errors.New("ws.allowed_origins is required in production")
	}
	for i, origin := range c.WS.AllowedOrigins {
		c.WS.AllowedOrigins[i] = strings.TrimSpace(origin)
	}
	return nil
}

func normalizeEnv(env string) string {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "", "dev", "development":
		return "development"
	case "test":
		return "test"
	case "prod", "production":
		return "production"
	default:
		return strings.ToLower(strings.TrimSpace(env))
	}
}

func normalizePublicBase(base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		return "/uploads/avatars"
	}
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}
	base = strings.TrimRight(base, "/")
	if base == "" {
		return "/uploads/avatars"
	}
	return base
}
