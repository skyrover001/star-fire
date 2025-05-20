package models

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

type RegisterToken struct {
	Token     string    `json:"token"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	Used      bool      `json:"used"`
}

type TokenStore struct {
	tokens map[string]*RegisterToken
	mutex  sync.RWMutex
}

func NewTokenStore() *TokenStore {
	return &TokenStore{
		tokens: make(map[string]*RegisterToken),
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

	s.mutex.Lock()
	s.tokens[tokenString] = token
	s.mutex.Unlock()

	return token, nil
}

func (s *TokenStore) ValidateAndUseToken(tokenString string) (string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	token, exists := s.tokens[tokenString]
	if !exists {
		return "", errors.New("invalid token")
	}

	if token.Used {
		return "", errors.New("token has already been used")
	}

	if time.Since(token.CreatedAt) > 10*time.Minute {
		return "", errors.New("token has expired")
	}

	token.Used = true
	return token.UserID, nil
}

func (s *TokenStore) CleanupExpiredTokens() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for key, token := range s.tokens {
		if token.Used || time.Since(token.CreatedAt) > 24*time.Hour {
			delete(s.tokens, key)
		}
	}
}
