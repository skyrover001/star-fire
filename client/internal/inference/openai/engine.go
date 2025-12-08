package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/sashabaranov/go-openai"
	"log"
	"star-fire/client/internal/config"
	"star-fire/pkg/public"
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
			Name:        model.ID,
			Type:        model.Root,
			Size:        "unknown",
			Arch:        model.Object,
			Engine:      "openai",
			IPPM:        conf.InputTokenPricePerMillion,
			OPPM:        conf.OutputTokenPricePerMillion,
			OpenAIModel: model,
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

	// 判断模型是否为kimi模型，kimi模型的request 和openai的request不同，stream response也不同
	if strings.Contains(request.Model, "kimi") || strings.Contains(request.Model, "moonshot") {
		log.Println("handle kimi model request [%s]: modle=%s, strem=%v, API BASE URL=%s")
		if request.ReasoningEffort == "none" {
			request.ReasoningEffort = ""
		}
		if request.Stream {
			log.Printf("use stream [%s]", fingerprint)

			// 使用自定义流处理以保留 usage 信息
			err := e.handleStreamWithRawSSE(ctx, fingerprint, request, responseConn)
			if err != nil {
				return err
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

	if request.Stream {
		log.Printf("use openai fomat stream [%s]", fingerprint)
		if strings.Contains(strings.ToLower(request.Model), "qwen") {
			if strings.Contains(strings.ToLower(request.Model), "think") || strings.Contains(strings.ToLower(request.Model), "qwen") {
				request.ReasoningEffort = "low"
			}
		}
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

		var lastResponse *openai.ChatCompletionStreamResponse
		hasFinishReason := false

		for {
			response, err := stream.Recv()
			if err != nil {
				if strings.Contains(err.Error(), "stream closed") || err.Error() == "EOF" {
					log.Printf("[%s] stream stop", fingerprint)
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

			// 发送流响应（包括可能只有 usage 的数据块）
			wsResp := public.WSMessage{
				Type:        public.MESSAGE_STREAM,
				Content:     response,
				FingerPrint: fingerprint,
			}

			if err := responseConn.WriteJSON(wsResp); err != nil {
				log.Printf("[%s] send response error: %v", fingerprint, err)
				return err
			}

			// 记录最后一个响应
			lastResponse = &response

			// 记录日志并检查 usage
			if response.Usage != nil && response.Usage.TotalTokens > 0 {
				log.Printf("[%s] received usage: prompt=%d, completion=%d, total=%d",
					fingerprint, response.Usage.PromptTokens, response.Usage.CompletionTokens, response.Usage.TotalTokens)

				// 如果已经收到了 finish_reason，现在收到 usage，可以结束了
				if hasFinishReason {
					log.Printf("[%s] received usage after finish_reason, stream complete", fingerprint)
					break
				}
			}

			// 检查 finish_reason
			if len(response.Choices) > 0 && response.Choices[0].FinishReason != "" {
				log.Printf("[%s] received finish_reason: %s", fingerprint, response.Choices[0].FinishReason)
				hasFinishReason = true

				// 如果这个数据块同时包含 usage，可以结束
				if response.Usage != nil && response.Usage.TotalTokens > 0 {
					log.Printf("[%s] finish_reason and usage in same block, stream complete", fingerprint)
					break
				}

				// 否则继续等待下一个可能包含 usage 的数据块
				log.Printf("[%s] finish_reason received, waiting for usage block...", fingerprint)
			}
		}

		// 确保最后发送了包含 usage 的数据块（如果有的话）
		if lastResponse != nil && lastResponse.Usage != nil && lastResponse.Usage.TotalTokens > 0 {
			log.Printf("[%s] stream ended with usage info available", fingerprint)
		} else {
			log.Printf("[%s] warning: stream ended without usage info", fingerprint)
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

// handleStreamWithRawSSE 直接处理 SSE 流以保留完整的 JSON 数据（包括 Kimi 的 usage）
func (e *Engine) handleStreamWithRawSSE(ctx context.Context, fingerprint string,
	request *openai.ChatCompletionRequest, responseConn *websocket.Conn) error {

	// 构造请求
	reqBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("marshal request error: %w", err)
	}

	baseURL := e.baseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	url := strings.TrimSuffix(baseURL, "/") + "/chat/completions"

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("create http request error: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+e.apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		errMsg := fmt.Sprintf("create chat completion stream error: %v", err)
		log.Printf("[%s] %s", fingerprint, errMsg)
		_ = responseConn.WriteJSON(public.WSMessage{
			Type:        public.MODEL_ERROR,
			Content:     errMsg,
			FingerPrint: fingerprint,
		})
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		errMsg := fmt.Sprintf("API error: status=%d, body=%s", resp.StatusCode, string(body))
		log.Printf("[%s] %s", fingerprint, errMsg)
		_ = responseConn.WriteJSON(public.WSMessage{
			Type:        public.MODEL_ERROR,
			Content:     errMsg,
			FingerPrint: fingerprint,
		})
		return fmt.Errorf(errMsg)
	}

	// 读取 SSE 流
	var accumulatedUsage *openai.Usage
	reader := bufio.NewReader(resp.Body)

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				log.Printf("[%s] stream EOF", fingerprint)
				break
			}
			errMsg := fmt.Sprintf("read stream error: %v", err)
			log.Printf("[%s] %s", fingerprint, errMsg)
			_ = responseConn.WriteJSON(public.WSMessage{
				Type:        public.MODEL_ERROR,
				Content:     errMsg,
				FingerPrint: fingerprint,
			})
			return err
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		// SSE 格式: "data: {... }"
		if !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}

		data := bytes.TrimPrefix(line, []byte("data: "))

		// 结束标记
		if bytes.Equal(data, []byte("[DONE]")) {
			log.Printf("[%s] stream done", fingerprint)
			break
		}

		// 解析为兼容结构
		var kimiResp KimiStreamResponse
		if err := json.Unmarshal(data, &kimiResp); err != nil {
			log.Printf("[%s] unmarshal chunk error: %v, data: %s", fingerprint, err, string(data))
			continue
		}

		// 提取 usage
		if usage := kimiResp.ExtractUsage(); usage != nil {
			accumulatedUsage = usage
			log.Printf("[%s] extracted usage: prompt=%d, completion=%d, total=%d",
				fingerprint, usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
		}

		// 转换为标准格式推送
		standardResp := kimiResp.ToStandardResponse(accumulatedUsage)
		wsResp := public.WSMessage{
			Type:        public.MESSAGE_STREAM,
			Content:     standardResp,
			FingerPrint: fingerprint,
		}

		if err := responseConn.WriteJSON(wsResp); err != nil {
			log.Printf("[%s] send response error: %v", fingerprint, err)
			return err
		}

		// 检查是否结束
		if len(kimiResp.Choices) > 0 && kimiResp.Choices[0].FinishReason != "" {
			log.Printf("[%s] stream response over, reason: %s", fingerprint, kimiResp.Choices[0].FinishReason)

			// 推送 usage
			if accumulatedUsage != nil {
				usageMsg := public.WSMessage{
					Type:        public.MESSAGE_STREAM,
					Content:     accumulatedUsage,
					FingerPrint: fingerprint,
				}
				if err := responseConn.WriteJSON(usageMsg); err != nil {
					log.Printf("[%s] send usage error: %v", fingerprint, err)
				} else {
					log.Printf("[%s] sent usage info", fingerprint)
				}
			}
			break
		}
	}

	return nil
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

func (e *Engine) SupportsEmbedding(modelName string) bool {
	embeddingModels := []string{
		"text-embedding-ada-002",
		"text-embedding-3-small",
		"text-embedding-3-large",
		// ... 其他 embedding 模型
	}

	for _, embeddingModel := range embeddingModels {
		if strings.Contains(modelName, embeddingModel) || modelName == embeddingModel {
			return true
		}
	}

	for _, model := range e.modelList {
		if model.ID == modelName && (strings.Contains(strings.ToLower(model.ID), "embed") ||
			strings.Contains(strings.ToLower(model.ID), "similarity") ||
			strings.Contains(strings.ToLower(model.ID), "search") ||
			strings.Contains(strings.ToLower(model.ID), "bge") ||
			strings.Contains(strings.ToLower(model.ID), "reranker")) {
			return true
		}
	}

	modelLower := strings.ToLower(modelName)
	if strings.Contains(modelLower, "bge-") ||
		strings.Contains(modelLower, "baai/bge") ||
		strings.Contains(modelLower, "bge_") {
		return true
	}

	return false
}
