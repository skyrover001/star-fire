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
	resp, err := e.client.ListRunning(ctx)
	if err != nil {
		return nil, fmt.Errorf("get ollama running models error: %w", err)
	}

	models := make([]*public.Model, 0, len(resp.Models))
	for _, model := range resp.Models {
		e.models[model.Name] = model
		publicModel := &public.Model{
			Name: model.Name,
			Type: "ollama",
			Size: fmt.Sprintf("%d", model.Size),
			Arch: model.Details.QuantizationLevel,
			PPM:  conf.PricePerMillion,
		}
		models = append(models, publicModel)
	}

	log.Println("Ollama models:", models)
	return models, nil
}

func (e *Engine) SupportsModel(modelName string, conf *config.Config) bool {
	if _, ok := e.models[modelName]; ok {
		return true
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := e.ListModels(ctx, conf); err != nil {
		return false
	}
	_, ok := e.models[modelName]
	return ok
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
