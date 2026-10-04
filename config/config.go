package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `yaml:"server" mapstructure:"server"`
	Logger   LoggerConfig   `yaml:"logger" mapstructure:"logger"`
	Database DatabaseConfig `yaml:"database" mapstructure:"database"`
	Redis    RedisConfig    `yaml:"redis" mapstructure:"redis"`
	Auth     AuthConfig     `yaml:"auth" mapstructure:"auth"`
}

type ServerConfig struct {
	ServerName   string `yaml:"server_name" mapstructure:"server_name"`
	ReadTimeout  int64  `yaml:"read_timeout" mapstructure:"read_timeout"`
	WriteTimeout int64  `yaml:"write_timeout" mapstructure:"write_timeout"`
	IdleTimeout  int64  `yaml:"idle_timeout" mapstructure:"idle_timeout"`
	Port         int    `yaml:"port" mapstructure:"port"`
	ServerMode   string `yaml:"server_mode" mapstructure:"server_mode"`
}

type LoggerConfig struct {
	Level      string `yaml:"level" mapstructure:"level"`             // 日志等级 INFO、DEBUG、ERROR
	LogDir     string `yaml:"log_dir" mapstructure:"log_dir"`         // 日志文件夹
	LogFile    string `yaml:"log_file" mapstructure:"log_file"`       // 日志文件名
	MaxSize    int    `yaml:"max_size" mapstructure:"max_size"`       // 单个日志文件最大体积 MB
	MaxAge     int    `yaml:"max_age" mapstructure:"max_age"`         // 日志旧文件最多保留时间 天
	MaxBackups int    `yaml:"max_backups" mapstructure:"max_backups"` // 最多保留几个日志文件
	Compress   bool   `yaml:"compress" mapstructure:"compress"`       // 是否压缩旧日志文件
}

type DatabaseConfig struct {
	File string `yaml:"file" mapstructure:"file"`
}

type RedisConfig struct {
	Addr         string        `yaml:"addr" mapstructure:"addr"`
	Password     string        `yaml:"password" mapstructure:"password"`
	DB           int           `yaml:"db" mapstructure:"db"`
	DialTimeout  time.Duration `yaml:"dial_timeout" mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `yaml:"read_timeout" mapstructure:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout" mapstructure:"write_timeout"`
	PoolSize     int           `yaml:"pool_size" mapstructure:"pool_size"`
}

type AuthConfig struct {
	CookieName  string `yaml:"cookie_name" mapstructure:"cookie_name"`
	ExpireHours int    `yaml:"expire_hours" mapstructure:"expire_hours"`
	Secure      bool   `yaml:"secure" mapstructure:"secure"`
}

func Load() (*Config, error) {
	viper.SetConfigType("yaml")
	viper.SetConfigName("ginblog")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config

	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}
