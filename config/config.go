package config

import(
	"github.com/spf13/viper"
	"log"
	"strings"
)

type Config struct {
	Mysql Mysql `mapstructure:"mysql"`
	Redis Redis `mapstructure:"redis"`
	JWT   JWT   `mapstructure:"jwt"`
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

func LoadConfig() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")

	viper.SetEnvPrefix("IM")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("Error unmarshalling config: %v", err)
	}

	return &cfg
}

