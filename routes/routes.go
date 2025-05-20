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
	authService := service.NewAuthService(server.UserStore)
	apiKeyService := service.NewAPIKeyService(server.APIKeyStore)
	tokenService := service.NewTokenService(server.TokenStore, server.UserStore)

	authHandler := user_handlers.NewAuthHandler(authService)
	apiKeyHandler := user_handlers.NewAPIKeyHandler(apiKeyService)
	tokenHandler := client_handlers.NewTokenHandler(tokenService)
	clientHandler := client_handlers.NewClientHandler(server, tokenService)

	// login route
	r.POST("/api/login", authHandler.Login)

	// client routes
	r.GET("/register/:id", clientHandler.RegisterClient)
	r.GET("/response/:fingerprint", clientHandler.ResponseClient)

	// user routes
	userAPI := r.Group("/api/user")
	userAPI.Use(middleware.JWTAuth())
	{
		userAPI.POST("/register-token", tokenHandler.GenerateRegisterToken)

		userAPI.POST("/keys", apiKeyHandler.CreateAPIKey)
		userAPI.GET("/keys", apiKeyHandler.GetAPIKeys)
		userAPI.DELETE("/keys/:id", apiKeyHandler.RevokeAPIKey)
	}

	api := r.Group("/v1")
	api.Use(middleware.AuthRequired(apiKeyService))
	{
		// chat
		api.POST("/chat/completions", func(c *gin.Context) {
			user_handlers.ChatHandler(c, server)
		})
		// models
		api.POST("/models", func(c *gin.Context) {
			user_handlers.ModelsHandler(c, server)
		})
	}

	// TODO: Admin routes
	admin := r.Group("/admin")
	admin.Use(middleware.JWTAuth(), middleware.AdminRequired())
	{
		// admin handler
	}
}
