package config

import(
	"github.com/spf13/viper"
	"strings"
)

type Config struct {
	Mysql Mysql `mapstructure:"mysql"`
	Redis Redis `mapstructure:"redis"`
	JWT   JWT   `mapstructure:"jwt"`
	RabbitMQ RabbitMQ `mapstructure:"rabbitmq"`
}

type Mysql struct {
	DSN string `mapstructure:"dsn"`
}

type Redis struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type JWT struct {
	Secret string `mapstructure:"secret"`
	Expiry int64  `mapstructure:"expiry"`
}

type RabbitMQ struct {
	URL string `mapstructure:"url"`
}

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

