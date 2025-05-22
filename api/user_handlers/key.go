package user_handlers

import (
	"net/http"
	"star-fire/internal/service"

	"github.com/gin-gonic/gin"
)

type APIKeyHandler struct {
	apiKeyService *service.APIKeyService
}

func NewAPIKeyHandler(apiKeyService *service.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{
		apiKeyService: apiKeyService,
	}
}

func (h *APIKeyHandler) CreateAPIKey(c *gin.Context) {
	// for test e.g.: curl -X POST http://localhost:8080/api/user/keys -H "Authorization: Bearer ......"  -H "Content-Type:application/json" -d "{\"name\": \"My API Key\", \"expiry_days\": 30}"
	var req service.CreateAPIKeyRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	resp, err := h.apiKeyService.CreateAPIKey(userID.(string), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *APIKeyHandler) GetAPIKeys(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	keys := h.apiKeyService.GetUserKeys(userID.(string))

	c.JSON(http.StatusOK, gin.H{
		"keys": keys,
	})
}

func (h *APIKeyHandler) RevokeAPIKey(c *gin.Context) {
	keyID := c.Param("id")
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}
	err := h.apiKeyService.RevokeAPIKey(userID.(string), keyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "API key revoked successfully",
	})
}
