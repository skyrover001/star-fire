package user_handlers

import (
	"github.com/gin-gonic/gin"
	"star-fire/internal/models"
)

type MarketHandler struct {
	server *models.Server
}

func NewMarketHandler(server *models.Server) *MarketHandler {
	return &MarketHandler{
		server: server,
	}
}

// ModelsHandler handles requests for available models
func (h *MarketHandler) ModelsHandler(c *gin.Context) {
	// For test: curl -X POST http://localhost:8080/api/market/models
	allModels := h.server.GetAllModels()
	c.JSON(200, allModels)
}

// get trends
func (h *MarketHandler) TrendsHandler(c *gin.Context) {
	// For test: curl -X GET http://localhost:8080/api/market/trends
	// 获取开始和结束时间
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	trends := h.server.GetTrends(startDate, endDate)
	c.JSON(200, trends)
}
