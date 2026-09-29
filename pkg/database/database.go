package database

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

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

	db, err := gorm.Open(sqlite.Open(cfg.File), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to open db file: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	for _, p := range []string{
		"PRAGMA journal_mode = WAL;",   // WAL 读写并发，性能好很多
		"PRAGMA busy_timeout = 5000;",  // 锁等待 5s，而不是立刻报错
		"PRAGMA foreign_keys = ON;",    // 外键约束默认是关的！
		"PRAGMA synchronous = NORMAL;", // WAL 下安全且更快
	} {
		if _, err := sqlDB.Exec(p); err != nil {
			return nil, fmt.Errorf("failed to exec pragma: %w", err)
		}
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
