package user_handlers

import (
	"net/http"
	"time"

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
	allModels := server.GetModels()
	// Anthropic 客户端（带 anthropic-version 头，或仅 x-api-key 无 Bearer）→ Anthropic 格式。
	if isAnthropicModelsRequest(c) {
		c.JSON(http.StatusOK, anthropicModelsList(allModels))
		return
	}
	c.JSON(http.StatusOK, allModels)
}

// isAnthropicModelsRequest 判断 /v1/models 请求是否来自 Anthropic SDK。
// 官方 Anthropic SDK 会携带 anthropic-version 头（以及 x-api-key）；OpenAI SDK
// 使用 Authorization: Bearer。据此区分返回格式。
func isAnthropicModelsRequest(c *gin.Context) bool {
	if c.GetHeader("anthropic-version") != "" {
		return true
	}
	return c.GetHeader("x-api-key") != "" && c.GetHeader("Authorization") == ""
}

// anthropicModelsList 将 OpenAI 格式的模型列表转为 Anthropic 格式。
func anthropicModelsList(openaiList map[string]interface{}) map[string]interface{} {
	data, _ := openaiList["data"].([]map[string]interface{})
	out := make([]map[string]interface{}, 0, len(data))
	var firstID, lastID string
	for i, m := range data {
		id, _ := m["id"].(string)
		out = append(out, map[string]interface{}{
			"type":         "model",
			"id":           id,
			"display_name": id,
			"created_at":   time.Now().UTC().Format(time.RFC3339),
		})
		if i == 0 {
			firstID = id
		}
		lastID = id
	}
	return map[string]interface{}{
		"data":     out,
		"has_more": false,
		"first_id": firstID,
		"last_id":  lastID,
	}
}
