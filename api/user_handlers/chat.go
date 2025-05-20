package user_handlers

import (
	"net/http"
	"star-fire/internal/models"
	"star-fire/internal/service"

	"github.com/gin-gonic/gin"
)

// chat handler
func ChatHandler(c *gin.Context, server *models.Server) {
	// for test e.g. : curl -X POST http://localhost:8080/v1/chat/completions -H "Content-Type: application/json" -H "Authorization: Bearer sk-dAr989DwY+YOxQjOdUJiIicHWEAvasoFlPHkflNF4Nw=" -d "{\"model\":\"qwen3:0.6b\",\"messages\": [{\"role\": \"user\", \"content\": \"请给我一个关于健康饮食的建议\"}]}"
	service.HandleChatRequest(c, server)
}

// model handler
func ModelsHandler(c *gin.Context, server *models.Server) {
	// for test e.g. : curl -X POST http://localhost:8080/v1/models -H "Authorization: Bearer sk-dAr989DwY+YOxQjOdUJiIicHWEAvasoFlPHkflNF4Nw="
	allModels := server.GetAllModels()
	c.JSON(http.StatusOK, allModels)
}
