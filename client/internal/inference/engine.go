// internal/inference/engine.go
package inference

import (
	"context"
	"star-fire/pkg/public"

	"github.com/gorilla/websocket"
	"github.com/sashabaranov/go-openai"
)

type Engine interface {
	Name() string
	Initialize(ctx context.Context) error
	ListModels(ctx context.Context) ([]*public.Model, error)
	SupportsModel(modelName string) bool
	HandleChat(ctx context.Context, fingerprint string,
		request *openai.ChatCompletionRequest,
		responseConn *websocket.Conn) error
}
