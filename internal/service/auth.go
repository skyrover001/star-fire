package service

import (
	"errors"
	"star-fire/internal/models"
	"star-fire/pkg/utils"
)

type AuthService struct {
	userDB *models.UserDB
}

func NewAuthService(userDB *models.UserDB) *AuthService {
	return &AuthService{
		userDB: userDB,
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
	// 使用UserDB验证用户名和密码
	user, err := s.userDB.ValidatePassword(req.Username, req.Password)
	if err != nil {
		return nil, errors.New("用户名或密码无效")
	}

	// 生成JWT令牌
	token, err := utils.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, errors.New("生成令牌失败")
	}

	return &LoginResponse{
		Token:     token,
		User:      user,
		ExpiresIn: 86400, // 24小时
	}, nil
}
