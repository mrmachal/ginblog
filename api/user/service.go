package user

import (
	"context"
	"errors"
	"fmt"
	"ginblog/pkg/session"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo Repository
	sess *session.Store
}

func NewUserService(repo Repository, sess *session.Store) *Service {
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

	return token, MeResponse{
		ID:        user.ID,
		UserName:  user.UserName,
		NickName:  user.NickName,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
	}, nil
}

func (s *Service) Logout(c context.Context, token string) error {
	err := s.sess.Delete(c, token)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}
