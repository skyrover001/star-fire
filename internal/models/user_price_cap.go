package models

import (
	"errors"
	"log"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserModelPriceCap stores per-user, per-model price cap configuration.
// Unique constraint: (UserID, Model).
type UserModelPriceCap struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	UserID    string    `gorm:"uniqueIndex:idx_user_model;not null" json:"user_id"`
	Model     string    `gorm:"uniqueIndex:idx_user_model;not null" json:"model"`
	MaxIPPM   float64   `gorm:"not null" json:"max_ippm"` // max input price per million tokens
	MaxOPPM   float64   `gorm:"not null" json:"max_oppm"` // max output price per million tokens
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// priceCapCacheEntry is an in-memory TTL cache entry.
type priceCapCacheEntry struct {
	maxIPPM   float64
	maxOPPM   float64
	expiresAt time.Time
}

const priceCapCacheTTL = 60 * time.Second

// UserPriceCapDB manages UserModelPriceCap persistence with an in-process write-through cache.
type UserPriceCapDB struct {
	db    *gorm.DB
	cache sync.Map // key: "userID:model" → *priceCapCacheEntry
}

func NewUserPriceCapDB(db *gorm.DB) *UserPriceCapDB {
	if err := db.AutoMigrate(&UserModelPriceCap{}); err != nil {
		log.Fatalf("migrate UserModelPriceCap failed: %v", err)
	}
	return &UserPriceCapDB{db: db}
}

func priceCapCacheKey(userID, model string) string {
	return userID + ":" + model
}

// GetPriceCap returns the configured price caps for a user+model pair.
// Returns math.MaxFloat64 for both if no cap is configured (= unlimited).
func (p *UserPriceCapDB) GetPriceCap(userID, model string) (maxIPPM, maxOPPM float64) {
	key := priceCapCacheKey(userID, model)

	if v, ok := p.cache.Load(key); ok {
		entry := v.(*priceCapCacheEntry)
		if time.Now().Before(entry.expiresAt) {
			return entry.maxIPPM, entry.maxOPPM
		}
		p.cache.Delete(key)
	}

	var cap UserModelPriceCap
	err := p.db.Where("user_id = ? AND model = ?", userID, model).First(&cap).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			p.setCache(key, math.MaxFloat64, math.MaxFloat64)
			return math.MaxFloat64, math.MaxFloat64
		}
		log.Printf("GetPriceCap query error: %v", err)
		return math.MaxFloat64, math.MaxFloat64
	}

	p.setCache(key, cap.MaxIPPM, cap.MaxOPPM)
	return cap.MaxIPPM, cap.MaxOPPM
}

func (p *UserPriceCapDB) setCache(key string, maxIPPM, maxOPPM float64) {
	p.cache.Store(key, &priceCapCacheEntry{
		maxIPPM:   maxIPPM,
		maxOPPM:   maxOPPM,
		expiresAt: time.Now().Add(priceCapCacheTTL),
	})
}

// Upsert creates or updates the price cap for a user+model pair.
func (p *UserPriceCapDB) Upsert(userID, model string, maxIPPM, maxOPPM float64) (*UserModelPriceCap, error) {
	var cap UserModelPriceCap
	err := p.db.Where("user_id = ? AND model = ?", userID, model).First(&cap).Error
	now := time.Now()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		cap = UserModelPriceCap{
			ID:        uuid.NewString(),
			UserID:    userID,
			Model:     model,
			MaxIPPM:   maxIPPM,
			MaxOPPM:   maxOPPM,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := p.db.Create(&cap).Error; err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		cap.MaxIPPM = maxIPPM
		cap.MaxOPPM = maxOPPM
		cap.UpdatedAt = now
		if err := p.db.Save(&cap).Error; err != nil {
			return nil, err
		}
	}
	p.cache.Delete(priceCapCacheKey(userID, model))
	return &cap, nil
}

// Delete removes the price cap for a user+model pair and invalidates the cache.
func (p *UserPriceCapDB) Delete(userID, model string) error {
	result := p.db.Where("user_id = ? AND model = ?", userID, model).Delete(&UserModelPriceCap{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("price cap not found")
	}
	p.cache.Delete(priceCapCacheKey(userID, model))
	return nil
}

// GetByUser returns all price caps configured for a user, ordered by model name.
func (p *UserPriceCapDB) GetByUser(userID string) ([]*UserModelPriceCap, error) {
	var caps []*UserModelPriceCap
	return caps, p.db.Where("user_id = ?", userID).Order("model").Find(&caps).Error
}
