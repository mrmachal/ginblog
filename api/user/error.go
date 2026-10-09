package user

import "errors"

var (
	// ErrUserNotFound 表示用户不存在
	ErrUserNotFound = errors.New("user not found")
	// ErrInvalidCredential 表示登录失败
	ErrInvalidCredential = errors.New("invalid credential")
	// ErrUserNameAlreadyExists 表示用户名已存在
	ErrUserNameAlreadyExists = errors.New("user name already exists")
	// ErrEmailAlreadyExists 表示邮箱已存在
	ErrEmailAlreadyExists = errors.New("email already exists")
)
