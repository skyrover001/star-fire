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
	"github.com/gorilla/websocket"
	"github.com/sashabaranov/go-openai"
)

// handle user chat request
func HandleChatRequest(c *gin.Context, server *models.Server) {
	//扩展结构体（在 go-openai 标准请求之上承载 thinking / enable_thinking）
	var extendedRequest public.ExtendedChatRequest
	err := c.ShouldBindJSON(&extendedRequest)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// 仅在用户未显式提供 reasoning_effort 时才填充默认值，
	// 避免覆盖调用方（如 hermes/opencode）自带的值。
	// 注意：不要对空值强制设 "none"，因为某些后端（如 vLLM/GLM-5.1）不支持该参数，
	// 强制设置可能导致模型输出混乱（think块和tool_call混杂）导致截断。
	if extendedRequest.ReasoningEffort == "" {
		if extendedRequest.EnableThinking != nil && *extendedRequest.EnableThinking {
			extendedRequest.ReasoningEffort = "medium"
		}
		// qwen 系列模型兼容 reasoning effort
		modelLower := strings.ToLower(extendedRequest.Model)
		if strings.Contains(modelLower, "qwen") && (strings.Contains(modelLower, "think") || strings.Contains(modelLower, "235b")) {
			extendedRequest.ReasoningEffort = "low"
		}
	}

	request := extendedRequest.ChatCompletionRequest

	userID, _ := c.Get("user_id")
	userIDStr, _ := userID.(string)

	// Balance pre-check: reject if balance insufficient (OpenAI-compatible error)
	balance, _, _ := server.UserDB.GetBalance(userIDStr)
	if balance <= 0 {
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error": gin.H{
				"message": "You exceeded your current quota, please check your plan and billing details. For more information on this error, see https://platform.openai.com/docs/guides/error-codes/api-errors.",
				"type":    "insufficient_quota",
				"param":   nil,
				"code":    "insufficient_quota",
			},
		})
		return
	}

	if request.Stream {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
	}

	handleChatWithRetry(c, server, extendedRequest, userIDStr)
}

// handleChatWithRetry 在"第一个 token 前"对失败的请求进行自动重试。
// 每次重试重新 LoadBalance（排除已失败的 client）、重新生成 fingerprint、
// 重新建立响应通道。一旦读到第一条消息（MESSAGE/MESSAGE_STREAM），
// 即进入正常处理流程，不再重试（此时用户可能已收到内容）。
func handleChatWithRetry(c *gin.Context, server *models.Server, extendedRequest public.ExtendedChatRequest, userIDStr string) {
	request := extendedRequest.ChatCompletionRequest
	failedClients := map[string]bool{}
	start := time.Now()

	for attempt := 0; attempt < public.MAX_CHAT_RETRY; attempt++ {
		// 全局超时检查，避免极端情况下重试耗时过长
		if time.Since(start) > public.CHAT_RETRY_TOTAL_TIMEOUT*time.Second {
			break
		}

		// 1. 选 client（排除已失败的）
		client := server.LoadBalanceExcluding(request.Model, userIDStr, failedClients)
		if client == nil {
			break
		}
		failedClients[client.ID] = true

		// 2. 从该 client 提取价格（计费使用实际服务的 client 价格）
		ippm := 9.0  // 输入tokens价格（未命中缓存部分）
		oppm := 9.0  // 输出tokens价格
		cippm := 0.0 // 缓存命中输入tokens价格
		for _, m := range client.Models {
			if m.Name == request.Model {
				ippm = m.IPPM
				oppm = m.OPPM
				cippm = m.CIPPM
				break
			}
		}

		// 3. 生成新 fingerprint（每次重试必须重新生成）
		fingerPrint := uuid.NewString()

		if err := server.ClientFingerprintDB.SaveFingerprint(fingerPrint, client.ID, "preparing"); err != nil {
			log.Printf("save fingerprint and client relation failed: %v", err)
		}

		log.Println("Client ID:", client.ID, "Model:", request.Model, "IPPM:", ippm, "OPPM:", oppm, "CIPPM:", cippm)

		// 4. 发送请求到 client
		if err := client.ControlConn.WriteJSON(public.WSMessage{
			Type:        public.MESSAGE,
			Content:     extendedRequest,
			FingerPrint: fingerPrint,
		}); err != nil {
			log.Printf("attempt %d: send to client %s failed: %v", attempt, client.ID, err)
			server.ClientFingerprintDB.DeleteFingerprint(fingerPrint)
			time.Sleep(backoff(attempt))
			continue
		}

		// 5. 等待响应连接就绪（替代自旋），最多等 CHAT_MAX_TIME
		readyCh := server.AddRespClientChan(fingerPrint)
		select {
		case <-readyCh:
		case <-time.After(public.CHAT_MAX_TIME * time.Second):
			server.RemoveRespClientChan(fingerPrint)
			log.Printf("attempt %d: response conn timeout for client %s", attempt, client.ID)
			abortClientRequest(client, fingerPrint)
			server.ClientFingerprintDB.DeleteFingerprint(fingerPrint)
			time.Sleep(backoff(attempt))
			continue
		}

		// 6. 获取响应连接
		respConn, ok := server.GetRespClient(fingerPrint)
		if !ok {
			abortClientRequest(client, fingerPrint)
			server.ClientFingerprintDB.DeleteFingerprint(fingerPrint)
			time.Sleep(backoff(attempt))
			continue
		}

		// 7. 更新 fingerprint 状态为 transmitting
		if err := server.ClientFingerprintDB.UpdateFingerprint(fingerPrint, client.ID, "transmitting"); err != nil {
			log.Printf("save fingerprint and client relation failed: %v", err)
			respConn.Close()
			server.RemoveRespClient(fingerPrint)
			abortClientRequest(client, fingerPrint)
			server.ClientFingerprintDB.DeleteFingerprint(fingerPrint)
			time.Sleep(backoff(attempt))
			continue
		}

		// 8. 读取第一条消息（判断类型）
		var response public.WSMessage
		if err := respConn.ReadJSON(&response); err != nil {
			log.Printf("attempt %d: read first msg from client %s failed: %v", attempt, client.ID, err)
			respConn.Close()
			server.RemoveRespClient(fingerPrint)
			abortClientRequest(client, fingerPrint)
			server.ClientFingerprintDB.DeleteFingerprint(fingerPrint)
			time.Sleep(backoff(attempt))
			continue
		}

		// 9. 判断第一条消息类型
		switch response.Type {
		case public.MESSAGE, public.MESSAGE_STREAM:
			// 成功！进入正常处理流程
			handleChatResponseWithFirst(c, server, fingerPrint, time.Now(), client.ID, ippm, oppm, cippm, request.Model, response, respConn)
			return
		case public.CLOSE:
			log.Printf("attempt %d: client %s closed before first token", attempt, client.ID)
			respConn.Close()
			server.RemoveRespClient(fingerPrint)
			server.ClientFingerprintDB.DeleteFingerprint(fingerPrint)
			time.Sleep(backoff(attempt))
			continue
		case public.MODEL_ERROR:
			log.Printf("attempt %d: model error from client %s: %v", attempt, client.ID, response.Content)
			respConn.Close()
			server.RemoveRespClient(fingerPrint)
			server.ClientFingerprintDB.DeleteFingerprint(fingerPrint)
			time.Sleep(backoff(attempt))
			continue
		default:
			log.Printf("attempt %d: unexpected first msg type %s from client %s", attempt, response.Type, client.ID)
			respConn.Close()
			server.RemoveRespClient(fingerPrint)
			server.ClientFingerprintDB.DeleteFingerprint(fingerPrint)
			time.Sleep(backoff(attempt))
			continue
		}
	}

	// 重试耗尽，返回明确错误
	c.JSON(http.StatusServiceUnavailable, gin.H{"error": "All clients failed, please retry"})
}

// backoff 指数退避：attempt=0 -> 100ms, 1 -> 200ms, 2 -> 400ms
func backoff(attempt int) time.Duration {
	return time.Duration(public.CHAT_RETRY_BASE_DELAY*(1<<attempt)) * time.Millisecond
}

// abortClientRequest 通知 client 停止处理指定 fingerprint 的请求（尽力而为）。
// 用于 server 放弃某 client 时，避免 client 继续生成孤儿 token 浪费算力。
func abortClientRequest(client *models.Client, fingerPrint string) {
	if client == nil || client.ControlConn == nil {
		return
	}
	client.ControlConnMutex.Lock()
	defer client.ControlConnMutex.Unlock()
	if err := client.ControlConn.WriteJSON(public.WSMessage{
		Type:        public.CLOSE,
		Content:     public.ABORT,
		FingerPrint: fingerPrint,
	}); err != nil {
		log.Printf("send abort to client %s failed: %v", client.ID, err)
	}
}

// handleChatResponseWithFirst 处理已读取的第一条响应消息（不再重复 ReadJSON）。
// 由 handleChatWithRetry 在成功读到第一条消息后调用。
func handleChatResponseWithFirst(c *gin.Context, server *models.Server, fingerPrint string, waitStart time.Time, clientID string, ippm, oppm, cippm float64, reqModel string, response public.WSMessage, respConn *websocket.Conn) {
	switch response.Type {
	case public.MESSAGE:
		handleStandardChatResponse(c, server, fingerPrint, response, clientID, ippm, oppm, cippm, reqModel, respConn)
		return

	case public.MESSAGE_STREAM:
		finished := handleStreamChatResponse(c, server, fingerPrint, response, clientID, ippm, oppm, cippm, reqModel, respConn)
		if finished {
			return
		}
		// continue reading stream
		readStreamLoop(c, server, fingerPrint, respConn, waitStart, clientID, ippm, oppm, cippm, reqModel)
		return

	case public.CLOSE:
		log.Println("Client closed connection")
		// 向前端发送 [DONE] 标记，确保 SSE 流正常终止
		if c.Writer.Header().Get("Content-Type") == "text/event-stream" {
			_, _ = c.Writer.Write([]byte("data: [DONE]\n\n"))
			c.Writer.Flush()
		}
		respConn.Close()
		server.RemoveRespClient(fingerPrint)
		_ = server.ClientFingerprintDB.UpdateFingerprint(fingerPrint, clientID, "completed")
		return

	case public.MODEL_ERROR:
		log.Println("Model error:", response.Content)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Model error: " + response.Content.(string)})
		respConn.Close()
		server.RemoveRespClient(fingerPrint)
		return

	default:
		log.Println("Unknown message type:", response.Type)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unknown message type: " + response.Type})
		respConn.Close()
		server.RemoveRespClient(fingerPrint)
		return
	}
}

// readStreamLoop 持续读取 stream 消息
func readStreamLoop(c *gin.Context, server *models.Server, fingerPrint string, respConn *websocket.Conn, waitStart time.Time, clientID string, ippm, oppm, cippm float64, reqModel string) {
	for {
		var response public.WSMessage
		err := respConn.ReadJSON(&response)
		if err != nil {
			log.Println("Error while reading json from client:", err)
			return
		}
		switch response.Type {
		case public.MESSAGE_STREAM:
			finished := handleStreamChatResponse(c, server, fingerPrint, response, clientID, ippm, oppm, cippm, reqModel, respConn)
			if finished {
				return
			}
		case public.CLOSE:
			log.Println("Client closed connection")
			if c.Writer.Header().Get("Content-Type") == "text/event-stream" {
				_, _ = c.Writer.Write([]byte("data: [DONE]\n\n"))
				c.Writer.Flush()
			}
			respConn.Close()
			server.RemoveRespClient(fingerPrint)
			_ = server.ClientFingerprintDB.UpdateFingerprint(fingerPrint, clientID, "completed")
			return
		case public.MODEL_ERROR:
			log.Println("Model error:", response.Content)
			respConn.Close()
			server.RemoveRespClient(fingerPrint)
			return
		default:
			log.Println("Unknown message type:", response.Type)
			respConn.Close()
			server.RemoveRespClient(fingerPrint)
			return
		}
		if time.Since(waitStart) > public.CHAT_MAX_TIME*time.Second {
			log.Println("Chat timeout")
			respConn.Close()
			server.RemoveRespClient(fingerPrint)
			return
		}
	}
}

// handle standard chat response
func handleStandardChatResponse(c *gin.Context, server *models.Server, fingerPrint string, response public.WSMessage, clientID string, ippm, oppm, cippm float64, reqModel string, conn *websocket.Conn) {
	if content, ok := response.Content.(map[string]interface{}); ok {
		jsonData, err := json.Marshal(content)
		if err != nil {
			log.Println("Error marshaling content:", err)
			conn.Close()
			server.RemoveRespClient(fingerPrint)
			return
		}

		var chatResponse openai.ChatCompletionResponse
		err = json.Unmarshal(jsonData, &chatResponse)
		if err != nil {
			log.Println("Error unmarshaling content into ChatResponse struct:", err)
			conn.Close()
			server.RemoveRespClient(fingerPrint)
			_ = server.ClientFingerprintDB.UpdateFingerprint(fingerPrint, clientID, "completed")
			return
		}

		c.JSON(http.StatusOK, content)

		// 提取缓存命中tokens
		cachedTokens := 0
		if chatResponse.Usage.PromptTokensDetails != nil && chatResponse.Usage.PromptTokensDetails.CachedTokens > 0 {
			cachedTokens = chatResponse.Usage.PromptTokensDetails.CachedTokens
		}

		recordTokenUsage(c, server, fingerPrint, reqModel,
			chatResponse.Usage.PromptTokens, chatResponse.Usage.CompletionTokens,
			chatResponse.Usage.TotalTokens, cachedTokens, clientID, ippm, oppm, cippm)
	} else {
		log.Println("Invalid message content format")
		conn.Close()
		server.RemoveRespClient(fingerPrint)
	}
}

// handle stream chat response
func handleStreamChatResponse(c *gin.Context, server *models.Server, fingerPrint string, response public.WSMessage, clientID string, ippm, oppm, cippm float64, reqModel string, conn *websocket.Conn) bool {
	if content, ok := response.Content.(map[string]interface{}); ok {
		jsonData, err := json.Marshal(content)
		if err != nil {
			log.Println("Error marshaling content:", err)
			conn.Close()
			server.RemoveRespClient(fingerPrint)
			return true
		}

		var chatResponse openai.ChatCompletionStreamResponse
		err = json.Unmarshal(jsonData, &chatResponse)
		if err != nil {
			log.Println("Error unmarshaling content into ChatResponse struct:", err)
			conn.Close()
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
			conn.Close()
			server.RemoveRespClient(fingerPrint)
			return true
		}
		c.Writer.Flush()

		// 检查是否有 usage 信息（可能在 finish_reason 之后的单独数据块中）
		if chatResponse.Usage != nil && chatResponse.Usage.TotalTokens > 0 {
			log.Printf("Recording usage: prompt=%d, completion=%d, total=%d",
				chatResponse.Usage.PromptTokens, chatResponse.Usage.CompletionTokens, chatResponse.Usage.TotalTokens)

			// 提取缓存命中tokens
			cachedTokens := 0
			if chatResponse.Usage.PromptTokensDetails != nil && chatResponse.Usage.PromptTokensDetails.CachedTokens > 0 {
				cachedTokens = chatResponse.Usage.PromptTokensDetails.CachedTokens
			}

			recordTokenUsage(c, server, fingerPrint, reqModel,
				chatResponse.Usage.PromptTokens, chatResponse.Usage.CompletionTokens,
				chatResponse.Usage.TotalTokens, cachedTokens, clientID, ippm, oppm, cippm)

			// 收到 usage 后发送 [DONE] 并结束
			_, err = c.Writer.Write([]byte("data: [DONE]\n\n"))
			c.Writer.Flush()
			conn.Close()
			server.RemoveRespClient(fingerPrint)
			_ = server.ClientFingerprintDB.UpdateFingerprint(fingerPrint, clientID, "completed")
			return true
		}

		// 检查是否完成（finish_reason 为 stop、tool_calls 或 length）
		if len(chatResponse.Choices) > 0 && chatResponse.Choices[0].FinishReason != "" {
			log.Printf("Received finish_reason: %s", chatResponse.Choices[0].FinishReason)
			// 如果这个数据块中已经有 usage，直接处理
			if usage, hasUsage := content["usage"].(map[string]interface{}); hasUsage {
				promptTokens := int(usage["prompt_tokens"].(float64))
				completionTokens := int(usage["completion_tokens"].(float64))
				totalTokens := int(usage["total_tokens"].(float64))

				// 从 prompt_tokens_details 中提取 cached_tokens
				cachedTokens := 0
				if ptd, ok := usage["prompt_tokens_details"].(map[string]interface{}); ok {
					if ct, ok := ptd["cached_tokens"].(float64); ok && ct > 0 {
						cachedTokens = int(ct)
					}
				}

				log.Printf("Recording usage from finish block: prompt=%d, completion=%d, total=%d, cached=%d",
					promptTokens, completionTokens, totalTokens, cachedTokens)

				recordTokenUsage(c, server, fingerPrint, reqModel,
					promptTokens, completionTokens, totalTokens, cachedTokens, clientID, ippm, oppm, cippm)

				_, err = c.Writer.Write([]byte("data: [DONE]\n\n"))
				c.Writer.Flush()
				conn.Close()
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
		conn.Close()
		server.RemoveRespClient(fingerPrint)
		return true
	}
}

func recordTokenUsage(c *gin.Context, server *models.Server, requestID string, model string, inputTokens, outputTokens, totalTokens, cachedTokens int, clientID string, ippm, oppm, cippm float64) {
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
		CachedTokens: cachedTokens,
		TotalTokens:  totalTokens,
		IPPM:         ippm,
		OPPM:         oppm,
		CIPPM:        cippm,
		Timestamp:    time.Now(),
	}

	// Calculate cost: (non-cached input * ippm + cached input * cippm + output * oppm) / 1e6
	cost := (float64(inputTokens-cachedTokens)*ippm + float64(cachedTokens)*cippm + float64(outputTokens)*oppm) / 1000000
	if cost < 0 {
		cost = 0
	}
	usage.Cost = cost

	// Check and deduct balance before saving usage
	userIDStr := userID.(string)
	if err := server.UserDB.DeductBalance(userIDStr, cost); err != nil {
		log.Printf("余额扣费失败: user=%s, cost=%.6f, error=%v", userIDStr, cost, err)
		// Return OpenAI-compatible insufficient balance error
		// We can't set HTTP status here since this is called after streaming starts,
		// so we log and continue. The balance check should happen before sending to client.
		// For non-stream requests we return error; for stream it's best-effort.
		return
	}

	err := server.TokenUsageDB.SaveTokenUsage(usage)
	if err != nil {
		log.Printf("保存token使用记录失败: %v", err)
		return
	}
	log.Printf("记录用户 %s 使用 %s 模型，消耗 %d tokens", userID, model, totalTokens)

	// 根据client的用户userid 获取最新的总收入（异步执行，避免阻塞聊天请求）
	chatClient := server.GetClientByModel(model, clientID)
	if chatClient == nil {
		log.Printf("client %s not found for model %s", clientID, model)
		return
	}

	chatClient.ControlConnMutex.Lock()
	conn := chatClient.ControlConn
	chatClient.ControlConnMutex.Unlock()

	if conn == nil {
		log.Printf("client %s ControlConn is nil", clientID)
		return
	}

	// 异步通知 client 收益更新，避免全表扫描阻塞聊天响应
	go func(clientID, model string, income float64, inputTokens, outputTokens, totalTokens, cachedTokens int) {
		totalIncomeResult, totalErr := server.TokenUsageDB.GetTotalIncomeByUserID(chatClient.User.ID, server.ClientDB)
		if totalErr != nil {
			log.Printf("获取用户 %s 总收入失败: %v", chatClient.User.ID, totalErr)
			return
		}
		totalIncome, _ := totalIncomeResult.(float64)
		_ = conn.WriteJSON(public.WSMessage{
			Type: public.INCOME,
			Content: map[string]interface{}{
				"model": model,
				"usage": map[string]interface{}{
					"prompt_tokens":     inputTokens,
					"completion_tokens": outputTokens,
					"total_tokens":      totalTokens,
					"cached_tokens":     cachedTokens,
				},
				"income":       income,
				"total_income": totalIncome,
				"timestamp":    strconv.Itoa(int(time.Now().Unix())),
			},
		})
	}(clientID, model,
		(ippm*float64(inputTokens-cachedTokens)+cippm*float64(cachedTokens)+oppm*float64(outputTokens))/1000000,
		inputTokens, outputTokens, totalTokens, cachedTokens)
}
