package user_handlers

import (
	"net/http"
	"star-fire/internal/models"

	"github.com/gin-gonic/gin"
)

type ModelPriceHandler struct {
	server *models.Server
}

func NewModelPriceHandler(server *models.Server) *ModelPriceHandler {
	return &ModelPriceHandler{server: server}
}

// ListMyModels returns all models provided by the current user's connected clients.
// GET /api/user/my-models
func (h *ModelPriceHandler) ListMyModels(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	result := h.server.GetUserModels(userID.(string))
	c.JSON(http.StatusOK, gin.H{"models": result})
}

// UpdateModelPrice updates the price for a model across all of the user's clients.
// PUT /api/user/model-price/:model
func (h *ModelPriceHandler) UpdateModelPrice(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	model := c.Param("model")
	if model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model name is required"})
		return
	}

	var req struct {
		IPPM  float64 `json:"ippm" binding:"min=0"`
		OPPM  float64 `json:"oppm" binding:"min=0"`
		CIPPM float64 `json:"cippm" binding:"min=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	count, err := h.server.UpdateModelPrice(userID.(string), model, req.IPPM, req.OPPM, req.CIPPM)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "price updated",
		"updated_clients": count,
	})
}
