package config

import (
	"strings"

	"github.com/spf13/viper"
)

// Config 是整个应用的顶层配置结构，由 config.yaml 映射而来。
type Config struct {
	App   App   `mapstructure:"app"`
	Mysql Mysql `mapstructure:"mysql"`
	Redis Redis `mapstructure:"redis"`
	JWT   JWT   `mapstructure:"jwt"`
}

// App 保存应用级配置。
type App struct {
	// Env 标识运行环境。设为 production/prod 时会启用更严格的启动校验。
	Env string `mapstructure:"env"`
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

// JWT 保存签名密钥和有效期（小时）。
type JWT struct {
	Secret string `mapstructure:"secret"`
	Expiry int64  `mapstructure:"expiry"`
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
