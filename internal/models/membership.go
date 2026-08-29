package models

import (
	"time"

	configs "star-fire/config"
)

// 会员等级常量
const (
	MembershipNormal = "normal"
	MembershipVIP    = "vip"
	MembershipSVIP   = "svip"
)

// 各等级年费（元）
const (
	VIPPrice  = 20.0
	SVIPPrice = 50.0
)

// 各等级同时处理链接数上限（连接池大小）。
// normal=1, vip=5, svip=50（最高 100），均可通过环境变量配置。
func GetMaxConnections(membership string) int {
	switch membership {
	case MembershipSVIP:
		limit := configs.Config.SVIPMaxConnections
		if limit <= 0 {
			limit = 50
		}
		if limit > 100 {
			limit = 100
		}
		return limit
	case MembershipVIP:
		limit := configs.Config.VIPMaxConnections
		if limit <= 0 {
			limit = 5
		}
		return limit
	default:
		limit := configs.Config.NormalMaxConnections
		if limit <= 0 {
			limit = 1
		}
		return limit
	}
}

// GetMembershipPrice 返回指定会员等级的年费（元），未知等级返回 0
func GetMembershipPrice(membership string) float64 {
	switch membership {
	case MembershipVIP:
		return VIPPrice
	case MembershipSVIP:
		return SVIPPrice
	default:
		return 0
	}
}

// CalculateUpgradePrice 计算从 currentLevel 升级到 targetLevel 所需支付的金额。
// 规则：升级只需补差价（targetPrice - currentPrice）；同级续费付全价。
func CalculateUpgradePrice(currentLevel, targetLevel string) float64 {
	targetPrice := GetMembershipPrice(targetLevel)
	currentPrice := GetMembershipPrice(currentLevel)
	// 升级（不同等级且目标更贵）：补差价
	if currentLevel != targetLevel && currentPrice > 0 && targetPrice > currentPrice {
		return targetPrice - currentPrice
	}
	// 同级续费或普通购买：付全价
	return targetPrice
}

// CalculateExpireAt 计算购买/升级后的会员到期时间。
// 规则：升级（等级变化）重置为 1 年后；同级续费在原到期时间上延长 1 年（已过期则从今天算）。
// currentExpireAt 为 nil 表示未开通（MySQL 严格模式禁止零日期，会员到期时间以指针+NULL 存储）。
func CalculateExpireAt(currentLevel, targetLevel string, currentExpireAt *time.Time) time.Time {
	now := time.Now()
	if currentLevel == targetLevel && currentExpireAt != nil && now.Before(*currentExpireAt) {
		// 同级续费且未过期：延长 1 年
		return currentExpireAt.AddDate(1, 0, 0)
	}
	// 升级或已过期：重置为 1 年后
	return now.AddDate(1, 0, 0)
}
