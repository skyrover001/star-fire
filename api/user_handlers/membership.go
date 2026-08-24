package user_handlers

import (
	"fmt"
	"net/http"
	configs "star-fire/config"
	"star-fire/internal/models"
	"time"

	"github.com/gin-gonic/gin"
)

// rateLimitFor 根据消费者会员等级返回限流配置（RPM/TPM）。
// 优先使用环境变量配置，未配置时使用默认值。
func rateLimitFor(membership string) models.RateLimitConfig {
	switch membership {
	case models.MembershipSVIP:
		if cfg := models.ParseRateLimit(configs.Config.RateLimitSVIP); cfg.RPM > 0 || cfg.TPM > 0 {
			return cfg
		}
		return models.DefaultRateLimit(models.MembershipSVIP)
	case models.MembershipVIP:
		if cfg := models.ParseRateLimit(configs.Config.RateLimitVIP); cfg.RPM > 0 || cfg.TPM > 0 {
			return cfg
		}
		return models.DefaultRateLimit(models.MembershipVIP)
	default:
		if cfg := models.ParseRateLimit(configs.Config.RateLimitNormal); cfg.RPM > 0 || cfg.TPM > 0 {
			return cfg
		}
		return models.DefaultRateLimit(models.MembershipNormal)
	}
}

// MembershipHandler 处理会员购买与查询
type MembershipHandler struct {
	server *models.Server
}

func NewMembershipHandler(server *models.Server) *MembershipHandler {
	return &MembershipHandler{server: server}
}

// BuyMembershipRequest 购买会员请求
type BuyMembershipRequest struct {
	Level string `json:"level" binding:"required,oneof=vip svip"`
	// Type 会员类型：consumer（消费者，默认）/ contributor（贡献者）
	Type string `json:"type" binding:"omitempty,oneof=consumer contributor"`
}

// BuyMembership 用余额购买/升级会员。
// 价格规则：升级补差价（VIP→SVIP 只需 30 元），同级续费付全价。
// 有效期规则：升级重置为 1 年，同级续费在原到期时间上延长 1 年。
// 消费者会员与贡献者会员相互独立，通过 Type 区分。
func (h *MembershipHandler) BuyMembership(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return
	}
	userIDStr := userID.(string)

	var req BuyMembershipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if req.Type == "" {
		req.Type = "consumer"
	}
	isContributor := req.Type == "contributor"

	// 获取当前有效会员等级
	var currentLevel string
	if isContributor {
		currentLevel = h.server.UserDB.GetEffectiveContributorMembership(userIDStr)
	} else {
		currentLevel = h.server.UserDB.GetEffectiveMembership(userIDStr)
	}

	// 计算实际应付金额（升级补差价 / 续费全价）
	price := models.CalculateUpgradePrice(currentLevel, req.Level)
	if price <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的会员等级"})
		return
	}

	// 检查余额是否足够
	balance, _, err := h.server.UserDB.GetBalance(userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取余额失败"})
		return
	}
	if balance < price {
		c.JSON(http.StatusBadRequest, gin.H{"error": "余额不足，请先充值"})
		return
	}

	// 扣款
	if err := h.server.UserDB.DeductBalance(userIDStr, price); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "扣款失败"})
		return
	}

	// 获取当前到期时间，计算新的到期时间
	user, _ := h.server.UserDB.GetUserByID(userIDStr)
	var expireAt time.Time
	if isContributor {
		expireAt = models.CalculateExpireAt(currentLevel, req.Level, user.ContributorMembershipExpireAt)
		if err := h.server.UserDB.SetContributorMembership(userIDStr, req.Level, expireAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "设置会员失败"})
			return
		}
	} else {
		expireAt = models.CalculateExpireAt(currentLevel, req.Level, user.MembershipExpireAt)
		if err := h.server.UserDB.SetMembership(userIDStr, req.Level, expireAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "设置会员失败"})
			return
		}
	}

	// 会员等级变化后重置限流桶，使新的限流额度立即生效
	if h.server.RateLimiter != nil {
		h.server.RateLimiter.Reset("user:" + userIDStr)
	}

	// 通知用户会员升级成功
	if h.server.NotificationDB != nil {
		levelName := "VIP"
		if req.Level == models.MembershipSVIP {
			levelName = "SVIP"
		}
		typeName := "消费者"
		if isContributor {
			typeName = "贡献者"
		}
		_ = h.server.NotificationDB.Create(
			userIDStr,
			"membership_upgrade",
			"会员升级成功",
			fmt.Sprintf("恭喜！您已升级为%s %s 会员，有效期至 %s", typeName, levelName, expireAt.Format("2006-01-02")),
		)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "购买成功",
		"type":            req.Type,
		"membership":      req.Level,
		"expire_at":       expireAt,
		"max_connections": models.GetMaxConnections(req.Level),
		"paid":            price,
	})
}

// GetMembership 查询当前会员状态（消费者 + 贡献者）
func (h *MembershipHandler) GetMembership(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return
	}
	userIDStr := userID.(string)

	user, err := h.server.UserDB.GetUserByID(userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户失败"})
		return
	}

	// 消费者会员
	effective := h.server.UserDB.GetEffectiveMembership(userIDStr)
	// 贡献者会员
	contributorEffective := h.server.UserDB.GetEffectiveContributorMembership(userIDStr)

	// 消费者限流配置（RPM/TPM）
	consumerRateLimit := rateLimitFor(effective)

	// 汇总该用户所有在线 client 的当前连接数（贡献者连接池）
	currentConnections, onlineClients := h.server.GetUserClientConnections(userIDStr)
	maxConnections := models.GetMaxConnections(contributorEffective)

	// 连接池使用程度（百分比），SVIP 无限时返回 -1
	usagePercent := -1
	if maxConnections > 0 {
		usagePercent = int(float64(currentConnections) / float64(maxConnections) * 100)
		if usagePercent > 100 {
			usagePercent = 100
		}
	}

	c.JSON(http.StatusOK, gin.H{
		// 消费者会员（向后兼容）
		"membership":         effective,
		"expire_at":          user.MembershipExpireAt,
		"vip_upgrade_price":  models.CalculateUpgradePrice(effective, models.MembershipVIP),
		"svip_upgrade_price": models.CalculateUpgradePrice(effective, models.MembershipSVIP),
		// 消费者限流配置
		"rate_limit_rpm": consumerRateLimit.RPM,
		"rate_limit_tpm": consumerRateLimit.TPM,
		// 贡献者会员
		"contributor_membership":         contributorEffective,
		"contributor_expire_at":          user.ContributorMembershipExpireAt,
		"contributor_vip_upgrade_price":  models.CalculateUpgradePrice(contributorEffective, models.MembershipVIP),
		"contributor_svip_upgrade_price": models.CalculateUpgradePrice(contributorEffective, models.MembershipSVIP),
		// 贡献者连接池
		"max_connections":     maxConnections,
		"current_connections": currentConnections,
		"online_clients":      onlineClients,
		"usage_percent":       usagePercent,
		"vip_price":           models.VIPPrice,
		"svip_price":          models.SVIPPrice,
	})
}
