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

// Handle 持有初始化后的日志器
type Handle struct {
	App    *slog.Logger // 应用日志，带 source 信息
	Gorm   *slog.Logger // GORM 日志，不带 source（调用方恒为本包，无参考价值）
	closer io.Closer
}

// Close 关闭底层日志文件
func (h *Handle) Close() error {
	if h == nil || h.closer == nil {
		return nil
	}
	return h.closer.Close()
}

func Init(cfg Config) (*Handle, error) {
	if err := os.MkdirAll(cfg.LogDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create log dir: %w", err)
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
	level := parseLevel(cfg.Level)

	newHandler := func(addSource bool) slog.Handler {
		return slog.NewJSONHandler(writer, &slog.HandlerOptions{
			Level:       level,
			AddSource:   addSource,
			ReplaceAttr: replaceAttr,
		})
	}

	appLogger := slog.New(newHandler(true))
	gormSlogLogger := slog.New(newHandler(false))

	// 应用代码（含 middleware）统一走 slog 默认 logger
	slog.SetDefault(appLogger)

	return &Handle{
		App:    appLogger,
		Gorm:   gormSlogLogger,
		closer: rollingFile,
	}, nil
}

// NewSlogWriter 返回一个 io.Writer，把写入的内容转发到 slog，
// 用于接管第三方库（如 gin）的日志输出，保持格式统一
func NewSlogWriter(tag string, level slog.Level) io.Writer {
	return slogWriter{tag: tag, level: level}
}

type slogWriter struct {
	tag   string
	level slog.Level
}

func (w slogWriter) Write(p []byte) (int, error) {
	if msg := strings.TrimSpace(string(p)); msg != "" {
		slog.Log(context.Background(), w.level, w.tag, slog.String("detail", msg))
	}
	return len(p), nil
}

// parseLevel 解析配置的日志等级，silent 表示不输出任何日志
func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warning", "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "silent":
		return slog.LevelError + 1 // 高于最高等级，过滤掉一切日志
	case "info", "":
		return slog.LevelInfo
	default:
		return slog.LevelInfo
	}
}

// replaceAttr 将 Duration 格式化为可读字符串（如 "64.1ms"），避免输出裸纳秒数字
func replaceAttr(_ []string, a slog.Attr) slog.Attr {
	if a.Value.Kind() == slog.KindDuration {
		a.Value = slog.StringValue(a.Value.Duration().String())
	}
	return a
}

// GormLogger 将 GORM 的日志桥接到 slog
type GormLogger struct {
	slog          *slog.Logger
	level         gormlogger.LogLevel
	slowThreshold time.Duration
}

// NewGormLogger 根据应用日志等级创建 GORM logger
// l 为 nil 时回退到 slog.Default()（仅作兜底，建议显式传入 Handle.Gorm）
//
//	debug  -> Info   打印每条 SQL
//	info   -> Warn   只打印错误和慢查询
//	warn   -> Warn   同上
//	error  -> Error  只打印错误
//	silent -> Silent 不打印
func NewGormLogger(l *slog.Logger, level string, slowThreshold time.Duration) *GormLogger {
	if l == nil {
		l = slog.Default()
	}

	var lg gormlogger.LogLevel
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		lg = gormlogger.Info
	case "error":
		lg = gormlogger.Error
	case "silent":
		lg = gormlogger.Silent
	case "info", "warn", "warning", "":
		lg = gormlogger.Warn
	default:
		lg = gormlogger.Warn
	}

	if slowThreshold <= 0 {
		slowThreshold = 200 * time.Millisecond
	}

	return &GormLogger{
		slog:          l,
		level:         lg,
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
	l.slog.InfoContext(ctx, "[GORM] "+format(msg, data...))
}

func (l *GormLogger) Warn(ctx context.Context, msg string, data ...any) {
	if l.level < gormlogger.Warn {
		return
	}
	l.slog.WarnContext(ctx, "[GORM] "+format(msg, data...))
}

func (l *GormLogger) Error(ctx context.Context, msg string, data ...any) {
	if l.level < gormlogger.Error {
		return
	}
	l.slog.ErrorContext(ctx, "[GORM] "+format(msg, data...))
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
		l.slog.ErrorContext(ctx, "gorm: query failed",
			slog.String("error", err.Error()),
			slog.String("sql", sql),
			slog.Int64("rows", rows),
			slog.Duration("elapsed", elapsed),
		)
	case elapsed >= l.slowThreshold && l.level >= gormlogger.Warn:
		l.slog.WarnContext(ctx, "gorm: slow query",
			slog.String("sql", sql),
			slog.Int64("rows", rows),
			slog.Duration("elapsed", elapsed),
		)
	case l.level >= gormlogger.Info:
		l.slog.DebugContext(ctx, "gorm: query",
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
