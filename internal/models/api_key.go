package models

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	configs "star-fire/config"
	"time"
)

// APIKey 表示用户的API密钥
type APIKey struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	Key       string    `json:"key,omitempty"` // 只在创建时返回给用户
	Prefix    string    `json:"prefix"`        // 密钥前缀，用于识别
	LastUsed  time.Time `json:"last_used,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Revoked   bool      `json:"revoked"`
}

// APIKeyStore 是一个简单的内存API密钥存储
// 注意：生产环境应使用数据库
type APIKeyStore struct {
	keys        map[string]*APIKey   // key ID -> API Key
	keysByUser  map[string][]*APIKey // user ID -> API Keys
	keysByValue map[string]*APIKey   // actual key value -> API Key (for validation)
}

// NewAPIKeyStore 创建新的API密钥存储
func NewAPIKeyStore() *APIKeyStore {
	return &APIKeyStore{
		keys:        make(map[string]*APIKey),
		keysByUser:  make(map[string][]*APIKey),
		keysByValue: make(map[string]*APIKey),
	}
}

// GenerateAPIKey 为用户生成新的API密钥
func (s *APIKeyStore) GenerateAPIKey(userID, name string, expiryDays int) (*APIKey, error) {
	// 检查用户的密钥数量是否已达上限
	if keys, ok := s.keysByUser[userID]; ok && len(keys) >= configs.Config.MaxAPIKeysPerUser {
		return nil, errors.New("maximum number of API keys reached for this user")
	}

	// 生成随机密钥
	keyBytes := make([]byte, 32)
	_, err := rand.Read(keyBytes)
	if err != nil {
		return nil, err
	}

	// 使用base64编码密钥，并添加sk-前缀以模拟OpenAI格式
	keyString := "sk-" + base64.StdEncoding.EncodeToString(keyBytes)

	// 密钥前缀用于标识
	prefix := keyString[:10]

	// 创建新的API密钥
	now := time.Now()
	apiKey := &APIKey{
		ID:        fmt.Sprintf("key-%d", time.Now().UnixNano()),
		UserID:    userID,
		Name:      name,
		Key:       keyString,
		Prefix:    prefix,
		CreatedAt: now,
		ExpiresAt: now.AddDate(0, 0, expiryDays),
		Revoked:   false,
	}

	// 存储密钥
	s.keys[apiKey.ID] = apiKey
	s.keysByValue[keyString] = apiKey

	if _, ok := s.keysByUser[userID]; !ok {
		s.keysByUser[userID] = []*APIKey{}
	}
	s.keysByUser[userID] = append(s.keysByUser[userID], apiKey)

	return apiKey, nil
}

// GetAPIKeysByUser 获取用户的所有API密钥
func (s *APIKeyStore) GetAPIKeysByUser(userID string) []*APIKey {
	keys, ok := s.keysByUser[userID]
	if !ok {
		return []*APIKey{}
	}

	// 创建一个没有敏感数据的副本
	safeKeys := make([]*APIKey, len(keys))
	for i, k := range keys {
		// 创建复制，避免修改原始对象
		keyCopy := *k
		keyCopy.Key = "" // 不包含实际密钥
		safeKeys[i] = &keyCopy
	}

	return safeKeys
}

// ValidateAPIKey 验证API密钥
func (s *APIKeyStore) ValidateAPIKey(keyString string) (*APIKey, error) {
	key, ok := s.keysByValue[keyString]
	if !ok {
		return nil, errors.New("invalid API key")
	}

	// 检查密钥是否已被撤销
	if key.Revoked {
		return nil, errors.New("API key has been revoked")
	}

	// 检查密钥是否已过期
	if time.Now().After(key.ExpiresAt) {
		return nil, errors.New("API key has expired")
	}

	// 更新最后使用时间
	key.LastUsed = time.Now()

	return key, nil
}

// RevokeAPIKey 撤销API密钥
func (s *APIKeyStore) RevokeAPIKey(userID, keyID string) error {
	key, ok := s.keys[keyID]
	if !ok {
		return errors.New("API key not found")
	}

	// 确保用户拥有此密钥
	if key.UserID != userID {
		return errors.New("unauthorized to revoke this API key")
	}

	key.Revoked = true
	return nil
}
