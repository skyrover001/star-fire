package service

import (
	"errors"
	"star-fire/internal/models"
	"star-fire/pkg/utils"
)

type AuthService struct {
	userStore *models.UserStore
}

func NewAuthService(userStore *models.UserStore) *AuthService {
	return &AuthService{
		userStore: userStore,
	}
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token     string       `json:"token"`
	User      *models.User `json:"user"`
	ExpiresIn int64        `json:"expires_in"` // 过期时间（秒）
}

func (s *AuthService) Login(req *LoginRequest) (*LoginResponse, error) {
	user, err := s.userStore.ValidatePassword(req.Username, req.Password)
	if err != nil {
		return nil, errors.New("invalid username or password")
	}

	// generate JWT token
	token, err := utils.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, errors.New("failed to generate token")
	}

	return &LoginResponse{
		Token:     token,
		User:      user,
		ExpiresIn: 86400, // 24 hours
	}, nil
}
