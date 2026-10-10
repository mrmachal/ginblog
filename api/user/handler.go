package user

import (
	"errors"
	"fmt"
	"ginblog/config"
	"ginblog/middleware"
	"ginblog/pkg/response"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	serv *Service
	auth config.AuthConfig
}

func NewUserHandler(serv *Service, auth config.AuthConfig) *Handler {
	return &Handler{serv: serv, auth: auth}
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Fail(response.CodeInvalidParam, response.MessageOf(response.CodeInvalidParam)))
		return
	}
	token, res, err := h.serv.Login(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidCredential) {
			c.JSON(http.StatusUnauthorized, response.Fail(response.CodeUnauthorized, "用户名或密码错误"))
			return
		}
		_ = c.Error(fmt.Errorf("failed to login: %w", err))
		c.JSON(http.StatusInternalServerError, response.Fail(response.CodeInternal, "登录失败"))
		return
	}

	c.SetCookie(h.auth.CookieName, token, h.auth.ExpireHours*3600, "/", "", h.auth.Secure, true)

	c.JSON(http.StatusOK, response.Success(res))
}

func (h *Handler) Logout(c *gin.Context) {
	token, err := c.Cookie(h.auth.CookieName)
	if err != nil || token == "" {
		c.JSON(http.StatusOK, response.Success(&LogoutResponse{OK: true}))
		return
	}

	if err := h.serv.Logout(c.Request.Context(), token); err != nil {
		_ = c.Error(fmt.Errorf("failed to logout: %w", err))
		c.JSON(http.StatusInternalServerError, response.Fail(response.CodeInternal, "登出失败"))
		return
	}

	c.SetCookie(h.auth.CookieName, "", -1, "/", "", h.auth.Secure, true)
	c.JSON(http.StatusOK, response.Success(&LogoutResponse{OK: true}))
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Fail(response.CodeInvalidParam, response.MessageOf(response.CodeInvalidParam)))
		return
	}

	res, err := h.serv.Register(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrUserNameAlreadyExists) {
			c.JSON(http.StatusConflict, response.Fail(response.CodeConflict, "用户名已存在"))
			return
		}
		if errors.Is(err, ErrEmailAlreadyExists) {
			c.JSON(http.StatusConflict, response.Fail(response.CodeConflict, "邮箱已存在"))
			return
		}
		_ = c.Error(fmt.Errorf("failed to register: %w", err))
		c.JSON(http.StatusInternalServerError, response.Fail(response.CodeInternal, "注册失败"))
		return
	}

	c.JSON(http.StatusOK, response.Success(res))
}

func (h *Handler) Me(c *gin.Context) {
	userID := c.GetUint(middleware.KeyUserID)
	res, err := h.serv.Me(c.Request.Context(), userID)
	if err != nil {
		_ = c.Error(fmt.Errorf("failed to get me: %w", err))
		c.JSON(http.StatusInternalServerError, response.Fail(response.CodeInternal, "获取用户信息失败"))
		return
	}

	c.JSON(http.StatusOK, response.Success(res))
}

func (h *Handler) ChangePassword(c *gin.Context) {
	userID := c.GetUint(middleware.KeyUserID)
	var req ChangePWRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Fail(response.CodeInvalidParam, response.MessageOf(response.CodeInvalidState)))
		return
	}

	res, err := h.serv.ChangePassword(c.Request.Context(), userID, req)
	if err != nil {
		if errors.Is(err, ErrInvalidCredential) {
			c.JSON(http.StatusConflict, response.Fail(response.CodeConflict, "密码不正确"))
			return
		}

		_ = c.Error(fmt.Errorf("failed to change password: %w", err))
		c.JSON(http.StatusInternalServerError, response.Fail(response.CodeInternal, "修改密码失败"))
		return
	}

	c.JSON(http.StatusOK, response.Success(res))
}

func (h *Handler) UpdateMe(c *gin.Context) {
	userID := c.GetUint(middleware.KeyUserID)
	var req UpdateMeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Fail(response.CodeInvalidParam, response.MessageOf(response.CodeInvalidParam)))
		return
	}
	res, err := h.serv.UpdateMe(c.Request.Context(), userID, req)
	if err != nil {
		if errors.Is(err, ErrEmailAlreadyExists) {
			c.JSON(http.StatusConflict, response.Fail(response.CodeConflict, "邮箱已存在"))
			return
		}
		_ = c.Error(fmt.Errorf("failed to update userinfo: %w", err))
		c.JSON(http.StatusInternalServerError, response.Fail(response.CodeInternal, "更新信息失败"))
		return
	}

	c.JSON(http.StatusOK, response.Success(res))
}

func (h *Handler) ChangeUser(c *gin.Context) {
	var query struct {
		ID uint `form:"id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, response.Fail(response.CodeInvalidParam, response.MessageOf(response.CodeInvalidParam)))
		return
	}
	var req ChangeUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Fail(response.CodeInvalidParam, response.MessageOf(response.CodeInvalidParam)))
		return
	}

	res, err := h.serv.ChangeUser(c.Request.Context(), query.ID, req)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			c.JSON(http.StatusNotFound, response.Fail(response.CodeNotFound, "用户不存在"))
			return
		}
		_ = c.Error(fmt.Errorf("failed to change userinfo: %w", err))
		c.JSON(http.StatusInternalServerError, response.Fail(response.CodeInternal, "修改用户信息出错"))
		return
	}

	c.JSON(http.StatusOK, response.Success(res))

}

func (h *Handler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Fail(response.CodeInvalidParam, response.MessageOf(response.CodeInvalidParam)))
		return
	}
	res, err := h.serv.CreateUser(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrUserNameAlreadyExists) {
			c.JSON(http.StatusConflict, response.Fail(response.CodeConflict, "用户名已存在"))
			return
		}

		if errors.Is(err, ErrEmailAlreadyExists) {
			c.JSON(http.StatusConflict, response.Fail(response.CodeConflict, "邮箱已存在"))
			return
		}

		_ = c.Error(fmt.Errorf("failed to create user: %w", err))
		c.JSON(http.StatusInternalServerError, response.Fail(response.CodeInternal, "创建用户失败"))
		return
	}

	c.JSON(http.StatusOK, response.Success(res))
}

func (h *Handler) ListUser(c *gin.Context) {
	var query ListUserQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, response.Fail(response.CodeInvalidParam, response.MessageOf(response.CodeInvalidParam)))
		return
	}

	res, total, err := h.serv.ListUser(c.Request.Context(), query)
	if err != nil {
		_ = c.Error(fmt.Errorf("failed to get user list: %w", err))
		c.JSON(http.StatusInternalServerError, response.Fail(response.CodeInternal, "获取用户列表失败"))
		return
	}

	c.JSON(http.StatusOK, response.SuccessPage(total, res, query.Page, query.PageSize))
}

func (h *Handler) GetDetail(c *gin.Context) {
	var query struct {
		ID uint `form:"id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, response.Fail(response.CodeInvalidParam, response.MessageOf(response.CodeInvalidParam)))
		return
	}

	user, err := h.serv.GetDetail(c.Request.Context(), query.ID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			c.JSON(http.StatusNotFound, response.Fail(response.CodeNotFound, "用户不存在"))
			return
		}

		_ = c.Error(fmt.Errorf("failed to get userinfo: %w", err))
		c.JSON(http.StatusInternalServerError, response.Fail(response.CodeInternal, "获取用户信息失败"))
		return
	}

	c.JSON(http.StatusOK, response.Success(user))
}

func (h *Handler) DeleteUser(c *gin.Context) {
	var query struct {
		ID uint `form:"id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, response.Fail(response.CodeInvalidParam, response.MessageOf(response.CodeInvalidParam)))
		return
	}

	if err := h.serv.DeleteUser(c.Request.Context(), query.ID); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			c.JSON(http.StatusNotFound, response.Fail(response.CodeNotFound, "用户不存在"))
			return
		}

		_ = c.Error(fmt.Errorf("failed to delete user: %w", err))
		c.JSON(http.StatusInternalServerError, response.Fail(response.CodeInternal, "删除用户失败"))
		return
	}

	c.JSON(http.StatusOK, response.Success(nil))
}
