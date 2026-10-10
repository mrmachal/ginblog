package article

import (
	"context"
	"errors"
	"fmt"
	"ginblog/model"
	"strings"

	"gorm.io/gorm"
)

type RoleProvider interface {
	Role(c context.Context, userID uint) (model.UserRole, bool, error)
}

type Service struct {
	repo  Repository
	roles RoleProvider
}

func NewService(repo Repository, roles RoleProvider) *Service {
	return &Service{repo: repo, roles: roles}
}

func (s *Service) authorize(c context.Context, actorID, articleID uint) (*model.Article, error) {
	role, found, err := s.roles.Role(c, actorID)
	if err != nil {
		return nil, fmt.Errorf("failed to load actor role: %w", err)
	}
	if !found {
		return nil, ErrUserNotFound
	}

	article, err := s.repo.GetById(c, articleID)
	if err != nil {
		if errors.Is(err, ErrArticleNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get article: %w", err)
	}

	if role == model.RoleAdmin || article.UserID == actorID {
		return article, nil
	}
	return nil, ErrAuthorNotOwnsArticle
}

func (s *Service) List(c context.Context, q ListArticleQuery) ([]SummaryResponse, int64, error) {
	articleList, total, err := s.repo.List(c, q)
	if err != nil {
		return nil, 0, fmt.Errorf("list articles failed: %w", err)
	}
	out := make([]SummaryResponse, 0, len(articleList))
	for _, item := range articleList {
		out = append(out, SummaryResponse{
			ID:          item.ID,
			Title:       item.Title,
			Description: item.Description,
			UpdatedAt:   item.UpdatedAt,
			CreatedAt:   item.CreatedAt,
			AuthorName:  item.User.NickName,
		})
	}
	return out, total, nil
}

func (s *Service) GetByID(c context.Context, articleID uint) (DetailArticleResponse, error) {
	article, err := s.repo.GetById(c, articleID)
	if err != nil {
		if errors.Is(err, ErrArticleNotFound) {
			return DetailArticleResponse{}, err
		}
		return DetailArticleResponse{}, fmt.Errorf("failed to get article detail: %w", err)
	}
	return DetailArticleResponse{
		ID:          article.ID,
		Title:       article.Title,
		Description: article.Description,
		Content:     article.Content,
		CreatedAt:   article.CreatedAt,
		UpdatedAt:   article.UpdatedAt,
		AuthorID:    article.UserID,
		AuthorName:  article.User.NickName,
	}, nil
}

func (s *Service) Create(c context.Context, userID uint, articleReq CreateArticleRequest) (SummaryResponse, error) {
	title := strings.TrimSpace(articleReq.Title)
	if title == "" {
		return SummaryResponse{}, fmt.Errorf("title must not be blank")
	}
	if strings.TrimSpace(articleReq.Content) == "" {
		return SummaryResponse{}, fmt.Errorf("content must not be blank")
	}

	if userID == 0 {
		return SummaryResponse{}, ErrUserNotFound
	}
	article := model.Article{
		Title:       articleReq.Title,
		Description: articleReq.Description,
		Content:     articleReq.Content,
		UserID:      userID,
	}
	if err := s.repo.Create(c, &article); err != nil {
		// repo 已把外键违约翻译成 ErrUserNotFound；这里保留对原始 gorm 错误的判断，
		// 兜住驱动/方言未翻译的情况，避免漏成 500
		if errors.Is(err, ErrUserNotFound) || errors.Is(err, gorm.ErrForeignKeyViolated) {
			return SummaryResponse{}, ErrUserNotFound
		}
		return SummaryResponse{}, fmt.Errorf("failed to create article: %w", err)
	}
	return SummaryResponse{
		ID:          article.ID,
		Title:       article.Title,
		Description: article.Description,
		CreatedAt:   article.CreatedAt,
		UpdatedAt:   article.UpdatedAt,
		AuthorName:  article.User.NickName,
	}, nil
}

func (s *Service) Delete(c context.Context, actorID, articleID uint) error {
	// 鉴权同时完成存在性校验（文章不存在 → ErrArticleNotFound）
	if _, err := s.authorize(c, actorID, articleID); err != nil {
		return err
	}

	if err := s.repo.Delete(c, articleID); err != nil {
		if errors.Is(err, ErrArticleNotFound) {
			return err
		}
		return fmt.Errorf("failed to delete article: %w", err)
	}
	return nil
}

// Update 更新文章：作者本人或管理员。
func (s *Service) Update(c context.Context, actorID, articleID uint, req UpdateArticleRequest) (SummaryResponse, error) {
	// 1. 鉴权 + 载入实体：安全门先过，再谈业务校验
	article, err := s.authorize(c, actorID, articleID)
	if err != nil {
		return SummaryResponse{}, err
	}

	// 2. 业务校验：binding tag 只能在 handler 层校验格式/长度，这里补语义校验
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return SummaryResponse{}, fmt.Errorf("title must not be blank")
	}
	if strings.TrimSpace(req.Content) == "" {
		return SummaryResponse{}, fmt.Errorf("content must not be blank")
	}

	// 3. 只覆盖业务字段；ID/CreatedAt/DeletedAt 等系统字段保持数据库原值
	article.Title = title
	article.Description = strings.TrimSpace(req.Description)
	article.Content = req.Content

	// 4. 全量回写（load-modify-save）：article 来自完整查询，不存在零值覆盖问题；
	//    UpdatedAt 由 gorm.Model 的 autoUpdateTime 自动刷新
	if err := s.repo.Save(c, article); err != nil {
		return SummaryResponse{}, fmt.Errorf("failed to update article: %w", err)
	}

	// 5. 回显更新后的结果
	return SummaryResponse{
		ID:          article.ID,
		Title:       article.Title,
		Description: article.Description,
		CreatedAt:   article.CreatedAt,
		UpdatedAt:   article.UpdatedAt,
	}, nil
}
