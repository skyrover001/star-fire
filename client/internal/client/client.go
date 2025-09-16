// internal/client/client.go
package client

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"net"
	"star-fire/client/internal/config"
	"star-fire/client/internal/inference"
	"star-fire/client/internal/inference/ollama"
	"star-fire/client/internal/inference/openai"
	"star-fire/pkg/public"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Client struct {
	ID           string `json:"id"`
	engines      []inference.Engine
	controlConn  *websocket.Conn
	starFireHost string
	joinToken    string
	Models       []*public.Model `json:"models"`
	ctx          context.Context
	cancel       context.CancelFunc
	cfg          *config.Config
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

	case "all":
		ollamaEngine, err := ollama.NewEngine(c.ctx, cfg.OllamaHost, cfg)
		if err != nil {
			log.Printf("init ollama engine error: %v", err)
		} else {
			c.engines = append(c.engines, ollamaEngine)
		}

		if cfg.OpenAIKey != "" {
			openaiEngine, err := openai.NewEngine(c.ctx, cfg.OpenAIKey, cfg.OpenAIBaseURL, cfg)
			if err != nil {
				log.Printf("init openai engine error: %v", err)
			} else {
				c.engines = append(c.engines, openaiEngine)
			}
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
	c.Models = make([]*public.Model, 0)

	log.Println("c.engines:", c.engines, len(c.engines))
	for _, engine := range c.engines {
		models, err := engine.ListModels(c.ctx, c.cfg)
		if err != nil {
			log.Printf("get models from %s error: %v", engine.Name(), err)
			continue
		}

		// 为每个模型检���是否支持embedding
		for _, model := range models {
			// 标记embedding支持
			if c.isEmbeddingModel(model.Name) && c.engineSupportsEmbedding(engine, model.Name) {
				log.Printf("Found embedding model: %s from engine: %s", model.Name, engine.Name())
				model.Type = "embedding" // 设置模型类型为embedding
			}
		}

		c.Models = append(c.Models, models...)
	}

	log.Printf("discovery %d models (including embedding models)", len(c.Models))
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

// Close 关闭客户端连接
func (c *Client) Close() error {
	// 取消 context，这会停止所有使用该 context 的操作
	if c.cancel != nil {
		c.cancel()
	}

	// 关闭 WebSocket 连接
	if c.controlConn != nil {
		return c.controlConn.Close()
	}

	return nil
}
