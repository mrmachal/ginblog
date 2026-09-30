package article

import "github.com/gin-gonic/gin"

func SetRouter(r *gin.RouterGroup, h *Handler) {
	article := r.Group("/article")
	{
		article.GET("/list", h.List)
		article.GET("/detail", h.Get)
		article.POST("/create", h.Create)
		article.POST("/delete", h.Delete)
		article.POST("/update", h.Update)
	}
}
