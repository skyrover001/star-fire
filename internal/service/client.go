package service

import (
	"encoding/json"
	"fmt"
	"github.com/ollama/ollama/api"
	"log"
	"star-fire/internal/models"
	"star-fire/pkg/public"
	"star-fire/pkg/utils"
	"strconv"
	"time"
)

func HandleClientConnection(client *models.Client, server *models.Server) {
	// if client is registered then keep the connection alive
	go keepAliveClient(client)

	// 处理客户端消息
	handleClientMessages(client, server)
}

func keepAliveClient(client *models.Client) {
	ticker := time.NewTicker(public.KEEPALIVE_TIME * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := client.ControlConn.WriteJSON(public.WSMessage{
				Type: public.KEEPALIVE,
				Content: public.PPMessage{
					Type:      public.PING,
					Timestamp: strconv.Itoa(int(time.Now().Unix())),
				},
			})
			if err != nil {
				log.Println("Error while writing ping message:", err)
				client.Status = "offline"
				return
			}

		case pong := <-client.PongChan:
			if pong == nil {
				log.Println("Client pong message is nil")
				client.Status = "offline"
				return
			}

			end := time.Now()
			timestamp, _ := strconv.ParseInt(pong.Timestamp, 10, 64)
			latency := end.Unix() - timestamp
			client.Latency = int(latency)

			if latency > public.MAXLATENCE {
				log.Println("Client latency is too high, closing connection")
				client.ControlConn.Close()
				client.Status = "offline"
				return
			} else {
				client.Models = pong.AvaliableModels
				client.Status = "online"
			}
		}
	}
}

// handle client messages
func handleClientMessages(client *models.Client, server *models.Server) {
	for {
		var message public.WSMessage
		err := client.ControlConn.ReadJSON(&message)
		if err != nil {
			log.Println("Error while reading message:", err)
			client.Status = "offline"
			client.ControlConn = nil
			return
		}

		switch message.Type {
		case public.KEEPALIVE:
			handleKeepAliveMessage(client, message)

		case public.REGISTER:
			handleRegisterMessage(client, server, message)

		case public.MESSAGE:
			handleChatMessage(client, message)

		case public.MODEL_ERROR:
			if content, ok := message.Content.(string); ok {
				client.ErrChan <- fmt.Errorf("model error: %s", content)
			}

		case public.CLOSE:
			log.Println("Client closed connection")
			client.Status = "offline"
			return

		default:
			log.Println("Unknown message type:", message.Type)
		}
	}
}

// keep alive message and update client info
func handleKeepAliveMessage(client *models.Client, message public.WSMessage) {
	if content, ok := message.Content.(map[string]interface{}); ok {
		var pong public.PPMessage
		if err := utils.ParseJSON(content, &pong); err != nil {
			log.Println("Error mapping content to PPMessage struct:", err)
			return
		}

		timestamp, err := strconv.ParseInt(pong.Timestamp, 10, 64)
		if err != nil {
			log.Println("Error parsing pong.Timestamp:", err)
			return
		}

		client.Latency = int(time.Now().Unix() - timestamp)
		if client.Latency > public.MAXLATENCE {
			log.Println("Client latency is too high, closing connection")
			client.ControlConn.Close()
			client.Status = "offline"
			return
		}

		pong.Timestamp = strconv.Itoa(int(time.Now().Unix()))
		pong.Type = public.PONG
		client.PongChan <- &pong
	}
}

// handle client register message
func handleRegisterMessage(client *models.Client, server *models.Server, message public.WSMessage) {
	if content, ok := message.Content.(map[string]interface{}); ok {
		var registerInfo models.Client
		if err := utils.ParseJSON(content, &registerInfo); err != nil {
			log.Println("Error mapping content to Client struct:", err)
			return
		}

		client.ID = registerInfo.ID
		client.IP = registerInfo.IP
		client.Token = registerInfo.Token
		client.Models = registerInfo.Models
		client.Status = "online"
		client.RegisterTime = time.Now().Format("2006-01-02 15:04:05")

		// 注册客户端的所有模型
		for _, m := range client.Models {
			model := public.Model{
				Name: m.Name,
				Type: m.Type,
				Size: m.Size,
				Arch: m.Arch,
			}
			server.RegisterModel(&model, client)
		}
	}
}

// handle client chat message
func handleChatMessage(client *models.Client, message public.WSMessage) {
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

		client.MessageChan <- &chatResponse
	}
}
