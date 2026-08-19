package models

import (
	"strconv"
	"time"

	"gorm.io/gorm"
)

// SystemConfig 系统动态配置项（KV 结构）
type SystemConfig struct {
	Key       string    `gorm:"primaryKey" json:"key"`
	Value     string    `gorm:"not null" json:"value"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// SystemConfigDB 提供系统配置项的读写方法
type SystemConfigDB struct {
	db *gorm.DB
}

// NewSystemConfigDB 初始化 SystemConfigDB
func NewSystemConfigDB(db *gorm.DB) *SystemConfigDB {
	db.AutoMigrate(&SystemConfig{})
	return &SystemConfigDB{db: db}
}

// 配置项 key 常量
const (
	// ConfigKeyRegisterBonus 新注册会员赠送余额（元），0 表示不赠送
	ConfigKeyRegisterBonus = "register_bonus_balance"
)

// GetFloat 读取配置项并解析为 float64，不存在或解析失败返回默认值
func (s *SystemConfigDB) GetFloat(key string, defaultVal float64) float64 {
	var cfg SystemConfig
	if err := s.db.Where("key = ?", key).First(&cfg).Error; err != nil {
		return defaultVal
	}
	val, err := strconv.ParseFloat(cfg.Value, 64)
	if err != nil {
		return defaultVal
	}
	return val
}

// Set 写入配置项（主键存在则更新，不存在则插入）
func (s *SystemConfigDB) Set(key, value string) error {
	cfg := SystemConfig{Key: key, Value: value}
	return s.db.Save(&cfg).Error
}
