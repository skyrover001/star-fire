package user_handlers

import (
	"github.com/gin-gonic/gin"
	"star-fire/internal/models"
	"strconv"
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
	// For test: curl -X GET "http://localhost:8080/api/market/trends?start_date=2025-01-01&end_date=2025-01-31&page=1&size=10"
	// 获取开始和结束时间
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// 获取分页参数
	pageStr := c.Query("page")
	sizeStr := c.Query("size")

	page := 1
	size := 10

	// 解析page参数
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	// 解析size参数
	if sizeStr != "" {
		if s, err := strconv.Atoi(sizeStr); err == nil && s > 0 {
			size = s
		}
	}

	// 限制size的最大值，防止查询过多数据
	if size > 100 {
		size = 100
	}

	trendsResponse := h.server.GetTrendsWithPagination(startDate, endDate, page, size)
	c.JSON(200, trendsResponse)
}
