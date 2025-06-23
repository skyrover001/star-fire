// internal/models/api_key.go
package models

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type APIKey struct {
	ID        string `gorm:"primaryKey"`
	UserID    string `gorm:"index;not null"`
	Name      string `gorm:"not null"`
	Key       string `gorm:"column:key_value;uniqueIndex;not null"`
	Prefix    string `gorm:"not null"`
	LastUsed  *time.Time
	CreatedAt time.Time `gorm:"not null"`
	ExpiresAt time.Time `gorm:"not null"`
	Revoked   bool      `gorm:"default:false;not null"`
}

type APIKeyDB struct {
	db *gorm.DB
}

func NewAPIKeyDB(db *gorm.DB) *APIKeyDB {
	db.AutoMigrate(&APIKey{})
	return &APIKeyDB{db: db}
}

func (kdb *APIKeyDB) SaveAPIKey(key *APIKey) error {
	return kdb.db.Create(key).Error
}

func (kdb *APIKeyDB) GetAPIKeyByValue(keyValue string) (*APIKey, error) {
	var key APIKey
	result := kdb.db.Where("key_value = ?", keyValue).First(&key)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("API key not found")
		}
		return nil, result.Error
	}
	return &key, nil
}

func (kdb *APIKeyDB) GetAPIKeysByUser(userID string) ([]*APIKey, error) {
	var keys []*APIKey
	result := kdb.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&keys)
	if result.Error != nil {
		return nil, result.Error
	}
	return keys, nil
}

func (kdb *APIKeyDB) GetAPIKeysByUserID(userID string) ([]*APIKey, error) {
	var keys []*APIKey
	result := kdb.db.Where("user_id = ? AND revoked = ?", userID, false).
		Order("created_at DESC").
		Find(&keys)
	if result.Error != nil {
		return nil, result.Error
	}
	return keys, nil
}

func (kdb *APIKeyDB) GetAPIKeyByID(keyID string) (*APIKey, error) {
	var key APIKey
	result := kdb.db.Where("id = ?", keyID).First(&key)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("API key not found")
		}
		return nil, result.Error
	}
	return &key, nil

}

func (kdb *APIKeyDB) DeleteAPIKey(keyID string) error {
	result := kdb.db.Where("id = ?", keyID).Delete(&APIKey{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("API key not found")
	}
	return nil
}

func (kdb *APIKeyDB) UpdateLastUsed(keyID string) error {
	now := time.Now()
	return kdb.db.Model(&APIKey{}).
		Where("id = ?", keyID).
		Update("last_used", now).Error
}

func (kdb *APIKeyDB) RevokeAPIKey(userID string, keyID string) error {
	result := kdb.db.Model(&APIKey{}).
		Where("id = ? AND user_id = ?", keyID, userID).
		Update("revoked", true)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("API key not found or not authorized")
	}

	return nil
}

func (kdb *APIKeyDB) CountUserAPIKeys(userID string) (int, error) {
	var count int64
	result := kdb.db.Model(&APIKey{}).
		Where("user_id = ? AND revoked = ?", userID, false).
		Count(&count)

	return int(count), result.Error
}
