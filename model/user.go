package model

import "gorm.io/gorm"

type UserRole string

const (
	RoleUser      UserRole = "user"
	RoleAdmin     UserRole = "admin"
	RoleModerator UserRole = "moderator"
)

type UserInfo struct {
	gorm.Model
	UserName     string   `json:"user_name" gorm:"varchar(100);uniqueIndex"`
	NickName     string   `json:"nick_name" gorm:"varchar(100);not null"`
	Email        string   `json:"email" gorm:"uniqueIndex"`
	PasswordHash string   `json:"-" gorm:"not null"`
	Role         UserRole `json:"role" gorm:"not null"`
}
