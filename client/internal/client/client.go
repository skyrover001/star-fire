// internal/client/client.go
package client

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"star-fire/client/internal/config"
	"star-fire/client/internal/inference"
	"star-fire/client/internal/inference/ollama"
	"star-fire/client/internal/inference/openai"
	"star-fire/pkg/public"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Client struct {
	ID              string `json:"id"`
	engines         []inference.Engine
	controlConn     *websocket.Conn
	starFireHost    string
	joinToken       string
	Models          []*public.Model `json:"models"`
	ctx             context.Context
	cancel          context.CancelFunc
	cfg             *config.Config
	ModelPriceScope map[string]struct {
		inputPriceMax       float64
		outputPriceMax      float64
		cachedInputPriceMax float64
	}
	AppClient net.Conn
}

func NewClient(cfg *config.Config) (*Client, error) {
	ctx, cancel := context.WithCancel(context.Background())
	client := &Client{
		starFireHost: cfg.StarFireHost,
		joinToken:    cfg.JoinToken,
		ctx:          ctx,
		cancel:       cancel,
		engines:      []inference.Engine{},
		Models:       []*public.Model{},
		cfg:          cfg,
		ModelPriceScope: make(map[string]struct {
			inputPriceMax       float64
			outputPriceMax      float64
			cachedInputPriceMax float64
		}),
	}
	if err := client.generateID(); err != nil {
		return nil, fmt.Errorf("generate id error: %w", err)
	}
	if err := client.initializeEngines(cfg); err != nil {
		return nil, fmt.Errorf("init engine error: %w", err)
	}
	if err := client.refreshModels(); err != nil {
		log.Printf("alert: refresh models error: %v", err)
	}

	// 初始化 TCP 连接到应用服务器
	if err := client.initTCPConnection(); err != nil {
		log.Printf("alert: init TCP connection error: %v", err)
	}

	return client, nil
}

func (c *Client) generateID() error {
	interfaces, err := net.Interfaces()
	if err != nil {
		c.ID = uuid.NewString()
		return nil
	}
	for _, iface := range interfaces {
		mac := iface.HardwareAddr.String()
		if mac != "" {
			hash := sha256.Sum256([]byte(mac))
			c.ID = fmt.Sprintf("%x", hash)
			return nil
		}
	}
	c.ID = uuid.NewString()
	return nil
}

func (c *Client) initializeEngines(cfg *config.Config) error {
	switch cfg.LocalInferenceType {
	case "ollama":
		ollamaEngine, err := ollama.NewEngine(c.ctx, cfg.OllamaHost, cfg)
		if err != nil {
			return fmt.Errorf("init Ollama engine error: %w", err)
		}
		c.engines = append(c.engines, ollamaEngine)

	case "openai":
		if cfg.OpenAIKey == "" {
			return fmt.Errorf("not set OpenAIKey")
		}
		openaiEngine, err := openai.NewEngine(c.ctx, cfg.OpenAIKey, cfg.OpenAIBaseURL, cfg)
		if err != nil {
			return fmt.Errorf("init openai engine error: %w", err)
		}
		c.engines = append(c.engines, openaiEngine)

		// 如果不是 OpenAIOnly 模式，也尝试初始化本地 Ollama 引擎
		if !cfg.OpenAIOnly {
			log.Println("OpenAI mode with local engines enabled, attempting to init Ollama...")
			ollamaEngine, err := ollama.NewEngine(c.ctx, cfg.OllamaHost, cfg)
			if err != nil {
				log.Printf("⚠️ Init Ollama engine error (optional): %v", err)
			} else {
				c.engines = append(c.engines, ollamaEngine)
				log.Println("✓ Ollama engine added alongside OpenAI")
			}
		} else {
			log.Println("OpenAI-only mode enabled, skipping local engines")
		}

	case "all":
		// 先初始化 OpenAI（如果配置了）
		if cfg.OpenAIKey != "" {
			openaiEngine, err := openai.NewEngine(c.ctx, cfg.OpenAIKey, cfg.OpenAIBaseURL, cfg)
			if err != nil {
				log.Printf("init openai engine error: %v", err)
			} else {
				c.engines = append(c.engines, openaiEngine)
			}
		}

		// 如果不是 OpenAIOnly 模式，才初始化本地引擎
		if !cfg.OpenAIOnly {
			ollamaEngine, err := ollama.NewEngine(c.ctx, cfg.OllamaHost, cfg)
			if err != nil {
				log.Printf("init ollama engine error: %v", err)
			} else {
				c.engines = append(c.engines, ollamaEngine)
			}
		} else {
			log.Println("OpenAI-only mode enabled in 'all' engine mode, skipping Ollama")
		}

	default:
		return fmt.Errorf("no avaliable engine : %s", cfg.LocalInferenceType)
	}

	if len(c.engines) == 0 {
		return fmt.Errorf("no engine found")
	}

	return nil
}

func (c *Client) refreshModels() error {
	// 创建一个 map 来快速查找现有模型（使用 engine+model 作为 key）
	existingModels := make(map[string]*public.Model)
	for _, model := range c.Models {
		key := fmt.Sprintf("%s:%s", model.Engine, model.Name)
		existingModels[key] = model
	}

	// 收集所有引擎的新模型列表
	newModels := make([]*public.Model, 0)
	newModelKeys := make(map[string]bool)

	log.Println("c.engines:", c.engines, len(c.engines))
	for _, engine := range c.engines {
		models, err := engine.ListModels(c.ctx, c.cfg)
		if err != nil {
			log.Printf("get models from %s error: %v", engine.Name(), err)
			continue
		}

		// 为每个模型检查是否支持embedding
		for _, model := range models {
			key := fmt.Sprintf("%s:%s", model.Engine, model.Name)
			newModelKeys[key] = true

			// 标记embedding支持
			if c.isEmbeddingModel(model.Name) && c.engineSupportsEmbedding(engine, model.Name) {
				log.Printf("Found embedding model: %s from engine: %s", model.Name, engine.Name())
				model.Type = "embedding" // 设置模型类型为embedding
			}

			// 检查是否已存在该模型
			if existingModel, exists := existingModels[key]; exists {
				// 模型已存在，保留价格信息，更新其他信息
				log.Printf("🔄 Updating existing model: %s/%s (preserving prices: ippm=%.6f, oppm=%.6f, cippm=%.6f)",
					model.Engine, model.Name, existingModel.IPPM, existingModel.OPPM, existingModel.CIPPM)

				// 保留价格信息
				model.IPPM = existingModel.IPPM
				model.OPPM = existingModel.OPPM
				model.CIPPM = existingModel.CIPPM
			} else {
				// 新模型，使用默认价格
				log.Printf("➕ New model discovered: %s/%s (default prices: ippm=%.6f, oppm=%.6f, cippm=%.6f)",
					model.Engine, model.Name, model.IPPM, model.OPPM, model.CIPPM)
			}

			newModels = append(newModels, model)
		}
	}

	// 检查被移除的模型
	for key, model := range existingModels {
		if !newModelKeys[key] {
			log.Printf("➖ Model removed: %s/%s", model.Engine, model.Name)
		}
	}

	// 更新模型列表
	c.Models = newModels

	if c.cfg.OpenAIOnly {
		log.Printf("📊 OpenAI-only mode: discovered %d models (OpenAI + running local models)", len(c.Models))
	} else {
		log.Printf("📊 Discovery %d models (including all local models)", len(c.Models))
	}

	// 记录发现的embedding模型
	embeddingCount := 0
	for _, model := range c.Models {
		if model.Type == "embedding" || c.isEmbeddingModel(model.Name) {
			log.Printf("Embedding model discovered: %s", model.Name)
			embeddingCount++
		}
	}
	log.Printf("Total embedding models found: %d", embeddingCount)

	return nil
}

// isEmbeddingModel 检查模型名称是否为embedding模型
func (c *Client) isEmbeddingModel(modelName string) bool {
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

	// 精确匹配
	for _, embeddingModel := range embeddingModels {
		if modelName == embeddingModel {
			return true
		}
	}

	// 关键词匹配（增加BGE相关关键词）
	embeddingKeywords := []string{"embed", "embedding", "similarity", "search", "bge", "reranker"}
	modelLower := strings.ToLower(modelName)
	for _, keyword := range embeddingKeywords {
		if strings.Contains(modelLower, keyword) {
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

// engineSupportsEmbedding 检查引擎是否支持embedding
func (c *Client) engineSupportsEmbedding(engine inference.Engine, modelName string) bool {
	// 检查引擎是否实现了SupportsEmbedding方法
	type EmbeddingSupporter interface {
		SupportsEmbedding(modelName string) bool
	}

	if embeddingEngine, ok := engine.(EmbeddingSupporter); ok {
		return embeddingEngine.SupportsEmbedding(modelName)
	}

	// 如果引擎没有实现SupportsEmbedding方法，基于引擎类型判断
	switch engine.Name() {
	case "openai":
		return c.isEmbeddingModel(modelName)
	case "ollama":
		// Ollama目前不支持embedding
		return false
	default:
		return false
	}
}

func (c *Client) findEngineForModel(modelName string) (inference.Engine, error) {
	for _, engine := range c.engines {
		if engine.SupportsModel(modelName, c.cfg) {
			return engine, nil
		}
	}

	return nil, fmt.Errorf("no enging support model: %s", modelName)
}

// SetModelPrice 设置模型价格信息，支持按模型定价
func (c *Client) SetModelPrice(engine string, inputTokenPrice float64, outputTokenPrice float64, cachedInputTokenPrice float64, modelName string) interface{} {
	type TIP struct {
		InputPrice       float64 `json:"inputPriceMax"`
		OutputPrice      float64 `json:"outputPriceMax"`
		CachedInputPrice float64 `json:"cachedInputPriceMax"`
		ModelName        string  `json:"modelName"`
		Engine           string  `json:"engine"`
		Msg              string  `json:"tip"`
	}
	tip := TIP{
		InputPrice:       0,
		OutputPrice:      0,
		CachedInputPrice: 0,
		ModelName:        modelName,
		Engine:           engine,
		Msg:              "",
	}

	// 获取价格上限
	ippmM, oppmM, cippmM := c.getModelPriceScope(modelName)

	// 确保价格非负
	if inputTokenPrice < 0 {
		inputTokenPrice = 0
	}
	if outputTokenPrice < 0 {
		outputTokenPrice = 0
	}
	if cachedInputTokenPrice < 0 {
		cachedInputTokenPrice = 0
	}

	// 查找并更新对应的模型
	modelFound := false
	for i := range c.Models {
		if c.Models[i].Name == modelName && c.Models[i].Engine == engine {
			modelFound = true
			log.Printf("🎯 Found model to update: %s/%s (current: ippm=%.6f, oppm=%.6f, cippm=%.6f)",
				engine, modelName, c.Models[i].IPPM, c.Models[i].OPPM, c.Models[i].CIPPM)

			// 检查价格是否超过上限
			if inputTokenPrice > ippmM && ippmM > 0 {
				tip.InputPrice = ippmM
				tip.Msg = fmt.Sprintf("输入token价格过高，最大允许值为%.6f", ippmM)
				c.Models[i].IPPM = ippmM
			} else {
				c.Models[i].IPPM = inputTokenPrice
			}

			if outputTokenPrice > oppmM && oppmM > 0 {
				if tip.Msg != "" {
					tip.Msg += "; "
				}
				tip.OutputPrice = oppmM
				tip.Msg += fmt.Sprintf("输出token价格过高，最大允许值为%.6f", oppmM)
				c.Models[i].OPPM = oppmM
			} else {
				c.Models[i].OPPM = outputTokenPrice
			}

			if cachedInputTokenPrice > cippmM && cippmM > 0 {
				if tip.Msg != "" {
					tip.Msg += "; "
				}
				tip.CachedInputPrice = cippmM
				tip.Msg += fmt.Sprintf("缓存输入token价格过高，最大允许值为%.6f", cippmM)
				c.Models[i].CIPPM = cippmM
			} else {
				c.Models[i].CIPPM = cachedInputTokenPrice
			}

			if tip.Msg == "" {
				tip.Msg = "设置成功"
			}

			log.Printf("✅ Model price updated: %s/%s (new: ippm=%.6f, oppm=%.6f, cippm=%.6f)",
				engine, modelName, c.Models[i].IPPM, c.Models[i].OPPM, c.Models[i].CIPPM)
			break
		}
	}

	if !modelFound {
		log.Printf("⚠️ Model not found: %s/%s (available models: %d)", engine, modelName, len(c.Models))
		tip.Msg = fmt.Sprintf("模型未找到: %s/%s", engine, modelName)
	}

	return &tip
}

func (c *Client) getModelPriceScope(modelName string) (float64, float64, float64) {
	for m, scope := range c.ModelPriceScope {
		if m == modelName {
			return scope.inputPriceMax, scope.outputPriceMax, scope.cachedInputPriceMax
		}
	}
	fmt.Println("server model price scope not found, use client default value")
	return c.cfg.InputTokenPricePerMillion, c.cfg.OutputTokenPricePerMillion, c.cfg.CachedInputTokenPricePerMillion
}

// initTCPConnection 初始化到应用服务器的 TCP 连接
func (c *Client) initTCPConnection() error {
	if c.cfg.APPPort == 0 {
		log.Println("APPPort not configured, skipping TCP connection initialization")
		return nil
	}

	address := fmt.Sprintf("127.0.0.1:%d", c.cfg.APPPort)
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return fmt.Errorf("connect to TCP server %s error: %w", address, err)
	}

	c.AppClient = conn
	log.Printf("✓ TCP connection to app server established: %s", address)

	// 启动消息接收 goroutine
	go c.receiveTCPMessages()

	return nil
}

// ensureTCPConnection 确保 TCP 连接可用，如果断开则重连
func (c *Client) ensureTCPConnection() error {
	if c.cfg.APPPort == 0 {
		return fmt.Errorf("APPPort not configured")
	}

	// 检查现有连接是否有效
	if c.AppClient != nil {
		// 尝试设置一个很短的超时来测试连接
		if err := c.AppClient.SetReadDeadline(time.Now().Add(1 * time.Millisecond)); err == nil {
			// 重置超时
			_ = c.AppClient.SetReadDeadline(time.Time{})
			return nil // 连接正常
		}
		// 连接无效，关闭它
		_ = c.AppClient.Close()
		c.AppClient = nil
		log.Println("TCP connection lost, reconnecting...")
	}

	// 重新建立连接
	if err := c.initTCPConnection(); err != nil {
		return err
	}

	return nil
}

// Close 关闭客户端连接
func (c *Client) Close() error {
	var errors []error

	// 取消上下文
	if c.cancel != nil {
		c.cancel()
	}

	// 关闭 TCP 连接
	if c.AppClient != nil {
		if err := c.AppClient.Close(); err != nil {
			errors = append(errors, fmt.Errorf("close TCP connection error: %w", err))
		}
		c.AppClient = nil
		log.Println("TCP connection closed")
	}

	// 关闭 WebSocket 连接
	if c.controlConn != nil {
		if err := c.controlConn.Close(); err != nil {
			errors = append(errors, fmt.Errorf("close WebSocket connection error: %w", err))
		}
		log.Println("WebSocket connection closed")
	}

	if len(errors) > 0 {
		return fmt.Errorf("close client errors: %v", errors)
	}

	return nil
}

// ModelPriceMessage TCP服务器发送的模型价格设置消息
type ModelPriceMessage struct {
	Engine           string  `json:"engine"`
	Model            string  `json:"model"`
	InputPrice       float64 `json:"ippm"`
	OutputPrice      float64 `json:"oppm"`
	CachedInputPrice float64 `json:"cippm"`
}

// UnmarshalJSON 自定义 JSON 反序列化，支持字符串格式的价格
func (m *ModelPriceMessage) UnmarshalJSON(data []byte) error {
	// 定义一个临时结构体，将价格字段定义为 interface{} 类型
	type Alias struct {
		Engine           string      `json:"engine"`
		Model            string      `json:"model"`
		InputPrice       interface{} `json:"ippm"`
		OutputPrice      interface{} `json:"oppm"`
		CachedInputPrice interface{} `json:"cippm"`
	}

	var alias Alias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	m.Engine = alias.Engine
	m.Model = alias.Model

	// 处理 InputPrice - 可能是字符串或数字
	m.InputPrice = parseFloat(alias.InputPrice)

	// 处理 OutputPrice - 可能是字符串或数字
	m.OutputPrice = parseFloat(alias.OutputPrice)

	// 处理 CachedInputPrice - 可能是字符串或数字
	m.CachedInputPrice = parseFloat(alias.CachedInputPrice)

	return nil
}

// parseFloat 将 interface{} 转换为 float64，支持字符串和数字类型
func parseFloat(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case string:
		var f float64
		_, _ = fmt.Sscanf(val, "%f", &f)
		return f
	case int:
		return float64(val)
	case int64:
		return float64(val)
	default:
		return 0
	}
}

// receiveTCPMessages 接收来自TCP服务器的消息
func (c *Client) receiveTCPMessages() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("receiveTCPMessages panic recovered: %v", r)
		}
	}()

	log.Println("TCP message receiver started")
	for {
		log.Println("waiting for message")
		select {
		case <-c.ctx.Done():
			log.Println("TCP message receiver stopped due to context cancellation")
			return
		default:
			if c.AppClient == nil {
				log.Println("TCP connection is nil, stopping message receiver")
				return
			}
			if err := c.AppClient.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
				log.Printf("set read deadline error: %v", err)
				return
			}
			// 读取消息长度头（4字节）
			lengthBytes := make([]byte, 4)
			n, err := c.AppClient.Read(lengthBytes)
			if err != nil {
				// 超时错误可以忽略，继续循环
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					fmt.Println("read timeout, continuing")
					continue
				}
				log.Printf("read length header error: %v", err)
				return
			}

			if n != 4 {
				log.Printf("invalid length header, expected 4 bytes, got %d", n)
				continue
			}

			// 解析消息长度（网络字节序）
			length := uint32(lengthBytes[0])<<24 | uint32(lengthBytes[1])<<16 |
				uint32(lengthBytes[2])<<8 | uint32(lengthBytes[3])

			if length == 0 || length > 1024*1024 { // 最大1MB
				log.Printf("invalid message length: %d", length)
				continue
			}

			// 读取消息内容
			messageBytes := make([]byte, length)
			totalRead := 0
			for totalRead < int(length) {
				n, err := c.AppClient.Read(messageBytes[totalRead:])
				if err != nil {
					log.Printf("read message content error: %v", err)
					return
				}
				totalRead += n
			}

			// 处理接收到的消息
			c.handleTCPMessage(messageBytes)
		}
	}
}

// handleTCPMessage 处理接收到的TCP消息
func (c *Client) handleTCPMessage(data []byte) {
	// 尝试解析为模型价格消息
	log.Printf("📥 Received TCP message: %s", string(data))
	// 定义完整的消息结构
	type TcpMessageWithPrices struct {
		Type      string              `json:"type"`
		Data      []ModelPriceMessage `json:"data"`
		TimeStamp int64               `json:"timestamp"`
		ID        string              `json:"id"`
	}

	var messages TcpMessageWithPrices
	if err := json.Unmarshal(data, &messages); err != nil {
		log.Printf("unmarshal TCP message error: %v, data: %s", err, string(data))
		return
	}

	if messages.Type != "model_prices" {
		log.Printf("invalid message type: %s", messages.Type)
		return
	}

	log.Printf("Processing price config: ID=%s, Timestamp=%d, Models=%d",
		messages.ID, messages.TimeStamp, len(messages.Data))

	// 更新每个模型的价格
	for _, priceMsg := range messages.Data {
		// 验证消息字段
		if priceMsg.Engine == "" || priceMsg.Model == "" {
			log.Printf("⚠ Invalid model price message: engine or model is empty")
			continue
		}

		log.Printf("📊 Updating price: engine=%s, model=%s, ippm=%.6f, oppm=%.6f, cippm=%.6f",
			priceMsg.Engine, priceMsg.Model, priceMsg.InputPrice, priceMsg.OutputPrice, priceMsg.CachedInputPrice)

		// 调用 SetModelPrice 方法更新价格
		result := c.SetModelPrice(priceMsg.Engine, priceMsg.InputPrice, priceMsg.OutputPrice, priceMsg.CachedInputPrice, priceMsg.Model)
		if result != nil {
			log.Printf("✓ Model price updated: %s/%s - %v", priceMsg.Engine, priceMsg.Model, result)
		} else {
			log.Printf("✓ Model price updated: %s/%s", priceMsg.Engine, priceMsg.Model)
		}
	}

	log.Printf("✅ Price configuration applied successfully for %d models", len(messages.Data))
}
