package routes

import (
	client_handlers "star-fire/api/client_handlers"
	user_handlers "star-fire/api/user_handlers"
	"star-fire/internal/models"
	"star-fire/internal/service"
	"star-fire/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, server *models.Server) {
	authService := service.NewAuthService(server.UserDB)
	apiKeyService := service.NewAPIKeyService(server.APIKeyDB)
	registerTokenService := service.NewRegisterTokenService(server.RegisterTokenStore, server.UserDB)

	authHandler := user_handlers.NewAuthHandler(authService)
	apiKeyHandler := user_handlers.NewAPIKeyHandler(apiKeyService)

	clientHandler := client_handlers.NewClientHandler(server, registerTokenService)
	tokenUsageHandler := user_handlers.NewTokenUsageHandler(server)

	// 登录路由
	r.POST("/api/login", authHandler.Login)

	// 客户端路由
	r.GET("/register/:id", clientHandler.RegisterClient)
	r.GET("/response/:fingerprint", clientHandler.ResponseClient)

	// 用户路由
	userAPI := r.Group("/api/user")
	userAPI.Use(middleware.JWTAuth(server.UserDB))
	{
		userAPI.POST("/register-token", clientHandler.GenerateRegisterToken)

		userAPI.POST("/keys", apiKeyHandler.CreateAPIKey)
		userAPI.GET("/keys", apiKeyHandler.GetAPIKeys)
		userAPI.DELETE("/keys/:id", apiKeyHandler.RevokeAPIKey)

		userAPI.GET("/token-usage", tokenUsageHandler.GetUserTokenUsage)
	}

	api := r.Group("/v1")
	api.Use(middleware.AuthRequired(apiKeyService, server.UserDB))
	{
		// 聊天
		api.POST("/chat/completions", func(c *gin.Context) {
			service.HandleChatRequest(c, server)
		})
		// 模型
		api.POST("/models", func(c *gin.Context) {
			user_handlers.ModelsHandler(c, server)
		})
	}

	// 管理员路由
	admin := r.Group("/admin")
	admin.Use(middleware.JWTAuth(server.UserDB), middleware.AdminRequired())
	{
		// 管理员处理器
	}
}
