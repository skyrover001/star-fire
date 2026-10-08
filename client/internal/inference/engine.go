// internal/inference/engine.go
package inference

import (
	"context"
	"star-fire/client/internal/config"
	"star-fire/pkg/public"

	"github.com/gorilla/websocket"
	"github.com/sashabaranov/go-openai"
)

type Engine interface {
	Name() string
	// Format 返回该引擎的上游协议格式（openai | anthropic | responses）。
	// 用于多格式路由：server 发送的 WSMessage.Format 决定选哪个引擎。
	Format() string
	Initialize(ctx context.Context, conf *config.Config) error
	ListModels(ctx context.Context, conf *config.Config) ([]*public.Model, error)
	SupportsModel(modelName string, conf *config.Config) bool
	HandleChat(ctx context.Context, fingerprint string,
		request *public.ExtendedChatRequest,
		responseConn *websocket.Conn) error
	// 添加embedding支持
	HandleEmbedding(ctx context.Context, fingerprint string,
		request *openai.EmbeddingRequest,
		responseConn *websocket.Conn) error
	SupportsEmbedding(modelName string) bool
}
