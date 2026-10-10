package article

import (
	"context"
	"errors"
	"testing"

	"ginblog/model"
)

// ---- 测试替身 ----------------------------------------------------------
//
// 鉴权是纯策略（管理员 || 作者本人），不需要数据库。用 fake 覆盖规则分支，
// 比在 repository_test.go 里插用户行 + 文章行的成本低一个数量级——
// 这正是「策略放 service」的实际收益。

type fakeRoleProvider struct {
	role  model.UserRole
	found bool
	err   error
}

func (f fakeRoleProvider) Role(context.Context, uint) (model.UserRole, bool, error) {
	return f.role, f.found, f.err
}

type fakeRepo struct {
	article *model.Article // nil 表示文章不存在
	gotErr  error          // 非 nil 时 GetById 返回它

	saved   *model.Article
	deleted bool
}

func (f *fakeRepo) GetById(context.Context, uint) (*model.Article, error) {
	if f.gotErr != nil {
		return nil, f.gotErr
	}
	if f.article == nil {
		return nil, ErrArticleNotFound
	}
	return f.article, nil
}

func (f *fakeRepo) Save(_ context.Context, a *model.Article) error {
	f.saved = a
	return nil
}

func (f *fakeRepo) Delete(context.Context, uint) error {
	f.deleted = true
	return nil
}

func (f *fakeRepo) Create(context.Context, *model.Article) error { return nil }
func (f *fakeRepo) List(context.Context, ListArticleQuery) ([]*model.Article, int64, error) {
	return nil, 0, nil
}

// ---- 鉴权矩阵 ----------------------------------------------------------

// TestAuthorizeMatrix 覆盖「作者或管理员才能改/删」的全部规则分支。
func TestAuthorizeMatrix(t *testing.T) {
	const (
		articleOwner = uint(10)
		otherUser    = uint(20)
	)

	tests := []struct {
		name    string
		role    model.UserRole
		found   bool
		roleErr error
		article *model.Article // nil = 文章不存在
		actorID uint
		wantErr error // nil 表示放行
	}{
		{
			name: "作者改自己的文章 → 放行",
			role: model.RoleUser, found: true,
			article: &model.Article{UserID: articleOwner}, actorID: articleOwner,
			wantErr: nil,
		},
		{
			name: "管理员改别人的文章 → 放行",
			role: model.RoleAdmin, found: true,
			article: &model.Article{UserID: articleOwner}, actorID: otherUser,
			wantErr: nil,
		},
		{
			name: "管理员改不存在的文章 → 404",
			role: model.RoleAdmin, found: true,
			article: nil, actorID: otherUser,
			wantErr: ErrArticleNotFound,
		},
		{
			name: "普通用户改别人的文章 → 403",
			role: model.RoleUser, found: true,
			article: &model.Article{UserID: articleOwner}, actorID: otherUser,
			wantErr: ErrAuthorNotOwnsArticle,
		},
		{
			name: "普通用户改不存在的文章 → 404（不是 403，条件判断会掩盖文章不存在）",
			role: model.RoleUser, found: true,
			article: nil, actorID: otherUser,
			wantErr: ErrArticleNotFound,
		},
		{
			name: "moderator 不在策略内，视同普通用户 → 403",
			role: model.RoleModerator, found: true,
			article: &model.Article{UserID: articleOwner}, actorID: otherUser,
			wantErr: ErrAuthorNotOwnsArticle,
		},
		{
			name: "会话还在但账号已删 → 401 语义，不是 403",
			role: "", found: false,
			article: &model.Article{UserID: articleOwner}, actorID: articleOwner,
			wantErr: ErrUserNotFound,
		},
		{
			name: "查角色时 DB 故障 → 透传非领域错误（上层 500）",
			role: "", found: false, roleErr: errors.New("db down"),
			article: &model.Article{UserID: articleOwner}, actorID: articleOwner,
			wantErr: nil, // 用下面的断言单独判定
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{article: tt.article}
			svc := NewService(repo, fakeRoleProvider{role: tt.role, found: tt.found, err: tt.roleErr})

			_, err := svc.authorize(context.Background(), tt.actorID, 1)

			if tt.roleErr != nil {
				// 基础设施故障必须原样透出，不能被翻译成 401/403 而被掩盖
				if err == nil || errors.Is(err, ErrUserNotFound) || errors.Is(err, ErrAuthorNotOwnsArticle) {
					t.Fatalf("DB 故障被误判为领域错误: %v", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("authorize() err = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// TestDeleteRequiresOwnership 删除路径同样受鉴权约束，且无权时不得触达 repo。
func TestDeleteRequiresOwnership(t *testing.T) {
	repo := &fakeRepo{article: &model.Article{UserID: 10}}
	svc := NewService(repo, fakeRoleProvider{role: model.RoleUser, found: true})

	err := svc.Delete(context.Background(), 20, 1)
	if !errors.Is(err, ErrAuthorNotOwnsArticle) {
		t.Fatalf("err = %v, want ErrAuthorNotOwnsArticle", err)
	}
	if repo.deleted {
		t.Fatal("鉴权失败仍然执行了删除")
	}
}

// TestUpdateByAdminAllowed 管理员改他人文章应真正落库。
func TestUpdateByAdminAllowed(t *testing.T) {
	repo := &fakeRepo{article: &model.Article{UserID: 10, Title: "old", Content: "old"}}
	svc := NewService(repo, fakeRoleProvider{role: model.RoleAdmin, found: true})

	res, err := svc.Update(context.Background(), 99, 1, UpdateArticleRequest{
		Title: "new", Description: "d", Content: "c",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if repo.saved == nil || repo.saved.Title != "new" || repo.saved.UserID != 10 {
		t.Fatalf("保存结果不符：%+v", repo.saved)
	}
	// 管理员改他人文章不得顺手改掉归属
	if res.Title != "new" {
		t.Fatalf("res.Title = %q", res.Title)
	}
}
