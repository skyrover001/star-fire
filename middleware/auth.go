package middleware

import (
	"net/http"
	"star-fire/internal/models"
	"star-fire/internal/service"
	"star-fire/pkg/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

// JWTAuth
func JWTAuth(userDB *models.UserDB) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "需要Authorization头"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization头格式必须为Bearer <token>"})
			c.Abort()
			return
		}

		tokenString := parts[1]
		claims, err := utils.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效或过期的令牌"})
			c.Abort()
			return
		}

		user, err := userDB.GetUserByID(claims.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在或已被删除"})
			c.Abort()
			return
		}

		c.Set("user_id", user.ID)
		c.Set("username", user.Username)
		c.Set("user_role", user.Role)
		c.Set("user", user)

		c.Next()
	}
}

func APIKeyAuth(apiKeyService *service.APIKeyService, userDB *models.UserDB) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no Authorization header"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorazation header key must be: Bearer <api-key>"})
			c.Abort()
			return
		}

		apiKeyString := parts[1]
		apiKey, err := apiKeyService.ValidateAPIKey(apiKeyString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid api Key: " + err.Error()})
			c.Abort()
			return
		}

		user, err := userDB.GetUserByID(apiKey.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found or deleted"})
			c.Abort()
			return
		}
		c.Set("api_key", apiKeyString)
		c.Set("api_key_id", apiKey.ID)
		c.Set("user_id", user.ID)
		c.Set("username", user.Username)
		c.Set("user_role", user.Role)
		c.Set("user", user)
		c.Next()
	}
}

// AuthRequired
func AuthRequired(apiKeyService *service.APIKeyService, userDB *models.UserDB) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no Authorization header"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer <token>"})
			c.Abort()
			return
		}

		tokenString := parts[1]
		claims, err := utils.ValidateToken(tokenString)
		if err == nil {
			user, err := userDB.GetUserByID(claims.UserID)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found or deleted"})
				c.Abort()
				return
			}

			c.Set("user_id", user.ID)
			c.Set("username", user.Username)
			c.Set("user_role", user.Role)
			c.Set("user", user)
			c.Next()
			return
		}
		apiKey, err := apiKeyService.ValidateAPIKey(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid api Key: " + err.Error()})
			c.Abort()
			return
		}
		user, err := userDB.GetUserByID(apiKey.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found or deleted"})
			c.Abort()
			return
		}

		c.Set("api_key", tokenString)
		c.Set("api_key_id", apiKey.ID)
		c.Set("user_id", user.ID)
		c.Set("username", user.Username)
		c.Set("user_role", user.Role)
		c.Set("user", user)
		c.Next()
	}
}

// AdminRequired
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("user_role")
		if !exists || role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "need admin role"})
			c.Abort()
			return
		}
		c.Next()
	}
}
