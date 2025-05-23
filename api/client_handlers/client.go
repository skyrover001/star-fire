package client_handlers

import (
	"fmt"
	"log"
	"net/http"
	"star-fire/internal/models"
	"star-fire/internal/service"
	"star-fire/internal/websocket"

	"github.com/gin-gonic/gin"
)

type ClientHandler struct {
	server               *models.Server
	registerTokenService *service.RegisterTokenService
}

// NewClientHandler
func NewClientHandler(server *models.Server, registerTokenService *service.RegisterTokenService) *ClientHandler {
	return &ClientHandler{
		server:               server,
		registerTokenService: registerTokenService,
	}
}

// RegisterClient
func (h *ClientHandler) RegisterClient(c *gin.Context) {
	tokenString := c.GetHeader("X-Registration-Token")
	log.Printf("Token from server: %s", tokenString)
	id := c.Param("id")

	if tokenString == "" {
		c.JSON(400, gin.H{"error": "Registration token is required"})
		return
	}
	user, err := h.registerTokenService.ValidateRegisterToken(tokenString)
	fmt.Println("user:", user, "err:", err, " id:", id, " token:", tokenString)
	if err != nil {
		c.JSON(401, gin.H{"error": "Invalid registration token: " + err.Error()})
		return
	}
	conn, err := websocket.Upgrade(c.Writer, c.Request)
	if err != nil {
		log.Println("Error while upgrading to websocket:", err)
		return
	}
	log.Printf("Client connected: %s for user: %s (%s)", id, user.Username, user.Email)
	client := models.NewClient(id, c.ClientIP(), conn)

	client.SetUser(user)
	if err := h.server.ClientDB.SaveClient(client); err != nil {
		log.Printf("save client to database failed: %v", err)
	}
	service.HandleClientConnection(client, h.server)
}

// ResponseClient
func (h *ClientHandler) ResponseClient(c *gin.Context) {
	fingerPrint := c.Param("fingerprint")
	conn, err := websocket.Upgrade(c.Writer, c.Request)
	if err != nil {
		log.Println("Error while upgrading to websocket:", err)
		return
	}
	h.server.RespClients[fingerPrint] = conn
}

func (h *ClientHandler) GenerateRegisterToken(c *gin.Context) {
	// for text e.g. :curl -X POST http://localhost:8080/api/user/register-token -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}
	resp, err := h.registerTokenService.GenerateRegisterToken(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}
