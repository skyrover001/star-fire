package models

import (
	"gorm.io/gorm"
	"time"
)

// TokenUsage
type TokenUsage struct {
	ID           uint   `gorm:"primaryKey"`
	RequestID    string `gorm:"index;not null"`
	UserID       string `gorm:"index;not null"`
	APIKey       string `gorm:"index"`
	ClientID     string
	ClientIP     string
	Model        string    `gorm:"not null"`
	InputTokens  int       `gorm:"not null"`
	OutputTokens int       `gorm:"not null"`
	TotalTokens  int       `gorm:"not null"`
	Timestamp    time.Time `gorm:"index;not null"`
}

// TokenUsageDB
type TokenUsageDB struct {
	db *gorm.DB
}

// NewTokenUsageDB
func NewTokenUsageDB(db *gorm.DB) *TokenUsageDB {
	// AutoMigrate will create the table if it doesn't exist
	err := db.AutoMigrate(&TokenUsage{})
	if err != nil {
		return nil
	}
	return &TokenUsageDB{db: db}
}

// SaveTokenUsage
func (tdb *TokenUsageDB) SaveTokenUsage(usage *TokenUsage) error {
	return tdb.db.Create(usage).Error
}

// GetUserTokenUsage
func (tdb *TokenUsageDB) GetUserTokenUsage(userID string, startTime, endTime time.Time) ([]*TokenUsage, error) {
	var usages []*TokenUsage
	result := tdb.db.Where("user_id = ? AND timestamp BETWEEN ? AND ?",
		userID, startTime, endTime).
		Order("timestamp DESC").
		Find(&usages)

	if result.Error != nil {
		return nil, result.Error
	}

	return usages, nil
}

// GetUserTokenStats
func (tdb *TokenUsageDB) GetUserTokenStats(userID string, startTime, endTime time.Time) (map[string]int, error) {
	type Result struct {
		TotalInputTokens  int
		TotalOutputTokens int
		TotalTokens       int
	}

	var result Result
	err := tdb.db.Model(&TokenUsage{}).
		Select("SUM(input_tokens) as total_input_tokens, SUM(output_tokens) as total_output_tokens, SUM(total_tokens) as total_tokens").
		Where("user_id = ? AND timestamp BETWEEN ? AND ?", userID, startTime, endTime).
		Scan(&result).Error

	if err != nil {
		return nil, err
	}

	return map[string]int{
		"input_tokens":  result.TotalInputTokens,
		"output_tokens": result.TotalOutputTokens,
		"total_tokens":  result.TotalTokens,
	}, nil
}
