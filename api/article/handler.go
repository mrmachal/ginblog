package article

import (
	"errors"
	"fmt"
	"ginblog/middleware"
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

func writeArticleWriteError(c *gin.Context, err error, notFoundMsg, failMsg string) {
	switch {
	case errors.Is(err, ErrArticleNotFound):
		c.JSON(http.StatusNotFound, response.Fail(response.CodeNotFound, notFoundMsg))
	case errors.Is(err, ErrAuthorNotOwnsArticle):
		c.JSON(http.StatusForbidden, response.Fail(response.CodeForbidden, response.MessageOf(response.CodeForbidden)))
	case errors.Is(err, ErrUserNotFound):
		// session 还活着，但账号已不存在：按登录态失效处理，让前端重新登录
		c.JSON(http.StatusUnauthorized, response.Fail(response.CodeUnauthorized, "登录状态已失效，请重新登录"))
	default:
		_ = c.Error(fmt.Errorf("%s: %w", failMsg, err))
		c.JSON(http.StatusInternalServerError, response.Fail(response.CodeInternal, failMsg))
	}
}

func actorIDFromContext(c *gin.Context) (uint, bool) {
	userID := c.GetUint(middleware.KeyUserID)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, response.Fail(response.CodeUnauthorized, response.MessageOf(response.CodeUnauthorized)))
		return 0, false
	}
	return userID, true
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
	userID, ok := actorIDFromContext(c)
	if !ok {
		return
	}
	var req CreateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(fmt.Errorf("failed to bind request body: %w", err))
		c.JSON(http.StatusBadRequest, response.Fail(response.CodeInvalidParam, response.MessageOf(response.CodeInvalidParam)))
		return
	}
	res, err := h.serv.Create(c, userID, req)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			c.JSON(http.StatusUnauthorized, response.Fail(response.CodeUnauthorized, "登录状态已失效，请重新登录"))
			return
		}
		_ = c.Error(fmt.Errorf("failed to create article: %w", err))
		c.JSON(http.StatusInternalServerError, response.Fail(response.CodeInternal, "创建文章失败"))
		return
	}

	c.JSON(http.StatusOK, response.Success(res))
}

func (h *Handler) Update(c *gin.Context) {
	actorID, ok := actorIDFromContext(c)
	if !ok {
		return
	}
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
	result, err := h.serv.Update(c, actorID, query.ID, req)
	if err != nil {
		writeArticleWriteError(c, err, "要更新的文章不存在", "更新文章失败")
		return
	}

	c.JSON(http.StatusOK, response.Success(result))
}

func (h *Handler) Delete(c *gin.Context) {
	actorID, ok := actorIDFromContext(c)
	if !ok {
		return
	}
	var query struct {
		ID uint `form:"id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		_ = c.Error(fmt.Errorf("failed to bind request: %w", err))
		c.JSON(http.StatusBadRequest, response.Fail(response.CodeInvalidParam, response.MessageOf(response.CodeInvalidParam)))
		return
	}
	if err := h.serv.Delete(c, actorID, query.ID); err != nil {
		writeArticleWriteError(c, err, "要删除的文章不存在", "删除文章失败")
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
		if errors.Is(err, ErrArticleNotFound) {
			c.JSON(http.StatusNotFound, response.Fail(response.CodeNotFound, "文章不存在"))
			return
		}
		_ = c.Error(fmt.Errorf("failed to get article: %w", err))
		c.JSON(http.StatusInternalServerError, response.Fail(response.CodeInternal, response.MessageOf(response.CodeInternal)))
		return
	}
	c.JSON(http.StatusOK, response.Success(result))
}
