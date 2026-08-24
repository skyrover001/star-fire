package models

import (
	"time"

	"gorm.io/gorm"
)

// Notification 系统通知
type Notification struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    string    `gorm:"index;not null" json:"user_id"` // 接收者用户ID
	Type      string    `gorm:"not null" json:"type"`          // register / membership_expire / membership_upgrade
	Title     string    `gorm:"not null" json:"title"`
	Content   string    `gorm:"type:text" json:"content"`
	IsRead    bool      `gorm:"default:false" json:"is_read"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// NotificationDB 提供通知的读写方法
type NotificationDB struct {
	db *gorm.DB
}

func NewNotificationDB(db *gorm.DB) *NotificationDB {
	db.AutoMigrate(&Notification{})
	return &NotificationDB{db: db}
}

// Create 创建通知
func (n *NotificationDB) Create(userID, ntype, title, content string) error {
	notif := &Notification{
		UserID:  userID,
		Type:    ntype,
		Title:   title,
		Content: content,
	}
	return n.db.Create(notif).Error
}

// ListByUser 获取用户的通知列表（分页）
func (n *NotificationDB) ListByUser(userID string, page, size int) ([]*Notification, int64, error) {
	var list []*Notification
	var total int64
	n.db.Model(&Notification{}).Where("user_id = ?", userID).Count(&total)
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	err := n.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset((page - 1) * size).Limit(size).
		Find(&list).Error
	return list, total, err
}

// CountUnread 获取用户未读通知数
func (n *NotificationDB) CountUnread(userID string) (int64, error) {
	var count int64
	err := n.db.Model(&Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&count).Error
	return count, err
}

// MarkRead 标记通知为已读
func (n *NotificationDB) MarkRead(userID string, id uint) error {
	return n.db.Model(&Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("is_read", true).Error
}

// MarkAllRead 标记用户所有通知为已读
func (n *NotificationDB) MarkAllRead(userID string) error {
	return n.db.Model(&Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Update("is_read", true).Error
}
