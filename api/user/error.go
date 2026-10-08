package user

import "errors"

var (
	// ErrUserNotFound 表示用户不存在
	ErrUserNotFound = errors.New("user not found")
	// ErrInvalidCredential 表示登录失败
	ErrInvalidCredential = errors.New("invalid credential")
)
