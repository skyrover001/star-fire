package user_handlers

import (
	"net/http"
	"star-fire/internal/models"
	"time"

	"github.com/gin-gonic/gin"
)

type TokenUsageHandler struct {
	server *models.Server
}

func NewTokenUsageHandler(server *models.Server) *TokenUsageHandler {
	return &TokenUsageHandler{
		server: server,
	}
}

// GetUserTokenUsage
func (h *TokenUsageHandler) GetUserTokenUsage(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return
	}
	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无效的用户ID"})
		return
	}
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	var startTime, endTime time.Time
	var err error

	if startDate != "" {
		startTime, err = time.Parse("2006-01-02", startDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "start_date is invalid"})
			return
		}
	} else {
		startTime = time.Now().AddDate(0, 0, -30)
	}

	if endDate != "" {
		endTime, err = time.Parse("2006-01-02", endDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "end_date is invalid"})
			return
		}
		endTime = endTime.Add(24*time.Hour - time.Second)
	} else {
		endTime = time.Now()
	}

	usages, err := h.server.TokenUsageDB.GetUserTokenUsage(userIDStr, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query token usage failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(usages),
		"data":  usages,
	})
}

func (h *TokenUsageHandler) GetUserIncome(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return
	}
	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无效的用户ID"})
		return
	}
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	var startTime, endTime time.Time
	var err error

	if startDate != "" {
		startTime, err = time.Parse("2006-01-02", startDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "start_date is invalid"})
			return
		}
	} else {
		startTime = time.Now().AddDate(0, 0, -30)
	}

	if endDate != "" {
		endTime, err = time.Parse("2006-01-02", endDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "end_date is invalid"})
			return
		}
		endTime = endTime.Add(24*time.Hour - time.Second)
	} else {
		endTime = time.Now()
	}

	// 先找到该用户所有的client
	// 再查询所有client的token使用情况
	userClients, err := h.server.ClientDB.GetClientsByUserID(userIDStr)
	if userClients == nil {
		c.JSON(http.StatusOK, gin.H{
			"total": 0,
			"data":  []string{},
		})
		return
	}
	// 获取所有client的ID
	clientIDs := make([]string, 0, len(userClients))
	for _, client := range userClients {
		clientIDs = append(clientIDs, client.ID)
	}
	// 查询所有client的token使用情况
	if len(clientIDs) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"total": 0,
			"data":  []string{},
		})
		return
	}
	// 获取所有client的token使用情况
	usages, err := h.server.TokenUsageDB.GetIncomeTokenUsage(clientIDs, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query token usage failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(usages),
		"data":  usages,
	})
}
