package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// checkPragma 读取连接级 pragma 的当前值并断言期望值，不符合直接返回错误。
// 这里查的是刚从 DSN 建出来的新连接，能同时发现「DSN 参数拼错」与「参数没生效」。
func checkPragma(sqlDB *sql.DB, name string, want int64) error {
	var got int64
	if err := sqlDB.QueryRow("PRAGMA " + name).Scan(&got); err != nil {
		return fmt.Errorf("failed to check pragma %s: %w", name, err)
	}
	if got != want {
		return fmt.Errorf("pragma %s = %d, want %d", name, got, want)
	}
	return nil
}

type Config struct {
	File   string
	Logger gormlogger.Interface // 可选，nil 时使用 GORM 默认 logger
}

func Init(cfg Config) (*gorm.DB, error) {
	if cfg.File == "" {
		return nil, fmt.Errorf("database file is empty")
	}

	// 确保数据库文件所在目录存在
	if dir := filepath.Dir(cfg.File); dir != "." && dir != "" && cfg.File != ":memory:" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create db dir: %w", err)
		}
	}
	gormCfg := &gorm.Config{
		PrepareStmt:    true,
		TranslateError: true,
	}
	if cfg.Logger != nil {
		gormCfg.Logger = cfg.Logger
	}

	dsn := cfg.File
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	dsn += sep + "_pragma=busy_timeout(5000)" +
		"&_pragma=foreign_keys(1)" +
		"&_pragma=synchronous(NORMAL)"

	db, err := gorm.Open(sqlite.Open(dsn), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to open db file: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// journal_mode 是文件级、持久化的（写进 DB 文件头），不属于连接级，
	// 用 Exec 设一次即可；WAL 读写并发，性能好很多
	if _, err := sqlDB.Exec("PRAGMA journal_mode = WAL;"); err != nil {
		return nil, fmt.Errorf("failed to exec pragma: %w", err)
	}

	// 启动期校验（fail-fast）：连接级 pragma 必须已生效。
	// 宁可启动失败，也不要让外键约束静默放开——那会让写入路径悄悄产生脏数据
	if err := checkPragma(sqlDB, "foreign_keys", 1); err != nil {
		return nil, err
	}
	if err := checkPragma(sqlDB, "busy_timeout", 5000); err != nil {
		return nil, err
	}

	// 连接池：SQLite 单文件，写是串行的，限制并发避免自锁
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxIdleTime(time.Hour)

	return db, nil
}

func Migrate(db *gorm.DB, models []any) error {
	return db.AutoMigrate(
		models...,
	)
}
