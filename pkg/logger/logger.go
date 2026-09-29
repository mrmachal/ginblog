package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
	gormlogger "gorm.io/gorm/logger"
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

// GormLogger 将 GORM 的日志桥接到 slog，复用 Init 建立的默认 logger
type GormLogger struct {
	level         gormlogger.LogLevel
	slowThreshold time.Duration
}

// NewGormLogger 根据应用日志等级创建 GORM logger
//
//	debug  -> Info   打印每条 SQL
//	info   -> Warn   只打印错误和慢查询
//	warn   -> Warn   同上
//	error  -> Error  只打印错误
//	silent -> Silent 不打印
func NewGormLogger(level string, slowThreshold time.Duration) *GormLogger {
	var l gormlogger.LogLevel
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		l = gormlogger.Info
	case "error":
		l = gormlogger.Error
	case "silent":
		l = gormlogger.Silent
	case "info", "warn", "warning", "":
		l = gormlogger.Warn
	default:
		l = gormlogger.Warn
	}

	if slowThreshold <= 0 {
		slowThreshold = 200 * time.Millisecond
	}

	return &GormLogger{
		level:         l,
		slowThreshold: slowThreshold,
	}
}

// LogMode 返回一个副本，避免修改共享的 logger 实例
func (l *GormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newLogger := *l
	newLogger.level = level
	return &newLogger
}

func (l *GormLogger) Info(ctx context.Context, msg string, data ...any) {
	if l.level < gormlogger.Info {
		return
	}
	slog.InfoContext(ctx, "[GORM] "+format(msg, data...))
}

func (l *GormLogger) Warn(ctx context.Context, msg string, data ...any) {
	if l.level < gormlogger.Warn {
		return
	}
	slog.WarnContext(ctx, "[GORM] "+format(msg, data...))
}

func (l *GormLogger) Error(ctx context.Context, msg string, data ...any) {
	if l.level < gormlogger.Error {
		return
	}
	slog.ErrorContext(ctx, "[GORM] "+format(msg, data...))
}

// Trace 记录 SQL 执行情况：错误 -> Error，超过慢查询阈值 -> Warn，普通查询 -> Debug
func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.level <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	switch {
	case err != nil && l.level >= gormlogger.Error:
		slog.ErrorContext(ctx, "gorm: query failed",
			slog.String("error", err.Error()),
			slog.String("sql", sql),
			slog.Int64("rows", rows),
			slog.Duration("elapsed", elapsed),
		)
	case elapsed >= l.slowThreshold && l.level >= gormlogger.Warn:
		slog.WarnContext(ctx, "gorm: slow query",
			slog.String("sql", sql),
			slog.Int64("rows", rows),
			slog.Duration("elapsed", elapsed),
		)
	case l.level >= gormlogger.Info:
		slog.DebugContext(ctx, "gorm: query",
			slog.String("sql", sql),
			slog.Int64("rows", rows),
			slog.Duration("elapsed", elapsed),
		)
	}
}

func format(msg string, data ...any) string {
	if len(data) == 0 {
		return msg
	}
	return fmt.Sprintf(msg, data...)
}
