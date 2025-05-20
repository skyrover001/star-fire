package middleware

import (
	"net/http"
	"star-fire/internal/service"
	"strings"

	"github.com/gin-gonic/gin"
)

func APIKeyAuth(apiKeyService *service.APIKeyService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}
		// get api key from header e.g.：Bearer sk-xxxxx or Bearer xxxxx
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer <api_key>"})
			c.Abort()
			return
		}

		apiKey := parts[1]
		// validate the API key
		key, err := apiKeyService.ValidateAPIKey(apiKey)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key: " + err.Error()})
			c.Abort()
			return
		}

		// set user_id in context
		c.Set("user_id", key.UserID)
		c.Next()
	}
}

func AuthRequired(apiKeyService *service.APIKeyService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		// get api key
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer <token>"})
			c.Abort()
			return
		}

		tokenType := parts[0]
		token := parts[1]

		// 根据不同的认证类型选择不同的处理方式
		if tokenType == "Bearer" {
			// 检查是否是JWT格式
			if strings.Count(token, ".") == 2 {
				// 调用JWT验证
				jwtAuth := JWTAuth()
				jwtAuth(c)
			} else {
				// 调用API Key验证
				apiKeyAuth := APIKeyAuth(apiKeyService)
				apiKeyAuth(c)
			}
			// 检查是否已经中止
			if c.IsAborted() {
				return
			}
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unsupported authorization type"})
			c.Abort()
			return
		}

		c.Next()
	}
}
