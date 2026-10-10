package database

import (
	"errors"
	"path/filepath"
	"testing"

	"ginblog/model"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestForeignKeysSurviveConnectionRebuild 是连接级 pragma 的回归测试：
// 连接池里的连接被销毁重建后，foreign_keys 必须仍然生效。
//
// 历史 bug：pragma 原来是用 sqlDB.Exec("PRAGMA ...") 设置的，只作用于当时那一条
// 连接；SetConnMaxIdleTime 触发重建后，新连接默认 foreign_keys=OFF，
// 插入不存在的 user_id 不再报错，而是静默写出孤儿数据。
func TestForeignKeysSurviveConnectionRebuild(t *testing.T) {
	db, err := Init(Config{
		File:   filepath.Join(t.TempDir(), "test.db"),
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, e := db.DB(); e == nil {
			_ = sqlDB.Close()
		}
	})

	if err := Migrate(db, []any{&model.Article{}, &model.UserInfo{}}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB(): %v", err)
	}

	// 不缓存空闲连接：下一条查询必然新建连接，等价于「旧连接被回收后重建」
	sqlDB.SetMaxIdleConns(0)
	if _, err := sqlDB.Exec("SELECT 1"); err != nil {
		t.Fatalf("warm up: %v", err)
	}

	// 1) 重建出来的连接上，pragma 必须仍然生效
	var fk int64
	if err := sqlDB.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatalf("query pragma: %v", err)
	}
	if fk != 1 {
		t.Fatalf("foreign_keys = %d on rebuilt connection, want 1", fk)
	}

	// 2) 外键违约必须被检出，并被 TranslateError 翻译成 gorm.ErrForeignKeyViolated
	//    （否则 repository 层的 ErrUserNotFound 映射永远不会触发）
	bad := model.Article{Title: "t", Content: "c", UserID: 999999}
	err = db.Create(&bad).Error
	if !errors.Is(err, gorm.ErrForeignKeyViolated) {
		t.Fatalf("create with unknown user_id: err = %v, want gorm.ErrForeignKeyViolated", err)
	}

	// 3) 约束生效时不能留下孤儿数据
	var n int64
	if err := db.Model(&model.Article{}).Count(&n).Error; err != nil {
		t.Fatalf("count articles: %v", err)
	}
	if n != 0 {
		t.Fatalf("articles = %d, want 0 (FK violation must not insert)", n)
	}
}
