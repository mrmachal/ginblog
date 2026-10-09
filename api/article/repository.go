package article

import (
	"context"
	"ginblog/model"
	"strings"

	"gorm.io/gorm"
)

type Repository interface {
	Create(c context.Context, article *model.Article) error
	List(c context.Context, q ListArticleQuery) ([]*model.Article, int64, error)
	GetById(c context.Context, articleId uint) (*model.Article, error)
	Save(c context.Context, u *model.Article) error
	Delete(c context.Context, id uint) error
}

type articleRepository struct {
	db *gorm.DB
}

func NewArticleRepository(db *gorm.DB) Repository {
	return &articleRepository{db: db}
}

func (r *articleRepository) Create(c context.Context, article *model.Article) error {
	return r.db.WithContext(c).Create(article).Error
}

func (r *articleRepository) List(c context.Context, query ListArticleQuery) ([]*model.Article, int64, error) {
	var (
		articleList []*model.Article
		total       int64
	)

	db := r.db.WithContext(c).Preload("User", func(db *gorm.DB) *gorm.DB {
		return db.Select("id", "nick_name")
	})
	db = db.Model(&model.Article{}).Select("id", "title", "description", "updated_at", "created_at", "user")
	if query.Keyword != "" {
		kw := "%" + query.Keyword + "%"
		db = db.Where("title LIKE ? OR description LIKE ?", kw, kw)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	order := "id DESC"
	if query.Sort != "" {
		desc := strings.HasPrefix(query.Sort, "-")
		key := strings.TrimPrefix(query.Sort, "-")
		if key == "title" || key == "description" || key == "updated_at" || key == "created_at" {
			if desc {
				order = key + " DESC"
			} else {
				order = key + " ASC"
			}
		}
	}
	db = db.Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Order(order)
	err := db.Find(&articleList).Error
	return articleList, total, err
}

func (r *articleRepository) GetById(c context.Context, articleId uint) (*model.Article, error) {
	var article model.Article
	db := r.db.WithContext(c).
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "user_name", "nick_name")
		})
	db = db.First(&article, articleId)
	if err := db.Error; err != nil {
		return nil, err
	}
	return &article, nil
}

func (r *articleRepository) Delete(c context.Context, articleID uint) error {
	res := r.db.WithContext(c).Delete(&model.Article{}, articleID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *articleRepository) Save(c context.Context, article *model.Article) error {
	return r.db.WithContext(c).Save(article).Error
}

func (r *articleRepository) UserOwnsArticle(c context.Context, articleID, userID uint) (bool, error) {
	db := r.db.WithContext(c).Select("id", "user_id").Where("id = ? AND user_id = ?", articleID, userID)
	var count int64
	err := db.Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}
