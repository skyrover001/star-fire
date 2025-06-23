package service

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"star-fire/internal/models"
	"star-fire/pkg/public"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
)

// handle user chat request
func HandleChatRequest(c *gin.Context, server *models.Server) {
	var request public.OpenAIRequest
	fingerPrint := uuid.NewString()

	err := c.ShouldBindJSON(&request)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	client := server.LoadBalance(request.Model)
	if client == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No available client"})
		return
	}

	if err := server.ClientFingerprintDB.SaveFingerprint(fingerPrint, client.ID); err != nil {
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
	handleChatResponse(c, server, fingerPrint, waitStart, client.ID)
}

// handle chat response
func handleChatResponse(c *gin.Context, server *models.Server, fingerPrint string, waitStart time.Time, clientID string) {
	for {
		if server.RespClients[fingerPrint] == nil {
			time.Sleep(1 * time.Millisecond)
			continue
		}

		var response public.WSMessage
		err := server.RespClients[fingerPrint].ReadJSON(&response)
		if err != nil {
			log.Println("Error while reading json from client:", err)
			return
		}

		switch response.Type {
		case public.MESSAGE:
			handleStandardChatResponse(c, server, fingerPrint, response, clientID)
			return

		case public.MESSAGE_STREAM:
			finished := handleStreamChatResponse(c, server, fingerPrint, response, clientID)
			if finished {
				return
			}

		case public.CLOSE:
			log.Println("Client closed connection")
			server.RespClients[fingerPrint].Close()
			server.RemoveRespClient(fingerPrint)
			return

		case public.MODEL_ERROR:
			log.Println("Model error:", response.Content)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Model error: " + response.Content.(string)})
			server.RespClients[fingerPrint].Close()
			server.RemoveRespClient(fingerPrint)
			return

		default:
			log.Println("Unknown message type:", response.Type)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Unknown message type: " + response.Type})
			server.RespClients[fingerPrint].Close()
			server.RemoveRespClient(fingerPrint)
			return
		}

		if time.Since(waitStart) > public.CHAT_MAX_TIME*time.Second {
			log.Println("Chat timeout")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Chat timeout"})
			server.RespClients[fingerPrint].Close()
			server.RemoveRespClient(fingerPrint)
			return
		}
	}
}

// handle standard chat response
func handleStandardChatResponse(c *gin.Context, server *models.Server, fingerPrint string, response public.WSMessage, clientID string) {
	if content, ok := response.Content.(map[string]interface{}); ok {
		jsonData, err := json.Marshal(content)
		if err != nil {
			log.Println("Error marshaling content:", err)
			server.RespClients[fingerPrint].Close()
			server.RemoveRespClient(fingerPrint)
			return
		}

		var chatResponse openai.ChatCompletionResponse
		err = json.Unmarshal(jsonData, &chatResponse)
		if err != nil {
			log.Println("Error unmarshaling content into ChatResponse struct:", err)
			server.RespClients[fingerPrint].Close()
			server.RemoveRespClient(fingerPrint)
			return
		}

		c.JSON(http.StatusOK, content)

		recordTokenUsage(c, server, fingerPrint, chatResponse.Model,
			chatResponse.Usage.PromptTokens, chatResponse.Usage.CompletionTokens,
			chatResponse.Usage.TotalTokens, clientID)
	} else {
		log.Println("Invalid message content format")
		server.RespClients[fingerPrint].Close()
		server.RemoveRespClient(fingerPrint)
	}
}

// handle stream chat response
func handleStreamChatResponse(c *gin.Context, server *models.Server, fingerPrint string, response public.WSMessage, clientID string) bool {
	if content, ok := response.Content.(map[string]interface{}); ok {
		jsonData, err := json.Marshal(content)
		if err != nil {
			log.Println("Error marshaling content:", err)
			server.RespClients[fingerPrint].Close()
			server.RemoveRespClient(fingerPrint)
			return true
		}

		var chatResponse openai.ChatCompletionStreamResponse
		err = json.Unmarshal(jsonData, &chatResponse)
		fmt.Println("chatResponse:", chatResponse, "chatResponse.Choices:", chatResponse.Choices,
			" chatResponse.Choices[0].Delta:", chatResponse.Choices[0].Delta, " chatResponse.Choices[0].FinishReason:",
			chatResponse.Choices[0].FinishReason)
		if err != nil {
			log.Println("Error unmarshaling content into ChatResponse struct:", err)
			server.RespClients[fingerPrint].Close()
			server.RemoveRespClient(fingerPrint)
			return true
		}

		_, err = c.Writer.Write([]byte("data: " + string(jsonData) + "\n\n"))
		if err != nil {
			log.Println("Error while writing response:", err)
			server.RespClients[fingerPrint].Close()
			server.RemoveRespClient(fingerPrint)
			return true
		}
		c.Writer.Flush()

		if chatResponse.Choices[0].FinishReason == "stop" {
			_, err = c.Writer.Write([]byte("data:[DONE]\n\n"))
			if usage, hasUsage := content["usage"].(map[string]interface{}); hasUsage {
				promptTokens := int(usage["prompt_tokens"].(float64))
				completionTokens := int(usage["completion_tokens"].(float64))
				totalTokens := int(usage["total_tokens"].(float64))
				// Record token usage
				fmt.Println("chatResponse:", chatResponse, " promptTokens:", promptTokens,
					" completionTokens:", completionTokens, " totalTokens:", totalTokens)
				recordTokenUsage(c, server, fingerPrint, chatResponse.Model,
					promptTokens, completionTokens, totalTokens, clientID)
			}
			server.RespClients[fingerPrint].Close()
			server.RemoveRespClient(fingerPrint)
			return true
		}

		return false
	} else {
		log.Println("Invalid message content format")
		server.RespClients[fingerPrint].Close()
		server.RemoveRespClient(fingerPrint)
		return true
	}
}

func recordTokenUsage(c *gin.Context, server *models.Server, requestID string, model string, inputTokens, outputTokens, totalTokens int, clientID string) {
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
		Timestamp:    time.Now(),
	}

	err := server.TokenUsageDB.SaveTokenUsage(usage)
	if err != nil {
		log.Printf("保存token使用记录失败: %v", err)
	} else {
		log.Printf("记录用户 %s 使用 %s 模型，消耗 %d tokens", userID, model, totalTokens)
	}
}
