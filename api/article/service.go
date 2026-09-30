package article

import (
	"context"
	"errors"
	"fmt"
	"ginblog/model"
	"strings"

	"gorm.io/gorm"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
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
		})
	}
	return out, total, nil
}

func (s *Service) GetByID(c context.Context, articleID uint) (*DetailArticleResponse, error) {
	article, err := s.repo.GetById(c, articleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get article detail: %w", err)
	}
	return &DetailArticleResponse{
		ID:          article.ID,
		Title:       article.Title,
		Description: article.Description,
		Content:     article.Content,
		CreatedAt:   article.CreatedAt,
		UpdatedAt:   article.UpdatedAt,
	}, nil
}

func (s *Service) Create(c context.Context, articleReq CreateArticleRequest) (SummaryResponse, error) {
	title := strings.TrimSpace(articleReq.Title)
	if title == "" {
		return SummaryResponse{}, fmt.Errorf("title must not be blank")
	}
	if strings.TrimSpace(articleReq.Content) == "" {
		return SummaryResponse{}, fmt.Errorf("content must not be blank")
	}
	article := model.Article{
		Title:       articleReq.Title,
		Description: articleReq.Description,
		Content:     articleReq.Content,
	}
	if err := s.repo.Create(c, &article); err != nil {
		return SummaryResponse{}, fmt.Errorf("failed to create article: %w", err)
	}
	return SummaryResponse{
		ID:          article.ID,
		Title:       article.Title,
		Description: article.Description,
		CreatedAt:   article.CreatedAt,
		UpdatedAt:   article.UpdatedAt,
	}, nil
}

func (s *Service) Delete(c context.Context, articleID uint) error {
	err := s.repo.Delete(c, articleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("article not found: %w", err)
		}
		return fmt.Errorf("failed to delete article: %w", err)
	}
	return nil
}

func (s *Service) Update(c context.Context, articleID uint, req UpdateArticleRequest) (SummaryResponse, error) {
	// 1. 业务校验：binding tag 只能在 handler 层校验格式/长度，这里补语义校验
	if articleID == 0 {
		return SummaryResponse{}, fmt.Errorf("invalid article id")
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return SummaryResponse{}, fmt.Errorf("title must not be blank")
	}
	if strings.TrimSpace(req.Content) == "" {
		return SummaryResponse{}, fmt.Errorf("content must not be blank")
	}

	// 2. 存在性校验（load）：不存在返回 404 语义
	article, err := s.repo.GetById(c, articleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return SummaryResponse{}, fmt.Errorf("article not found: %w", err)
		}
		return SummaryResponse{}, fmt.Errorf("failed to get article: %w", err)
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
