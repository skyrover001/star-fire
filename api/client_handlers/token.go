package client_handlers

import (
	"net/http"
	"star-fire/internal/service"

	"github.com/gin-gonic/gin"
)

// TokenHandler 处理注册Token相关HTTP请求
type TokenHandler struct {
	tokenService *service.TokenService
}

// NewTokenHandler 创建一个新的Token处理器
func NewTokenHandler(tokenService *service.TokenService) *TokenHandler {
	return &TokenHandler{
		tokenService: tokenService,
	}
}

// GenerateRegisterToken 处理生成注册Token请求
func (h *TokenHandler) GenerateRegisterToken(c *gin.Context) {
	// for text e.g. :curl -X POST http://localhost:8080/api/user/register-token -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	// 生成注册Token
	resp, err := h.tokenService.GenerateRegisterToken(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}
