package user

import "time"

// 以下长度均按字符（rune）数计，需与 model.UserInfo 的 gorm 标签保持一致：
// UserName ≤ 100、NickName ≤ 100、Email ≤ 100。
//
// 例外：Password 的 max=72 来自 x/crypto/bcrypt 的 72 字节硬限制，
// 超过会返回 ErrPasswordTooLong（在 service 层报错），故在 handler 层提前拦截。
// 注意 validator 的 max 按 rune 计，多字节密码仍可能超 72 字节，故收紧到 64。
//
// 以下规则由 binding 在 handler 层校验；跨字段/查库的规则归 service：
//   - UserName / Email 唯一性 → CodeConflict(20002)，靠 TranslateError 把
//     uniqueIndex 冲突转成 gorm.ErrDuplicatedKey，不要先 SELECT 再 INSERT（有竞态）
//   - 用户名字符集 ^[\w.-]+$ → CodeInvalidParam(10001)，validator 无内置标签
//   - 旧密码是否正确 → CodeUnauthorized(10002)
//   - 改的是否是自己的资料 → CodeForbidden(10003)

//-------Request-------

// RegisterRequest 用户注册。
type RegisterRequest struct {
	UserName        string `json:"user_name"        binding:"required,min=3,max=100"`
	NickName        string `json:"nick_name"        binding:"required,max=100"`
	Email           string `json:"email"            binding:"required,email,max=100"`
	Password        string `json:"password"         binding:"required,min=8,max=64"`
	ConfirmPassword string `json:"confirm_password" binding:"required,eqfield=Password"`
}

// LoginRequest 登录。成功后凭证经 Set-Cookie 下发，不进响应体。
type LoginRequest struct {
	UserName string `json:"user_name" binding:"required,max=100"`
	Password string `json:"password"  binding:"required,max=64"`
}

// UpdateMeRequest 修改个人资料，全部可选。
type UpdateMeRequest struct {
	NickName string `json:"nick_name" binding:"omitempty,max=100"`
	Email    string `json:"email"     binding:"omitempty,email,max=100"`
}

// ChangePWRequest 修改密码。新旧密码不得相同由 binding 直接校验。
type ChangePWRequest struct {
	OldPassword     string `json:"old_password" binding:"required,max=64"`
	NewPassword     string `json:"new_password" binding:"required,min=8,max=64,nefield=OldPassword"`
	ConfirmPassword string `json:"confirm_password" binding:"required,eqfield=NewPassword"`
}

// CreateUserRequest 管理员创建用户。比 ChangeUserRequest 多出账号与密码。
type CreateUserRequest struct {
	UserName string `json:"user_name" binding:"required,min=3,max=100"`
	NickName string `json:"nick_name" binding:"required,max=100"`
	Email    string `json:"email"     binding:"required,email,max=100"`
	Password string `json:"password"  binding:"required,min=8,max=64"`
	Role     string `json:"role"      binding:"omitempty,oneof=user moderator admin"`
}

// ChangeUserRequest 管理员修改他人资料。Role 可改是与 UpdateMeRequest 的唯一区别。
type ChangeUserRequest struct {
	NickName string `json:"nick_name" binding:"omitempty,max=100"`
	Email    string `json:"email"     binding:"omitempty,email,max=100"`
	Role     string `json:"role"      binding:"omitempty,oneof=user moderator admin"`
}

// ListUserQuery 管理员查询用户列表（query 参数用 form 标签）。
type ListUserQuery struct {
	Page     int `form:"page,default=1" binding:"min=1"`
	PageSize int `form:"page_size,default=20" binding:"min=1,max=100"`
	// 字段名或 -字段名（升序/降序），白名单校验，需与 repository.List 的排序白名单一致
	Sort    string `form:"sort" binding:"omitempty,oneof=user_name -user_name nick_name -nick_name email -email role -role created_at -created_at"`
	Keyword string `form:"keyword" binding:"omitempty,max=100"`
}

//-------Response-------

// MeResponse 当前登录用户（/api/user/me）。
// 故意不含 PasswordHash：模型上 json:"-" 是第一道防线，DTO 不给字段是第二道。
type MeResponse struct {
	ID        uint      `json:"id"`
	UserName  string    `json:"user_name"`
	NickName  string    `json:"nick_name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// UserResponse 用户列表项 / 管理员视角的单个用户。
// 与 MeResponse 的差别在于暴露 UpdatedAt 与最后活跃时间，供管理端使用。
type UserResponse struct {
	ID        uint      `json:"id"`
	UserName  string    `json:"user_name"`
	NickName  string    `json:"nick_name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// LoginResponse 登录响应。
// session 方案下凭证由 handler 用 c.SetCookie 写进 Set-Cookie 头（HttpOnly），
// 不再有 token / expires_at 字段——cookie 是 HTTP 关注点，不是业务数据。
type LoginResponse struct {
	User MeResponse `json:"user"`
}

// LogoutResponse 登出响应。真正动作是清 cookie + 删 session 记录。
type LogoutResponse struct {
	OK bool `json:"ok"`
}

// SessionUserResponse 会话中的用户摘要（可选，用于 /api/user/sessions 返回当前会话列表）。
type SessionUserResponse struct {
	SessionID string    `json:"session_id"` // 前端展示用的短标识，非原始 token
	UserAgent string    `json:"user_agent"`
	IP        string    `json:"ip"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}
