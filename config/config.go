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
	Level string `yaml:"level"`
	File  string `yaml:"file"`
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
