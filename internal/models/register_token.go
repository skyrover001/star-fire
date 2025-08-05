package models

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"star-fire/pkg/utils"
	"time"
)

type RegisterToken struct {
	Token          string    `json:"token"`
	UserID         string    `json:"user_id"`
	CreatedAt      time.Time `json:"created_at"`
	Used           bool      `json:"used"`
	ExpiredSeconds int64     `json:"expired_seconds"` // 可选字段，用于设置令牌的过期时间
}

type RegisterTokenStore struct {
	cache *utils.Cache
}

func NewRegisterTokenStore() *RegisterTokenStore {
	return &RegisterTokenStore{
		cache: utils.NewCache(),
	}
}

func (s *RegisterTokenStore) GenerateToken(userID string, expiredSeconds int64) (*RegisterToken, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, err
	}

	tokenString := hex.EncodeToString(tokenBytes)
	token := &RegisterToken{
		Token:          tokenString,
		UserID:         userID,
		CreatedAt:      time.Now(),
		Used:           false,
		ExpiredSeconds: expiredSeconds,
	}

	s.cache.Set(tokenString, token)
	return token, nil
}

func (s *RegisterTokenStore) ValidateAndUseToken(tokenString string) (string, error) {
	value, exists := s.cache.Get(tokenString)
	if !exists {
		return "", errors.New("invalid token")
	}

	token, ok := value.(*RegisterToken)
	if !ok {
		return "", errors.New("cached token is not valid")
	}

	if token.Used {
		return "", errors.New("token already used")
	}

	if token.ExpiredSeconds > 0 {
		if time.Since(token.CreatedAt) > time.Duration(token.ExpiredSeconds)*time.Second {
			return "", errors.New("token expired")
		}
	}

	token.Used = true
	s.cache.Set(tokenString, token)

	return token.UserID, nil
}

func (s *RegisterTokenStore) CleanupExpiredTokens() {
	allTokens := s.cache.GetAll()

	for key, value := range allTokens {
		token, ok := value.(*RegisterToken)
		if !ok {
			continue
		}

		if token.ExpiredSeconds > 0 {
			if token.Used || time.Since(token.CreatedAt) > 24*time.Hour {
				s.cache.Delete(key)
			}
		}
	}
}
