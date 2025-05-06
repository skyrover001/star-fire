// 开启星火算力计划，支持个人用户的PC大模型（星火）汇入算力银河为其他需要的用户提供大模型服务，共享分成。
// server接受client的注册，同时接受用户端的达模型请求，将这些请求转发到client端。client注册采用websocket的方式，不需要client用户提供任何端口。
// 定义一个struct 包含client的注册信息，包含client的ip和端口，client的id，client的token等信息。
// server端使用一个map来存储client的注册信息，key为client的id，value为client的注册信息。
package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Client struct {
	Id    string `json:"id"`    // client的id
	Ip    string `json:"ip"`    // client的ip
	Port  string `json:"port"`  // client的端口
	Token string `json:"token"` // client的token
	Model string `json:"model"` // client的模型
	// 其他信息
	// 例如client的状态，client的注册时间，延迟等
	Status       string `json:"status"`        // client的状态
	RegisterTime string `json:"register_time"` // client的注册时间
	Latency      int    `json:"latency"`       // client的延迟
}

// 每个client connect都有一个keepalive的协程，只要server正常运行则定时向client发送ping消息，client收到ping消息后返回pong消息，server收到pong消息后更新client的状态。同时更新client的延迟。
func (c *Client) KeepAlive(conn *websocket.Conn) {
	// 定时向client发送ping消息
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			start := time.Now()
			err := conn.WriteMessage(websocket.PingMessage, []byte("ping"))
			if err != nil {
				log.Println("Error while writing ping message:", err)
				return
			}
			// 等待client返回pong消息
			_, _, err = conn.ReadMessage()
			if err != nil {
				log.Println("Error while reading pong message:", err)
				return
			}
			c.Status = "online"
			end := time.Now()
			c.Latency = int(end.Sub(start).Milliseconds())
		}
	}
}

// 定义一个http server struct 支持websocket
type Server struct {
	Clients map[Client]*websocket.Conn // 存储client的注册信息
	Port    string                     // server的端口
}

// 使用gin框架来实现http server
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (s *Server) RegisterClient(c *gin.Context) {
	// 升级到websocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Error while upgrading to websocket:", err)
		return
	}
	defer conn.Close()

	// 读取client的注册信息
	var client Client
	err = conn.ReadJSON(&client)
	if err != nil {
		log.Println("Error while reading json:", err)
		return
	}

	// 将client的注册信息存储到map中
	s.Clients[client] = conn

	// 返回注册成功的信息
	err = conn.WriteJSON(gin.H{"status": "success", "message": "client registered successfully"})
	if err != nil {
		log.Println("Error while writing json:", err)
		return
	}

	log.Println("Client registered:", client)
}

// 定义一个负载均衡器，根据用户请求的模型来选择client，支持多种负载均衡算法
// 例如轮询，随机，最小连接数等
// 这里使用简单的轮询算法来选择client
func (s *Server) LoadBalance(model string) *Client {
	// 遍历所有的client，选择一个符合条件的client
	for client, conn := range s.Clients {
		if client.Model == model {
			return &client
		}
	}
	return nil
}

func (s *Server) Chat(c *gin.Context) {
	// 读取用户端的请求
	var request map[string]interface{}
	err := c.ShouldBindJSON(&request)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	// 获取请求的模型
	model, ok := request["model"].(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid model"})
		return
	}

	// 负载均衡器选择client
	client := s.LoadBalance(model)
	if client == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No available client"})
		return
	}
	// 转发请求到client端
	for client, conn := range s.Clients {
		err = conn.WriteJSON(request)
		if err != nil {
			log.Println("Error while writing json:", err)
			continue
		}
		log.Println("Request forwarded to client:", client)
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "request forwarded to client"})
}

func main() {
	// 创建server实例
	server := &Server{
		Clients: make(map[Client]*websocket.Conn),
		Port:    ":8080",
	}

	// 创建gin实例
	r := gin.Default()

	// 注册client的路由
	r.GET("/register", server.RegisterClient)
	// 定义一个路由用于接收用户端openAI API格式的大模型请求
	r.POST("/v1/chat/completions", server.Chat)

	// 启动http server
	err := r.Run(server.Port)
	if err != nil {
		log.Fatal("Error while starting server:", err)
	}
}
