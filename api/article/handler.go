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
		_ = c.Error(fmt.Errorf("failed to list articles: %w", err))
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

func (h *Handler) Update(c *gin.Context) {
	var query struct {
		ID uint `form:"id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		_ = c.Error(fmt.Errorf("failed to bind request: %w", err))
		c.JSON(http.StatusBadRequest, response.Fail(response.CodeInvalidParam, response.MessageOf(response.CodeInvalidParam)))
		return
	}
	var req UpdateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(fmt.Errorf("failed to bind request body: %w", err))
		c.JSON(http.StatusBadRequest, response.Fail(response.CodeInvalidParam, response.MessageOf(response.CodeInvalidParam)))
		return
	}
	result, err := h.serv.Update(c, query.ID, req)
	if err != nil {
		_ = c.Error(fmt.Errorf("failed to update article: %w", err))
		c.JSON(http.StatusInternalServerError, response.Fail(response.CodeInternal, "更新文章失败"))
		return
	}

	c.JSON(http.StatusOK, response.Success(result))
}

func (h *Handler) Delete(c *gin.Context) {
	var query struct {
		ID uint `form:"id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		_ = c.Error(fmt.Errorf("failed to bind request: %w", err))
		c.JSON(http.StatusBadRequest, response.Fail(response.CodeInvalidParam, response.MessageOf(response.CodeInvalidParam)))
		return
	}
	if err := h.serv.Delete(c, query.ID); err != nil {
		_ = c.Error(fmt.Errorf("failed to delete article: %w", err))
		c.JSON(http.StatusInternalServerError, response.Fail(response.CodeInternal, response.MessageOf(response.CodeInternal)))
		return
	}
	c.JSON(http.StatusOK, response.Success(map[string]any{
		"id": query.ID,
	}))
}

func (h *Handler) Get(c *gin.Context) {
	var query struct {
		ID uint `form:"id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		_ = c.Error(fmt.Errorf("failed to bind request: %w", err))
		c.JSON(http.StatusBadRequest, response.Fail(response.CodeInvalidParam, response.MessageOf(response.CodeInvalidParam)))
		return
	}
	result, err := h.serv.GetByID(c, query.ID)
	if err != nil {
		_ = c.Error(fmt.Errorf("failed to get article: %w", err))
		c.JSON(http.StatusInternalServerError, response.Fail(response.CodeInternal, response.MessageOf(response.CodeInternal)))
		return
	}
	c.JSON(http.StatusOK, response.Success(result))
}
