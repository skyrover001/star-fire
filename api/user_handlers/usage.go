package user_handlers

import (
	"fmt"
	"net/http"
	"star-fire/internal/models"
	"strconv"
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

// parseTimeRange 解析 start_date/end_date 查询参数，返回 (startTime, endTime, err)
// start_date 为空时默认最近 defaultDays 天；end_date 为空时默认当前时间
func parseTimeRange(c *gin.Context, defaultDays int) (time.Time, time.Time, error) {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	var startTime, endTime time.Time
	var err error

	if startDate != "" {
		startTime, err = time.Parse("2006-01-02", startDate)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("start_date is invalid")
		}
	} else {
		startTime = time.Now().AddDate(0, 0, -defaultDays)
	}

	if endDate != "" {
		endTime, err = time.Parse("2006-01-02", endDate)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("end_date is invalid")
		}
		endTime = endTime.Add(24*time.Hour - time.Second)
	} else {
		endTime = time.Now()
	}

	return startTime, endTime, nil
}

// parsePageParams 解析 page/size 查询参数，返回 (page, size)
// 默认 page=1, size=50, size 上限 100
func parsePageParams(c *gin.Context) (int, int) {
	page := 1
	size := 50

	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		page = p
	}
	if s, err := strconv.Atoi(c.Query("size")); err == nil && s > 0 {
		size = s
	}
	if size > 100 {
		size = 100
	}

	return page, size
}

// getUserIDFromContext 从 gin.Context 中提取 user_id 字符串
func getUserIDFromContext(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return "", false
	}
	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无效的用户ID"})
		return "", false
	}
	return userIDStr, true
}

// ==================== 使用统计（my-usage）====================

// GetUserTokenUsage 获取用户使用详单（分页）
func (h *TokenUsageHandler) GetUserTokenUsage(c *gin.Context) {
	userIDStr, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	startTime, endTime, err := parseTimeRange(c, 30)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	page, size := parsePageParams(c)

	usages, total, err := h.server.TokenUsageDB.GetUserTokenUsagePaged(userIDStr, startTime, endTime, page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query token usage failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total": total,
		"page":  page,
		"size":  size,
		"data":  usages,
	})
}

// GetUserUsageTotal 获取用户总计使用统计（无时间过滤，真·总计）
func (h *TokenUsageHandler) GetUserUsageTotal(c *gin.Context) {
	userIDStr, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	stats, err := h.server.TokenUsageDB.GetUsageTotalStatsByUserID(userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query usage total stats failed"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetUserUsageStats 获取用户指定时间段的聚合使用统计
func (h *TokenUsageHandler) GetUserUsageStats(c *gin.Context) {
	userIDStr, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	startTime, endTime, err := parseTimeRange(c, 30)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	stats, err := h.server.TokenUsageDB.GetUsageStatsByUserID(userIDStr, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query usage stats failed"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetUserUsageTrend 获取用户使用趋势（按天聚合）
func (h *TokenUsageHandler) GetUserUsageTrend(c *gin.Context) {
	userIDStr, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	startTime, endTime, err := parseTimeRange(c, 90)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	points, err := h.server.TokenUsageDB.GetUsageTrendByDay(userIDStr, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query usage trend failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(points),
		"data":  points,
	})
}

// GetUserUsageModels 获取用户按模型聚合的使用统计
func (h *TokenUsageHandler) GetUserUsageModels(c *gin.Context) {
	userIDStr, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	startTime, endTime, err := parseTimeRange(c, 30)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	stats, err := h.server.TokenUsageDB.GetUsageStatsByModel(userIDStr, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query usage model stats failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(stats),
		"data":  stats,
	})
}

// GetUserUsageModelDetail 获取用户某模型的使用详单（分页）
func (h *TokenUsageHandler) GetUserUsageModelDetail(c *gin.Context) {
	userIDStr, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	model := c.Param("model")
	if model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model is required"})
		return
	}

	startTime, endTime, err := parseTimeRange(c, 30)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	page, size := parsePageParams(c)

	usages, total, err := h.server.TokenUsageDB.GetUserTokenUsageByModelPaged(userIDStr, model, startTime, endTime, page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query model usage detail failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total": total,
		"page":  page,
		"size":  size,
		"data":  usages,
	})
}

// ==================== 收益统计（my-contribution）====================

// GetUserIncome 获取用户收益详单（分页）
func (h *TokenUsageHandler) GetUserIncome(c *gin.Context) {
	userIDStr, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	startTime, endTime, err := parseTimeRange(c, 30)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 先找到该用户所有的 client
	userClients, err := h.server.ClientDB.GetClientsByUserID(userIDStr)
	if userClients == nil || len(userClients) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"total": 0,
			"page":  1,
			"size":  50,
			"data":  []string{},
		})
		return
	}

	clientIDs := make([]string, 0, len(userClients))
	for _, client := range userClients {
		clientIDs = append(clientIDs, client.ID)
	}

	page, size := parsePageParams(c)

	usages, total, err := h.server.TokenUsageDB.GetIncomeTokenUsagePaged(clientIDs, startTime, endTime, page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query token usage failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total": total,
		"page":  page,
		"size":  size,
		"data":  usages,
	})
}

// GetUserIncomeTotal 获取用户总计收益统计（无时间过滤，真·总计）
func (h *TokenUsageHandler) GetUserIncomeTotal(c *gin.Context) {
	userIDStr, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	stats, err := h.server.TokenUsageDB.GetTotalIncomeStatsByUserID(userIDStr, h.server.ClientDB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query income total stats failed"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetUserIncomeTrend 获取用户收益趋势（按天聚合）
func (h *TokenUsageHandler) GetUserIncomeTrend(c *gin.Context) {
	userIDStr, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	startTime, endTime, err := parseTimeRange(c, 90)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userClients, err := h.server.ClientDB.GetClientsByUserID(userIDStr)
	if userClients == nil || len(userClients) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"total": 0,
			"data":  []string{},
		})
		return
	}

	clientIDs := make([]string, 0, len(userClients))
	for _, client := range userClients {
		clientIDs = append(clientIDs, client.ID)
	}

	points, err := h.server.TokenUsageDB.GetIncomeTrendByDay(clientIDs, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query income trend failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(points),
		"data":  points,
	})
}

// GetUserIncomeModels 获取用户按模型聚合的收益统计
func (h *TokenUsageHandler) GetUserIncomeModels(c *gin.Context) {
	userIDStr, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	startTime, endTime, err := parseTimeRange(c, 30)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userClients, err := h.server.ClientDB.GetClientsByUserID(userIDStr)
	if userClients == nil || len(userClients) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"total": 0,
			"data":  []string{},
		})
		return
	}

	clientIDs := make([]string, 0, len(userClients))
	for _, client := range userClients {
		clientIDs = append(clientIDs, client.ID)
	}

	stats, err := h.server.TokenUsageDB.GetIncomeStatsByModel(clientIDs, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query income model stats failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(stats),
		"data":  stats,
	})
}
