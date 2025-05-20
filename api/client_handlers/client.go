package client_handlers

import (
	"fmt"
	"log"
	"star-fire/internal/models"
	"star-fire/internal/service"
	"star-fire/internal/websocket"

	"github.com/gin-gonic/gin"
)

//// register client
//func RegisterClient(c *gin.Context, server *models.Server) {
//	// 获取客户端设备关联的唯一ID，是否合理呢，如果用户换了设备呢
//	id := c.Param("id")
//	conn, err := websocket.Upgrade(c.Writer, c.Request)
//	if err != nil {
//		log.Println("Error while upgrading to websocket:", err)
//		return
//	}
//
//	log.Println("Client connected:", id)
//	client := models.NewClient(id, c.ClientIP(), conn)
//	service.HandleClientConnection(client, server)
//}
//
//// chat response websocket, 每次chat请求都会创建一个新的连接
//func ResponseClient(c *gin.Context, server *models.Server) {
//	fingerPrint := c.Param("fingerprint")
//
//	conn, err := websocket.Upgrade(c.Writer, c.Request)
//	if err != nil {
//		log.Println("Error while upgrading to websocket:", err)
//		return
//	}
//
//	server.RespClients[fingerPrint] = conn
//}

// 添加token服务
type ClientHandler struct {
	server       *models.Server
	tokenService *service.TokenService
}

// NewClientHandler 创建一个新的客户端处理器
func NewClientHandler(server *models.Server, tokenService *service.TokenService) *ClientHandler {
	return &ClientHandler{
		server:       server,
		tokenService: tokenService,
	}
}

// RegisterClient 处理客户端注册请求
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
	// 处理客户端连接
	service.HandleClientConnection(client, h.server)
}

// ResponseClient 处理客户端响应连接请求
func (h *ClientHandler) ResponseClient(c *gin.Context) {
	fingerPrint := c.Param("fingerprint")

	// 升级连接到WebSocket
	conn, err := websocket.Upgrade(c.Writer, c.Request)
	if err != nil {
		log.Println("Error while upgrading to websocket:", err)
		return
	}

	// 存储响应连接
	h.server.RespClients[fingerPrint] = conn
}
