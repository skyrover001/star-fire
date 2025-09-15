package openai

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/sashabaranov/go-openai"
	"log"
	"star-fire/client/internal/config"
	"star-fire/pkg/public"
	"strings"
	"time"
)

type Engine struct {
	client    *openai.Client
	baseURL   string
	apiKey    string
	modelList []openai.Model
}

func NewEngine(ctx context.Context, apiKey, baseURL string, conf *config.Config) (*Engine, error) {
	engine := &Engine{
		apiKey:  apiKey,
		baseURL: baseURL,
	}
	if err := engine.Initialize(ctx, conf); err != nil {
		return nil, err
	}
	return engine, nil
}

func (e *Engine) Name() string {
	return "openai"
}

func (e *Engine) Initialize(ctx context.Context, conf *config.Config) error {
	clientConfig := openai.DefaultConfig(e.apiKey)
	if e.baseURL != "" && e.baseURL != "https://api.openai.com/v1" {
		clientConfig.BaseURL = e.baseURL
	}
	e.client = openai.NewClientWithConfig(clientConfig)
	fmt.Println("openai client created", e.client, clientConfig.BaseURL, clientConfig.APIType, e.apiKey, e.baseURL)
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	models, err := e.client.ListModels(timeoutCtx)
	if err != nil {
		return fmt.Errorf("connect to open ai engine error: %w", err)
	}

	e.modelList = models.Models
	log.Printf("connect to openai engine，discovery %d models", len(e.modelList))
	return nil
}

func (e *Engine) ListModels(ctx context.Context, conf *config.Config) ([]*public.Model, error) {
	models, err := e.client.ListModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("get models frome openai engine error: %w", err)
	}
	e.modelList = models.Models

	publicModels := make([]*public.Model, 0)
	for _, model := range e.modelList {
		publicModel := &public.Model{
			Name: model.ID,
			Type: model.Root,
			Size: "unknown",
			Arch: model.Object,
			IPPM: conf.InputTokenPricePerMillion,  // 每百万输入tokens价格，默认值4.0
			OPPM: conf.OutputTokenPricePerMillion, // 每百万输出tokens价格，默认值8.0
		}
		publicModels = append(publicModels, publicModel)
	}
	return publicModels, nil
}

func (e *Engine) SupportsModel(modelName string, conf *config.Config) bool {
	for _, model := range e.modelList {
		if model.ID == modelName {
			return true
		}
	}
	return false
}

func (e *Engine) HandleChat(ctx context.Context, fingerprint string,
	request *openai.ChatCompletionRequest, responseConn *websocket.Conn) error {
	log.Printf("handle chat request [%s]: modle=%s, strem=%v, API BASE URL=%s",
		fingerprint, request.Model, request.Stream, e.baseURL)

	if request.Stream {
		log.Printf("use stream [%s]", fingerprint)
		stream, err := e.client.CreateChatCompletionStream(ctx, *request)
		if err != nil {
			errMsg := fmt.Sprintf("create chat complation error: %v", err)
			log.Printf("[%s] %s", fingerprint, errMsg)
			err = responseConn.WriteJSON(public.WSMessage{
				Type:        public.MODEL_ERROR,
				Content:     errMsg,
				FingerPrint: fingerprint,
			})
			return err
		}
		defer stream.Close()

		for {
			response, err := stream.Recv()
			if err != nil {
				if strings.Contains(err.Error(), "stream closed") || err.Error() == "EOF" {
					log.Printf("[%s] stram stop", fingerprint)
					break
				}
				errMsg := fmt.Sprintf("read stream error: %v", err)
				log.Printf("[%s] %s", fingerprint, errMsg)

				err = responseConn.WriteJSON(public.WSMessage{
					Type:        public.MODEL_ERROR,
					Content:     errMsg,
					FingerPrint: fingerprint,
				})
				return err
			}
			fmt.Println("stream response:", response, "response.Choices[0].FinishReason:", response.Choices[0].FinishReason,
				"response.usage:", response.Usage, "response.choices:", response.Choices)
			wsResp := public.WSMessage{
				Type:        public.MESSAGE_STREAM,
				Content:     response,
				FingerPrint: fingerprint,
			}

			if err := responseConn.WriteJSON(wsResp); err != nil {
				log.Printf("[%s] send response error: %v", fingerprint, err)
				return err
			}

			if len(response.Choices) > 0 && response.Choices[0].FinishReason != "" {
				log.Printf("[%s] stream response over，on: %s", fingerprint, response.Choices[0].FinishReason)
				break
			}
		}
	} else {
		log.Printf("not stream request [%s]", fingerprint)
		resp, err := e.client.CreateChatCompletion(ctx, *request)
		if err != nil {
			errMsg := fmt.Sprintf("create chat error: %v", err)
			log.Printf("[%s] %s", fingerprint, errMsg)
			err = responseConn.WriteJSON(public.WSMessage{
				Type:        public.MODEL_ERROR,
				Content:     errMsg,
				FingerPrint: fingerprint,
			})
			return err
		}

		log.Printf("[%s] recieve response", fingerprint)
		err = responseConn.WriteJSON(public.WSMessage{
			Type:        public.MESSAGE,
			Content:     resp,
			FingerPrint: fingerprint,
		})

		if err != nil {
			log.Printf("[%s] send response error: %v", fingerprint, err)
			return err
		}
	}
	log.Printf("[%s] send close message", fingerprint)
	err := responseConn.WriteJSON(public.WSMessage{
		Type:        public.CLOSE,
		Content:     nil,
		FingerPrint: fingerprint,
	})
	log.Printf("[%s] send close message success", fingerprint)
	return err
}

func (e *Engine) HandleEmbedding(ctx context.Context, fingerprint string,
	request *openai.EmbeddingRequest, responseConn *websocket.Conn) error {
	log.Printf("handle embedding request [%s]: model=%s, input=%v, API BASE URL=%s",
		fingerprint, request.Model, request.Input, e.baseURL)

	resp, err := e.client.CreateEmbeddings(ctx, *request)
	if err != nil {
		errMsg := fmt.Sprintf("create embedding error: %v", err)
		log.Printf("[%s] %s", fingerprint, errMsg)
		err = responseConn.WriteJSON(public.WSMessage{
			Type:        public.MODEL_ERROR,
			Content:     errMsg,
			FingerPrint: fingerprint,
		})
		return err
	}

	log.Printf("[%s] receive embedding response", fingerprint)
	err = responseConn.WriteJSON(public.WSMessage{
		Type:        public.EMBEDDING_RESPONSE,
		Content:     resp,
		FingerPrint: fingerprint,
	})

	if err != nil {
		log.Printf("[%s] send embedding response error: %v", fingerprint, err)
		return err
	}

	log.Printf("[%s] send close message", fingerprint)
	err = responseConn.WriteJSON(public.WSMessage{
		Type:        public.CLOSE,
		Content:     nil,
		FingerPrint: fingerprint,
	})
	log.Printf("[%s] send close message success", fingerprint)
	return err
}

// 检查模型是否支持embedding
func (e *Engine) SupportsEmbedding(modelName string) bool {
	// 常见的embedding模型
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
		if strings.Contains(modelName, embeddingModel) || modelName == embeddingModel {
			return true
		}
	}

	// 检查模型列表中是否包含embedding相关关键词（包括BGE）
	for _, model := range e.modelList {
		if model.ID == modelName && (strings.Contains(strings.ToLower(model.ID), "embed") ||
			strings.Contains(strings.ToLower(model.ID), "similarity") ||
			strings.Contains(strings.ToLower(model.ID), "search") ||
			strings.Contains(strings.ToLower(model.ID), "bge") ||
			strings.Contains(strings.ToLower(model.ID), "reranker")) {
			return true
		}
	}

	// BGE模型的特殊模式匹配
	modelLower := strings.ToLower(modelName)
	if strings.Contains(modelLower, "bge-") ||
		strings.Contains(modelLower, "baai/bge") ||
		strings.Contains(modelLower, "bge_") {
		return true
	}

	return false
}
