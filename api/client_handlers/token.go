package client_handlers

import (
	"net/http"
	"star-fire/internal/service"

	"github.com/gin-gonic/gin"
)

// TokenHandler
type TokenHandler struct {
	tokenService *service.TokenService
}

// NewTokenHandler
func NewTokenHandler(tokenService *service.TokenService) *TokenHandler {
	return &TokenHandler{
		tokenService: tokenService,
	}
}

// GenerateRegisterToken
func (h *TokenHandler) GenerateRegisterToken(c *gin.Context) {
	// for text e.g. :curl -X POST http://localhost:8080/api/user/register-token -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}
	resp, err := h.tokenService.GenerateRegisterToken(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}
