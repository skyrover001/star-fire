package service

import (
	"encoding/json"
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

	// 负载均衡选择客户端
	client := server.LoadBalance(request.Model)
	if client == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No available client"})
		return
	}

	// 发送请求到客户端
	err = client.ControlConn.WriteJSON(public.WSMessage{
		Type:        public.MESSAGE,
		Content:     request,
		FingerPrint: fingerPrint,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while writing json to client:" + err.Error()})
		return
	}

	// 设置流式响应头部
	if request.Stream {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
	}

	waitStart := time.Now()
	handleChatResponse(c, server, fingerPrint, waitStart)
}

// handle chat response
func handleChatResponse(c *gin.Context, server *models.Server, fingerPrint string, waitStart time.Time) {
	for {
		// 等待响应连接建立
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
			handleStandardChatResponse(c, server, fingerPrint, response)
			return

		case public.MESSAGE_STREAM:
			finished := handleStreamChatResponse(c, server, fingerPrint, response)
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

		// 检查超时
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
func handleStandardChatResponse(c *gin.Context, server *models.Server, fingerPrint string, response public.WSMessage) {
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
	} else {
		log.Println("Invalid message content format")
		server.RespClients[fingerPrint].Close()
		server.RemoveRespClient(fingerPrint)
	}
}

// handle stream chat response
func handleStreamChatResponse(c *gin.Context, server *models.Server, fingerPrint string, response public.WSMessage) bool {
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
