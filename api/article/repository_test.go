package article

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"ginblog/model"
	"ginblog/pkg/database"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTestRepo(t *testing.T) (Repository, *gorm.DB) {
	t.Helper()
	db, err := database.Init(database.Config{
		File:   filepath.Join(t.TempDir(), "test.db"),
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, e := db.DB(); e == nil {
			_ = sqlDB.Close()
		}
	})
	if err := database.Migrate(db, []any{&model.Article{}, &model.UserInfo{}}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewArticleRepository(db), db
}

// TestCreateRejectsMissingAuthor 作者不存在时必须返回领域错误，
// 且不能落库——否则 session 里过期的 userID 会静默写出孤儿文章。
func TestCreateRejectsMissingAuthor(t *testing.T) {
	repo, db := newTestRepo(t)

	err := repo.Create(context.Background(), &model.Article{
		Title:   "t",
		Content: "c",
		UserID:  999999,
	})
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("create with unknown author: err = %v, want ErrUserNotFound", err)
	}

	var n int64
	if err := db.Model(&model.Article{}).Count(&n).Error; err != nil {
		t.Fatalf("count articles: %v", err)
	}
	if n != 0 {
		t.Fatalf("articles = %d, want 0", n)
	}
}

// TestCreateFillsAuthorName 作者存在时创建成功，并回填昵称用于回显
// （gorm Create 只写 user_id 外键列，不会填充 User 关联）。
func TestCreateFillsAuthorName(t *testing.T) {
	repo, db := newTestRepo(t)

	author := model.UserInfo{
		UserName:     "u1",
		NickName:     "Nick",
		Email:        "u1@example.com",
		PasswordHash: "hash",
		Role:         model.RoleUser,
	}
	if err := db.Create(&author).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	article := &model.Article{Title: "t", Content: "c", UserID: author.ID}
	if err := repo.Create(context.Background(), article); err != nil {
		t.Fatalf("create article: %v", err)
	}
	if article.ID == 0 {
		t.Fatal("article.ID = 0, want inserted id")
	}
	if article.User.NickName != "Nick" {
		t.Fatalf("author name = %q, want %q", article.User.NickName, "Nick")
	}
}
