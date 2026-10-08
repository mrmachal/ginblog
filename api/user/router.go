package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetRouter(r *gin.RouterGroup, h *Handler) {
	article := r.Group("/user")
	{
		article.GET("/", func(context *gin.Context) {
			context.JSON(http.StatusOK, gin.H{
				"message": "this interface is not useful",
			})
		})
	}
}
