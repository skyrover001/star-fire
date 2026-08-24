package models

import (
	"log"
	"time"

	"gorm.io/gorm"
)

// ClientStats 记录 client 的在线稳定性（SLA）与服务等级统计。
// 用于 smart 负载均衡算法的 stability（在线稳定性）与 service（服务等级）维度。
type ClientStats struct {
	ClientID           string    `gorm:"primaryKey"`
	TotalOnlineSeconds int64     // 累计在线秒数
	DisconnectCount    int64     // 累计掉线次数
	LastOnlineAt       time.Time // 最近一次上线时间
	UpdatedAt          time.Time
}

// ClientStatsDB
type ClientStatsDB struct {
	db *gorm.DB
}

// NewClientStatsDB
func NewClientStatsDB(db *gorm.DB) *ClientStatsDB {
	if err := db.AutoMigrate(&ClientStats{}); err != nil {
		log.Fatalf("migrate client stats table: %v", err)
	}
	return &ClientStatsDB{db: db}
}

// GetOrCreate 获取指定 client 的统计记录，不存在则创建默认记录。
func (csdb *ClientStatsDB) GetOrCreate(clientID string) (*ClientStats, error) {
	var stats ClientStats
	err := csdb.db.Where("client_id = ?", clientID).First(&stats).Error
	if err == gorm.ErrRecordNotFound {
		stats = ClientStats{
			ClientID:     clientID,
			LastOnlineAt: time.Now(),
		}
		if err := csdb.db.Create(&stats).Error; err != nil {
			return nil, err
		}
		return &stats, nil
	}
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

// RecordOnline 记录 client 上线：设置 LastOnlineAt = now。
func (csdb *ClientStatsDB) RecordOnline(clientID string) error {
	stats, err := csdb.GetOrCreate(clientID)
	if err != nil {
		return err
	}
	stats.LastOnlineAt = time.Now()
	stats.UpdatedAt = time.Now()
	return csdb.db.Save(stats).Error
}

// RecordOffline 记录 client 掉线：DisconnectCount++，TotalOnlineSeconds += now - LastOnlineAt。
func (csdb *ClientStatsDB) RecordOffline(clientID string) error {
	stats, err := csdb.GetOrCreate(clientID)
	if err != nil {
		return err
	}
	now := time.Now()
	if !stats.LastOnlineAt.IsZero() {
		onlineSec := int64(now.Sub(stats.LastOnlineAt).Seconds())
		if onlineSec > 0 {
			stats.TotalOnlineSeconds += onlineSec
		}
	}
	stats.DisconnectCount++
	stats.UpdatedAt = now
	return csdb.db.Save(stats).Error
}

// GetStats 获取指定 client 的统计记录。
func (csdb *ClientStatsDB) GetStats(clientID string) (*ClientStats, error) {
	return csdb.GetOrCreate(clientID)
}
