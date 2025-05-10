// 开启星火算力计划，支持个人用户的PC大模型（星火）汇入算力银河为其他需要的用户提供大模型服务，共享分成。
// server东侧接受client的注册，西侧接受用户端的大模型请求，通过分配算法将这些请求转发到client端。
// client注册和问答都是通过客户端websocket的方式，不需要西侧client用户提供任何互联网入口。
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"star-fire/public"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/ollama/ollama/api"
	"github.com/sashabaranov/go-openai"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Client struct {
	ID     string          `json:"id"`
	IP     string          `json:"ip"`
	Token  string          `json:"token"`
	Models []*public.Model `json:"models"`

	Status       string `json:"status"`
	RegisterTime string `json:"register_time"`
	Latency      int    `json:"latency"`

	ControlConn *websocket.Conn
	MessageChan chan *api.ChatResponse
	PongChan    chan *public.PPMessage
	ErrChan     chan error
}

// keep aliving client control channel
func (c *Client) KeepAlive(conn *websocket.Conn) {
	ticker := time.NewTicker(public.KEEPALIVE_TIME * time.Second)
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
			log.Println("response connection is:", len(server.RespClients))
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
				// update available models
				availableModels := make([]*public.Model, 0)
				for _, model := range c.Models {
					if model.Name == pong.AvaliableModels[0].Name {
						availableModels = append(availableModels, model)
					}
				}
				c.Models = availableModels
				log.Printf("Client available models: %v", c.Models)
			}
		}
	}
}

func parseJSON(input interface{}, output interface{}) error {
	data, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}
	err = json.Unmarshal(data, output)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %v", err)
	}
	return nil
}

// handle control channel message
func (c *Client) HandleMessage() {
	for {
		var message public.WSMessage
		err := c.ControlConn.ReadJSON(&message)
		if err != nil {
			log.Println("Error while reading message:", err)
			c.ControlConn = nil
			return
		}
		log.Printf("recv: %s", message)
		switch message.Type {
		case public.KEEPALIVE:
			log.Println("Received keepalive message", message)
			if content, ok := message.Content.(map[string]interface{}); ok {
				var pong public.PPMessage
				if err := parseJSON(content, &pong); err != nil {
					log.Println("Error mapping content to PPMessage struct:", err)
					return
				}
				log.Println("Pong message:", pong)
				timestamp, err := strconv.ParseInt(pong.Timestamp, 10, 64)
				if err != nil {
					log.Println("Error parsing pong.Timestamp:", err)
					return
				}
				c.Latency = int(time.Now().Unix() - timestamp)
				log.Printf("Client latency: %d", c.Latency)
				if c.Latency > public.MAXLATENCE {
					log.Println("Client latency is too high, closing connection")
					err := c.ControlConn.Close()
					if err != nil {
						log.Println("Error while closing connection:", err)
						return
					}
					c.Status = "offline"
					log.Printf("Client %s is offline", c.ID)
					return
				}
				pong.Timestamp = strconv.Itoa(int(time.Now().Unix()))
				pong.Type = public.PONG
				log.Println("Sending pong message:", pong)
				c.PongChan <- &pong
			} else {
				log.Println("Invalid message content format")
				return
			}
		case public.REGISTER:
			if content, ok := message.Content.(map[string]interface{}); ok {
				var client Client
				if err := parseJSON(content, &client); err != nil {
					log.Println("Error mapping content to Client struct:", err)
					return
				}

				c.ID = client.ID
				c.IP = client.ID
				c.Token = client.Token
				c.Models = client.Models
				c.Status = "online"
				c.RegisterTime = time.Now().Format("2006-01-02 15:04:05")
				c.Latency = public.MAXLATENCE
				log.Printf("Client registered: %s", c.ID)
				for _, m := range c.Models {
					fmt.Println("model:", m)
					model := public.Model{
						Name: m.Name,
						Type: m.Type,
						Size: m.Size,
						Arch: m.Arch,
					}
					server.Clients[&model] = c
				}
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
				fmt.Println("chatResponse:", chatResponse)
				c.MessageChan <- &chatResponse
			} else {
				log.Println("Invalid message content format")
				return
			}
		case public.MODEL_ERROR:
			log.Println("Received model error message:", message)
			c.ErrChan <- fmt.Errorf("model error: %s", message.Content.(string))
		case public.CLOSE:
			log.Println("Client closed connection")
			return
		default:
			log.Println("Unknown message type:", message.Type)
		}
	}
}

type Server struct {
	Clients     map[*public.Model]*Client
	Port        string
	RespClients map[string]*websocket.Conn
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024 * 1024,
	WriteBufferSize: 1024 * 1024,
}

// register client to server
func (s *Server) Register(c *gin.Context) {
	id := c.Param("id")
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Error while upgrading to websocket:", err)
		return
	}
	log.Println("Client connected:", id)
	// init client
	client := Client{
		ID:          id,
		IP:          c.ClientIP(),
		ControlConn: conn,
		PongChan:    make(chan *public.PPMessage),
		MessageChan: make(chan *api.ChatResponse),
		ErrChan:     make(chan error),
	}
	// ping pong to keep alive
	go client.KeepAlive(conn)
	// handling message
	client.HandleMessage()
}

// chat channel for response
func (s *Server) Response(c *gin.Context) {
	fingerPrint := c.Param("fingerprint")
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Error while upgrading to websocket:", err)
		return
	}
	s.RespClients[fingerPrint] = conn
}

// load balance to get client for user chat request
func (s *Server) LoadBalance(model string) *Client {
	// 遍历所有的client，选择一个符合条件的client
	for k, v := range s.Clients {
		log.Println("Client:", k.Name, "Latency:", v.Latency, k, v, *v)
		// 判断websocket连接是否正
		if k.Name == model && v.Status == "online" && v.Latency < public.MAXLATENCE && v.ControlConn != nil {
			return v
		}
	}
	return nil
}

// chat request
func (s *Server) Chat(c *gin.Context) {
	var request public.OpenAIRequest
	fingerPrint := uuid.NewString()
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
	err = client.ControlConn.WriteJSON(public.WSMessage{
		Type:        public.MESSAGE,
		Content:     request,
		FingerPrint: fingerPrint,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while writing json to client:" + err.Error()})
		return
	}
	// set timeout for request
	waitStart := time.Now()
	if request.Stream {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
	}
	for {
		if s.RespClients[fingerPrint] == nil {
			time.Sleep(1 * time.Millisecond)
			continue
		}
		var response public.WSMessage
		err := s.RespClients[fingerPrint].ReadJSON(&response)
		if err != nil {
			log.Println("Error while reading json from client:", err)
			return
		}
		fmt.Println("response:", response)
		if response.Type == public.MESSAGE {
			if content, ok := response.Content.(map[string]interface{}); ok {
				jsonData, err := json.Marshal(content)
				if err != nil {
					log.Println("Error marshaling content:", err)
					delete(s.RespClients, fingerPrint)
					return
				}
				var chatResponse openai.ChatCompletionResponse
				err = json.Unmarshal(jsonData, &chatResponse)
				if err != nil {
					log.Println("Error unmarshaling content into ChatResponse struct:", err)
					delete(s.RespClients, fingerPrint)
					return
				}
				c.JSON(http.StatusOK, content)
			} else {
				log.Println("Invalid message content format")
				delete(s.RespClients, fingerPrint)
				return
			}
		} else if response.Type == public.CLOSE {
			log.Println("Client closed connection")
			s.RespClients[fingerPrint].Close()
			delete(s.RespClients, fingerPrint)
			return
		} else if response.Type == public.MODEL_ERROR {
			log.Println("Model error:", response.Content)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Model error: " + response.Content.(string)})
			s.RespClients[fingerPrint].Close()
			delete(s.RespClients, fingerPrint)
			return
		} else if response.Type == public.MESSAGE_STREAM {
			if content, ok := response.Content.(map[string]interface{}); ok {
				jsonData, err := json.Marshal(content)
				if err != nil {
					log.Println("Error marshaling content:", err)
					delete(s.RespClients, fingerPrint)
					return
				}
				var chatResponse openai.ChatCompletionStreamResponse
				err = json.Unmarshal(jsonData, &chatResponse)
				if err != nil {
					log.Println("Error unmarshaling content into ChatResponse struct:", err)
					delete(s.RespClients, fingerPrint)
					return
				}
				_, err = c.Writer.Write([]byte("data: " + string(jsonData) + "\n\n"))
				if err != nil {
					log.Println("Error while writing response:", err)
					delete(s.RespClients, fingerPrint)
					return
				}
				c.Writer.Flush()
				if chatResponse.Choices[0].FinishReason == "stop" {
					_, err = c.Writer.Write([]byte("data:[DONE]\n\n"))
					s.RespClients[fingerPrint].Close()
					delete(s.RespClients, fingerPrint)
					return
				}
			} else {
				log.Println("Invalid message content format")
				delete(s.RespClients, fingerPrint)
				return
			}
		} else {
			log.Println("Unknown message type:", response.Type)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Unknown message type: " + response.Type})
			s.RespClients[fingerPrint].Close()
			delete(s.RespClients, fingerPrint)
			return
		}
		if time.Since(waitStart) > public.CHAT_MAX_TIME*time.Second {
			log.Println("Chat timeout")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Chat timeout"})
			s.RespClients[fingerPrint].Close()
			delete(s.RespClients, fingerPrint)
			return
		}
	}
}

var server *Server

func main() {
	server = &Server{
		Clients:     make(map[*public.Model]*Client),
		Port:        ":8080",
		RespClients: make(map[string]*websocket.Conn),
	}
	r := gin.Default()
	r.GET("/register/:id", server.Register)
	r.GET("/response/:fingerprint", server.Response)
	r.POST("/v1/chat/completions", server.Chat)
	err := r.Run(server.Port)
	if err != nil {
		log.Fatal("Error while starting server:", err)
	}
}
