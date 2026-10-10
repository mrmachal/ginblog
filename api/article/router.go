package article

import (
	"ginblog/config"
	"ginblog/middleware"

	"github.com/gin-gonic/gin"
)

func SetRouter(r *gin.RouterGroup, h *Handler, sess middleware.SessionStore, auth config.AuthConfig) {
	article := r.Group("/article")
	{
		article.GET("/list", h.List)
		article.GET("/detail", h.Get)
		authGroup := article.Group("")
		authGroup.Use(middleware.RequireAuth(sess, auth))
		{
			authGroup.POST("/create", h.Create)
			authGroup.POST("/delete", h.Delete)
			authGroup.POST("/update", h.Update)
		}
	}
}
