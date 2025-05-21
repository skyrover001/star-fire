package client_handlers

import (
	"fmt"
	"log"
	"star-fire/internal/models"
	"star-fire/internal/service"
	"star-fire/internal/websocket"

	"github.com/gin-gonic/gin"
)

type ClientHandler struct {
	server       *models.Server
	tokenService *service.TokenService
}

// NewClientHandler
func NewClientHandler(server *models.Server, tokenService *service.TokenService) *ClientHandler {
	return &ClientHandler{
		server:       server,
		tokenService: tokenService,
	}
}

// RegisterClient
func (h *ClientHandler) RegisterClient(c *gin.Context) {
	tokenString := c.GetHeader("X-Registration-Token")
	log.Printf("Token from header: %s", tokenString)
	fmt.Println("RegisterClient")
	id := c.Param("id")
	fmt.Println("id:", id, " token:", tokenString)

	if tokenString == "" {
		c.JSON(400, gin.H{"error": "Registration token is required"})
		return
	}
	user, err := h.tokenService.ValidateRegisterToken(tokenString)
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
