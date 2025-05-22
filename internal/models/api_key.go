package models

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	configs "star-fire/config"
	"time"
)

type APIKey struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	Key       string    `json:"key,omitempty"`
	Prefix    string    `json:"prefix"`
	LastUsed  time.Time `json:"last_used,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Revoked   bool      `json:"revoked"`
}

// 注意：生产环境应使用数据库
type APIKeyStore struct {
	keys        map[string]*APIKey
	keysByUser  map[string][]*APIKey
	keysByValue map[string]*APIKey
}

// NewAPIKeyStore
func NewAPIKeyStore() *APIKeyStore {
	return &APIKeyStore{
		keys:        make(map[string]*APIKey),
		keysByUser:  make(map[string][]*APIKey),
		keysByValue: make(map[string]*APIKey),
	}
}

// GenerateAPIKey
func (s *APIKeyStore) GenerateAPIKey(userID, name string, expiryDays int) (*APIKey, error) {
	if keys, ok := s.keysByUser[userID]; ok && len(keys) >= configs.Config.MaxAPIKeysPerUser {
		return nil, errors.New("maximum number of API keys reached for this user")
	}

	keyBytes := make([]byte, 32)
	_, err := rand.Read(keyBytes)
	if err != nil {
		return nil, err
	}

	keyString := "sk-" + base64.StdEncoding.EncodeToString(keyBytes)
	prefix := keyString[:10]

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

	s.keys[apiKey.ID] = apiKey
	s.keysByValue[keyString] = apiKey

	if _, ok := s.keysByUser[userID]; !ok {
		s.keysByUser[userID] = []*APIKey{}
	}
	s.keysByUser[userID] = append(s.keysByUser[userID], apiKey)

	return apiKey, nil
}

// GetAPIKeysByUser
func (s *APIKeyStore) GetAPIKeysByUser(userID string) []*APIKey {
	keys, ok := s.keysByUser[userID]
	if !ok {
		return []*APIKey{}
	}

	safeKeys := make([]*APIKey, len(keys))
	for i, k := range keys {
		keyCopy := *k
		keyCopy.Key = ""
		safeKeys[i] = &keyCopy
	}

	return safeKeys
}

// ValidateAPIKey
func (s *APIKeyStore) ValidateAPIKey(keyString string) (*APIKey, error) {
	key, ok := s.keysByValue[keyString]
	if !ok {
		return nil, errors.New("invalid API key")
	}

	if key.Revoked {
		return nil, errors.New("API key has been revoked")
	}
	if time.Now().After(key.ExpiresAt) {
		return nil, errors.New("API key has expired")
	}

	key.LastUsed = time.Now()
	return key, nil
}

// RevokeAPIKey
func (s *APIKeyStore) RevokeAPIKey(userID, keyID string) error {
	key, ok := s.keys[keyID]
	if !ok {
		return errors.New("API key not found")
	}

	if key.UserID != userID {
		return errors.New("unauthorized to revoke this API key")
	}

	key.Revoked = true
	return nil
}
