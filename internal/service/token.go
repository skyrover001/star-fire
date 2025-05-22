package service

import (
	"star-fire/internal/models"
)

// TokenService
type TokenService struct {
	tokenStore *models.TokenStore
	userStore  *models.UserStore
}

// NewTokenService
func NewTokenService(tokenStore *models.TokenStore, userStore *models.UserStore) *TokenService {
	return &TokenService{
		tokenStore: tokenStore,
		userStore:  userStore,
	}
}

// GenerateTokenResponse
type GenerateTokenResponse struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expires_in"` // 过期时间（秒）
}

// GenerateRegisterToken
func (s *TokenService) GenerateRegisterToken(userID string) (*GenerateTokenResponse, error) {
	_, err := s.userStore.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	token, err := s.tokenStore.GenerateToken(userID)
	if err != nil {
		return nil, err
	}

	return &GenerateTokenResponse{
		Token:     token.Token,
		ExpiresIn: 600, // 10分钟
	}, nil
}

// ValidateRegisterToken
func (s *TokenService) ValidateRegisterToken(tokenString string) (*models.User, error) {
	userID, err := s.tokenStore.ValidateAndUseToken(tokenString)
	if err != nil {
		return nil, err
	}

	user, err := s.userStore.GetUserByID(userID)
	if err != nil {
		return nil, err
	}
	return user, nil
}
