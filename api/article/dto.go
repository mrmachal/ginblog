package article

import "time"

// 以下长度均按字符（rune）数计，上限需与 model.Article 的 gorm 标签保持一致。

type CreateArticleRequest struct {
	Title       string `json:"title" binding:"required,max=100"`        // 必填，≤100 字符
	Description string `json:"description" binding:"omitempty,max=255"` // 摘要可为空，若填则 ≤255 字符
	Content     string `json:"content" binding:"required,max=50000"`    // 必填，≤50000 字符
}

type UpdateArticleRequest struct {
	Title       string `json:"title" binding:"required,max=100"`
	Description string `json:"description" binding:"omitempty,max=255"`
	Content     string `json:"content" binding:"required,max=50000"`
}

type ListArticleQuery struct {
	Page     int `form:"page,default=1" binding:"min=1"`
	PageSize int `form:"page_size,default=20" binding:"min=1,max=100"`
	// 字段名或 -字段名（升序/降序），白名单校验，与 repository.List 的排序白名单一致
	Sort    string `form:"sort" binding:"omitempty,oneof=title -title description -description updated_at -updated_at created_at -created_at"`
	Keyword string `form:"keyword" binding:"omitempty,max=100"` // 关键词最长 100 字符
}

type DetailArticleResponse struct {
	ID          uint      `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type SummaryResponse struct {
	ID          uint      `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
