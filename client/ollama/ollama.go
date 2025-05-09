package ollama

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/ollama/ollama/api"
	"github.com/sashabaranov/go-openai"
	"log"
	"time"
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

func ConvertOllamaResponseToOpenAIResponse(respone *api.ChatResponse) (*openai.ChatCompletionResponse, error) {
	// Convert Ollama response to OpenAI response
	openaiResponse := openai.ChatCompletionResponse{
		ID:      string(time.Now().Unix()),
		Object:  "chat.completion",
		Created: respone.CreatedAt.Unix(),
		Model:   respone.Model,
		Choices: make([]openai.ChatCompletionChoice, 0),
		Usage: openai.Usage{
			PromptTokens:     respone.PromptEvalCount,
			CompletionTokens: respone.EvalCount,
			TotalTokens:      respone.PromptEvalCount + respone.EvalCount,
		},
	}
	openaiResponse.Choices = append(openaiResponse.Choices, openai.ChatCompletionChoice{
		Index: 0,
		Message: openai.ChatCompletionMessage{
			Role:             respone.Message.Role,
			Content:          respone.Message.Content,
			Refusal:          "",
			MultiContent:     nil,
			Name:             "",
			ReasoningContent: "",
			FunctionCall:     nil,
			ToolCalls:        nil, // 如果ollama支持函数调用，可以在这里填充
			ToolCallID:       "",
		},
		FinishReason:         "",
		LogProbs:             nil,
		ContentFilterResults: openai.ContentFilterResults{},
	})
	return &openaiResponse, nil
}
