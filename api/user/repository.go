package user

import (
	"context"
	"ginblog/model"

	"gorm.io/gorm"
)

type Repository interface {
	Create(c context.Context, user *model.UserInfo) error
	Delete(c context.Context, userID uint) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *userRepository {
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
