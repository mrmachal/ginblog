package article

import "errors"

var (
	// ErrArticleNotFound 文章不存在（含已软删），上层按 404 处理。
	ErrArticleNotFound = errors.New("article not found")

	// ErrUserNotFound 用户不存在或无效，上层按 401 处理（当前登录身份已失效）。
	// 两个场景归到这里，因为对调用方的含义相同——都是「这个 user id 不对」：
	//   - 创建文章时 user_id 外键违约（作者不存在）
	//   - 更新/删除时当前操作者不存在（session 未过期但账号已被删除）
	// 注意：它不代表「用户存在但无权限」，那是 ErrAuthorNotOwnsArticle（403）。
	ErrUserNotFound = errors.New("user not found")

	// ErrAuthorNotOwnsArticle 操作者既不是作者也不是管理员，上层按 403 处理。
	// 判定逻辑在 service.authorize，不在 repository——repo 只回数据事实，不回权限结论。
	ErrAuthorNotOwnsArticle = errors.New("author not own article")
)
