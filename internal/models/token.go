package models

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"star-fire/cache"
	"time"
)

type RegisterToken struct {
	Token     string    `json:"token"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	Used      bool      `json:"used"`
}

type TokenStore struct {
	cache *cache.Cache
}

func NewTokenStore() *TokenStore {
	return &TokenStore{
		cache: cache.NewCache(),
	}
}

func (s *TokenStore) GenerateToken(userID string) (*RegisterToken, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, err
	}

	tokenString := hex.EncodeToString(tokenBytes)
	token := &RegisterToken{
		Token:     tokenString,
		UserID:    userID,
		CreatedAt: time.Now(),
		Used:      false,
	}

	s.cache.Set(tokenString, token)
	return token, nil
}

func (s *TokenStore) ValidateAndUseToken(tokenString string) (string, error) {
	value, exists := s.cache.Get(tokenString)
	if !exists {
		return "", errors.New("无效的令牌")
	}

	token, ok := value.(*RegisterToken)
	if !ok {
		return "", errors.New("缓存中存储了无效的令牌类型")
	}

	if token.Used {
		return "", errors.New("令牌已被使用")
	}

	if time.Since(token.CreatedAt) > 10*time.Minute {
		return "", errors.New("令牌已过期")
	}

	// 更新已使用状态
	token.Used = true
	s.cache.Set(tokenString, token)

	return token.UserID, nil
}

func (s *TokenStore) CleanupExpiredTokens() {
	allTokens := s.cache.GetAll()

	for key, value := range allTokens {
		token, ok := value.(*RegisterToken)
		if !ok {
			continue
		}

		if token.Used || time.Since(token.CreatedAt) > 24*time.Hour {
			s.cache.Delete(key)
		}
	}
}
