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
	priceCapHandler := user_handlers.NewPriceCapHandler(server.UserPriceCapDB)
	modelPriceHandler := user_handlers.NewModelPriceHandler(server)

	clientHandler := client_handlers.NewClientHandler(server, registerTokenService)
	tokenUsageHandler := user_handlers.NewTokenUsageHandler(server)

	marketHandler := user_handlers.NewMarketHandler(server)
	userHandler := user_handlers.NewUserHandler(server)
	balanceHandler := user_handlers.NewBalanceHandler(server)

	// 登录和注册路由
	r.POST("/api/login", authHandler.Login)
	r.POST("/api/send-code", func(c *gin.Context) {
		userHandler.SendVerificationCode(c)
	})
	r.POST("/api/register", func(c *gin.Context) {
		userHandler.Register(c, server)
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

		// Balance and recharge
		userAPI.GET("/balance", balanceHandler.GetBalance)
		userAPI.POST("/recharge", balanceHandler.CreateRechargeOrder)
		userAPI.POST("/recharge/confirm", balanceHandler.ConfirmRecharge)
		userAPI.GET("/recharge/history", balanceHandler.GetRechargeHistory)

		// Price cap configuration: userID is taken from JWT, not from the request body.
		userAPI.GET("/price-caps", priceCapHandler.ListPriceCaps)
		userAPI.PUT("/price-caps/:model", priceCapHandler.UpsertPriceCap)
		userAPI.DELETE("/price-caps/:model", priceCapHandler.DeletePriceCap)

		// Model price management: set prices for your own provided models.
		userAPI.GET("/my-models", modelPriceHandler.ListMyModels)
		userAPI.PUT("/model-price/:model", modelPriceHandler.UpdateModelPrice)
	}

	api := r.Group("/v1")
	api.Use(middleware.AuthRequired(apiKeyService, server.UserDB))
	{
		// 聊天
		api.POST("/chat/completions", func(c *gin.Context) {
			service.HandleChatRequest(c, server)
		})
		// Embedding
		api.POST("/embeddings", func(c *gin.Context) {
			service.HandleEmbeddingRequest(c, server)
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
