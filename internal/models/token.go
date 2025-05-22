package models

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"star-fire/pkg/utils"
	"time"
)

type RegisterToken struct {
	Token     string    `json:"token"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	Used      bool      `json:"used"`
}

type TokenStore struct {
	cache *utils.Cache
}

func NewTokenStore() *TokenStore {
	return &TokenStore{
		cache: utils.NewCache(),
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
		return "", errors.New("invalid token")
	}

	token, ok := value.(*RegisterToken)
	if !ok {
		return "", errors.New("cached token is not valid")
	}

	if token.Used {
		return "", errors.New("token already used")
	}

	if time.Since(token.CreatedAt) > 10*time.Minute {
		return "", errors.New("token expired")
	}

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
