package service

import (
	"encoding/json"
	"fmt"
	"log"
	"star-fire/internal/models"
	"star-fire/pkg/public"
	"star-fire/pkg/utils"
	"strconv"
	"time"

	"github.com/ollama/ollama/api"
)

func HandleClientConnection(client *models.Client, server *models.Server) {
	// if client is registered then keep the connection alive
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("keep alive goroutine recovered from panic: %v", r)
				client.Status = "offline"
			}
		}()
		keepAliveClient(client, server)
	}()
	handleClientMessages(client, server)

	// 连接断开，主动清理该 client 注册的所有模型
	for _, m := range client.Models {
		server.RemoveClient(m.Name, client.ID)
	}
}

func keepAliveClient(client *models.Client, server *models.Server) {
	ticker := time.NewTicker(public.KEEPALIVE_TIME * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 如果客户端连接断开，则关闭连接
			client.ControlConnMutex.Lock()
			if client.ControlConn == nil {
				client.ControlConnMutex.Unlock()
				log.Println("Client control connection is nil, closing connection")
				client.Status = "offline"
				return
			}
			err := client.ControlConn.WriteJSON(public.WSMessage{
				Type: public.KEEPALIVE,
				Content: public.PPMessage{
					Type:      public.PING,
					Timestamp: strconv.Itoa(int(time.Now().Unix())),
				},
			})
			client.ControlConnMutex.Unlock()
			if err != nil {
				log.Println("Error while writing ping message:", err)
				client.Status = "offline"
				return
			}

		case pong := <-client.PongChan:
			fmt.Println("<UNK>:", pong)
			if pong == nil {
				log.Println("Client pong message is nil")
				client.Status = "offline"
				return
			}

			end := time.Now()
			timestamp, _ := strconv.ParseInt(pong.Timestamp, 10, 64)
			latency := end.Unix() - timestamp
			client.Latency = int(latency)
			fmt.Println("Client latency:", client.Latency)

			if latency > public.MAXLATENCE {
				log.Println("Client latency is too high, closing connection")
				client.ControlConn.Close()
				client.Status = "offline"
				return
			} else {
				client.Models = pong.AvailableModels
				var trends []*models.Trend
				for _, m := range client.Models {
					if m.OPPM > server.Conf.AllModelOutPutMaxPrice {
						log.Println("OPPM price is too high, set starfire platform default value!")
						m.OPPM = server.Conf.AllModelOutPutMaxPrice
					}
					if m.IPPM > server.Conf.AllModelInputMaxPrice {
						log.Println("IPPM price is too high, set starfire platform default value!")
						m.IPPM = server.Conf.AllModelInputMaxPrice
					}
					server.RegisterModel(m, client)
					fmt.Println("Client available model:", m.Name, m)
					// add trend for client keep alive
					trends = append(trends, &models.Trend{
						Name:        fmt.Sprintf("%s_%s", client.User.Username, "keep alive model: "+m.Name),
						Description: "用户 " + client.User.Username + " 保持模型: " + m.Name + " 在线",
						CreatedAt:   time.Now().Format("2006-01-02 15:04:05"),
						UpdatedAt:   "",
						DeletedAt:   "",
						Active:      true,
						User:        client.User,
					})
				}
				// batch save all trends in a single transaction
				if err := server.TrendDB.SaveTrends(trends); err != nil {
					log.Println("Error saving trends:", err)
				} else {
					log.Printf("Trends saved successfully: %d records", len(trends))
				}
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
			client.ControlConnMutex.Lock()
			client.ControlConn = nil
			client.ControlConnMutex.Unlock()
			client.Status = "offline"
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
	fmt.Println("Registering client:", message)
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
		client.RegisterTime = time.Now()

		var trends []*models.Trend
		for _, m := range client.Models {
			fmt.Println("Registering model:", m.Name, "Type:", m.Type, "IPPM:", m.IPPM, "OPPM:", m.OPPM, server.Conf.AllModelOutPutMaxPrice, server.Conf.AllModelInputMaxPrice)
			if m.OPPM > server.Conf.AllModelOutPutMaxPrice {
				log.Println("OPPM price is too high, set starfire platform default value!")
				m.OPPM = server.Conf.AllModelOutPutMaxPrice
			}
			if m.IPPM > server.Conf.AllModelInputMaxPrice {
				log.Println("IPPM price is too high, set starfire platform default value!")
				m.IPPM = server.Conf.AllModelInputMaxPrice
			}
			model := public.Model{
				Name:  m.Name,
				Type:  m.Type,
				Size:  m.Size,
				Arch:  m.Arch,
				IPPM:  m.IPPM,  // 确保传递IPPM价格
				OPPM:  m.OPPM,  // 确保传递OPPM价格
				CIPPM: m.CIPPM, // 确保传递CIPPM价格
			}
			fmt.Println("model is", model, m.OPPM, m.IPPM)
			server.RegisterModel(&model, client)

			// 为embedding模型添加特殊的trend记录
			var description string
			if m.Type == "embedding" || server.IsEmbeddingModel(m.Name) {
				description = fmt.Sprintf("用户 %s 贡献了embedding模型: %s", client.User.Username, m.Name)
			} else {
				description = fmt.Sprintf("用户 %s 贡献了模型: %s", client.User.Username, m.Name)
			}

			trends = append(trends, &models.Trend{
				Name:        fmt.Sprintf("%s_%s", client.User.Username, "register model: "+m.Name),
				Description: description,
				CreatedAt:   time.Now().Format("2006-01-02 15:04:05"),
				UpdatedAt:   "",
				DeletedAt:   "",
				Active:      true,
				User:        client.User,
				Client:      client,
			})
		}
		// batch save all registration trends in a single transaction
		if err := server.TrendDB.SaveTrends(trends); err != nil {
			log.Println("Error saving trends:", err)
		} else {
			log.Printf("Trends saved successfully: %d records", len(trends))
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

func getClientByFingerprint(server *models.Server, fingerprint string) (*models.Client, error) {
	clientID, err := server.ClientFingerprintDB.GetClientID(fingerprint)
	if err != nil {
		return nil, fmt.Errorf("获取客户端ID失败: %w", err)
	}

	return server.ClientDB.GetClient(clientID)
}
