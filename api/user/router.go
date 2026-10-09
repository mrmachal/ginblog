package user

import (
	"ginblog/config"
	"ginblog/middleware"
	"ginblog/model"

	"github.com/gin-gonic/gin"
)

func SetRouter(r *gin.RouterGroup, h *Handler, sess middleware.SessionStore, auth config.AuthConfig, repo middleware.UserReader) {
	// 不需要认证的接口
	{
		r.POST("/login", h.Login)
		r.POST("/register", h.Register)
	}

	// 需要认证的接口
	authGroup := r.Group("")
	authGroup.Use(middleware.RequireAuth(sess, auth))
	{
		authGroup.POST("/logout", h.Logout)
		authGroup.GET("/me", h.Me)
		authGroup.POST("/change_password", h.ChangePassword)
		authGroup.POST("/update_me", h.UpdateMe)
		userManage := authGroup.Group("/user")
		userManage.Use(middleware.RequireAdmin(repo, model.RoleAdmin))
		{
			userManage.GET("/list", h.ListUser)
			userManage.GET("/detail", h.GetDetail)
			userManage.POST("/create", h.CreateUser)
			userManage.POST("/update", h.ChangeUser)
			userManage.POST("/delete", h.DeleteUser)
		}
	}
}
