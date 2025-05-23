package service

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"star-fire/config"
	"star-fire/internal/models"
	"time"
)

type APIKeyService struct {
	apiKeyDB *models.APIKeyDB
}

func NewAPIKeyService(apiKeyDB *models.APIKeyDB) *APIKeyService {
	return &APIKeyService{
		apiKeyDB: apiKeyDB,
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
	// 使用默认过期时间
	expiryDays := req.ExpiryDays
	if expiryDays <= 0 {
		expiryDays = configs.Config.DefaultKeyExpiry
	}

	// 检查用户API Key数量是否达到上限
	count, err := s.apiKeyDB.CountUserAPIKeys(userID)
	if err != nil {
		return nil, err
	}

	if count >= configs.Config.MaxAPIKeysPerUser {
		return nil, errors.New("已达到最大API Key数量限制")
	}

	// 生成新的API Key
	keyBytes := make([]byte, 32)
	_, err = rand.Read(keyBytes)
	if err != nil {
		return nil, err
	}

	keyString := "sk-" + base64.StdEncoding.EncodeToString(keyBytes)
	prefix := keyString[:10]

	// 创建新的API Key记录
	now := time.Now()
	apiKey := &models.APIKey{
		ID:        fmt.Sprintf("key-%d", now.UnixNano()),
		UserID:    userID,
		Name:      req.Name,
		Key:       keyString,
		Prefix:    prefix,
		CreatedAt: now,
		ExpiresAt: now.AddDate(0, 0, expiryDays),
		Revoked:   false,
	}

	// 保存到数据库
	err = s.apiKeyDB.SaveAPIKey(apiKey)
	if err != nil {
		return nil, err
	}

	return &APIKeyResponse{
		Key:       apiKey,
		CreatedAt: now,
	}, nil
}

// 获取用户的所有API Key
func (s *APIKeyService) GetUserKeys(userID string) ([]*models.APIKey, error) {
	return s.apiKeyDB.GetAPIKeysByUser(userID)
}

// 撤销API Key
func (s *APIKeyService) RevokeAPIKey(userID, keyID string) error {
	return s.apiKeyDB.RevokeAPIKey(userID, keyID)
}

// 验证API Key
func (s *APIKeyService) ValidateAPIKey(apiKey string) (*models.APIKey, error) {
	key, err := s.apiKeyDB.GetAPIKeyByValue(apiKey)
	if err != nil {
		return nil, err
	}

	if key.Revoked {
		return nil, errors.New("API Key已被撤销")
	}

	if time.Now().After(key.ExpiresAt) {
		return nil, errors.New("API Key已过期")
	}

	// 更新最后使用时间
	err = s.apiKeyDB.UpdateLastUsed(key.ID)
	if err != nil {
		// 只记录错误但不中断流程
		fmt.Printf("更新API Key最后使用时间失败: %v\n", err)
	}

	return key, nil
}
