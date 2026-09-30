package article

import (
	"fmt"
	"ginblog/pkg/response"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	serv *Service
}

func NewHandler(serv *Service) *Handler {
	return &Handler{serv: serv}
}

func (h *Handler) List(c *gin.Context) {
	var q ListArticleQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		_ = c.Error(fmt.Errorf("failed to bind query params: %w", err))
		c.JSON(http.StatusBadRequest, response.Fail(response.CodeInvalidParam, response.MessageOf(response.CodeInvalidParam)))
		return
	}
	articleList, total, err := h.serv.List(c, q)
	if err != nil {
		_ = c.Error(fmt.Errorf("failed to list articles:%w", err))
		c.JSON(http.StatusInternalServerError, response.Fail(response.CodeInternal, "获取文章列表失败"))
		return
	}
	c.JSON(http.StatusOK, response.SuccessPage(total, articleList, q.Page, q.PageSize))
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(fmt.Errorf("failed to bind request body: %w", err))
		c.JSON(http.StatusBadRequest, response.Fail(response.CodeInvalidParam, response.MessageOf(response.CodeInvalidParam)))
		return
	}
	res, err := h.serv.Create(c, req)
	if err != nil {
		_ = c.Error(fmt.Errorf("failed to create article: %w", err))
		c.JSON(http.StatusInternalServerError, response.Fail(response.CodeInternal, "创建文章失败"))
		return
	}

	c.JSON(http.StatusOK, response.Success(res))
}
