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
	ClientID     string `gorm:"index"`
	ClientIP     string
	Model        string    `gorm:"not null"`
	IPPM         float64   `gorm:"not null"` // 输入tokens价格
	OPPM         float64   `gorm:"not null"` // 输出tokens价格
	InputTokens  int       `gorm:"not null"`
	OutputTokens int       `gorm:"not null"`
	TotalTokens  int       `gorm:"not null"`
	RequestType  string    `gorm:"not null;default:'chat'"` // 请求类型: chat, embedding
	Revenue      float64   `gorm:"not null;default:0"`      // 收益
	Fingerprint  string    `gorm:"index"`                   // 请求指纹
	Timestamp    time.Time `gorm:"index;not null"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
}

// 声明一个模型的unitprice表，包含模型名、输入token单价、输出token单价，用户折扣率，用户id
type ModelPrice struct {
	ModelName        string  `gorm:"primaryKey;not null"`
	InputTokenPrice  float64 `gorm:"not null"`       // 输入token单价
	OutputTokenPrice float64 `gorm:"not null"`       // 输出token单价
	UserDiscountRate float64 `gorm:"not null"`       // 用户折扣率
	UserID           string  `gorm:"index;not null"` // 用户ID
}

// TokenUsageDB
type TokenUsageDB struct {
	db *gorm.DB
}

// NewTokenUsageDB
func NewTokenUsageDB(db *gorm.DB) *TokenUsageDB {
	// AutoMigrate will create the table if it doesn't exist
	err := db.AutoMigrate(&TokenUsage{})
	// and ModelPrice{}
	err = db.AutoMigrate(&ModelPrice{})
	if err != nil {
		return nil
	}
	return &TokenUsageDB{db: db}
}

// SaveTokenUsage
func (tdb *TokenUsageDB) SaveTokenUsage(usage *TokenUsage) error {
	return tdb.db.Create(usage).Error
}

// RecordTokenUsage 记录token使用情况 (新增方法用于embedding和chat)
func (tdb *TokenUsageDB) RecordTokenUsage(usage TokenUsage) error {
	return tdb.db.Create(&usage).Error
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

func (tdb *TokenUsageDB) GetIncomeTokenUsage(clientIDs []string, startTime, endTime time.Time) ([]*TokenUsage, error) {
	var usages []*TokenUsage
	result := tdb.db.Where("client_id IN ? AND timestamp BETWEEN ? AND ?",
		clientIDs, startTime, endTime).
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

// GetModelPrice
func (tdb *TokenUsageDB) GetModelPrice(modelName string, userID string) (*ModelPrice, error) {
	var price ModelPrice
	result := tdb.db.Where("model_name = ? AND user_id = ?", modelName, userID).First(&price)

	if result.Error != nil {
		return nil, result.Error
	}

	return &price, nil
}

// SaveModelPrice
func (tdb *TokenUsageDB) SaveModelPrice(price *ModelPrice) error {
	return tdb.db.Save(price).Error
}

// DeleteModelPrice
func (tdb *TokenUsageDB) DeleteModelPrice(modelName string, userID string) error {
	return tdb.db.Where("model_name = ? AND user_id = ?", modelName, userID).Delete(&ModelPrice{}).Error
}

// update ModelPrice
func (tdb *TokenUsageDB) UpdateModelPrice(price *ModelPrice) error {
	return tdb.db.Save(price).Error
}

// GetTokenUsageByRequestType 根据请求类型获取token使用情况
func (tdb *TokenUsageDB) GetTokenUsageByRequestType(userID string, requestType string, startTime, endTime time.Time) ([]*TokenUsage, error) {
	var usages []*TokenUsage
	result := tdb.db.Where("user_id = ? AND request_type = ? AND timestamp BETWEEN ? AND ?",
		userID, requestType, startTime, endTime).
		Order("timestamp DESC").
		Find(&usages)

	if result.Error != nil {
		return nil, result.Error
	}

	return usages, nil
}

// GetRevenueStats 获取收益统计
func (tdb *TokenUsageDB) GetRevenueStats(clientIDs []string, startTime, endTime time.Time) (map[string]float64, error) {
	type Result struct {
		TotalRevenue     float64
		ChatRevenue      float64
		EmbeddingRevenue float64
	}

	var result Result
	err := tdb.db.Model(&TokenUsage{}).
		Select(`
			SUM(revenue) as total_revenue,
			SUM(CASE WHEN request_type = 'chat' THEN revenue ELSE 0 END) as chat_revenue,
			SUM(CASE WHEN request_type = 'embedding' THEN revenue ELSE 0 END) as embedding_revenue
		`).
		Where("client_id IN ? AND timestamp BETWEEN ? AND ?", clientIDs, startTime, endTime).
		Scan(&result).Error

	if err != nil {
		return nil, err
	}

	return map[string]float64{
		"total":     result.TotalRevenue,
		"chat":      result.ChatRevenue,
		"embedding": result.EmbeddingRevenue,
	}, nil
}
