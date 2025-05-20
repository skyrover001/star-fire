package service

import (
	"star-fire/config"
	"star-fire/internal/models"
	"time"
)

type APIKeyService struct {
	apiKeyStore *models.APIKeyStore
}

func NewAPIKeyService(apiKeyStore *models.APIKeyStore) *APIKeyService {
	return &APIKeyService{
		apiKeyStore: apiKeyStore,
	}
}

type CreateAPIKeyRequest struct {
	Name       string `json:"name" binding:"required"`
	ExpiryDays int    `json:"expiry_days"`
}

type APIKeyResponse struct {
	Key       *models.APIKey `json:"key"`
	CreatedAt time.Time      `json:"created_at"`
}

func (s *APIKeyService) CreateAPIKey(userID string, req *CreateAPIKeyRequest) (*APIKeyResponse, error) {
	// use default
	expiryDays := req.ExpiryDays
	if expiryDays <= 0 {
		expiryDays = configs.Config.DefaultKeyExpiry
	}

	// new api key
	apiKey, err := s.apiKeyStore.GenerateAPIKey(userID, req.Name, expiryDays)
	if err != nil {
		return nil, err
	}

	return &APIKeyResponse{
		Key:       apiKey,
		CreatedAt: time.Now(),
	}, nil
}

// get all API keys for a user
func (s *APIKeyService) GetUserKeys(userID string) []*models.APIKey {
	return s.apiKeyStore.GetAPIKeysByUser(userID)
}

// revoke an API key
func (s *APIKeyService) RevokeAPIKey(userID, keyID string) error {
	return s.apiKeyStore.RevokeAPIKey(userID, keyID)
}

// validate an API key
func (s *APIKeyService) ValidateAPIKey(apiKey string) (*models.APIKey, error) {
	return s.apiKeyStore.ValidateAPIKey(apiKey)
}
