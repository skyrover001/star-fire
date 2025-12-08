package service

import (
	"encoding/json"
	"log"
	"net/http"
	"star-fire/internal/models"
	"star-fire/pkg/public"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
)

// handle user chat request
func HandleChatRequest(c *gin.Context, server *models.Server) {
	type ExtendedChatRequest struct {
		openai.ChatCompletionRequest
		EnableThink bool `json:"enable_thinking,omitempty"`
	}

	//扩展结构体
	var extendedRequest ExtendedChatRequest
	err := c.ShouldBindJSON(&extendedRequest)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	request := extendedRequest.ChatCompletionRequest
	fingerPrint := uuid.NewString()

	if extendedRequest.EnableThink {
		request.ReasoningEffort = "medium"
	} else {
		request.ReasoningEffort = "none"
	}

	// qwen 系列模型兼容reasoning effort
	if strings.Contains(strings.ToLower(request.Model), "qwen") && (strings.Contains(strings.ToLower(request.Model), "think") || strings.Contains(strings.ToLower(request.Model), "235b")) {
		request.ReasoningEffort = "low"
	}

	// fmt.Println("request is ..............................", request.Metadata, request.ChatCompletionRequestExtensions, request.ReasoningEffort)
	client := server.LoadBalance(request.Model)
	if client == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No available client"})
		return
	}

	ippm := 9.0 // 输入tokens价格
	oppm := 9.0 // 输出tokens价格
	for _, m := range client.Models {
		if m.Name == request.Model {
			ippm = m.IPPM
			oppm = m.OPPM
			break
		}
	}

	log.Println("Client ID:", client.ID, "Model:", request.Model, "IPPM:", ippm, "OPPM:", oppm)

	if err := server.ClientFingerprintDB.SaveFingerprint(fingerPrint, client.ID, "preparing"); err != nil {
		log.Printf("save fingerprint and client relation failed: %v", err)
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

	if request.Stream {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
	}

	waitStart := time.Now()
	handleChatResponse(c, server, fingerPrint, waitStart, client.ID, ippm, oppm)
}

// handle chat response
func handleChatResponse(c *gin.Context, server *models.Server, fingerPrint string, waitStart time.Time, clientID string, ippm, oppm float64) {
	for {
		if server.RespClients[fingerPrint] == nil {
			time.Sleep(1 * time.Millisecond)
			continue
		}

		// save fingerprint and client relation and connect status to database
		if err := server.ClientFingerprintDB.UpdateFingerprint(fingerPrint, clientID, "transmitting"); err != nil {
			log.Printf("save fingerprint and client relation failed: %v", err)
			_ = server.RespClients[fingerPrint].Close()
			server.RemoveRespClient(fingerPrint)
			return
		}

		var response public.WSMessage
		err := server.RespClients[fingerPrint].ReadJSON(&response)
		if err != nil {
			log.Println("Error while reading json from client:", err)
			return
		}

		switch response.Type {
		case public.MESSAGE:
			handleStandardChatResponse(c, server, fingerPrint, response, clientID, ippm, oppm)
			return

		case public.MESSAGE_STREAM:
			finished := handleStreamChatResponse(c, server, fingerPrint, response, clientID, ippm, oppm)
			if finished {
				return
			}

		case public.CLOSE:
			log.Println("Client closed connection")
			_ = server.RespClients[fingerPrint].Close()
			server.RemoveRespClient(fingerPrint)
			return

		case public.MODEL_ERROR:
			log.Println("Model error:", response.Content)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Model error: " + response.Content.(string)})
			_ = server.RespClients[fingerPrint].Close()
			server.RemoveRespClient(fingerPrint)
			return

		default:
			log.Println("Unknown message type:", response.Type)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Unknown message type: " + response.Type})
			_ = server.RespClients[fingerPrint].Close()
			server.RemoveRespClient(fingerPrint)
			return
		}

		if time.Since(waitStart) > public.CHAT_MAX_TIME*time.Second {
			log.Println("Chat timeout")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Chat timeout"})
			_ = server.RespClients[fingerPrint].Close()
			server.RemoveRespClient(fingerPrint)
			return
		}
	}
}

// handle standard chat response
func handleStandardChatResponse(c *gin.Context, server *models.Server, fingerPrint string, response public.WSMessage, clientID string, ippm, oppm float64) {
	if content, ok := response.Content.(map[string]interface{}); ok {
		jsonData, err := json.Marshal(content)
		if err != nil {
			log.Println("Error marshaling content:", err)
			_ = server.RespClients[fingerPrint].Close()
			server.RemoveRespClient(fingerPrint)
			return
		}

		var chatResponse openai.ChatCompletionResponse
		err = json.Unmarshal(jsonData, &chatResponse)
		if err != nil {
			log.Println("Error unmarshaling content into ChatResponse struct:", err)
			_ = server.RespClients[fingerPrint].Close()
			server.RemoveRespClient(fingerPrint)
			_ = server.ClientFingerprintDB.UpdateFingerprint(fingerPrint, clientID, "completed")
			return
		}

		c.JSON(http.StatusOK, content)

		recordTokenUsage(c, server, fingerPrint, chatResponse.Model,
			chatResponse.Usage.PromptTokens, chatResponse.Usage.CompletionTokens,
			chatResponse.Usage.TotalTokens, clientID, ippm, oppm)
	} else {
		log.Println("Invalid message content format")
		server.RespClients[fingerPrint].Close()
		server.RemoveRespClient(fingerPrint)
	}
}

// handle stream chat response
func handleStreamChatResponse(c *gin.Context, server *models.Server, fingerPrint string, response public.WSMessage, clientID string, ippm, oppm float64) bool {
	if content, ok := response.Content.(map[string]interface{}); ok {
		jsonData, err := json.Marshal(content)
		if err != nil {
			log.Println("Error marshaling content:", err)
			_ = server.RespClients[fingerPrint].Close()
			server.RemoveRespClient(fingerPrint)
			return true
		}

		var chatResponse openai.ChatCompletionStreamResponse
		err = json.Unmarshal(jsonData, &chatResponse)
		if err != nil {
			log.Println("Error unmarshaling content into ChatResponse struct:", err)
			_ = server.RespClients[fingerPrint].Close()
			server.RemoveRespClient(fingerPrint)
			return true
		}

		if chatResponse.Usage != nil {
			log.Printf("chatResponse: usage prompt=%d, completion=%d, total=%d",
				chatResponse.Usage.PromptTokens, chatResponse.Usage.CompletionTokens, chatResponse.Usage.TotalTokens)
		}

		// 发送数据到客户端
		_, err = c.Writer.Write([]byte("data: " + string(jsonData) + "\n\n"))
		if err != nil {
			log.Println("Error while writing response:", err)
			_ = server.RespClients[fingerPrint].Close()
			server.RemoveRespClient(fingerPrint)
			return true
		}
		c.Writer.Flush()

		// 检查是否有 usage 信息（可能在 finish_reason 之后的单独数据块中）
		if chatResponse.Usage != nil && chatResponse.Usage.TotalTokens > 0 {
			log.Printf("Recording usage: prompt=%d, completion=%d, total=%d",
				chatResponse.Usage.PromptTokens, chatResponse.Usage.CompletionTokens, chatResponse.Usage.TotalTokens)

			recordTokenUsage(c, server, fingerPrint, chatResponse.Model,
				chatResponse.Usage.PromptTokens, chatResponse.Usage.CompletionTokens,
				chatResponse.Usage.TotalTokens, clientID, ippm, oppm)

			// 收到 usage 后发送 [DONE] 并结束
			_, err = c.Writer.Write([]byte("data: [DONE]\n\n"))
			c.Writer.Flush()
			_ = server.RespClients[fingerPrint].Close()
			server.RemoveRespClient(fingerPrint)
			_ = server.ClientFingerprintDB.UpdateFingerprint(fingerPrint, clientID, "completed")
			return true
		}

		// 检查是否完成（finish_reason 为 stop）
		if len(chatResponse.Choices) > 0 && chatResponse.Choices[0].FinishReason == "stop" {
			log.Printf("Received finish_reason: stop")
			// 如果这个数据块中已经有 usage，直接处理
			if usage, hasUsage := content["usage"].(map[string]interface{}); hasUsage {
				promptTokens := int(usage["prompt_tokens"].(float64))
				completionTokens := int(usage["completion_tokens"].(float64))
				totalTokens := int(usage["total_tokens"].(float64))

				log.Printf("Recording usage from finish block: prompt=%d, completion=%d, total=%d",
					promptTokens, completionTokens, totalTokens)

				recordTokenUsage(c, server, fingerPrint, chatResponse.Model,
					promptTokens, completionTokens, totalTokens, clientID, ippm, oppm)

				_, err = c.Writer.Write([]byte("data: [DONE]\n\n"))
				c.Writer.Flush()
				_ = server.RespClients[fingerPrint].Close()
				server.RemoveRespClient(fingerPrint)
				_ = server.ClientFingerprintDB.UpdateFingerprint(fingerPrint, clientID, "completed")
				return true
			}
			// 如果没有 usage，继续等待下一个可能包含 usage 的数据块
			log.Printf("Finish reason received but no usage yet, waiting for usage block...")
		}

		return false
	} else {
		log.Println("Invalid message content format")
		_ = server.RespClients[fingerPrint].Close()
		server.RemoveRespClient(fingerPrint)
		return true
	}
}

func recordTokenUsage(c *gin.Context, server *models.Server, requestID string, model string, inputTokens, outputTokens, totalTokens int, clientID string, ippm, oppm float64) {
	if server.TokenUsageDB == nil {
		log.Println("Token usage database not initialized")
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		log.Println("User ID not found in context")
		return
	}

	apiKeyID := ""
	if id, exists := c.Get("api_key_id"); exists {
		apiKeyID = id.(string)
	}

	clientIP := c.ClientIP()
	usage := &models.TokenUsage{
		RequestID:    requestID,
		UserID:       userID.(string),
		APIKey:       apiKeyID,
		ClientIP:     clientIP,
		ClientID:     clientID,
		Model:        model,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  totalTokens,
		IPPM:         ippm,
		OPPM:         oppm,
		Timestamp:    time.Now(),
	}

	err := server.TokenUsageDB.SaveTokenUsage(usage)
	if err != nil {
		log.Printf("保存token使用记录失败: %v", err)
	} else {
		log.Printf("记录用户 %s 使用 %s 模型，消耗 %d tokens", userID, model, totalTokens)
		go func(server *models.Server, model string, clientID string, inputTokens, outputTokens, totalTokens int, ippm, oppm float64) {
			// 检查 clients 是否存在
			if server.Clients == nil {
				log.Printf("server.Clients is nil, cannot send income message")
				return
			}

			// 检查模型是否存在
			modelClients, modelExists := server.Clients[model]
			if !modelExists || modelClients == nil {
				log.Printf("model %s not found in server.Clients", model)
				return
			}

			// 检查客户端是否存在
			client, clientExists := modelClients[clientID]
			if !clientExists || client == nil {
				log.Printf("client %s not found for model %s", clientID, model)
				return
			}

			// 检查控制连接是否存在
			if client.ControlConn == nil {
				log.Printf("client %s ControlConn is nil", clientID)
				return
			}

			// 根据client的用户userid 获取最新的总收入
			totalIncome, err := server.TokenUsageDB.GetTotalIncomeByUserID(client.User.ID, server.ClientDB)
			if err != nil {
				log.Printf("获取用户 %s 总收入失败: %v", client.User.ID, err)
				return
			}
			_ = client.ControlConn.WriteJSON(public.WSMessage{
				Type: public.INCOME,
				Content: map[string]interface{}{
					"model": model,
					"usage": map[string]interface{}{
						"prompt_tokens":     inputTokens,
						"completion_tokens": outputTokens,
						"total_tokens":      totalTokens,
					},
					"income":       (ippm*float64(inputTokens) + oppm*float64(outputTokens)) / 1000000,
					"total_income": totalIncome,
					"timestamp":    strconv.Itoa(int(time.Now().Unix())),
				},
			})
		}(server, model, clientID, inputTokens, outputTokens, totalTokens, ippm, oppm)
	}
}
