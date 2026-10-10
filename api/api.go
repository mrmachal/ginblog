package api

import (
	"ginblog/api/article"
	"ginblog/api/user"
	"ginblog/config"
	"ginblog/pkg/session"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Init 是全局唯一的 composition root：repo、会话存储等共享资源只在这里 new，
// 各模块只接收依赖、挂路由，拿不到 *gorm.DB，因此不可能偷偷构造别的模块的 repository。
func Init(db *gorm.DB, r *gin.Engine, sess *session.Store, auth config.AuthConfig) {
	api := r.Group("/api")

	userRepo := user.NewUserRepository(db)
	articleRepo := article.NewArticleRepository(db)

	article.Init(api, sess, auth, articleRepo, userRepo)
	user.Init(api, sess, auth, userRepo)
}
