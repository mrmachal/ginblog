package article

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Init(db *gorm.DB, r *gin.RouterGroup) {
	repo := NewArticleRepository(db)
	serv := NewService(repo)
	handler := NewHandler(serv)
	SetRouter(r, handler)
}
