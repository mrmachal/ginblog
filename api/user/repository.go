package user

import (
	"context"
	"errors"
	"ginblog/model"
	"strings"

	"gorm.io/gorm"
)

type Repository interface {
	Create(c context.Context, user *model.UserInfo) error
	Delete(c context.Context, userID uint) error
	GetByID(c context.Context, userID uint) (*model.UserInfo, error)
	GetByUserName(c context.Context, userName string) (*model.UserInfo, error)
	GetByEmail(c context.Context, email string) (*model.UserInfo, error)
	Save(c context.Context, newUserInfo *model.UserInfo) error
	List(c context.Context, query ListUserQuery) ([]*model.UserInfo, int64, error)
	Role(c context.Context, userID uint) (model.UserRole, bool, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) Repository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(c context.Context, user *model.UserInfo) error {
	return r.db.WithContext(c).Create(user).Error
}

func (r *userRepository) Delete(c context.Context, userID uint) error {
	res := r.db.WithContext(c).Delete(&model.UserInfo{}, userID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *userRepository) GetByID(c context.Context, userID uint) (*model.UserInfo, error) {
	var userInfo model.UserInfo
	db := r.db.WithContext(c).First(&userInfo, userID)
	if err := db.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &userInfo, nil
}

func (r *userRepository) Role(c context.Context, userID uint) (model.UserRole, bool, error) {
	var userInfo model.UserInfo
	// 只取 id/role 两列，别把 PasswordHash 拉进内存
	err := r.db.WithContext(c).Select("id", "role").First(&userInfo, userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return userInfo.Role, true, nil
}

func (r *userRepository) GetByUserName(c context.Context, userName string) (*model.UserInfo, error) {
	var userInfo model.UserInfo
	db := r.db.WithContext(c).Where("user_name = ?", userName).First(&userInfo)
	if err := db.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &userInfo, nil
}

func (r *userRepository) GetByEmail(c context.Context, email string) (*model.UserInfo, error) {
	var userInfo model.UserInfo
	if err := r.db.WithContext(c).
		Where("email = ?", email).
		First(&userInfo).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &userInfo, nil
}

func (r *userRepository) Save(c context.Context, newUserInfo *model.UserInfo) error {
	return r.db.WithContext(c).Save(newUserInfo).Error
}

func (r *userRepository) List(c context.Context, query ListUserQuery) ([]*model.UserInfo, int64, error) {
	var userList []*model.UserInfo
	var total int64
	db := r.db.WithContext(c).
		Model(&model.UserInfo{}).
		Select("id", "user_name", "nick_name", "email", "role", "updated_at", "created_at")
	if query.Keyword != "" {
		kw := "%" + query.Keyword + "%"
		db = db.Where("user_name LIKE ? OR nick_name LIKE ? OR email LIKE ?", kw, kw, kw)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	order := "id DESC"
	if query.Sort != "" {
		desc := strings.HasPrefix(query.Sort, "-")
		key := strings.TrimPrefix(query.Sort, "-")
		if key == "user_name" || key == "nick_name" || key == "email" || key == "role" || key == "created_at" {
			if desc {
				order = key + " DESC"
			} else {
				order = key + " ASC"
			}
		}
	}

	db = db.Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Order(order)
	err := db.Find(&userList).Error
	return userList, total, err
}
