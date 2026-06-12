package user_handlers

import (
	"fmt"
	"math/rand"
	"net/http"
	"star-fire/internal/models"
	"time"

	"github.com/gin-gonic/gin"
)

type BalanceHandler struct {
	server *models.Server
}

func NewBalanceHandler(server *models.Server) *BalanceHandler {
	return &BalanceHandler{server: server}
}

// RechargeRequest is the request body for creating a recharge order
type RechargeRequest struct {
	Amount        float64 `json:"amount" binding:"required,min=0.01,max=10000"`
	PaymentMethod string  `json:"payment_method" binding:"required,oneof=wechat alipay"`
}

// GetBalance returns user's balance and total spent
func (h *BalanceHandler) GetBalance(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return
	}
	userIDStr := userID.(string)

	balance, totalSpent, err := h.server.UserDB.GetBalance(userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取余额失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"balance":     balance,
		"total_spent": totalSpent,
	})
}

// CreateRechargeOrder creates a simulated recharge order and returns QR code info.
// In production this would call WeChat/Alipay API to generate a real QR code.
func (h *BalanceHandler) CreateRechargeOrder(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return
	}
	userIDStr := userID.(string)

	var req RechargeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	// Generate unique order ID
	orderID := fmt.Sprintf("RC%d%06d", time.Now().Unix(), rand.Intn(1000000))

	// Generate simulated QR code content.
	// In production this would be a WeChat/Alipay payment URL.
	qrCodeContent := fmt.Sprintf("https://pay.starfire.local/simulate?order=%s&amount=%.2f&method=%s",
		orderID, req.Amount, req.PaymentMethod)

	record := &models.RechargeRecord{
		UserID:        userIDStr,
		Amount:        req.Amount,
		PaymentMethod: req.PaymentMethod,
		OrderID:       orderID,
		Status:        "pending",
		QrCodeContent: qrCodeContent,
	}

	if err := h.server.RechargeDB.CreateRechargeOrder(record); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建订单失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"order_id":        orderID,
		"amount":          req.Amount,
		"payment_method":  req.PaymentMethod,
		"qr_code_content": qrCodeContent,
		"status":          "pending",
		"message":         "请使用微信/支付宝扫码支付",
	})
}

// ConfirmRecharge simulates a successful payment and adds balance.
// In production this would be a webhook callback from WeChat/Alipay.
func (h *BalanceHandler) ConfirmRecharge(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return
	}
	userIDStr := userID.(string)

	var confirmReq struct {
		OrderID string `json:"order_id"`
	}
	if err := c.ShouldBindJSON(&confirmReq); err != nil || confirmReq.OrderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少订单号"})
		return
	}
	orderID := confirmReq.OrderID

	// Find the order
	order, err := h.server.RechargeDB.GetRechargeOrder(orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "订单不存在"})
		return
	}

	// Verify order belongs to current user
	if order.UserID != userIDStr {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此订单"})
		return
	}

	if order.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "订单状态不正确: " + order.Status})
		return
	}

	// Complete the order
	if err := h.server.RechargeDB.CompleteRecharge(orderID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "确认支付失败"})
		return
	}

	// Add balance to user
	if err := h.server.UserDB.AddBalance(userIDStr, order.Amount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "充值失败"})
		return
	}

	// Get updated balance
	balance, totalSpent, _ := h.server.UserDB.GetBalance(userIDStr)

	c.JSON(http.StatusOK, gin.H{
		"message":     "充值成功",
		"order_id":    orderID,
		"amount":      order.Amount,
		"balance":     balance,
		"total_spent": totalSpent,
	})
}

// GetRechargeHistory returns user's recharge history
func (h *BalanceHandler) GetRechargeHistory(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return
	}
	userIDStr := userID.(string)

	page := 1
	size := 10
	if p, ok := c.GetQuery("page"); ok {
		fmt.Sscanf(p, "%d", &page)
	}
	if s, ok := c.GetQuery("size"); ok {
		fmt.Sscanf(s, "%d", &size)
	}
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}

	records, total, err := h.server.RechargeDB.GetUserRechargeHistory(userIDStr, page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询充值记录失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": records,
		"total":  total,
	})
}
