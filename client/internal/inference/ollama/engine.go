package ollama

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"star-fire/client/internal/config"
	"star-fire/pkg/public"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ollama/ollama/api"
	"github.com/sashabaranov/go-openai"
)

type Engine struct {
	client    *api.Client
	models    map[string]api.ProcessModelResponse
	ollamaURL string
}

func NewEngine(ctx context.Context, ollamaURL string, conf *config.Config) (*Engine, error) {
	engine := &Engine{
		models:    make(map[string]api.ProcessModelResponse),
		ollamaURL: ollamaURL,
	}

	if err := engine.Initialize(ctx, conf); err != nil {
		return nil, err
	}
	return engine, nil
}

func (e *Engine) Name() string {
	return "ollama"
}

func (e *Engine) Initialize(ctx context.Context, conf *config.Config) error {
	httpClient := &http.Client{
		Timeout: 3600 * time.Second,
	}

	ollamaURL, err := url.Parse(e.ollamaURL)
	if err != nil {
		return fmt.Errorf("invalid Ollama URL: %w", err)
	}
	e.client = api.NewClient(ollamaURL, httpClient)

	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := e.client.Heartbeat(timeoutCtx); err != nil {
		return fmt.Errorf("can not connect to ollama engine: %w", err)
	}
	if _, err := e.ListModels(ctx, conf); err != nil {
		log.Printf("alert: load models error: %v", err)
	}

	return nil
}

func (e *Engine) ListModels(ctx context.Context, conf *config.Config) ([]*public.Model, error) {
	var allModels []*public.Model

	// 1. 首先获取所有已下载的模型（用于发现embedding模型）
	allDownloadedResp, err := e.client.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("get ollama downloaded models error: %w", err)
	}

	// 2. 获取正在运行的模型（用于普通模型）
	runningResp, err := e.client.ListRunning(ctx)
	if err != nil {
		log.Printf("get ollama running models error: %v", err)
		// 如果获取运行中的模型失败，继续处理已下载的模型
	}

	// 创建运行中模型的映射，用于快速查找
	runningModels := make(map[string]api.ProcessModelResponse)
	if runningResp != nil {
		for _, model := range runningResp.Models {
			runningModels[model.Name] = model
			// 将运行中的模型信息存储到内部map中
			e.models[model.Name] = model
		}
	}

	// 3. 处理所有已下载的模型
	for _, model := range allDownloadedResp.Models {
		isEmbedding := isOllamaEmbeddingModel(model.Name)
		isRunning := false

		// 检查模型是否正在运行
		if runningModel, exists := runningModels[model.Name]; exists {
			isRunning = true
			// 使用运行中模型的详细信息
			processModel := runningModel
			e.models[model.Name] = processModel
		} else {
			// 对于非运行中的模型，创建基本信息
			processModel := api.ProcessModelResponse{
				Name:   model.Name,
				Size:   model.Size,
				Digest: model.Digest,
			}
			e.models[model.Name] = processModel
		}

		// 4. 决定是否注册模型
		shouldRegister := false
		modelType := "ollama"

		if isEmbedding {
			// Embedding模型：无论是否运行都注册
			shouldRegister = true
			modelType = "embedding"
			log.Printf("Found Ollama embedding model: %s (running: %v)", model.Name, isRunning)
		} else if isRunning {
			// 普通模型：只注册运行中的
			shouldRegister = true
			log.Printf("Found running Ollama model: %s", model.Name)
		} else {
			// 普通模型且未运行：不注册
			log.Printf("Skipping non-running model: %s", model.Name)
		}

		// 5. 如果决定注册，添加到结果列表
		if shouldRegister {
			publicModel := &public.Model{
				Name: model.Name,
				Type: modelType,
				Size: fmt.Sprintf("%d", model.Size),
				Arch: model.Details.QuantizationLevel,
				IPPM: conf.InputTokenPricePerMillion,  // 每百万输入tokens价格
				OPPM: conf.OutputTokenPricePerMillion, // 每百万输出tokens价格
			}
			allModels = append(allModels, publicModel)
		}
	}

	log.Printf("Ollama registered %d models (%d embedding models)",
		len(allModels), countEmbeddingModels(allModels))
	return allModels, nil
}

// countEmbeddingModels 统计embedding模型数量
func countEmbeddingModels(models []*public.Model) int {
	count := 0
	for _, model := range models {
		if model.Type == "embedding" {
			count++
		}
	}
	return count
}

// isOllamaEmbeddingModel 检查Ollama模型是否为embedding模型
func isOllamaEmbeddingModel(modelName string) bool {
	// Ollama中常见的embedding模型
	embeddingModels := []string{
		// BGE模型系列
		"bge-large",
		"bge-base",
		"bge-small",
		"bge-large-en",
		"bge-base-en",
		"bge-small-en",
		"bge-large-zh",
		"bge-base-zh",
		"bge-small-zh",
		"bge-m3",
		"bge-reranker",

		// 其他embedding模型
		"all-minilm",
		"all-mpnet",
		"nomic-embed",
		"snowflake-arctic-embed",
		"mxbai-embed",
		"paraphrase-multilingual",
		"sentence-transformers",

		// 多语言embedding模型
		"multilingual-e5",
		"e5-large",
		"e5-base",
		"e5-small",
	}

	modelLower := strings.ToLower(modelName)

	// 精确匹配或包含匹配
	for _, embeddingModel := range embeddingModels {
		if strings.Contains(modelLower, embeddingModel) {
			return true
		}
	}

	// 关键词匹配
	embeddingKeywords := []string{
		"embed", "embedding", "embeddings",
		"sentence", "semantic", "similarity",
		"retrieval", "rerank", "reranker",
		"vector", "encode", "encoder",
	}

	for _, keyword := range embeddingKeywords {
		if strings.Contains(modelLower, keyword) {
			return true
		}
	}

	return false
}

// 添加转换函数
func convertOpenAIToolToOllama(openaiTool openai.Tool) (api.Tool, error) {
	ollamaTool := api.Tool{
		Type: "function", // OpenAI 的 ToolType 是字符串类型，直接使用 "function"
		Function: api.ToolFunction{
			Name:        openaiTool.Function.Name,
			Description: openaiTool.Function.Description,
		},
	}

	// 转换 Parameters
	if openaiTool.Function.Parameters != nil {
		// 将 OpenAI 的 Parameters 转换为 JSON 再解析到 Ollama 格式
		paramBytes, err := json.Marshal(openaiTool.Function.Parameters)
		if err != nil {
			return api.Tool{}, fmt.Errorf("marshal parameters error: %v", err)
		}

		var paramMap map[string]interface{}
		if err := json.Unmarshal(paramBytes, &paramMap); err != nil {
			return api.Tool{}, fmt.Errorf("unmarshal parameters error: %v", err)
		}

		// 设置基本类型
		if typeVal, ok := paramMap["type"].(string); ok {
			ollamaTool.Function.Parameters.Type = typeVal
		} else {
			ollamaTool.Function.Parameters.Type = "object" // 默认值
		}

		// 设置 Required 字段
		if requiredVal, ok := paramMap["required"].([]interface{}); ok {
			required := make([]string, len(requiredVal))
			for i, req := range requiredVal {
				if reqStr, ok := req.(string); ok {
					required[i] = reqStr
				}
			}
			ollamaTool.Function.Parameters.Required = required
		}

		// 设置 Properties 字段
		if propertiesVal, ok := paramMap["properties"].(map[string]interface{}); ok {
			properties := make(map[string]struct {
				Type        api.PropertyType `json:"type"`
				Items       any              `json:"items,omitempty"`
				Description string           `json:"description"`
				Enum        []any            `json:"enum,omitempty"`
			})

			for propName, propVal := range propertiesVal {
				if propMap, ok := propVal.(map[string]interface{}); ok {
					property := struct {
						Type        api.PropertyType `json:"type"`
						Items       any              `json:"items,omitempty"`
						Description string           `json:"description"`
						Enum        []any            `json:"enum,omitempty"`
					}{}

					// 设置类型
					if typeVal, ok := propMap["type"]; ok {
						if typeStr, ok := typeVal.(string); ok {
							property.Type = api.PropertyType([]string{typeStr})
						} else if typeSlice, ok := typeVal.([]interface{}); ok {
							types := make([]string, len(typeSlice))
							for i, t := range typeSlice {
								if tStr, ok := t.(string); ok {
									types[i] = tStr
								}
							}
							property.Type = api.PropertyType(types)
						}
					}

					// 设置描述
					if descVal, ok := propMap["description"].(string); ok {
						property.Description = descVal
					}

					// 设置枚举
					if enumVal, ok := propMap["enum"].([]interface{}); ok {
						property.Enum = enumVal
					}

					// 设置 Items
					if itemsVal, ok := propMap["items"]; ok {
						property.Items = itemsVal
					}

					properties[propName] = property
				}
			}

			ollamaTool.Function.Parameters.Properties = properties
		}

		// 设置其他字段
		if defsVal, ok := paramMap["$defs"]; ok {
			ollamaTool.Function.Parameters.Defs = defsVal
		}

		if itemsVal, ok := paramMap["items"]; ok {
			ollamaTool.Function.Parameters.Items = itemsVal
		}
	}

	return ollamaTool, nil
}

func (e *Engine) HandleChat(ctx context.Context, fingerprint string,
	request *openai.ChatCompletionRequest, responseConn *websocket.Conn) error {
	ollamaReq := &api.ChatRequest{
		Model:    request.Model,
		Stream:   &request.Stream,
		Messages: convertToOllamaMessages(request.Messages),
		Options: map[string]interface{}{
			"temperature": request.Temperature,
		},
		Tools: make([]api.Tool, 0, len(request.Tools)),
	}
	for _, tool := range request.Tools {
		fmt.Println("Tool:", tool)
		ollamaTool, err := convertOpenAIToolToOllama(tool)
		if err != nil {
			log.Printf("convert tool error: %v", err)
			continue
		}
		ollamaReq.Tools = append(ollamaReq.Tools, ollamaTool)
	}

	respFunc := func(resp api.ChatResponse) error {
		if request.Stream {
			openAIStreamResp := convertToOpenAIStreamResponse(&resp, fingerprint)
			err := responseConn.WriteJSON(public.WSMessage{
				Type:        public.MESSAGE_STREAM,
				Content:     openAIStreamResp,
				FingerPrint: fingerprint,
			})
			if err != nil {
				log.Printf("send message with websocket error: %v", err)
				err = responseConn.WriteJSON(public.WSMessage{
					Type:        public.CLOSE,
					Content:     nil,
					FingerPrint: fingerprint,
				})
				return err
			}
		} else if resp.Done {
			openAIResp := convertToOpenAIResponse(&resp, fingerprint)
			err := responseConn.WriteJSON(public.WSMessage{
				Type:        public.MESSAGE,
				Content:     openAIResp,
				FingerPrint: fingerprint,
			})
			if err != nil {
				log.Printf("send message with websocket error: %v", err)
				return err
			}
			return responseConn.WriteJSON(public.WSMessage{
				Type:        public.CLOSE,
				Content:     nil,
				FingerPrint: fingerprint,
			})
		}

		return nil
	}

	err := e.client.Chat(ctx, ollamaReq, respFunc)
	if err != nil {
		log.Printf("Ollama chat error: %v", err)
		err = responseConn.WriteJSON(public.WSMessage{
			Type:        public.MODEL_ERROR,
			Content:     err.Error(),
			FingerPrint: fingerprint,
		})
		return err
	}

	return nil
}

func convertToOllamaMessages(messages []openai.ChatCompletionMessage) []api.Message {
	ollamaMessages := make([]api.Message, len(messages))
	for i, msg := range messages {
		ollamaMessage := api.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}

		// 处理多媒体内容（包括图片）
		if len(msg.MultiContent) > 0 {
			var images []api.ImageData
			var textContent string

			for _, part := range msg.MultiContent {
				if part.Type == openai.ChatMessagePartTypeText {
					if textContent != "" {
						textContent += "\n"
					}
					textContent += part.Text
				} else if part.Type == openai.ChatMessagePartTypeImageURL && part.ImageURL != nil {
					imageURL := part.ImageURL.URL
					if strings.HasPrefix(imageURL, "data:image/") {
						// 提取 base64 部分
						if commaIndex := strings.Index(imageURL, ","); commaIndex != -1 {
							base64Data := imageURL[commaIndex+1:]
							// 解码 base64 为二进制数据
							if decodedData, err := base64.StdEncoding.DecodeString(base64Data); err == nil {
								images = append(images, decodedData)
							}
						}
					}
				}
			}

			ollamaMessage.Content = textContent
			ollamaMessage.Images = images
		}

		if msg.FunctionCall != nil {
			ollamaMessage.ToolCalls = make([]api.ToolCall, 0, 1)
			var args api.ToolCallFunctionArguments
			if err := json.Unmarshal([]byte(msg.FunctionCall.Arguments), &args); err != nil {
				log.Printf("failed to unmarshal function call arguments: %v", err)
				args = make(api.ToolCallFunctionArguments)
			}
			ollamaMessage.ToolCalls = append(ollamaMessage.ToolCalls, api.ToolCall{
				Function: api.ToolCallFunction{
					Index:     0,
					Name:      msg.FunctionCall.Name,
					Arguments: args,
				},
			})
		}

		ollamaMessages[i] = ollamaMessage
	}
	return ollamaMessages
}

func convertToOpenAIStreamResponse(resp *api.ChatResponse, fingerprint string) openai.ChatCompletionStreamResponse {
	var finishReason openai.FinishReason
	if resp.Done {
		finishReason = openai.FinishReasonStop
	}

	realResp := openai.ChatCompletionStreamResponse{
		ID:      fingerprint,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   resp.Model,
		Choices: []openai.ChatCompletionStreamChoice{
			{
				Index: 0,
				Delta: openai.ChatCompletionStreamChoiceDelta{
					Content: resp.Message.Content,
					Role: func() string {
						if !resp.Done {
							return "assistant"
						}
						return ""
					}(),
					ToolCalls: func() []openai.ToolCall {
						if len(resp.Message.ToolCalls) > 0 {
							toolCalls := make([]openai.ToolCall, len(resp.Message.ToolCalls))
							for i, tc := range resp.Message.ToolCalls {
								toolCalls[i] = openai.ToolCall{
									ID:   fmt.Sprintf("toolcall-%d", i),
									Type: "function",
									Function: openai.FunctionCall{
										Name: tc.Function.Name,
										Arguments: func() string {
											argsBytes, err := json.Marshal(tc.Function.Arguments)
											if err != nil {
												return ""
											}
											return string(argsBytes)
										}(),
									},
								}
							}
							return toolCalls
						}
						return nil
					}(),
				},
				FinishReason: finishReason,
			},
		},
	}
	if resp.Done {
		realResp.Usage = &openai.Usage{
			PromptTokens:     resp.PromptEvalCount,
			CompletionTokens: resp.EvalCount,
			TotalTokens:      resp.PromptEvalCount + resp.EvalCount,
		}
	}
	return realResp
}

func convertToOpenAIResponse(resp *api.ChatResponse, fingerprint string) openai.ChatCompletionResponse {
	realResp := openai.ChatCompletionResponse{
		ID:      fingerprint,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   resp.Model,
		Choices: []openai.ChatCompletionChoice{
			{
				Index: 0,
				Message: openai.ChatCompletionMessage{
					Role:    resp.Message.Role,
					Content: resp.Message.Content,
					FunctionCall: func() *openai.FunctionCall {
						if len(resp.Message.ToolCalls) > 0 {
							tc := resp.Message.ToolCalls[0]
							return &openai.FunctionCall{
								Name: tc.Function.Name,
								Arguments: func() string {
									argsBytes, err := json.Marshal(tc.Function.Arguments)
									if err != nil {
										return ""
									}
									return string(argsBytes)
								}(),
							}
						}
						return nil
					}(),
				},
				FinishReason: openai.FinishReasonStop,
			},
		},
	}
	if resp.Done {
		realResp.Usage = openai.Usage{
			PromptTokens:     resp.PromptEvalCount,
			CompletionTokens: resp.EvalCount,
			TotalTokens:      resp.PromptEvalCount + resp.EvalCount,
		}
	}
	return realResp
}

func (e *Engine) HandleEmbedding(ctx context.Context, fingerprint string,
	request *openai.EmbeddingRequest, responseConn *websocket.Conn) error {
	log.Printf("handle embedding request [%s]: model=%s, input=%v", fingerprint, request.Model, request.Input)

	// 使用Ollama的embedding API
	var inputTexts []string

	// 处理不同类型的输入
	switch input := request.Input.(type) {
	case string:
		inputTexts = []string{input}
	case []string:
		inputTexts = input
	case []interface{}:
		for _, item := range input {
			if str, ok := item.(string); ok {
				inputTexts = append(inputTexts, str)
			}
		}
	default:
		errMsg := "unsupported input type for embedding"
		log.Printf("[%s] %s", fingerprint, errMsg)
		return responseConn.WriteJSON(public.WSMessage{
			Type:        public.MODEL_ERROR,
			Content:     errMsg,
			FingerPrint: fingerprint,
		})
	}

	// 为每个输入文本生成embedding
	var embeddings []openai.Embedding
	for i, text := range inputTexts {
		// 使用Ollama的embed API
		embedReq := &api.EmbedRequest{
			Model: string(request.Model),
			Input: text,
		}

		embedResp, err := e.client.Embed(ctx, embedReq)
		if err != nil {
			errMsg := fmt.Sprintf("create embedding error: %v", err)
			log.Printf("[%s] %s", fingerprint, errMsg)
			return responseConn.WriteJSON(public.WSMessage{
				Type:        public.MODEL_ERROR,
				Content:     errMsg,
				FingerPrint: fingerprint,
			})
		}

		embedding := openai.Embedding{
			Object:    "embedding",
			Index:     i,
			Embedding: embedResp.Embeddings[0], // Ollama返回���embedding向量
		}
		embeddings = append(embeddings, embedding)
	}

	// 构造OpenAI格式的响应
	response := openai.EmbeddingResponse{
		Object: "list",
		Model:  openai.EmbeddingModel(request.Model),
		Data:   embeddings,
		Usage: openai.Usage{
			PromptTokens: calculateInputTokens(inputTexts),
			TotalTokens:  calculateInputTokens(inputTexts),
		},
	}

	log.Printf("[%s] receive embedding response", fingerprint)
	err := responseConn.WriteJSON(public.WSMessage{
		Type:        public.EMBEDDING_RESPONSE,
		Content:     response,
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
	// 检查是否为embedding模型且在可用模型列表中
	if !isOllamaEmbeddingModel(modelName) {
		return false
	}

	// 检查模型是否已下载
	_, exists := e.models[modelName]
	return exists
}

func (e *Engine) SupportsModel(modelName string, conf *config.Config) bool {
	// 检查模型是否在已下载的模型列表中
	if _, exists := e.models[modelName]; exists {
		return true
	}

	// 如果模型不在缓存中，尝试重新获取模型列表
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 更新模型列表
	if _, err := e.ListModels(ctx, conf); err != nil {
		log.Printf("failed to refresh models: %v", err)
		return false
	}

	// 再次检查模型是否存在
	_, exists := e.models[modelName]
	return exists
}

// calculateInputTokens 估算输入文本的token数量
func calculateInputTokens(texts []string) int {
	totalTokens := 0
	for _, text := range texts {
		// 简单估算：平均每4个字符约等于1个token
		tokens := len(text) / 4
		if tokens < 1 {
			tokens = 1
		}
		totalTokens += tokens
	}
	return totalTokens
}
