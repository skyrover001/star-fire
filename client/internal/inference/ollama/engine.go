package ollama

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
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

func NewEngine(ctx context.Context, ollamaURL string) (*Engine, error) {
	engine := &Engine{
		models:    make(map[string]api.ProcessModelResponse),
		ollamaURL: ollamaURL,
	}

	if err := engine.Initialize(ctx); err != nil {
		return nil, err
	}
	return engine, nil
}

func (e *Engine) Name() string {
	return "ollama"
}

func (e *Engine) Initialize(ctx context.Context) error {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
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
	if _, err := e.ListModels(ctx); err != nil {
		log.Printf("alert: load models error: %v", err)
	}

	return nil
}

func (e *Engine) ListModels(ctx context.Context) ([]*public.Model, error) {
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
		}
		models = append(models, publicModel)
	}

	log.Println("Ollama models:", models)
	return models, nil
}

func (e *Engine) SupportsModel(modelName string) bool {
	if _, ok := e.models[modelName]; ok {
		return true
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := e.ListModels(ctx); err != nil {
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
				responseConn.WriteJSON(public.WSMessage{
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

		if resp.Done && request.Stream {
			finishMsg := constructFinishMessage(fingerprint, request.Model)
			return responseConn.WriteJSON(finishMsg)
		}

		return nil
	}

	err := e.client.Chat(ctx, ollamaReq, respFunc)
	if err != nil {
		log.Printf("Ollama chat error: %v", err)
		responseConn.WriteJSON(public.WSMessage{
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

func convertToOpenAIStreamResponse(resp *api.ChatResponse, fingerprint string) map[string]interface{} {
	return map[string]interface{}{
		"id":      fingerprint,
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   resp.Model,
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"delta": map[string]string{
					"content": resp.Message.Content,
				},
				"finish_reason": nil,
			},
		},
	}
}

func convertToOpenAIResponse(resp *api.ChatResponse, fingerprint string) map[string]interface{} {
	return map[string]interface{}{
		"id":      fingerprint,
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   resp.Model,
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]string{
					"role":    "assistant",
					"content": resp.Message.Content,
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]int{
			"prompt_tokens":     resp.PromptEvalCount,
			"completion_tokens": resp.EvalCount,
			"total_tokens":      resp.PromptEvalCount + resp.EvalCount,
		},
	}
}

func constructFinishMessage(fingerprint, model string) public.WSMessage {
	return public.WSMessage{
		Type: public.MESSAGE,
		Content: map[string]interface{}{
			"id":      fingerprint,
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   model,
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"delta":         map[string]string{},
					"finish_reason": "stop",
				},
			},
		},
		FingerPrint: fingerprint,
	}
}
