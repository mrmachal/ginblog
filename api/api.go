package api

import (
	"ginblog/api/article"
	"ginblog/api/user"
	"ginblog/config"
	"ginblog/pkg/session"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Init(db *gorm.DB, r *gin.Engine, sess *session.Store, auth config.AuthConfig) {
	api := r.Group("/api")
	article.Init(db, api)
	user.Init(db, api, sess, auth)
}
