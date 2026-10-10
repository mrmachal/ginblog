package article

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"ginblog/middleware"
	"ginblog/pkg/response"

	"github.com/gin-gonic/gin"
)

// 这一层测试的价值：领域错误 → HTTP 状态码的映射是最容易漏配的地方。
// 曾经 ErrAuthorNotOwnsArticle 没有 case，导致「改别人的文章」返回 500 而不是 403，
// 而它恰好不需要数据库、不需要真实 service，是最便宜的回归防线。

func newTestContext(t *testing.T) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/article/update?id=1", nil)
	return c, w
}

func decodeBody(t *testing.T, w *httptest.ResponseRecorder) response.Response {
	t.Helper()
	var got response.Response
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("响应不是合法 JSON: %v, body=%s", err, w.Body.String())
	}
	return got
}

// TestWriteArticleWriteErrorMapping Update/Delete 共用的错误映射必须覆盖全部领域错误。
func TestWriteArticleWriteErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   int
	}{
		{
			name:       "文章不存在 → 404",
			err:        ErrArticleNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   response.CodeNotFound,
		},
		{
			name:       "无权限（非作者非管理员）→ 403",
			err:        ErrAuthorNotOwnsArticle,
			wantStatus: http.StatusForbidden,
			wantCode:   response.CodeForbidden,
		},
		{
			name:       "账号已删/身份失效 → 401",
			err:        ErrUserNotFound,
			wantStatus: http.StatusUnauthorized,
			wantCode:   response.CodeUnauthorized,
		},
		{
			// service 会把 repo 的 DB 错误包一层，errors.Is 必须仍然认得
			name:       "被 fmt.Errorf 包裹的领域错误 → 仍然映射正确",
			err:        fmt.Errorf("failed to get article: %w", ErrArticleNotFound),
			wantStatus: http.StatusNotFound,
			wantCode:   response.CodeNotFound,
		},
		{
			name:       "未知错误 → 500",
			err:        errors.New("db down"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   response.CodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := newTestContext(t)

			writeArticleWriteError(c, tt.err, "要更新的文章不存在", "更新文章失败")

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if got := decodeBody(t, w); got.Code != tt.wantCode {
				t.Fatalf("body.code = %d, want %d", got.Code, tt.wantCode)
			}
		})
	}
}

// TestWriteArticleWriteErrorRecordsUnknownErr 未知错误必须进 c.Errors（供日志中间件记录），
// 领域错误则不该被记成 error 级别噪音。
func TestWriteArticleWriteErrorRecordsUnknownErr(t *testing.T) {
	c, _ := newTestContext(t)
	writeArticleWriteError(c, errors.New("db down"), "更新文章失败", "更新文章失败")
	if len(c.Errors) != 1 {
		t.Fatalf("未知错误未被记录到 c.Errors: %v", c.Errors)
	}

	c2, _ := newTestContext(t)
	writeArticleWriteError(c2, ErrAuthorNotOwnsArticle, "不存在", "更新文章失败")
	if len(c2.Errors) != 0 {
		t.Fatalf("领域错误不应记入 c.Errors: %v", c2.Errors)
	}
}

// TestActorIDFromContext 路由漏挂 RequireAuth 时 GetUint 返回 0，
// 必须在入口挡成 401，而不是让 userID=0 流进 service。
func TestActorIDFromContext(t *testing.T) {
	c, w := newTestContext(t)
	if _, ok := actorIDFromContext(c); ok {
		t.Fatal("空 context 应当返回 ok=false")
	}
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}

	c2, _ := newTestContext(t)
	c2.Set(middleware.KeyUserID, uint(42))
	id, ok := actorIDFromContext(c2)
	if !ok || id != 42 {
		t.Fatalf("id = %d, ok = %v; want 42, true", id, ok)
	}
}
