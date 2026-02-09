package claude

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"star-fire/client/internal/config"
	"star-fire/pkg/public"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/sashabaranov/go-openai"
)

type Engine struct {
	apiKey  string
	baseURL string
	models  []ClaudeModel
}

type ClaudeModel struct {
	ID   string
	Name string
}

// Claude API message structures
type ClaudeMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type ClaudeRequest struct {
	Model       string          `json:"model"`
	Messages    []ClaudeMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float32         `json:"temperature,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	System      string          `json:"system,omitempty"`
}

type ClaudeResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason,omitempty"`
	StopSequence string `json:"stop_sequence,omitempty"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type ClaudeStreamResponse struct {
	Type         string `json:"type"`
	Index        int    `json:"index,omitempty"`
	Delta        *Delta `json:"delta,omitempty"`
	Message      *ClaudeResponse `json:"message,omitempty"`
	Usage        *UsageInfo `json:"usage,omitempty"`
	ContentBlock *ContentBlock `json:"content_block,omitempty"`
}

type Delta struct {
	Type         string `json:"type,omitempty"`
	Text         string `json:"text,omitempty"`
	StopReason   string `json:"stop_reason,omitempty"`
	StopSequence string `json:"stop_sequence,omitempty"`
}

type UsageInfo struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
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
	return "claude"
}

func (e *Engine) Initialize(ctx context.Context, conf *config.Config) error {
	if e.baseURL == "" {
		e.baseURL = "https://api.anthropic.com/v1"
	}

	// Initialize with known Claude models
	e.models = []ClaudeModel{
		{ID: "claude-opus-4-6", Name: "claude-opus-4-6"},
		{ID: "claude-3-5-sonnet-20241022", Name: "claude-3-5-sonnet-20241022"},
		{ID: "claude-3-5-haiku-20241022", Name: "claude-3-5-haiku-20241022"},
		{ID: "claude-3-opus-20240229", Name: "claude-3-opus-20240229"},
		{ID: "claude-3-sonnet-20240229", Name: "claude-3-sonnet-20240229"},
		{ID: "claude-3-haiku-20240307", Name: "claude-3-haiku-20240307"},
	}

	log.Printf("Initialized Claude engine with %d models", len(e.models))
	return nil
}

func (e *Engine) ListModels(ctx context.Context, conf *config.Config) ([]*public.Model, error) {
	publicModels := make([]*public.Model, 0)
	for _, model := range e.models {
		publicModel := &public.Model{
			Name:   model.ID,
			Type:   "text",
			Size:   "unknown",
			Arch:   "claude",
			Engine: "claude",
			IPPM:   conf.InputTokenPricePerMillion,
			OPPM:   conf.OutputTokenPricePerMillion,
		}
		publicModels = append(publicModels, publicModel)
	}
	return publicModels, nil
}

func (e *Engine) SupportsModel(modelName string, conf *config.Config) bool {
	for _, model := range e.models {
		if model.ID == modelName {
			return true
		}
	}
	// Also check if it starts with "claude"
	return strings.HasPrefix(strings.ToLower(modelName), "claude")
}

// Convert OpenAI chat request to Claude format
func (e *Engine) convertToClaudeRequest(request *openai.ChatCompletionRequest) (*ClaudeRequest, error) {
	claudeReq := &ClaudeRequest{
		Model:     request.Model,
		MaxTokens: 4096, // Default max tokens
		Stream:    request.Stream,
	}

	if request.Temperature != 0 {
		claudeReq.Temperature = request.Temperature
	}

	if request.MaxTokens != 0 {
		claudeReq.MaxTokens = request.MaxTokens
	}

	// Extract system message and convert messages
	claudeMessages := []ClaudeMessage{}
	for _, msg := range request.Messages {
		if msg.Role == "system" {
			// Claude handles system messages separately
			claudeReq.System = msg.Content
		} else {
			// Convert message content to Claude format
			var contentBytes []byte
			
			// Check if MultiContent is used (for complex messages)
			if len(msg.MultiContent) > 0 {
				// Use MultiContent if available
				contentBytes, _ = json.Marshal(msg.MultiContent)
			} else {
				// Simple string content, wrap in Claude format
				claudeContent := []map[string]string{
					{"type": "text", "text": msg.Content},
				}
				contentBytes, _ = json.Marshal(claudeContent)
			}

			claudeMessages = append(claudeMessages, ClaudeMessage{
				Role:    msg.Role,
				Content: contentBytes,
			})
		}
	}

	claudeReq.Messages = claudeMessages
	return claudeReq, nil
}

// Convert Claude response to OpenAI format
func (e *Engine) convertToOpenAIResponse(claudeResp *ClaudeResponse) *openai.ChatCompletionResponse {
	var content string
	if len(claudeResp.Content) > 0 {
		content = claudeResp.Content[0].Text
	}

	return &openai.ChatCompletionResponse{
		ID:      claudeResp.ID,
		Object:  "chat.completion",
		Created: 0,
		Model:   claudeResp.Model,
		Choices: []openai.ChatCompletionChoice{
			{
				Index: 0,
				Message: openai.ChatCompletionMessage{
					Role:    claudeResp.Role,
					Content: content,
				},
				FinishReason: openai.FinishReason(claudeResp.StopReason),
			},
		},
		Usage: openai.Usage{
			PromptTokens:     claudeResp.Usage.InputTokens,
			CompletionTokens: claudeResp.Usage.OutputTokens,
			TotalTokens:      claudeResp.Usage.InputTokens + claudeResp.Usage.OutputTokens,
		},
	}
}

func (e *Engine) HandleChat(ctx context.Context, fingerprint string,
	request *openai.ChatCompletionRequest, responseConn *websocket.Conn) error {
	log.Printf("handle chat request [%s]: model=%s, stream=%v", fingerprint, request.Model, request.Stream)

	// Convert OpenAI request to Claude format
	claudeReq, err := e.convertToClaudeRequest(request)
	if err != nil {
		errMsg := fmt.Sprintf("convert request error: %v", err)
		log.Printf("[%s] %s", fingerprint, errMsg)
		return responseConn.WriteJSON(public.WSMessage{
			Type:        public.MODEL_ERROR,
			Content:     errMsg,
			FingerPrint: fingerprint,
		})
	}

	if claudeReq.Stream {
		return e.handleStreamRequest(ctx, fingerprint, claudeReq, responseConn)
	}
	return e.handleNonStreamRequest(ctx, fingerprint, claudeReq, responseConn)
}

func (e *Engine) handleNonStreamRequest(ctx context.Context, fingerprint string,
	claudeReq *ClaudeRequest, responseConn *websocket.Conn) error {

	reqBody, err := json.Marshal(claudeReq)
	if err != nil {
		return fmt.Errorf("marshal request error: %w", err)
	}

	url := strings.TrimSuffix(e.baseURL, "/") + "/messages"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("create http request error: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", e.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		errMsg := fmt.Sprintf("create chat completion error: %v", err)
		log.Printf("[%s] %s", fingerprint, errMsg)
		return responseConn.WriteJSON(public.WSMessage{
			Type:        public.MODEL_ERROR,
			Content:     errMsg,
			FingerPrint: fingerprint,
		})
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		errMsg := fmt.Sprintf("API error: status=%d, body=%s", resp.StatusCode, string(body))
		log.Printf("[%s] %s", fingerprint, errMsg)
		return responseConn.WriteJSON(public.WSMessage{
			Type:        public.MODEL_ERROR,
			Content:     errMsg,
			FingerPrint: fingerprint,
		})
	}

	var claudeResp ClaudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&claudeResp); err != nil {
		return fmt.Errorf("decode response error: %w", err)
	}

	// Convert to OpenAI format and send
	openaiResp := e.convertToOpenAIResponse(&claudeResp)
	if err := responseConn.WriteJSON(public.WSMessage{
		Type:        public.MESSAGE,
		Content:     openaiResp,
		FingerPrint: fingerprint,
	}); err != nil {
		log.Printf("[%s] send response error: %v", fingerprint, err)
		return err
	}

	log.Printf("[%s] send close message", fingerprint)
	return responseConn.WriteJSON(public.WSMessage{
		Type:        public.CLOSE,
		Content:     nil,
		FingerPrint: fingerprint,
	})
}

func (e *Engine) handleStreamRequest(ctx context.Context, fingerprint string,
	claudeReq *ClaudeRequest, responseConn *websocket.Conn) error {

	reqBody, err := json.Marshal(claudeReq)
	if err != nil {
		return fmt.Errorf("marshal request error: %w", err)
	}

	url := strings.TrimSuffix(e.baseURL, "/") + "/messages"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("create http request error: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", e.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Accept", "text/event-stream")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		errMsg := fmt.Sprintf("create chat completion stream error: %v", err)
		log.Printf("[%s] %s", fingerprint, errMsg)
		return responseConn.WriteJSON(public.WSMessage{
			Type:        public.MODEL_ERROR,
			Content:     errMsg,
			FingerPrint: fingerprint,
		})
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		errMsg := fmt.Sprintf("API error: status=%d, body=%s", resp.StatusCode, string(body))
		log.Printf("[%s] %s", fingerprint, errMsg)
		return responseConn.WriteJSON(public.WSMessage{
			Type:        public.MODEL_ERROR,
			Content:     errMsg,
			FingerPrint: fingerprint,
		})
	}

	// Read SSE stream
	reader := bufio.NewReader(resp.Body)
	var accumulatedContent strings.Builder
	var totalInputTokens, totalOutputTokens int
	var stopReason string

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				log.Printf("[%s] stream EOF", fingerprint)
				break
			}
			errMsg := fmt.Sprintf("read stream error: %v", err)
			log.Printf("[%s] %s", fingerprint, errMsg)
			return responseConn.WriteJSON(public.WSMessage{
				Type:        public.MODEL_ERROR,
				Content:     errMsg,
				FingerPrint: fingerprint,
			})
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		// SSE format: "data: {...}"
		if !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}

		data := bytes.TrimPrefix(line, []byte("data: "))

		var streamResp ClaudeStreamResponse
		if err := json.Unmarshal(data, &streamResp); err != nil {
			log.Printf("[%s] unmarshal chunk error: %v, data: %s", fingerprint, err, string(data))
			continue
		}

		// Handle different event types
		switch streamResp.Type {
		case "content_block_delta":
			if streamResp.Delta != nil && streamResp.Delta.Text != "" {
				accumulatedContent.WriteString(streamResp.Delta.Text)

				// Send as OpenAI stream format
				openaiStream := &openai.ChatCompletionStreamResponse{
					ID:      fingerprint,
					Object:  "chat.completion.chunk",
					Created: 0,
					Model:   claudeReq.Model,
					Choices: []openai.ChatCompletionStreamChoice{
						{
							Index: streamResp.Index,
							Delta: openai.ChatCompletionStreamChoiceDelta{
								Content: streamResp.Delta.Text,
							},
						},
					},
				}

				if err := responseConn.WriteJSON(public.WSMessage{
					Type:        public.MESSAGE_STREAM,
					Content:     openaiStream,
					FingerPrint: fingerprint,
				}); err != nil {
					log.Printf("[%s] send response error: %v", fingerprint, err)
					return err
				}
			}

		case "message_delta":
			if streamResp.Delta != nil && streamResp.Delta.StopReason != "" {
				stopReason = streamResp.Delta.StopReason
			}
			if streamResp.Usage != nil {
				totalInputTokens = streamResp.Usage.InputTokens
				totalOutputTokens = streamResp.Usage.OutputTokens
			}

		case "message_stop":
			// Send final message with finish reason and usage
			openaiStream := &openai.ChatCompletionStreamResponse{
				ID:      fingerprint,
				Object:  "chat.completion.chunk",
				Created: 0,
				Model:   claudeReq.Model,
				Choices: []openai.ChatCompletionStreamChoice{
					{
						Index: 0,
						Delta: openai.ChatCompletionStreamChoiceDelta{},
						FinishReason: openai.FinishReason(stopReason),
					},
				},
				Usage: &openai.Usage{
					PromptTokens:     totalInputTokens,
					CompletionTokens: totalOutputTokens,
					TotalTokens:      totalInputTokens + totalOutputTokens,
				},
			}

			if err := responseConn.WriteJSON(public.WSMessage{
				Type:        public.MESSAGE_STREAM,
				Content:     openaiStream,
				FingerPrint: fingerprint,
			}); err != nil {
				log.Printf("[%s] send final response error: %v", fingerprint, err)
			}
			break
		}
	}

	log.Printf("[%s] send close message", fingerprint)
	return responseConn.WriteJSON(public.WSMessage{
		Type:        public.CLOSE,
		Content:     nil,
		FingerPrint: fingerprint,
	})
}

func (e *Engine) HandleEmbedding(ctx context.Context, fingerprint string,
	request *openai.EmbeddingRequest, responseConn *websocket.Conn) error {
	// Claude doesn't support embedding models
	errMsg := "Claude does not support embedding models"
	log.Printf("[%s] %s", fingerprint, errMsg)
	return responseConn.WriteJSON(public.WSMessage{
		Type:        public.MODEL_ERROR,
		Content:     errMsg,
		FingerPrint: fingerprint,
	})
}

func (e *Engine) SupportsEmbedding(modelName string) bool {
	// Claude doesn't support embedding
	return false
}
