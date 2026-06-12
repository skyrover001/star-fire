package models

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// RechargeRecord represents a recharge/payment record
type RechargeRecord struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserID        string    `gorm:"index;not null" json:"user_id"`
	Amount        float64   `gorm:"not null" json:"amount"`                   // 充值金额（元）
	PaymentMethod string    `gorm:"not null" json:"payment_method"`           // wechat, alipay
	Status        string    `gorm:"not null;default:'pending'" json:"status"` // pending, completed, failed
	OrderID       string    `gorm:"uniqueIndex;not null" json:"order_id"`     // 订单号
	QrCodeContent string    `gorm:"type:text" json:"qr_code_content"`         // 模拟二维码内容
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// RechargeDB provides methods to interact with recharge records
type RechargeDB struct {
	db *gorm.DB
}

// NewRechargeDB initializes a new RechargeDB
func NewRechargeDB(db *gorm.DB) *RechargeDB {
	db.AutoMigrate(&RechargeRecord{})
	return &RechargeDB{db: db}
}

// CreateRechargeOrder creates a new recharge order
func (r *RechargeDB) CreateRechargeOrder(record *RechargeRecord) error {
	return r.db.Create(record).Error
}

// GetRechargeOrder gets a recharge order by order ID
func (r *RechargeDB) GetRechargeOrder(orderID string) (*RechargeRecord, error) {
	var record RechargeRecord
	result := r.db.Where("order_id = ?", orderID).First(&record)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("order not found")
		}
		return nil, result.Error
	}
	return &record, nil
}

// CompleteRecharge updates order status to completed
func (r *RechargeDB) CompleteRecharge(orderID string) error {
	return r.db.Model(&RechargeRecord{}).
		Where("order_id = ? AND status = 'pending'", orderID).
		Update("status", "completed").Error
}

// GetUserRechargeHistory gets user's recharge history
func (r *RechargeDB) GetUserRechargeHistory(userID string, page, size int) ([]*RechargeRecord, int64, error) {
	var records []*RechargeRecord
	var total int64

	r.db.Model(&RechargeRecord{}).Where("user_id = ?", userID).Count(&total)

	offset := (page - 1) * size
	result := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").Offset(offset).Limit(size).Find(&records)
	return records, total, result.Error
}
