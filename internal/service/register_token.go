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
	ExpiresIn int    `json:"expires_in"`
}

// GenerateRegisterToken
func (s *RegisterTokenService) GenerateRegisterToken(userID string) (*GenerateTokenResponse, error) {
	_, err := s.userDB.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	token, err := s.registerTokenStore.GenerateToken(userID)
	if err != nil {
		return nil, err
	}

	return &GenerateTokenResponse{
		Token:     token.Token,
		ExpiresIn: 600,
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
