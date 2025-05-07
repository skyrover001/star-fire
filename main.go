// 开启星火算力计划，支持个人用户的PC大模型（星火）汇入算力银河为其他需要的用户提供大模型服务，共享分成。
// server接受client的注册，同时接受用户端的达模型请求，将这些请求转发到client端。client注册采用websocket的方式，不需要client用户提供任何端口。
// 定义一个struct 包含client的注册信息，包含client的ip和端口，client的id，client的token等信息。
// server端使用一个map来存储client的注册信息，key为client的id，value为client的注册信息。
package main

import (
	"encoding/json"
	"fmt"
	"github.com/ollama/ollama/api"
	"log"
	"net/http"
	"star-fire/public"
	"strconv"
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

	Conn        *websocket.Conn        // client的websocket连接
	MessageChan chan *api.ChatResponse // 用于接收client的响应
	PongChan    chan *public.PPMessage // 用于接收client的pong消息
}

// 每个client connect都有一个keepalive的协程，只要server正常运行则定时向client发送ping消息，client收到ping消息后返回pong消息，server收到pong消息后更新client的状态。同时更新client的延迟。
func (c *Client) KeepAlive(conn *websocket.Conn) {
	// 定时向client发送ping消息
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			err := conn.WriteJSON(public.WSMessage{
				Type: public.KEEPALIVE,
				Content: public.PPMessage{
					Type:      public.PING,
					Timestamp: strconv.Itoa(int(time.Now().Unix())),
				},
			})
			if err != nil {
				log.Println("Error while writing ping message:", err)
				return
			}
		case pong := <-c.PongChan:
			if pong == nil {
				log.Println("Client pong message is nil")
				return
			}
			end := time.Now()
			timestamp, _ := strconv.ParseInt(pong.Timestamp, 10, 64)
			latency := end.Unix() - timestamp
			c.Latency = int(latency)
			log.Printf("Client latency: %d", c.Latency)
			if latency > public.MAXLATENCE {
				log.Println("Client latency is too high, closing connection")
				err := conn.Close()
				if err != nil {
					log.Println("Error while closing connection:", err)
					return
				}
			} else {
				log.Printf("Client latency is normal: %d", c.Latency)
			}
		}
	}
}

func (c *Client) HandleMessage(conn *websocket.Conn) {
	// 处理client发送的消息
	for {
		var message public.WSMessage
		err := conn.ReadJSON(&message)
		if err != nil {
			log.Println("Error while reading message:", err)
			c.Conn = nil
			return
		}
		log.Printf("recv: %s", message)
		switch message.Type {
		case public.KEEPALIVE:
			log.Println("Received keepalive message", message)
			if content, ok := message.Content.(map[string]interface{}); ok {
				pong := &public.PPMessage{
					Type:      content["type"].(string),
					Timestamp: content["timestamp"].(string),
				}
				c.PongChan <- pong
			} else {
				log.Println("Invalid message content format")
				return
			}
		case public.REGISTER:
			if content, ok := message.Content.(map[string]interface{}); ok {
				// Marshal the map back to JSON
				jsonData, err := json.Marshal(content)
				if err != nil {
					log.Println("Error marshaling content:", err)
					return
				}
				log.Println("Received register message")
				if err != nil {
					log.Println("Error marshaling content:", err)
					return
				}

				// Unmarshal JSON into the Client struct
				var client Client
				err = json.Unmarshal(jsonData, &client)
				if err != nil {
					log.Println("Error unmarshaling content into Client struct:", err)
					return
				}
				c.Id = client.Id
				c.Ip = client.Ip
				c.Port = client.Port
				c.Token = client.Token
				c.Model = client.Model
				c.Status = "online"
				c.RegisterTime = time.Now().Format("2006-01-02 15:04:05")
				c.Latency = public.MAXLATENCE
				log.Printf("Client registered: %s", c.Id)
				// 将client注册到server端
				model := public.Model{Name: client.Model}
				server.Clients[&model] = c
			} else {
				log.Println("Invalid message content format")
				return
			}
		case public.MESSAGE:
			log.Println("Received message from client:", message)
			if content, ok := message.Content.(map[string]interface{}); ok {
				// Marshal the map back to JSON
				jsonData, err := json.Marshal(content)
				if err != nil {
					log.Println("Error marshaling content:", err)
					return
				}

				// Unmarshal JSON into the api.ChatResponse struct
				var chatResponse api.ChatResponse
				err = json.Unmarshal(jsonData, &chatResponse)
				if err != nil {
					log.Println("Error unmarshaling content into ChatResponse struct:", err)
					return
				}

				c.MessageChan <- &chatResponse
			} else {
				log.Println("Invalid message content format")
				return
			}
		case public.CLOSE:
			log.Println("Client closed connection")
			return
		default:
			log.Println("Unknown message type:", message.Type)
		}
	}
}

// 定义一个http server struct 支持websocket
type Server struct {
	Clients map[*public.Model]*Client // 存储client的注册信息
	Port    string                    // server的端口
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
	id := c.Param("id")
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Error while upgrading to websocket:", err)
		return
	}
	fmt.Println("Client connected:", id)
	client := Client{
		Id:          id,
		Ip:          c.ClientIP(),
		Conn:        conn,
		PongChan:    make(chan *public.PPMessage),
		MessageChan: make(chan *api.ChatResponse),
	}
	go client.KeepAlive(conn)
	client.HandleMessage(conn)
}

func (s *Server) LoadBalance(model string) *Client {
	// 遍历所有的client，选择一个符合条件的client
	for k, v := range s.Clients {
		log.Println("Client:", k.Name, "Latency:", v.Latency, k, v, *v)
		// 判断websocket连接是否正
		if k.Name == model && v.Status == "online" && v.Latency < public.MAXLATENCE && v.Conn != nil {
			return v
		}
	}
	return nil
}

func (s *Server) Chat(c *gin.Context) {
	var request public.OpenAIRequest
	err := c.ShouldBindJSON(&request)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	client := s.LoadBalance(request.Model)
	if client == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No available client"})
		return
	}
	err = client.Conn.WriteJSON(public.WSMessage{
		Type:    public.MESSAGE,
		Content: request,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while writing json to client:" + err.Error()})
		return
	}
	response := <-client.MessageChan
	if response == nil {
		return
	}
	c.JSON(http.StatusOK, response)
}

var server *Server

func main() {
	// 创建server实例
	server = &Server{
		Clients: make(map[*public.Model]*Client),
		Port:    ":8080",
	}

	// 创建gin实例
	r := gin.Default()

	// 注册client的路由
	r.GET("/register/:id", server.RegisterClient)
	// 定义一个路由用于接收用户端openAI API格式的大模型请求
	r.POST("/v1/chat/completions", server.Chat)

	// 启动http server
	err := r.Run(server.Port)
	if err != nil {
		log.Fatal("Error while starting server:", err)
	}
}
