package model

import "gorm.io/gorm"

// Article 文章表。
// 字段长度限制与 api/article/dto.go 中 binding 校验保持一致：
// Title ≤ 100 字符、Description ≤ 255 字符、Content ≤ 50000 字符。
type Article struct {
	gorm.Model
	Title       string   `json:"title" gorm:"type:varchar(100);not null;default:''"`
	Description string   `json:"description" gorm:"type:varchar(255);not null;default:''"`
	Content     string   `json:"content" gorm:"type:mediumtext;not null"`
	UserID      uint     `json:"user_id" gorm:"index;not null"`
	User        UserInfo `json:"user,omitzero" gorm:"foreignKey:UserID"`
}
