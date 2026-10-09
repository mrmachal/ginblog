package user

import (
	"context"
	"errors"
	"fmt"
	"ginblog/model"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type SessionStore interface {
	Create(c context.Context, userID uint) (string, error)
	Delete(c context.Context, token string) error
}

type Service struct {
	repo Repository
	sess SessionStore
}

func NewUserService(repo Repository, sess SessionStore) *Service {
	return &Service{repo: repo, sess: sess}
}

func (s *Service) Login(c context.Context, req LoginRequest) (string, MeResponse, error) {
	user, err := s.repo.GetByUserName(c, req.UserName)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return "", MeResponse{}, ErrInvalidCredential
		}

		return "", MeResponse{}, fmt.Errorf("failed to get user_info: %w", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		return "", MeResponse{}, ErrInvalidCredential
	}

	token, err := s.sess.Create(c, user.ID)
	if err != nil {
		return "", MeResponse{}, fmt.Errorf("failed to create session: %w", err)
	}

	return token, toMeResponse(user), nil
}

func (s *Service) Logout(c context.Context, token string) error {
	if err := s.sess.Delete(c, token); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

func (s *Service) Register(c context.Context, req RegisterRequest) (MeResponse, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return MeResponse{}, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &model.UserInfo{
		UserName:     req.UserName,
		NickName:     strings.TrimSpace(req.NickName),
		Email:        req.Email,
		PasswordHash: string(passwordHash),
		Role:         model.RoleUser,
	}

	if err := s.repo.Create(c, user); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return MeResponse{}, s.conflict(c, req.UserName, req.Email)
		}
		return MeResponse{}, fmt.Errorf("failed to create user: %w", err)
	}

	return toMeResponse(user), nil
}

func (s *Service) Me(c context.Context, userID uint) (MeResponse, error) {
	user, err := s.repo.GetByID(c, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return MeResponse{}, err
		}
		return MeResponse{}, fmt.Errorf("failed to get userinfo: %w", err)
	}
	return toMeResponse(user), nil
}

func (s *Service) ChangePassword(c context.Context, userID uint, req ChangePWRequest) (MeResponse, error) {
	user, err := s.repo.GetByID(c, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return MeResponse{}, err
		}
		return MeResponse{}, fmt.Errorf("failed to get userinfo: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword))
	if err != nil {
		return MeResponse{}, ErrInvalidCredential
	}

	if req.ConfirmPassword != req.NewPassword {
		return MeResponse{}, fmt.Errorf("confirm password not equals new password")
	}

	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return MeResponse{}, fmt.Errorf("failed to hash password: %w", err)
	}

	user.PasswordHash = string(newHashedPassword)
	if err := s.repo.Save(c, user); err != nil {
		return MeResponse{}, fmt.Errorf("failed to change password: %w", err)
	}

	return toMeResponse(user), nil
}

func (s *Service) UpdateMe(c context.Context, userID uint, req UpdateMeRequest) (MeResponse, error) {
	user, err := s.repo.GetByID(c, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return MeResponse{}, err
		}
		return MeResponse{}, fmt.Errorf("failed to get userinfo: %w", err)
	}

	user.NickName = req.NickName
	user.Email = req.NickName

	err = s.repo.Save(c, user)
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return MeResponse{}, ErrEmailAlreadyExists
		}
		return MeResponse{}, fmt.Errorf("failed to change userinfo: %w", err)
	}
	return toMeResponse(user), nil
}

func (s *Service) ChangeUser(c context.Context, userID uint, req ChangeUserRequest) (UserResponse, error) {
	user, err := s.repo.GetByID(c, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return UserResponse{}, err
		}
		return UserResponse{}, fmt.Errorf("failed to get userinfo: %w", err)
	}

	user.NickName = req.NickName
	user.Email = req.Email
	user.Role = req.Role

	if err := s.repo.Save(c, user); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return UserResponse{}, ErrEmailAlreadyExists
		}

		return UserResponse{}, fmt.Errorf("failed to update userinfo: %w", err)
	}
	return toUserResponse(user), nil
}

func (s *Service) CreateUser(c context.Context, req CreateUserRequest) (UserResponse, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return UserResponse{}, fmt.Errorf("failed to hash password: %w", err)
	}
	user := &model.UserInfo{
		UserName:     req.UserName,
		NickName:     req.NickName,
		Email:        req.Email,
		PasswordHash: string(passwordHash),
		Role:         req.Role,
	}

	if err := s.repo.Create(c, user); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return UserResponse{}, s.conflict(c, req.UserName, req.Email)
		}
		return UserResponse{}, fmt.Errorf("failed to create user: %w", err)
	}

	return toUserResponse(user), nil
}

func (s *Service) ListUser(c context.Context, query ListUserQuery) ([]UserResponse, int64, error) {
	result, total, err := s.repo.List(c, query)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get user list: %w", err)
	}
	out := make([]UserResponse, 0, len(result))
	for _, u := range result {
		out = append(out, UserResponse{
			ID:        u.ID,
			UserName:  u.UserName,
			NickName:  u.NickName,
			Email:     u.Email,
			Role:      u.Role,
			CreatedAt: u.CreatedAt,
			UpdatedAt: u.UpdatedAt,
		})
	}

	return out, total, nil
}

func (s *Service) GetDetail(c context.Context, userID uint) (UserResponse, error) {
	user, err := s.repo.GetByID(c, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return UserResponse{}, err
		}
		return UserResponse{}, fmt.Errorf("failed to get userinfo: %w", err)
	}

	return toUserResponse(user), nil
}

func (s *Service) DeleteUser(c context.Context, userID uint) error {
	err := s.repo.Delete(c, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}

		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

func (s *Service) conflict(c context.Context, userName, email string) error {
	if _, err := s.repo.GetByUserName(c, userName); err == nil {
		return ErrUserNameAlreadyExists
	} else if !errors.Is(err, ErrUserNotFound) {
		return fmt.Errorf("failed to get user_info: %w", err)
	}

	if _, err := s.repo.GetByEmail(c, email); err == nil {
		return ErrEmailAlreadyExists
	} else if !errors.Is(err, ErrUserNotFound) {
		return fmt.Errorf("failed to get user_info: %w", err)
	}

	// 两个都查不到（并发事务回滚等罕见情况）：按用户名冲突兜底
	return ErrUserNameAlreadyExists
}

func toMeResponse(user *model.UserInfo) MeResponse {
	return MeResponse{
		ID:        user.ID,
		UserName:  user.UserName,
		NickName:  user.NickName,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
	}
}

func toUserResponse(user *model.UserInfo) UserResponse {
	return UserResponse{
		ID:        user.ID,
		UserName:  user.UserName,
		NickName:  user.NickName,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}
