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
	CookieName    string `yaml:"cookie_name" mapstructure:"cookie_name"`
	ExpireHours   int    `yaml:"expire_hours" mapstructure:"expire_hours"`
	Secure        bool   `yaml:"secure" mapstructure:"secure"`
	SessionPrefix string `yaml:"session_prefix" mapstructure:"session_prefix"`
}

func Load() (*Config, error) {
	viper.SetConfigType("yaml")
	viper.SetConfigName("ginblog")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// 默认值必须在 ReadInConfig 之前注册：
	// 配置文件整段缺失时（如没写 redis: 段），零值不会触发任何报错，
	// 只会在运行期变成 "redis addr is empty" 或空 cookie 名，排查成本高。
	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config

	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

// setDefaults 为全部配置项注册缺省值
func setDefaults() {
	// server：http.Server 的超时单位是秒
	viper.SetDefault("server.server_name", "ginblog")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.read_timeout", 15)
	viper.SetDefault("server.write_timeout", 15)
	viper.SetDefault("server.idle_timeout", 60)
	viper.SetDefault("server.server_mode", "release")

	viper.SetDefault("logger.level", "info")
	viper.SetDefault("logger.log_dir", "./logs")
	viper.SetDefault("logger.log_file", "ginblog.log")
	viper.SetDefault("logger.max_size", 100)
	viper.SetDefault("logger.max_age", 7)
	viper.SetDefault("logger.max_backups", 7)
	viper.SetDefault("logger.compress", true)

	viper.SetDefault("database.file", "./data/ginblog.db")

	viper.SetDefault("redis.addr", "127.0.0.1:6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.dial_timeout", "3s")
	viper.SetDefault("redis.read_timeout", "3s")
	viper.SetDefault("redis.write_timeout", "3s")
	viper.SetDefault("redis.pool_size", 10)

	viper.SetDefault("auth.cookie_name", "ginblog_sid")
	viper.SetDefault("auth.expire_hours", 168)
	viper.SetDefault("auth.secure", false)
	viper.SetDefault("auth.session_prefix", "ginblog:")
}

func (c *Config) validate() error {
	if c.Redis.Addr == "" {
		return fmt.Errorf("redis.addr must not be empty")
	}

	for _, d := range []struct {
		key string
		val time.Duration
	}{
		{"redis.dial_timeout", c.Redis.DialTimeout},
		{"redis.read_timeout", c.Redis.ReadTimeout},
		{"redis.write_timeout", c.Redis.WriteTimeout},
	} {
		if err := checkDuration(d.key, d.val); err != nil {
			return err
		}
	}

	// 0 或负数会让 cookie 立即过期、Redis SETEX 直接报 invalid expire time
	if c.Auth.ExpireHours <= 0 {
		return fmt.Errorf("auth.expire_hours must be greater than 0, got %d", c.Auth.ExpireHours)
	}
	if c.Auth.CookieName == "" {
		return fmt.Errorf("auth.cookie_name must not be empty")
	}

	return nil
}

// checkDuration 识别漏写时间单位的配置：
func checkDuration(key string, d time.Duration) error {
	if d > 0 && d < time.Millisecond {
		return fmt.Errorf("%s = %v looks like a missing time unit; write it with a unit such as 3s or 500ms", key, d)
	}
	return nil
}
