package user_handlers

import (
	"net/http"
	"star-fire/internal/models"
	"strconv"

	"github.com/gin-gonic/gin"
)

// NotificationHandler 处理通知相关接口
type NotificationHandler struct {
	server *models.Server
}

func NewNotificationHandler(server *models.Server) *NotificationHandler {
	return &NotificationHandler{server: server}
}

// ListNotifications 获取当前用户的通知列表
func (h *NotificationHandler) ListNotifications(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return
	}
	userIDStr := userID.(string)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	list, total, err := h.server.NotificationDB.ListByUser(userIDStr, page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取通知失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  list,
		"total": total,
	})
}

// GetUnreadCount 获取当前用户未读通知数
func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return
	}
	userIDStr := userID.(string)

	count, err := h.server.NotificationDB.CountUnread(userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取未读数失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"unread": count})
}

// MarkNotificationRead 标记单条通知为已读
func (h *NotificationHandler) MarkNotificationRead(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return
	}
	userIDStr := userID.(string)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的通知ID"})
		return
	}

	if err := h.server.NotificationDB.MarkRead(userIDStr, uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "标记已读失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// MarkAllNotificationsRead 标记所有通知为已读
func (h *NotificationHandler) MarkAllNotificationsRead(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return
	}
	userIDStr := userID.(string)

	if err := h.server.NotificationDB.MarkAllRead(userIDStr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "标记已读失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}
