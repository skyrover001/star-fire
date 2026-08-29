// internal/inference/responses/engine.go
package responses

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

// Engine 实现 OpenAI Responses 上游格式的引擎。
// server 端已把用户请求转成 Responses 格式（WSMessage.Content 是 Responses 请求 JSON），
// 本引擎只需把该 JSON 原样 POST 到上游，并把上游响应（含流式 SSE）原样转发回 server。
type Engine struct {
	baseURL string
	apiKey  string
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
	return "responses"
}

func (e *Engine) Format() string {
	return "responses"
}

func (e *Engine) Initialize(ctx context.Context, conf *config.Config) error {
	log.Printf("responses client created for %s", e.baseURL)
	return nil
}

func (e *Engine) ListModels(ctx context.Context, conf *config.Config) ([]*public.Model, error) {
	// 从上游 /v1/models 发现模型，并标记 UpstreamFormat=responses。
	models := make([]*public.Model, 0)
	url := e.baseURL + "/models"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err == nil {
		httpReq.Header.Set("Authorization", "Bearer "+e.apiKey)
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
							Arch:           "responses",
							Engine:         "responses",
							UpstreamFormat: "responses",
						})
					}
				}
			}
			resp.Body.Close()
		}
	}

	// 兜底：若上游未发现模型，用 registered_models 配置补齐。
	if len(models) == 0 && conf != nil {
		for _, name := range conf.RegisteredModels {
			models = append(models, &public.Model{
				Name:           name,
				Type:           "chat",
				Size:           "unknown",
				Arch:           "responses",
				Engine:         "responses",
				UpstreamFormat: "responses",
			})
		}
	}
	return models, nil
}

func (e *Engine) SupportsModel(modelName string, conf *config.Config) bool {
	return true
}

func (e *Engine) HandleChat(ctx context.Context, fingerprint string,
	request *public.ExtendedChatRequest,
	responseConn *websocket.Conn) error {
	rawBody := request.Thinking
	if len(rawBody) == 0 {
		return fmt.Errorf("responses engine requires raw body")
	}

	var probe struct {
		Stream bool `json:"stream"`
	}
	_ = json.Unmarshal(rawBody, &probe)

	url := e.baseURL + "/v1/responses"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(rawBody))
	if err != nil {
		return fmt.Errorf("create responses request error: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+e.apiKey)
	if probe.Stream {
		httpReq.Header.Set("Accept", "text/event-stream")
	}

	// ===== 链路日志：client 发给上游后端的 Responses 请求体 =====
	// log.Printf("[TRACE] client send to backend [%s] baseURL=%s body=%s", fingerprint, e.baseURL, string(rawBody))

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		errMsg := fmt.Sprintf("responses request error: %v", err)
		log.Printf("[%s] %s", fingerprint, errMsg)
		_ = responseConn.WriteJSON(public.WSMessage{Type: public.MODEL_ERROR, Content: errMsg, FingerPrint: fingerprint})
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		errMsg := fmt.Sprintf("responses upstream error %d: %s", resp.StatusCode, string(body))
		log.Printf("[%s] %s", fingerprint, errMsg)
		_ = responseConn.WriteJSON(public.WSMessage{Type: public.MODEL_ERROR, Content: errMsg, FingerPrint: fingerprint})
		return fmt.Errorf("%s", errMsg)
	}

	if !probe.Stream {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("read responses response error: %w", err)
		}
		var content map[string]interface{}
		if err := json.Unmarshal(body, &content); err != nil {
			return fmt.Errorf("unmarshal responses response error: %w", err)
		}
		if err := responseConn.WriteJSON(public.WSMessage{Type: public.MESSAGE, Content: content, FingerPrint: fingerprint}); err != nil {
			return err
		}
		return responseConn.WriteJSON(public.WSMessage{Type: public.CLOSE, Content: nil, FingerPrint: fingerprint})
	}

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("read responses stream error: %w", err)
		}
		line = bytes.TrimSpace(line)
		if len(line) == 0 || !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}
		data := bytes.TrimPrefix(line, []byte("data: "))
		if bytes.Equal(data, []byte("[DONE]")) {
			break
		}
		// ===== 链路日志：client 收到的上游后端流式 chunk（含 function_call 的 call_id/name）=====
		// log.Printf("[TRACE] client backend stream chunk [%s] data=%s", fingerprint, string(data))
		var content map[string]interface{}
		if err := json.Unmarshal(data, &content); err != nil {
			log.Printf("[%s] unmarshal responses chunk error: %v", fingerprint, err)
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
	return fmt.Errorf("responses engine does not support embedding")
}

func (e *Engine) SupportsEmbedding(modelName string) bool {
	return false
}
