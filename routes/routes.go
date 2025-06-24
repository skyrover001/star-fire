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

	marketHandler := user_handlers.NewMarketHandler(server)

	// 登录和注册路由
	r.POST("/api/login", authHandler.Login)
	r.POST("/api/send-code", func(c *gin.Context) {
		user_handlers.SendVerificationCode(c)
	})
	r.POST("/api/register", func(c *gin.Context) {
		user_handlers.Register(c, server)
	})

	// 客户端路由
	r.GET("/register/:id", clientHandler.RegisterClient)
	r.GET("/response/:fingerprint", clientHandler.ResponseClient)

	marketAPI := r.Group("/api/market")
	marketAPI.Use(middleware.JWTAuth(server.UserDB))
	{
		marketAPI.GET("/models", marketHandler.ModelsHandler)
		marketAPI.GET("/trends", marketHandler.TrendsHandler)
		// marketAPI.POST("/messages", apiKeyHandler.CreateAPIKey)
	}

	// 用户路由
	userAPI := r.Group("/api/user")
	userAPI.Use(middleware.JWTAuth(server.UserDB))
	{
		userAPI.POST("/register-token", clientHandler.GenerateRegisterToken)

		userAPI.POST("/keys", apiKeyHandler.CreateAPIKey)
		userAPI.GET("/keys", apiKeyHandler.GetAPIKeys)
		userAPI.PUT("/keys/:id", apiKeyHandler.RevokeAPIKey)
		userAPI.DELETE("/keys/:id", apiKeyHandler.DeleteAPIKey)

		userAPI.GET("/token-usage", tokenUsageHandler.GetUserTokenUsage)
		userAPI.GET("/income", tokenUsageHandler.GetUserIncome)
	}

	api := r.Group("/v1")
	api.Use(middleware.AuthRequired(apiKeyService, server.UserDB))
	{
		// 聊天
		api.POST("/chat/completions", func(c *gin.Context) {
			service.HandleChatRequest(c, server)
		})
		// 模型
		api.GET("/models", func(c *gin.Context) {
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
