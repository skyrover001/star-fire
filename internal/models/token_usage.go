package models

import (
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
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
	IPPM         float64   `gorm:"column:ip_pm;not null"`           // 输入tokens价格 - 数据库列名是 ip_pm
	OPPM         float64   `gorm:"column:oppm;not null"`            // 输出tokens价格 - 数据库列名是 oppm
	CIPPM        float64   `gorm:"column:cippm;not null;default:0"` // 缓存命中输入tokens价格
	InputTokens  int       `gorm:"not null"`
	OutputTokens int       `gorm:"not null"`
	CachedTokens int       `gorm:"not null;default:0"` // 缓存命中的输入tokens数
	TotalTokens  int       `gorm:"not null"`
	RequestType  string    `gorm:"not null;default:'chat'"` // 请求类型: chat, embedding
	Revenue      float64   `gorm:"not null;default:0"`      // 收益（client端收入）
	Cost         float64   `gorm:"not null;default:0"`      // 费用（user端支出）
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

	// 创建复合索引以大幅提升大表查询性能
	// (user_id, timestamp) — 使用量详单/统计/趋势查询
	db.Exec("CREATE INDEX IF NOT EXISTS idx_token_usages_user_timestamp ON token_usages(user_id, timestamp)")
	// (client_id, timestamp) — 收益详单/统计/趋势查询
	db.Exec("CREATE INDEX IF NOT EXISTS idx_token_usages_client_timestamp ON token_usages(client_id, timestamp)")
	// (user_id, model, timestamp) — 按模型的使用量查询
	db.Exec("CREATE INDEX IF NOT EXISTS idx_token_usages_user_model_timestamp ON token_usages(user_id, model, timestamp)")
	// (client_id, model, timestamp) — 按模型的收益查询
	db.Exec("CREATE INDEX IF NOT EXISTS idx_token_usages_client_model_timestamp ON token_usages(client_id, model, timestamp)")

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

func (tdb *TokenUsageDB) GetTotalIncomeByUserID(id string, clientDB *ClientDB) (interface{}, interface{}) {
	// 获取某个用户的总收益
	// 1. 先获取该用户的所有客户端
	// 2. 查询这些客户端的所有 token 使用记录
	// 3. 计算总收益 = (IPPM × InputTokens + OPPM × OutputTokens) / 1,000,000

	// 获取用户的所有客户端
	userClients, err := clientDB.GetClientsByUserID(id)
	if err != nil {
		return 0.0, err
	}

	if len(userClients) == 0 {
		// 没有客户端，返回 0
		return 0.0, nil
	}

	// 获取所有客户端 ID
	clientIDs := make([]string, 0, len(userClients))
	for _, client := range userClients {
		clientIDs = append(clientIDs, client.ID)
	}

	// 查询这些客户端的总收益（支持缓存命中分离计费）
	// non_cached_income = (input_tokens - cached_tokens) * ippm
	// cached_income = cached_tokens * cippm
	// output_income = output_tokens * oppm
	type Result struct {
		TotalIncome float64
	}

	var result Result
	err = tdb.db.Model(&TokenUsage{}).
		Select("SUM(((input_tokens - cached_tokens) * ip_pm + cached_tokens * cippm + output_tokens * oppm) / 1000000.0) as total_income").
		Where("client_id IN ?", clientIDs).
		Scan(&result).Error

	if err != nil {
		return 0.0, err
	}

	return result.TotalIncome, nil
}

// ==================== 收益统计（my-contribution）====================

// GetTotalIncomeStatsByUserID 获取用户总计收益统计（无时间过滤，真·总计）
// 返回 total_income/total_calls/input_tokens/output_tokens/cached_tokens/total_tokens/models
func (tdb *TokenUsageDB) GetTotalIncomeStatsByUserID(userID string, clientDB *ClientDB) (map[string]float64, error) {
	userClients, err := clientDB.GetClientsByUserID(userID)
	if err != nil {
		return nil, err
	}
	if len(userClients) == 0 {
		return map[string]float64{
			"total_income":  0,
			"total_calls":   0,
			"input_tokens":  0,
			"output_tokens": 0,
			"cached_tokens": 0,
			"total_tokens":  0,
			"models":        0,
			"client_count":  0,
			"unique_users":  0,
		}, nil
	}

	clientIDs := make([]string, 0, len(userClients))
	for _, client := range userClients {
		clientIDs = append(clientIDs, client.ID)
	}

	type Result struct {
		TotalIncome  float64
		TotalCalls   int64
		InputTokens  int64
		OutputTokens int64
		CachedTokens int64
		TotalTokens  int64
		Models       int64
		ClientCount  int64
		UniqueUsers  int64
	}

	var result Result
	err = tdb.db.Model(&TokenUsage{}).
		Select(`
			SUM(((input_tokens - cached_tokens) * ip_pm + cached_tokens * cippm + output_tokens * oppm) / 1000000.0) as total_income,
			COUNT(*) as total_calls,
			SUM(input_tokens) as input_tokens,
			SUM(output_tokens) as output_tokens,
			SUM(cached_tokens) as cached_tokens,
			SUM(total_tokens) as total_tokens,
			COUNT(DISTINCT model) as models,
			COUNT(DISTINCT client_id) as client_count,
			COUNT(DISTINCT user_id) as unique_users
		`).
		Where("client_id IN ?", clientIDs).
		Scan(&result).Error

	if err != nil {
		return nil, err
	}

	return map[string]float64{
		"total_income":  result.TotalIncome,
		"total_calls":   float64(result.TotalCalls),
		"input_tokens":  float64(result.InputTokens),
		"output_tokens": float64(result.OutputTokens),
		"cached_tokens": float64(result.CachedTokens),
		"total_tokens":  float64(result.TotalTokens),
		"models":        float64(result.Models),
		"client_count":  float64(result.ClientCount),
		"unique_users":  float64(result.UniqueUsers),
	}, nil
}

// GetIncomeStatsByTimeRange 获取用户指定时间段的收益统计（服务端聚合，不受分页/详单限制）
// 返回 total_income/total_calls/input_tokens/output_tokens/cached_tokens/total_tokens/models
func (tdb *TokenUsageDB) GetIncomeStatsByTimeRange(clientIDs []string, startTime, endTime time.Time) (map[string]float64, error) {
	if len(clientIDs) == 0 {
		return map[string]float64{
			"total_income":  0,
			"total_calls":   0,
			"input_tokens":  0,
			"output_tokens": 0,
			"cached_tokens": 0,
			"total_tokens":  0,
			"models":        0,
		}, nil
	}

	type Result struct {
		TotalIncome  float64
		TotalCalls   int64
		InputTokens  int64
		OutputTokens int64
		CachedTokens int64
		TotalTokens  int64
		Models       int64
	}

	var result Result
	err := tdb.db.Model(&TokenUsage{}).
		Select(`
			SUM(((input_tokens - cached_tokens) * ip_pm + cached_tokens * cippm + output_tokens * oppm) / 1000000.0) as total_income,
			COUNT(*) as total_calls,
			SUM(input_tokens) as input_tokens,
			SUM(output_tokens) as output_tokens,
			SUM(cached_tokens) as cached_tokens,
			SUM(total_tokens) as total_tokens,
			COUNT(DISTINCT model) as models
		`).
		Where("client_id IN ? AND timestamp BETWEEN ? AND ?", clientIDs, startTime, endTime).
		Scan(&result).Error

	if err != nil {
		return nil, err
	}

	return map[string]float64{
		"total_income":  result.TotalIncome,
		"total_calls":   float64(result.TotalCalls),
		"input_tokens":  float64(result.InputTokens),
		"output_tokens": float64(result.OutputTokens),
		"cached_tokens": float64(result.CachedTokens),
		"total_tokens":  float64(result.TotalTokens),
		"models":        float64(result.Models),
	}, nil
}

// GetIncomeTokenUsagePaged 分页获取收益详单
func (tdb *TokenUsageDB) GetIncomeTokenUsagePaged(clientIDs []string, startTime, endTime time.Time, page, size int) ([]*TokenUsage, int64, error) {
	var usages []*TokenUsage
	var total int64

	query := tdb.db.Model(&TokenUsage{}).Where("client_id IN ? AND timestamp BETWEEN ? AND ?",
		clientIDs, startTime, endTime)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	result := query.Order("timestamp DESC").Offset(offset).Limit(size).Find(&usages)
	if result.Error != nil {
		return nil, 0, result.Error
	}

	return usages, total, nil
}

// IncomeTrendPoint 按天收益趋势点
type IncomeTrendPoint struct {
	Date   string  `json:"date"`
	Income float64 `json:"income"`
	Calls  int64   `json:"calls"`
}

// GetIncomeTrendByDay 按天聚合收益趋势
func (tdb *TokenUsageDB) GetIncomeTrendByDay(clientIDs []string, startTime, endTime time.Time) ([]IncomeTrendPoint, error) {
	var points []IncomeTrendPoint
	err := tdb.db.Model(&TokenUsage{}).
		Select(`
			DATE(timestamp) as date,
			SUM(((input_tokens - cached_tokens) * ip_pm + cached_tokens * cippm + output_tokens * oppm) / 1000000.0) as income,
			COUNT(*) as calls
		`).
		Where("client_id IN ? AND timestamp BETWEEN ? AND ?", clientIDs, startTime, endTime).
		Group("DATE(timestamp)").
		Order("date ASC").
		Scan(&points).Error

	if err != nil {
		return nil, err
	}

	return points, nil
}

// ModelIncomeStat 按模型收益统计
type ModelIncomeStat struct {
	Model        string  `json:"model"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	CachedTokens int64   `json:"cached_tokens"`
	TotalTokens  int64   `json:"total_tokens"`
	Income       float64 `json:"income"`
	Calls        int64   `json:"calls"`
	ClientCount  int64   `json:"client_count"`
}

// GetIncomeStatsByModel 按模型聚合收益
func (tdb *TokenUsageDB) GetIncomeStatsByModel(clientIDs []string, startTime, endTime time.Time) ([]ModelIncomeStat, error) {
	var stats []ModelIncomeStat
	err := tdb.db.Model(&TokenUsage{}).
		Select(`
			model,
			SUM(input_tokens) as input_tokens,
			SUM(output_tokens) as output_tokens,
			SUM(cached_tokens) as cached_tokens,
			SUM(total_tokens) as total_tokens,
			SUM(((input_tokens - cached_tokens) * ip_pm + cached_tokens * cippm + output_tokens * oppm) / 1000000.0) as income,
			COUNT(*) as calls,
			COUNT(DISTINCT client_id) as client_count
		`).
		Where("client_id IN ? AND timestamp BETWEEN ? AND ?", clientIDs, startTime, endTime).
		Group("model").
		Order("income DESC").
		Scan(&stats).Error

	if err != nil {
		return nil, err
	}

	return stats, nil
}

// ==================== 使用统计（my-usage）====================

// GetUsageTotalStatsByUserID 获取用户总计使用统计（无时间过滤，真·总计）
func (tdb *TokenUsageDB) GetUsageTotalStatsByUserID(userID string) (map[string]float64, error) {
	type Result struct {
		TotalCalls   int64
		InputTokens  int64
		OutputTokens int64
		CachedTokens int64
		TotalTokens  int64
		TotalCost    float64
		ClientCount  int64
		ModelCount   int64
	}

	var result Result
	err := tdb.db.Model(&TokenUsage{}).
		Select(`
			COUNT(*) as total_calls,
			SUM(input_tokens) as input_tokens,
			SUM(output_tokens) as output_tokens,
			SUM(cached_tokens) as cached_tokens,
			SUM(total_tokens) as total_tokens,
			SUM(cost) as total_cost,
			COUNT(DISTINCT client_id) as client_count,
			COUNT(DISTINCT model) as model_count
		`).
		Where("user_id = ?", userID).
		Scan(&result).Error

	if err != nil {
		return nil, err
	}

	return map[string]float64{
		"total_calls":   float64(result.TotalCalls),
		"input_tokens":  float64(result.InputTokens),
		"output_tokens": float64(result.OutputTokens),
		"cached_tokens": float64(result.CachedTokens),
		"total_tokens":  float64(result.TotalTokens),
		"total_cost":    result.TotalCost,
		"client_count":  float64(result.ClientCount),
		"model_count":   float64(result.ModelCount),
	}, nil
}

// GetUsageStatsByUserID 获取用户指定时间段的聚合使用统计
func (tdb *TokenUsageDB) GetUsageStatsByUserID(userID string, startTime, endTime time.Time) (map[string]float64, error) {
	type Result struct {
		TotalCalls   int64
		InputTokens  int64
		OutputTokens int64
		CachedTokens int64
		TotalTokens  int64
		TotalCost    float64
		ClientCount  int64
		ModelCount   int64
	}

	var result Result
	err := tdb.db.Model(&TokenUsage{}).
		Select(`
			COUNT(*) as total_calls,
			SUM(input_tokens) as input_tokens,
			SUM(output_tokens) as output_tokens,
			SUM(cached_tokens) as cached_tokens,
			SUM(total_tokens) as total_tokens,
			SUM(cost) as total_cost,
			COUNT(DISTINCT client_id) as client_count,
			COUNT(DISTINCT model) as model_count
		`).
		Where("user_id = ? AND timestamp BETWEEN ? AND ?", userID, startTime, endTime).
		Scan(&result).Error

	if err != nil {
		return nil, err
	}

	return map[string]float64{
		"total_calls":   float64(result.TotalCalls),
		"input_tokens":  float64(result.InputTokens),
		"output_tokens": float64(result.OutputTokens),
		"cached_tokens": float64(result.CachedTokens),
		"total_tokens":  float64(result.TotalTokens),
		"total_cost":    result.TotalCost,
		"client_count":  float64(result.ClientCount),
		"model_count":   float64(result.ModelCount),
	}, nil
}

// GetUserTokenUsagePaged 分页获取用户使用详单
func (tdb *TokenUsageDB) GetUserTokenUsagePaged(userID string, startTime, endTime time.Time, page, size int) ([]*TokenUsage, int64, error) {
	var usages []*TokenUsage
	var total int64

	query := tdb.db.Model(&TokenUsage{}).Where("user_id = ? AND timestamp BETWEEN ? AND ?",
		userID, startTime, endTime)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	result := query.Order("timestamp DESC").Offset(offset).Limit(size).Find(&usages)
	if result.Error != nil {
		return nil, 0, result.Error
	}

	return usages, total, nil
}

// UsageTrendPoint 按天使用趋势点
type UsageTrendPoint struct {
	Date         string `json:"date"`
	InputTokens  int64  `json:"input_tokens"`
	OutputTokens int64  `json:"output_tokens"`
	TotalTokens  int64  `json:"total_tokens"`
	Calls        int64  `json:"calls"`
}

// GetUsageTrendByDay 按天聚合使用趋势
func (tdb *TokenUsageDB) GetUsageTrendByDay(userID string, startTime, endTime time.Time) ([]UsageTrendPoint, error) {
	var points []UsageTrendPoint
	err := tdb.db.Model(&TokenUsage{}).
		Select(`
			DATE(timestamp) as date,
			SUM(input_tokens) as input_tokens,
			SUM(output_tokens) as output_tokens,
			SUM(total_tokens) as total_tokens,
			COUNT(*) as calls
		`).
		Where("user_id = ? AND timestamp BETWEEN ? AND ?", userID, startTime, endTime).
		Group("DATE(timestamp)").
		Order("date ASC").
		Scan(&points).Error

	if err != nil {
		return nil, err
	}

	return points, nil
}

// ModelUsageStat 按模型使用统计
type ModelUsageStat struct {
	Model        string  `json:"model"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	CachedTokens int64   `json:"cached_tokens"`
	TotalTokens  int64   `json:"total_tokens"`
	TotalCost    float64 `json:"total_cost"`
	Calls        int64   `json:"calls"`
	ClientCount  int64   `json:"client_count"`
	LastUsed     string  `json:"last_used"`
}

// GetUsageStatsByModel 按模型聚合使用统计
func (tdb *TokenUsageDB) GetUsageStatsByModel(userID string, startTime, endTime time.Time) ([]ModelUsageStat, error) {
	var stats []ModelUsageStat
	err := tdb.db.Model(&TokenUsage{}).
		Select(`
			model,
			SUM(input_tokens) as input_tokens,
			SUM(output_tokens) as output_tokens,
			SUM(cached_tokens) as cached_tokens,
			SUM(total_tokens) as total_tokens,
			SUM(cost) as total_cost,
			COUNT(*) as calls,
			COUNT(DISTINCT client_id) as client_count,
			MAX(timestamp) as last_used
		`).
		Where("user_id = ? AND timestamp BETWEEN ? AND ?", userID, startTime, endTime).
		Group("model").
		Order("total_cost DESC").
		Scan(&stats).Error

	if err != nil {
		return nil, err
	}

	return stats, nil
}

// GetUserTokenUsageByModelPaged 分页获取某模型的使用详单
func (tdb *TokenUsageDB) GetUserTokenUsageByModelPaged(userID, model string, startTime, endTime time.Time, page, size int) ([]*TokenUsage, int64, error) {
	var usages []*TokenUsage
	var total int64

	query := tdb.db.Model(&TokenUsage{}).Where("user_id = ? AND model = ? AND timestamp BETWEEN ? AND ?",
		userID, model, startTime, endTime)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	result := query.Order("timestamp DESC").Offset(offset).Limit(size).Find(&usages)
	if result.Error != nil {
		return nil, 0, result.Error
	}

	return usages, total, nil
}

// ModelMarketStat 模型广场展示的全局使用统计（跨所有用户）
type ModelMarketStat struct {
	Model        string `json:"model"`
	InputTokens  int64  `json:"input_tokens"`
	OutputTokens int64  `json:"output_tokens"`
	CachedTokens int64  `json:"cached_tokens"`
	TotalTokens  int64  `json:"total_tokens"`
	Calls        int64  `json:"calls"`
	ClientCount  int64  `json:"client_count"`
	UserCount    int64  `json:"user_count"`
	LastUsed     string `json:"last_used"`
}

// PublicHomepageStats is the read-only aggregate displayed on the public landing page.
type PublicHomepageStats struct {
	TotalTokens int64              `json:"total_tokens"`
	TotalCalls  int64              `json:"total_calls"`
	TotalValue  float64            `json:"total_value"`
	ActiveUsers int64              `json:"active_users"`
	ModelStats  []ModelMarketStat  `json:"model_stats"`
	DailyTrend  []PublicDailyTrend `json:"daily_trend"`
	// ModelRank 模型调用次数与 tokens 排名（总的，跨所有用户）
	ModelRank []ModelRankEntry `json:"model_rank"`
	// ContributorRank 贡献者收益排名（前10，按总收益降序，单位 $）
	ContributorRank []ContributorRankEntry `json:"contributor_rank"`
}

type PublicDailyTrend struct {
	Date        string  `json:"date"`
	TotalTokens int64   `json:"total_tokens"`
	TotalValue  float64 `json:"total_value"`
}

// ModelRankEntry 模型调用排名条目（总的）
type ModelRankEntry struct {
	Model       string `json:"model"`
	Calls       int64  `json:"calls"`
	TotalTokens int64  `json:"total_tokens"`
}

// ContributorRankEntry 贡献者收益排名条目（前10）
type ContributorRankEntry struct {
	// DisplayName 脱敏后的用户名（隐藏中间字符）
	DisplayName string  `json:"display_name"`
	Income      float64 `json:"income"` // 单位 $
}

// maskUsername 隐藏用户名中间字符以保护隐私。
// 规则：长度<=2 时只保留首字符；否则保留首尾各1个字符，中间用 * 填充。
func maskUsername(name string) string {
	runes := []rune(strings.TrimSpace(name))
	n := len(runes)
	if n == 0 {
		return "***"
	}
	if n == 1 {
		return string(runes[0]) + "**"
	}
	if n == 2 {
		return string(runes[0]) + "*"
	}
	// 保留首尾各1个字符，中间用 * 填充
	return string(runes[0]) + strings.Repeat("*", n-2) + string(runes[n-1])
}

// GetModelRank 获取所有模型总的调用次数与 tokens 排名（跨所有用户，无时间过滤）
func (tdb *TokenUsageDB) GetModelRank() ([]ModelRankEntry, error) {
	var entries []ModelRankEntry
	err := tdb.db.Model(&TokenUsage{}).
		Select(`
			model,
			COUNT(*) as calls,
			SUM(total_tokens) as total_tokens
		`).
		Group("model").
		Order("calls DESC").
		Scan(&entries).Error
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// GetContributorRank 获取贡献者收益排名（前10，按总收益降序，单位 $）。
// 收益按 client 端收入口径计算：((input_tokens - cached_tokens) * ip_pm + cached_tokens * cippm + output_tokens * oppm) / 1e6。
// 通过 client 关联到其所属 user，并对用户名做脱敏处理。
func (tdb *TokenUsageDB) GetContributorRank(limit int, clientDB *ClientDB, userDB *UserDB) ([]ContributorRankEntry, error) {
	if limit <= 0 {
		limit = 10
	}

	type Row struct {
		ClientID string
		Income   float64
	}
	var rows []Row
	err := tdb.db.Model(&TokenUsage{}).
		Select(`
			client_id,
			SUM(((input_tokens - cached_tokens) * ip_pm + cached_tokens * cippm + output_tokens * oppm) / 1000000.0) as income
		`).
		Group("client_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	// 汇总每个 client 的收益，再映射到其所属 user
	clientIncome := make(map[string]float64, len(rows))
	for _, r := range rows {
		clientIncome[r.ClientID] += r.Income
	}

	// 聚合到用户维度
	userIncome := make(map[string]float64)
	for clientID, income := range clientIncome {
		client, err := clientDB.GetClient(clientID)
		if err != nil || client == nil {
			continue
		}
		userIncome[client.UserID] += income
	}

	// 按收益降序排序
	type kv struct {
		userID string
		income float64
	}
	sorted := make([]kv, 0, len(userIncome))
	for uid, income := range userIncome {
		sorted = append(sorted, kv{userID: uid, income: income})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].income > sorted[j].income
	})

	if len(sorted) > limit {
		sorted = sorted[:limit]
	}

	entries := make([]ContributorRankEntry, 0, len(sorted))
	for _, s := range sorted {
		displayName := "***"
		if user, err := userDB.GetUserByID(s.userID); err == nil {
			displayName = maskUsername(user.Username)
		}
		entries = append(entries, ContributorRankEntry{
			DisplayName: displayName,
			Income:      s.income,
		})
	}
	return entries, nil
}

// GetPublicHomepageStats aggregates all recorded usage without exposing user or client details.
func (tdb *TokenUsageDB) GetPublicHomepageStats(clientDB *ClientDB, userDB *UserDB) (*PublicHomepageStats, error) {
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -6)

	var totals struct {
		TotalTokens int64
		TotalCalls  int64
		TotalValue  float64
		ActiveUsers int64
	}
	if err := tdb.db.Model(&TokenUsage{}).
		Select(`
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COUNT(*) as total_calls,
			COALESCE(SUM(cost), 0) as total_value,
			COUNT(DISTINCT user_id) as active_users
		`).
		Scan(&totals).Error; err != nil {
		return nil, err
	}

	modelStats, err := tdb.GetModelMarketStats(startTime, endTime)
	if err != nil {
		return nil, err
	}

	var dailyTrend []PublicDailyTrend
	if err := tdb.db.Model(&TokenUsage{}).
		Select(`
			DATE(timestamp) as date,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COALESCE(SUM(cost), 0) as total_value
		`).
		Where("timestamp BETWEEN ? AND ?", startTime, endTime).
		Group("DATE(timestamp)").
		Order("DATE(timestamp)").
		Scan(&dailyTrend).Error; err != nil {
		return nil, err
	}

	// 模型调用次数与 tokens 排名（总的）
	modelRank, err := tdb.GetModelRank()
	if err != nil {
		return nil, err
	}

	// 贡献者收益排名（前10，单位 $）
	contributorRank, err := tdb.GetContributorRank(10, clientDB, userDB)
	if err != nil {
		return nil, err
	}

	return &PublicHomepageStats{
		TotalTokens:     totals.TotalTokens,
		TotalCalls:      totals.TotalCalls,
		TotalValue:      totals.TotalValue,
		ActiveUsers:     totals.ActiveUsers,
		ModelStats:      modelStats,
		DailyTrend:      dailyTrend,
		ModelRank:       modelRank,
		ContributorRank: contributorRank,
	}, nil
}

// GetModelMarketStats 获取所有模型在指定时间段的全局使用统计（跨所有用户）
func (tdb *TokenUsageDB) GetModelMarketStats(startTime, endTime time.Time) ([]ModelMarketStat, error) {
	var stats []ModelMarketStat
	err := tdb.db.Model(&TokenUsage{}).
		Select(`
			model,
			SUM(input_tokens) as input_tokens,
			SUM(output_tokens) as output_tokens,
			SUM(cached_tokens) as cached_tokens,
			SUM(total_tokens) as total_tokens,
			COUNT(*) as calls,
			COUNT(DISTINCT client_id) as client_count,
			COUNT(DISTINCT user_id) as user_count,
			MAX(timestamp) as last_used
		`).
		Where("timestamp BETWEEN ? AND ?", startTime, endTime).
		Group("model").
		Scan(&stats).Error

	if err != nil {
		return nil, err
	}

	return stats, nil
}
