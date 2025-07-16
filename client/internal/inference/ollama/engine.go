package ollama

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"star-fire/client/internal/config"
	"star-fire/pkg/public"
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

func (e *Engine) HandleChat(ctx context.Context, fingerprint string,
	request *openai.ChatCompletionRequest, responseConn *websocket.Conn) error {
	ollamaReq := &api.ChatRequest{
		Model:    request.Model,
		Stream:   &request.Stream,
		Messages: convertToOllamaMessages(request.Messages),
		Options: map[string]interface{}{
			"temperature": request.Temperature,
		},
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
		ollamaMessages[i] = api.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
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
						if !resp.Done && resp.Message.Content == resp.Message.Content {
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
