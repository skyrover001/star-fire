package service

import (
	"star-fire/internal/models"
)

// TokenService 处理注册Token相关业务逻辑
type TokenService struct {
	tokenStore *models.TokenStore
	userStore  *models.UserStore
}

// NewTokenService 创建一个新的Token服务
func NewTokenService(tokenStore *models.TokenStore, userStore *models.UserStore) *TokenService {
	return &TokenService{
		tokenStore: tokenStore,
		userStore:  userStore,
	}
}

// GenerateTokenResponse 生成Token响应
type GenerateTokenResponse struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expires_in"` // 过期时间（秒）
}

// GenerateRegisterToken 为用户生成注册Token
func (s *TokenService) GenerateRegisterToken(userID string) (*GenerateTokenResponse, error) {
	// 检查用户是否存在
	_, err := s.userStore.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	// 生成Token
	token, err := s.tokenStore.GenerateToken(userID)
	if err != nil {
		return nil, err
	}

	return &GenerateTokenResponse{
		Token:     token.Token,
		ExpiresIn: 600, // 10分钟
	}, nil
}

// ValidateRegisterToken 验证注册Token
func (s *TokenService) ValidateRegisterToken(tokenString string) (*models.User, error) {
	// 验证并使用Token
	userID, err := s.tokenStore.ValidateAndUseToken(tokenString)
	if err != nil {
		return nil, err
	}

	// 获取用户信息
	user, err := s.userStore.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	return user, nil
}
