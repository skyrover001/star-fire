package ollama

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/ollama/ollama/api"
	"github.com/sashabaranov/go-openai"
)

const ENV_CLIENT = "ENV"

type ChatRequest struct {
	FingerPrint string
	Request     *api.ChatRequest
}

type ChatClient struct {
	ResponseConn *websocket.Conn
	ChatReq      *ChatRequest
	ReqClient    *api.Client
}

type Client struct {
	Models     []api.ProcessModelResponse `json:"models"`
	Type       string                     `json:"type"`
	ChatClient *ChatClient
}

type Ollama struct {
	Clients []*Client
}

func (o *Ollama) Init() {
	fmt.Println("init ollama client")
	oClient, err := api.ClientFromEnvironment()
	if err != nil {
		log.Println("ollama client init error:", err)
	}
	ctx := context.Background()
	models, err := oClient.ListRunning(ctx)
	if err != nil {
		log.Println("ollama client list error:", err)
		return
	}
	client := Client{
		Models:     models.Models,
		Type:       ENV_CLIENT,
		ChatClient: nil,
	}
	o.Clients = []*Client{&client}
}

func ConvertOpenAIToOllamaRequest(request *openai.ChatCompletionRequest) (*api.ChatRequest, error) {
	// Convert OpenAI request to Ollama request
	ollamaRequest := api.ChatRequest{
		Model:    request.Model,
		Messages: make([]api.Message, len(request.Messages)),
		Stream:   &request.Stream,
		Tools:    make([]api.Tool, len(request.Functions)),
	}
	for i, message := range request.Messages {
		ollamaRequest.Messages[i] = api.Message{
			Role:    message.Role,
			Content: message.Content,
		}
	}
	return &ollamaRequest, nil
}

func ConvertOllamaResponseToOpenAIResponse(response *api.ChatResponse, stream bool) (*openai.ChatCompletionResponse, *openai.ChatCompletionStreamResponse, error) {
	if stream {
		// For streaming responses, use Delta instead of Message
		choices := make([]openai.ChatCompletionStreamChoice, 1)
		choices[0] = openai.ChatCompletionStreamChoice{
			Index: 0,
			Delta: openai.ChatCompletionStreamChoiceDelta{
				Role:    response.Message.Role,
				Content: response.Message.Content,
			},
			FinishReason: openai.FinishReason(response.DoneReason),
		}

		return nil, &openai.ChatCompletionStreamResponse{
			ID:      fmt.Sprintf("chatcmpl-%v", uuid.NewString()),
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Choices: choices,
			Model:   response.Model,
			Usage: &openai.Usage{
				PromptTokens:     response.PromptEvalCount,
				CompletionTokens: response.EvalCount,
				TotalTokens:      response.PromptEvalCount + response.EvalCount,
			},
		}, nil

	}

	// For non-streaming responses, use Message as before
	choices := make([]openai.ChatCompletionChoice, 1)
	choices[0] = openai.ChatCompletionChoice{
		Index: 0,
		Message: openai.ChatCompletionMessage{
			Role:    response.Message.Role,
			Content: response.Message.Content,
		},
		FinishReason: openai.FinishReason(response.DoneReason),
	}

	return &openai.ChatCompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-%v", uuid.NewString()),
		Object:  "chat.completion", // Fixed the trailing dot
		Created: time.Now().Unix(),
		Choices: choices,
		Model:   response.Model,
		Usage: openai.Usage{
			PromptTokens:     response.PromptEvalCount,
			CompletionTokens: response.EvalCount,
			TotalTokens:      response.PromptEvalCount + response.EvalCount,
		},
	}, nil, nil
}
