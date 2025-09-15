package service

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"star-fire/internal/models"
	"star-fire/pkg/public"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
)

// HandleEmbeddingRequest 处理embedding请求
func HandleEmbeddingRequest(c *gin.Context, server *models.Server) {
	var request openai.EmbeddingRequest
	fingerPrint := uuid.NewString()

	err := c.ShouldBindJSON(&request)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// 使用专门的embedding负载均衡器
	client := server.LoadBalanceEmbedding(string(request.Model))
	if client == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No available client for embedding model"})
		return
	}

	// 检查客户端是否支持embedding
	if !isEmbeddingModelSupported(client, request.Model) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Model does not support embedding"})
		return
	}

	// 获取embedding模型的定价 (只有输入tokens，没有输出tokens)
	ippm := 0.1 // 默认embedding输入tokens价格
	for _, m := range client.Models {
		if m.Name == string(request.Model) {
			ippm = m.IPPM
			break
		}
	}

	log.Println("Client ID:", client.ID, "Embedding Model:", request.Model, "IPPM:", ippm)

	// 保存fingerprint和客户端关系
	if err := server.ClientFingerprintDB.SaveFingerprint(fingerPrint, client.ID, "preparing"); err != nil {
		log.Printf("save fingerprint and client relation failed: %v", err)
	}

	// 发送embedding请求到客户端
	err = client.ControlConn.WriteJSON(public.WSMessage{
		Type:        public.EMBEDDING_REQUEST,
		Content:     request,
		FingerPrint: fingerPrint,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while writing json to client:" + err.Error()})
		return
	}

	waitStart := time.Now()
	handleEmbeddingResponse(c, server, fingerPrint, waitStart, client.ID, ippm)
}

// handleEmbeddingResponse 处理embedding响应
func handleEmbeddingResponse(c *gin.Context, server *models.Server, fingerPrint string, waitStart time.Time, clientID string, ippm float64) {
	for {
		if server.RespClients[fingerPrint] == nil {
			time.Sleep(1 * time.Millisecond)
			continue
		}

		// 更新fingerprint状态
		if err := server.ClientFingerprintDB.UpdateFingerprint(fingerPrint, clientID, "transmitting"); err != nil {
			log.Printf("update fingerprint failed: %v", err)
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
		case public.EMBEDDING_RESPONSE:
			handleStandardEmbeddingResponse(c, server, fingerPrint, response, clientID, ippm)
			return

		case public.MODEL_ERROR:
			log.Printf("Embedding model error: %v", response.Content)
			c.JSON(http.StatusInternalServerError, gin.H{"error": response.Content})
			cleanupEmbeddingRequest(server, fingerPrint)
			return

		case public.CLOSE:
			log.Printf("Embedding request closed by client")
			cleanupEmbeddingRequest(server, fingerPrint)
			return

		default:
			log.Printf("Unknown response type for embedding: %s", response.Type)
		}
	}
}

// handleStandardEmbeddingResponse 处理标准embedding响应
func handleStandardEmbeddingResponse(c *gin.Context, server *models.Server, fingerPrint string, response public.WSMessage, clientID string, ippm float64) {
	// 将响应内容转换为OpenAI embedding响应格式
	responseBytes, err := json.Marshal(response.Content)
	if err != nil {
		log.Printf("Error marshaling embedding response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error processing response"})
		cleanupEmbeddingRequest(server, fingerPrint)
		return
	}

	var embeddingResp openai.EmbeddingResponse
	err = json.Unmarshal(responseBytes, &embeddingResp)
	if err != nil {
		log.Printf("Error unmarshaling embedding response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error processing response"})
		cleanupEmbeddingRequest(server, fingerPrint)
		return
	}

	// 计算token使用量和收益
	inputTokens := calculateEmbeddingTokens(embeddingResp)
	revenue := float64(inputTokens) * ippm / 1000000 // embedding只有输入tokens

	// 获取用户信息和API Key信息（从中间件中获取）
	userID, _ := c.Get("user_id")
	apiKey, _ := c.Get("api_key")

	// 生成请求ID
	requestID := fmt.Sprintf("emb_%s_%d", fingerPrint, time.Now().Unix())

	// 记录token使用情况
	tokenUsage := models.TokenUsage{
		RequestID:    requestID,
		UserID:       userID.(string),
		APIKey:       apiKey.(string),
		ClientID:     clientID,
		ClientIP:     c.ClientIP(),
		Model:        string(embeddingResp.Model),
		IPPM:         ippm,
		OPPM:         0.0, // embedding没有输出tokens
		InputTokens:  inputTokens,
		OutputTokens: 0, // embedding没有输出tokens
		TotalTokens:  inputTokens,
		RequestType:  "embedding",
		Revenue:      revenue,
		Fingerprint:  fingerPrint,
		Timestamp:    time.Now(),
		CreatedAt:    time.Now(),
	}

	err = server.TokenUsageDB.RecordTokenUsage(tokenUsage)
	if err != nil {
		log.Printf("Error recording embedding token usage: %v", err)
		// 即使记录失败，也继续返回响应
	} else {
		log.Printf("Embedding usage recorded - User: %s, Model: %s, Tokens: %d, Revenue: %.6f",
			userID, embeddingResp.Model, inputTokens, revenue)
	}

	log.Printf("Embedding completed - Fingerprint: %s, Input Tokens: %d, Revenue: %.6f",
		fingerPrint, inputTokens, revenue)

	// 返回embedding响应
	c.JSON(http.StatusOK, embeddingResp)
	cleanupEmbeddingRequest(server, fingerPrint)
}

// calculateEmbeddingTokens 计算embedding请求的token数量
func calculateEmbeddingTokens(response openai.EmbeddingResponse) int {
	if response.Usage.TotalTokens > 0 {
		return response.Usage.TotalTokens
	}

	// 如果没有usage信息，基于输入估算token数量
	// 这是一个简单的估算，实际情况可能需要更精确���计算
	totalTokens := 0
	for _, embedding := range response.Data {
		// 估算：每个embedding大约对应输入文本的token数量
		// 这里使用embedding维度作为粗略估算（实际应该基于输入文本）
		if len(embedding.Embedding) > 0 {
			totalTokens += len(embedding.Embedding) / 10 // 粗略估算
		}
	}

	if totalTokens == 0 {
		totalTokens = 100 // 最小默认值
	}

	return totalTokens
}

// isEmbeddingModelSupported 检查模型是否支持embedding
func isEmbeddingModelSupported(client *models.Client, modelName openai.EmbeddingModel) bool {
	// 检查客户端是否有这个模型
	for _, model := range client.Models {
		if model.Name == string(modelName) {
			// 检查是否为embedding模型
			return isEmbeddingModel(string(modelName))
		}
	}
	return false
}

// isEmbeddingModel 判断模型名称是否为embedding模型
func isEmbeddingModel(modelName string) bool {
	embeddingModels := []string{
		// OpenAI embedding models
		"text-embedding-ada-002",
		"text-embedding-3-small",
		"text-embedding-3-large",
		"text-similarity-davinci-001",
		"text-similarity-curie-001",
		"text-similarity-babbage-001",
		"text-similarity-ada-001",
		"text-search-ada-doc-001",
		"text-search-ada-query-001",
		"text-search-babbage-doc-001",
		"text-search-babbage-query-001",
		"text-search-curie-doc-001",
		"text-search-curie-query-001",
		"text-search-davinci-doc-001",
		"text-search-davinci-query-001",
		"code-search-ada-code-001",
		"code-search-ada-text-001",
		"code-search-babbage-code-001",
		"code-search-babbage-text-001",

		// BGE (BAAI General Embedding) models
		"bge-large-en",
		"bge-base-en",
		"bge-small-en",
		"bge-large-zh",
		"bge-base-zh",
		"bge-small-zh",
		"bge-large-en-v1.5",
		"bge-base-en-v1.5",
		"bge-small-en-v1.5",
		"bge-large-zh-v1.5",
		"bge-base-zh-v1.5",
		"bge-small-zh-v1.5",
		"bge-m3",
		"bge-multilingual-gemma2",
		"bge-reranker-large",
		"bge-reranker-base",
		"bge-reranker-v2-m3",
		"bge-reranker-v2-gemma",

		// BGE model variations with different naming patterns
		"BAAI/bge-large-en",
		"BAAI/bge-base-en",
		"BAAI/bge-small-en",
		"BAAI/bge-large-zh",
		"BAAI/bge-base-zh",
		"BAAI/bge-small-zh",
		"BAAI/bge-large-en-v1.5",
		"BAAI/bge-base-en-v1.5",
		"BAAI/bge-small-en-v1.5",
		"BAAI/bge-large-zh-v1.5",
		"BAAI/bge-base-zh-v1.5",
		"BAAI/bge-small-zh-v1.5",
		"BAAI/bge-m3",
		"BAAI/bge-multilingual-gemma2",
		"BAAI/bge-reranker-large",
		"BAAI/bge-reranker-base",
		"BAAI/bge-reranker-v2-m3",
		"BAAI/bge-reranker-v2-gemma",
	}

	for _, embeddingModel := range embeddingModels {
		if modelName == embeddingModel {
			return true
		}
	}

	// 检查模型名称中是否包含embedding相关关键词（包括BGE）
	embeddingKeywords := []string{"embed", "embedding", "similarity", "search", "bge", "reranker"}
	modelLower := strings.ToLower(modelName)
	for _, keyword := range embeddingKeywords {
		if contains(modelName, keyword) {
			return true
		}
	}

	// BGE模型的特殊模式匹配
	if strings.Contains(modelLower, "bge-") ||
		strings.Contains(modelLower, "baai/bge") ||
		strings.Contains(modelLower, "bge_") {
		return true
	}

	return false
}

// contains 检查字符串是否包含子字符串（忽略大小写）
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && s[:len(substr)] == substr) ||
		(len(s) > len(substr) && s[len(s)-len(substr):] == substr) ||
		(len(s) > len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// cleanupEmbeddingRequest 清理embedding请求资源
func cleanupEmbeddingRequest(server *models.Server, fingerPrint string) {
	if server.RespClients[fingerPrint] != nil {
		_ = server.RespClients[fingerPrint].Close()
		server.RemoveRespClient(fingerPrint)
	}

	// 更新fingerprint状态为完成
	if err := server.ClientFingerprintDB.UpdateFingerprint(fingerPrint, "", "completed"); err != nil {
		log.Printf("update fingerprint to completed failed: %v", err)
	}
}
