package models

import (
	"gorm.io/gorm"
	"log"
)

// Trend represents a trend in the marketplace in sqlite
type Trend struct {
	ID          int64   `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string  `json:"name" gorm:"not null;Index"`
	Description string  `json:"description" gorm:"not null"`
	CreatedAt   string  `json:"created_at" gorm:"not null"`
	UpdatedAt   string  `json:"updated_at" gorm:"not null"`
	DeletedAt   string  `json:"deleted_at" gorm:"default:0"` // Use 0 for not deleted
	Active      bool    `json:"active" gorm:"default:true;not null"`
	User        *User   `json:"user" gorm:"-"`
	Client      *Client `json:"client" gorm:"-"`
}

// TrendDB provides methods to interact with the trends in the database.
type TrendDB struct {
	db *gorm.DB
}

// NewTrendDB initializes a new TrendDB instance and migrates the Trend model.
func NewTrendDB(db *gorm.DB) *TrendDB {
	db.AutoMigrate(&Trend{})
	return &TrendDB{db: db}
}

// SaveTrend saves a new trend to the database.
func (t *TrendDB) SaveTrend(trend *Trend) error {
	return t.db.Create(trend).Error
}

// GetTrendByID retrieves a trend by its ID.
func (t *TrendDB) GetTrendByID(id int64) (*Trend, error) {
	var trend Trend
	result := t.db.First(&trend, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil // Not found
		}
		return nil, result.Error // Other error
	}
	return &trend, nil
}

// GetTrendsByUserID retrieves all trends associated with a specific user ID.
func (t *TrendDB) GetTrendsByUserID(userID string) ([]*Trend, error) {
	var trends []*Trend
	result := t.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&trends)
	if result.Error != nil {
		return nil, result.Error
	}
	return trends, nil
}

// get trends by time range
func (t *TrendDB) GetTrendsByTimeRange(start, end string) ([]*Trend, error) {
	var trends []*Trend
	log.Println("GetTrendsByTimeRange", start, end)
	result := t.db.Where("created_at >= ? AND created_at <= ?", start, end).Order("created_at DESC").Find(&trends)
	if result.Error != nil {
		return nil, result.Error
	}
	return trends, nil
}

// get trends by time range with pagination
func (t *TrendDB) GetTrendsByTimeRangeWithPagination(start, end string, page, size int) ([]*Trend, int64, error) {
	var trends []*Trend
	var total int64

	log.Println("GetTrendsByTimeRangeWithPagination", start, end, "page:", page, "size:", size)

	// 构建基础查询
	query := t.db.Model(&Trend{})
	if start != "" && end != "" {
		query = query.Where("created_at >= ? AND created_at <= ?", start, end)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 计算偏移量
	offset := (page - 1) * size

	// 获取分页数据
	result := query.Order("created_at DESC").Offset(offset).Limit(size).Find(&trends)
	if result.Error != nil {
		return nil, 0, result.Error
	}

	return trends, total, nil
}

// TrendsResponse represents the paginated response for trends
type TrendsResponse struct {
	Data       []*Trend `json:"data"`
	Total      int64    `json:"total"`
	Page       int      `json:"page"`
	Size       int      `json:"size"`
	TotalPages int      `json:"total_pages"`
}
