package article

import "errors"

var (
	// ErrArticleNotFound 文章不存在（含已软删），上层按 404 处理。
	ErrArticleNotFound = errors.New("article not found")

	// ErrAuthorNotFound 作者不存在或无效：写入时 user_id 外键违约，
	// 典型场景是 session 尚未过期但对应用户已被删除，
	// 语义上是「当前登录身份失效」，上层按 401 处理。
	ErrAuthorNotFound = errors.New("author not found")
)
