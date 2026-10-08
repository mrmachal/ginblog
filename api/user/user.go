package user

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Init(db *gorm.DB, r *gin.RouterGroup) {
	repo := NewUserRepository(db)
	serv := NewUserService(repo)
	handler := NewUserHandler(serv)
	SetRouter(r, handler)
}
