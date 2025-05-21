package openai

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/sashabaranov/go-openai"
	"log"
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

func NewEngine(ctx context.Context, apiKey, baseURL string) (*Engine, error) {
	engine := &Engine{
		apiKey:  apiKey,
		baseURL: baseURL,
	}
	if err := engine.Initialize(ctx); err != nil {
		return nil, err
	}
	return engine, nil
}

func (e *Engine) Name() string {
	return "openai"
}

func (e *Engine) Initialize(ctx context.Context) error {
	config := openai.DefaultConfig(e.apiKey)
	if e.baseURL != "" && e.baseURL != "https://api.openai.com/v1" {
		config.BaseURL = e.baseURL
	}
	e.client = openai.NewClientWithConfig(config)
	fmt.Println("openai client created", e.client, config.BaseURL, config.APIType, e.apiKey, e.baseURL)
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

func (e *Engine) ListModels(ctx context.Context) ([]*public.Model, error) {
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
			Arch: model.Object}
		publicModels = append(publicModels, publicModel)
	}
	return publicModels, nil
}

func (e *Engine) SupportsModel(modelName string) bool {
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
			responseConn.WriteJSON(public.WSMessage{
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

				responseConn.WriteJSON(public.WSMessage{
					Type:        public.MODEL_ERROR,
					Content:     errMsg,
					FingerPrint: fingerprint,
				})
				return err
			}
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
			responseConn.WriteJSON(public.WSMessage{
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
