package user_handlers

import (
	"net/http"
	"star-fire/internal/models"

	"github.com/gin-gonic/gin"
)

type PriceCapHandler struct {
	db *models.UserPriceCapDB
}

func NewPriceCapHandler(db *models.UserPriceCapDB) *PriceCapHandler {
	return &PriceCapHandler{db: db}
}

// ListPriceCaps returns all price caps configured for the current user.
// GET /api/user/price-caps
func (h *PriceCapHandler) ListPriceCaps(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	caps, err := h.db.GetByUser(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch price caps: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"price_caps": caps})
}

// UpsertPriceCap creates or updates the price cap for a specific model.
// PUT /api/user/price-caps/:model
func (h *PriceCapHandler) UpsertPriceCap(c *gin.Context) {
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
		MaxIPPM float64 `json:"max_ippm" binding:"required,min=0"`
		MaxOPPM float64 `json:"max_oppm" binding:"required,min=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	cap, err := h.db.Upsert(userID.(string), model, req.MaxIPPM, req.MaxOPPM)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save price cap: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, cap)
}

// DeletePriceCap removes the price cap for a specific model (restores unlimited).
// DELETE /api/user/price-caps/:model
func (h *PriceCapHandler) DeletePriceCap(c *gin.Context) {
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

	if err := h.db.Delete(userID.(string), model); err != nil {
		if err.Error() == "price cap not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete price cap: " + err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
