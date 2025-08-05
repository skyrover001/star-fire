package service

import (
	"star-fire/internal/models"
)

type RegisterTokenService struct {
	registerTokenStore *models.RegisterTokenStore
	userDB             *models.UserDB
}

// NewRegisterTokenService
func NewRegisterTokenService(registerTokenStore *models.RegisterTokenStore, userDB *models.UserDB) *RegisterTokenService {
	return &RegisterTokenService{
		registerTokenStore: registerTokenStore,
		userDB:             userDB,
	}
}

// GenerateTokenResponse
type GenerateTokenResponse struct {
	Token     string `json:"token"`
	ExpiresIn int64  `json:"expires_in"`
}

// GenerateRegisterToken
func (s *RegisterTokenService) GenerateRegisterToken(userID string, expiredSeconds int64) (*GenerateTokenResponse, error) {
	_, err := s.userDB.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	token, err := s.registerTokenStore.GenerateToken(userID, expiredSeconds)
	if err != nil {
		return nil, err
	}

	return &GenerateTokenResponse{
		Token:     token.Token,
		ExpiresIn: token.ExpiredSeconds,
	}, nil
}

// ValidateRegisterToken
func (s *RegisterTokenService) ValidateRegisterToken(tokenString string) (*models.User, error) {
	userID, err := s.registerTokenStore.ValidateAndUseToken(tokenString)
	if err != nil {
		return nil, err
	}

	user, err := s.userDB.GetUserByID(userID)
	if err != nil {
		return nil, err
	}
	return user, nil
}
