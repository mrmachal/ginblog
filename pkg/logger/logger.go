package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

type Config struct {
	Level      string `yaml:"level"`       // 日志等级 INFO、DEBUG、ERROR
	LogDir     string `yaml:"log_dir"`     // 日志文件夹
	LogFile    string `yaml:"log_file"`    // 日志文件名
	MaxSize    int    `yaml:"max_size"`    // 单个日志文件最大体积 MB
	MaxAge     int    `yaml:"max_age"`     // 日志旧文件最多保留时间 天
	MaxBackups int    `yaml:"max_backups"` // 最多保留几个日志文件
	Compress   bool   `yaml:"compress"`    // 是否压缩旧日志文件
}

func Init(cfg Config) (io.Closer, error) {
	if err := os.MkdirAll(cfg.LogDir, 0755); err != nil {
		return nil, fmt.Errorf("faild to create log dir: %w", err)
	}

	logPath := filepath.Join(cfg.LogDir, cfg.LogFile)

	rollingFile := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
		LocalTime:  true,
	}

	writer := io.MultiWriter(os.Stdout, rollingFile)
	var level slog.Level
	switch strings.ToLower(strings.TrimSpace(cfg.Level)) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warning", "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	}

	ginblogLogger := slog.New(slog.NewJSONHandler(writer, opts))
	slog.SetDefault(ginblogLogger)

	return rollingFile, nil

}
