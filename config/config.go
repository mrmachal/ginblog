package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Logger   LoggerConfig   `yaml:"logger"`
	Database DatabaseConfig `yaml:"database"`
}

type ServerConfig struct {
	ServerName   string `yaml:"server_name"`
	ReadTimeout  int64  `yaml:"read_timeout"`
	WriteTimeout int64  `yaml:"write_timeout"`
	IdleTimeout  int64  `yaml:"idle_timeout"`
	Port         int    `yaml:"port"`
}

type LoggerConfig struct {
	Level      string `yaml:"level"`       // 日志等级 INFO、DEBUG、ERROR
	LogDir     string `yaml:"log_dir"`     // 日志文件夹
	LogFile    string `yaml:"log_file"`    // 日志文件名
	MaxSize    int    `yaml:"max_size"`    // 单个日志文件最大体积 MB
	MaxAge     int    `yaml:"max_age"`     // 日志旧文件最多保留时间 天
	MaxBackups int    `yaml:"max_backups"` // 最多保留几个日志文件
	Compress   bool   `yaml:"compress"`    // 是否压缩旧日志文件
}

type DatabaseConfig struct {
	File string `yaml:"file"`
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
