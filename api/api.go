package api

import (
	"ginblog/api/article"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Init(db *gorm.DB, r *gin.Engine) {
	api := r.Group("/api")
	article.Init(db, api)
}
