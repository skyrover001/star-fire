// internal/inference/anthropic/engine.go
package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"log"
	"star-fire/client/internal/config"
	"star-fire/pkg/public"

	"github.com/gorilla/websocket"
	"github.com/sashabaranov/go-openai"
)

// Engine 实现 Anthropic Messages 上游格式的引擎。
// server 端已把用户请求转成 Anthropic 格式（WSMessage.Content 是 Anthropic 请求 JSON），
// 本引擎只需把该 JSON 原样 POST 到上游，并把上游响应（含流式 SSE）原样转发回 server。
type Engine struct {
	baseURL string
	apiKey  string
	models  []string
}

func NewEngine(ctx context.Context, apiKey, baseURL string, conf *config.Config) (*Engine, error) {
	engine := &Engine{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
	if err := engine.Initialize(ctx, conf); err != nil {
		return nil, err
	}
	return engine, nil
}

func (e *Engine) Name() string {
	return "anthropic"
}

func (e *Engine) Format() string {
	return "anthropic"
}

func (e *Engine) Initialize(ctx context.Context, conf *config.Config) error {
	// Anthropic 没有标准的模型列表端点，这里不做模型发现。
	// 模型列表由 server 端根据用户请求的 model 决定。
	log.Printf("anthropic client created for %s", e.baseURL)
	return nil
}

func (e *Engine) ListModels(ctx context.Context, conf *config.Config) ([]*public.Model, error) {
	// 尝试从上游 /v1/models 发现模型（多数网关支持），并标记 UpstreamFormat=anthropic。
	models := make([]*public.Model, 0)
	url := e.baseURL + "/models"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err == nil {
		httpReq.Header.Set("x-api-key", e.apiKey)
		httpReq.Header.Set("anthropic-version", "2023-06-01")
		resp, err := http.DefaultClient.Do(httpReq)
		if err == nil {
			if resp.StatusCode == http.StatusOK {
				var list struct {
					Data []struct {
						ID string `json:"id"`
					} `json:"data"`
				}
				if json.NewDecoder(resp.Body).Decode(&list) == nil {
					for _, m := range list.Data {
						models = append(models, &public.Model{
							Name:           m.ID,
							Type:           "chat",
							Size:           "unknown",
							Arch:           "anthropic",
							Engine:         "anthropic",
							UpstreamFormat: "anthropic",
						})
					}
				}
			}
			resp.Body.Close()
		}
	}

	// 兜底：若上游未发现模型，用 registered_models 配置补齐（保证模型能注册到 server）。
	if len(models) == 0 && conf != nil {
		for _, name := range conf.RegisteredModels {
			models = append(models, &public.Model{
				Name:           name,
				Type:           "chat",
				Size:           "unknown",
				Arch:           "anthropic",
				Engine:         "anthropic",
				UpstreamFormat: "anthropic",
			})
		}
	}
	return models, nil
}

func (e *Engine) SupportsModel(modelName string, conf *config.Config) bool {
	// 该引擎支持所有模型（由 server 端按 model 路由到本引擎）。
	return true
}

func (e *Engine) HandleChat(ctx context.Context, fingerprint string,
	request *public.ExtendedChatRequest,
	responseConn *websocket.Conn) error {
	// 注意：多格式模式下，server 发送的 Content 是 Anthropic 格式的原始 JSON。
	// 但 HandleChat 接口接收的是 ExtendedChatRequest。这里我们通过 WSMessage.Format
	// 判断。为兼容，我们直接使用 request 的原始 body。
	// 实际上多格式路径下，client 端 handleChatMessage 需要根据 Format 选择引擎并传原始 JSON。
	// 这里实现一个基于原始 JSON 的调用（通过 request.BuildRequestBody 不可行，因为那是 OpenAI 格式）。
	// 因此本引擎的 HandleChat 由 client 端多格式分发逻辑调用，传入的是 Anthropic 原始 JSON。
	return e.handleRaw(ctx, fingerprint, request, responseConn)
}

// handleRaw 处理 Anthropic 原始 JSON 请求。
// 多格式路径下 client 端把原始 Anthropic JSON 放入 request.Thinking 字段传递。
func (e *Engine) handleRaw(ctx context.Context, fingerprint string, request *public.ExtendedChatRequest, responseConn *websocket.Conn) error {
	rawBody := request.Thinking
	if len(rawBody) == 0 {
		return fmt.Errorf("anthropic engine requires raw body")
	}

	// 判断是否流式
	var probe struct {
		Stream bool `json:"stream"`
	}
	_ = json.Unmarshal(rawBody, &probe)

	url := e.baseURL + "/v1/messages"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(rawBody))
	if err != nil {
		return fmt.Errorf("create anthropic request error: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", e.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	if probe.Stream {
		httpReq.Header.Set("Accept", "text/event-stream")
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		errMsg := fmt.Sprintf("anthropic request error: %v", err)
		log.Printf("[%s] %s", fingerprint, errMsg)
		_ = responseConn.WriteJSON(public.WSMessage{Type: public.MODEL_ERROR, Content: errMsg, FingerPrint: fingerprint})
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		errMsg := fmt.Sprintf("anthropic upstream error %d: %s", resp.StatusCode, string(body))
		log.Printf("[%s] %s", fingerprint, errMsg)
		_ = responseConn.WriteJSON(public.WSMessage{Type: public.MODEL_ERROR, Content: errMsg, FingerPrint: fingerprint})
		return fmt.Errorf("%s", errMsg)
	}

	if !probe.Stream {
		// 非流式：读取完整响应，作为 MESSAGE 转发
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("read anthropic response error: %w", err)
		}
		var content map[string]interface{}
		if err := json.Unmarshal(body, &content); err != nil {
			return fmt.Errorf("unmarshal anthropic response error: %w", err)
		}
		if err := responseConn.WriteJSON(public.WSMessage{Type: public.MESSAGE, Content: content, FingerPrint: fingerprint}); err != nil {
			return err
		}
		return responseConn.WriteJSON(public.WSMessage{Type: public.CLOSE, Content: nil, FingerPrint: fingerprint})
	}

	// 流式：逐行读取 SSE，作为 MESSAGE_STREAM 转发
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("read anthropic stream error: %w", err)
		}
		line = bytes.TrimSpace(line)
		if len(line) == 0 || !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}
		data := bytes.TrimPrefix(line, []byte("data: "))
		if bytes.Equal(data, []byte("[DONE]")) {
			break
		}
		var content map[string]interface{}
		if err := json.Unmarshal(data, &content); err != nil {
			log.Printf("[%s] unmarshal anthropic chunk error: %v", fingerprint, err)
			continue
		}
		if err := responseConn.WriteJSON(public.WSMessage{Type: public.MESSAGE_STREAM, Content: content, FingerPrint: fingerprint}); err != nil {
			return err
		}
	}
	return responseConn.WriteJSON(public.WSMessage{Type: public.CLOSE, Content: nil, FingerPrint: fingerprint})
}

func (e *Engine) HandleEmbedding(ctx context.Context, fingerprint string,
	request *openai.EmbeddingRequest,
	responseConn *websocket.Conn) error {
	return fmt.Errorf("anthropic engine does not support embedding")
}

func (e *Engine) SupportsEmbedding(modelName string) bool {
	return false
}
