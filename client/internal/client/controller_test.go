package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"star-fire/client/internal/config"
	"star-fire/client/internal/inference"
	"star-fire/pkg/public"

	"github.com/gorilla/websocket"
	openaiapi "github.com/sashabaranov/go-openai"
)

type fakeEngine struct {
	name   string
	models []*public.Model
}

type concurrentFakeEngine struct {
	fakeEngine
	active    int
	maxActive int
	mu        sync.Mutex
}

func (engine *concurrentFakeEngine) ListModels(ctx context.Context, cfg *config.Config) ([]*public.Model, error) {
	engine.mu.Lock()
	engine.active++
	if engine.active > engine.maxActive {
		engine.maxActive = engine.active
	}
	engine.mu.Unlock()
	time.Sleep(10 * time.Millisecond)
	models, err := engine.fakeEngine.ListModels(ctx, cfg)
	engine.mu.Lock()
	engine.active--
	engine.mu.Unlock()
	return models, err
}

func (engine *fakeEngine) Name() string                                     { return engine.name }
func (engine *fakeEngine) Initialize(context.Context, *config.Config) error { return nil }
func (engine *fakeEngine) ListModels(context.Context, *config.Config) ([]*public.Model, error) {
	models := make([]*public.Model, 0, len(engine.models))
	for _, model := range engine.models {
		copy := *model
		models = append(models, &copy)
	}
	return models, nil
}
func (engine *fakeEngine) SupportsModel(modelName string, _ *config.Config) bool {
	for _, model := range engine.models {
		if model.Name == modelName {
			return true
		}
	}
	return false
}
func (engine *fakeEngine) HandleChat(context.Context, string, *public.ExtendedChatRequest, *websocket.Conn) error {
	return nil
}
func (engine *fakeEngine) HandleEmbedding(context.Context, string, *openaiapi.EmbeddingRequest, *websocket.Conn) error {
	return nil
}
func (engine *fakeEngine) SupportsEmbedding(string) bool { return false }

func TestHandleMessagesReconnectUpdatesTokenAndKeepsConnection(t *testing.T) {
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		conn, err := upgrader.Upgrade(writer, request, nil)
		if err != nil {
			t.Errorf("upgrade websocket: %v", err)
			return
		}
		defer conn.Close()
		if err := conn.WriteJSON(public.WSMessage{Type: public.RECONNECT, FingerPrint: "new-token"}); err != nil {
			t.Errorf("write reconnect: %v", err)
		}
		if err := conn.WriteJSON(public.WSMessage{Type: public.CLOSE, Content: "test complete"}); err != nil {
			t.Errorf("write close: %v", err)
		}
	}))
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cfg := &config.Config{JoinToken: "old-token"}
	client := &Client{controlConn: conn, ctx: ctx, cfg: cfg}

	client.HandleMessages()
	if cfg.JoinToken != "new-token" {
		t.Fatalf("expected updated token, got %q", cfg.JoinToken)
	}
}

func TestPushModelUpdateUsesUnixMilliseconds(t *testing.T) {
	upgrader := websocket.Upgrader{}
	messages := make(chan public.WSMessage, 1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		conn, err := upgrader.Upgrade(writer, request, nil)
		if err != nil {
			t.Errorf("upgrade websocket: %v", err)
			return
		}
		defer conn.Close()
		var message public.WSMessage
		if err := conn.ReadJSON(&message); err != nil {
			t.Errorf("read model update: %v", err)
			return
		}
		messages <- message
	}))
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	client := &Client{controlConn: conn}
	client.pushModelUpdate()

	message := <-messages
	data, err := json.Marshal(message.Content)
	if err != nil {
		t.Fatalf("marshal message content: %v", err)
	}
	var pong public.PPMessage
	if err := json.Unmarshal(data, &pong); err != nil {
		t.Fatalf("unmarshal pong: %v", err)
	}
	timestamp, err := strconv.ParseInt(pong.Timestamp, 10, 64)
	if err != nil {
		t.Fatalf("parse timestamp: %v", err)
	}
	if difference := time.Now().UnixMilli() - timestamp; difference < 0 || difference > 1000 {
		t.Fatalf("timestamp is not current Unix milliseconds: %d (difference %dms)", timestamp, difference)
	}
}

func TestHandleAbortCancelsOnlyMatchingRequest(t *testing.T) {
	fingerprint := "request-to-cancel"
	ctx, cancel := context.WithCancel(context.Background())
	requestCancels.Store(fingerprint, context.CancelFunc(cancel))
	t.Cleanup(func() { requestCancels.Delete(fingerprint) })

	client := &Client{}
	client.handleAbort(fingerprint)

	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("matching request was not cancelled")
	}
	if _, exists := requestCancels.Load(fingerprint); exists {
		t.Fatal("cancelled request remained in request map")
	}
}

func TestMergeModelPricesPreservesExistingAndPricesNewModelsFromConfig(t *testing.T) {
	existing := []*public.Model{{
		Name: "existing", Engine: "ollama", IPPM: 1.25, OPPM: 2.5, CIPPM: 0.5,
	}}
	discovered := []*public.Model{
		{Name: "existing", Engine: "ollama", IPPM: 99, OPPM: 99, CIPPM: 99},
		{Name: "configured", Engine: "openai", IPPM: 99, OPPM: 99, CIPPM: 99},
		{Name: "defaulted", Engine: "openai", IPPM: 99, OPPM: 99, CIPPM: 99},
	}
	cfg := &config.Config{
		InputTokenPricePerMillion:       3.8,
		OutputTokenPricePerMillion:      8.3,
		CachedInputTokenPricePerMillion: 1.0,
		ModelPrices: map[string]config.ModelPrice{
			"configured": {InputPrice: 4.1, OutputPrice: 9.2, CachedInputPrice: 1.1},
		},
		RegisteredModels: []string{"existing", "configured", "defaulted"},
	}

	merged := mergeModelPrices(existing, discovered, cfg)
	assertPrice := func(index int, ippm, oppm, cippm float64) {
		t.Helper()
		model := merged[index]
		if model.IPPM != ippm || model.OPPM != oppm || model.CIPPM != cippm {
			t.Fatalf("model %s prices: got %.2f/%.2f/%.2f, want %.2f/%.2f/%.2f",
				model.Name, model.IPPM, model.OPPM, model.CIPPM, ippm, oppm, cippm)
		}
	}
	assertPrice(0, 1.25, 2.5, 0.5)
	assertPrice(1, 4.1, 9.2, 1.1)
	assertPrice(2, 3.8, 8.3, 1.0)
}

func TestMergeModelPricesAppliesChangedConfiguredPrice(t *testing.T) {
	existing := []*public.Model{{
		Name: "deepseek-v4-pro", Engine: "openai", IPPM: 2, OPPM: 8.3, CIPPM: 0.02,
	}}
	discovered := []*public.Model{{Name: "deepseek-v4-pro", Engine: "openai"}}
	cfg := &config.Config{ModelPrices: map[string]config.ModelPrice{
		"deepseek-v4-pro": {
			Engine: "openai", InputPrice: 2, OutputPrice: 4, CachedInputPrice: 0.02,
		},
	}}

	merged := mergeModelPrices(existing, discovered, cfg)
	if len(merged) != 1 || merged[0].IPPM != 2 || merged[0].OPPM != 4 || merged[0].CIPPM != 0.02 {
		t.Fatalf("configured price was not applied: %+v", merged)
	}
}

func TestHandleModelPriceUpdateUpdatesModelsAndConfig(t *testing.T) {
	cfg := &config.Config{ModelPrices: map[string]config.ModelPrice{}}
	client := &Client{
		cfg: cfg,
		Models: []*public.Model{
			{Name: "shared", Engine: "ollama", IPPM: 1},
			{Name: "shared", Engine: "openai", IPPM: 2},
			{Name: "other", Engine: "ollama", IPPM: 3},
		},
	}
	client.handleModelPriceUpdate(public.WSMessage{
		Content: public.ModelPriceUpdate{Model: "shared", IPPM: 4.2, OPPM: 8.4, CIPPM: 1.2},
	})

	models := client.modelsSnapshot()
	for _, model := range models[:2] {
		if model.IPPM != 4.2 || model.OPPM != 8.4 || model.CIPPM != 1.2 {
			t.Fatalf("model %s/%s did not receive update: %+v", model.Engine, model.Name, model)
		}
	}
	if models[2].IPPM != 3 {
		t.Fatalf("unrelated model price changed: %+v", models[2])
	}
	price := cfg.ModelPrices["shared"]
	if price.InputPrice != 4.2 || price.OutputPrice != 8.4 || price.CachedInputPrice != 1.2 {
		t.Fatalf("config price not updated: %+v", price)
	}
}

func TestDuplicateBackendModelsAreCanonicalAndRoundRobinRouted(t *testing.T) {
	backendA := &fakeEngine{name: "backend-a", models: []*public.Model{{Name: "shared-model", Engine: "openai"}}}
	backendB := &fakeEngine{name: "backend-b", models: []*public.Model{{Name: "shared-model", Engine: "openai"}}}
	client := &Client{
		ctx:       context.Background(),
		cfg:       &config.Config{ModelPrices: map[string]config.ModelPrice{}, RegisteredModels: []string{"shared-model"}},
		engines:   []inference.Engine{backendA, backendB},
		routingRR: map[string]int{},
	}

	if err := client.refreshModels(); err != nil {
		t.Fatalf("refresh models: %v", err)
	}
	if got := len(client.modelsSnapshot()); got != 1 {
		t.Fatalf("canonical models: got %d, want 1", got)
	}

	want := []inference.Engine{backendA, backendB, backendA}
	for index, expected := range want {
		got, err := client.findEngineForModel("shared-model")
		if err != nil {
			t.Fatalf("route %d: %v", index, err)
		}
		if got != expected {
			t.Fatalf("route %d: got %s, want %s", index, got.Name(), expected.Name())
		}
	}
}

func TestRefreshModelsRegistersOnlySelectedModels(t *testing.T) {
	engine := &fakeEngine{name: "openai", models: []*public.Model{
		{Name: "selected", Engine: "openai"},
		{Name: "not-selected", Engine: "openai"},
	}}
	client := &Client{
		ctx: context.Background(), cfg: &config.Config{
			RegisteredModels: []string{"selected"}, ModelPrices: map[string]config.ModelPrice{},
		},
		engines: []inference.Engine{engine}, routingRR: map[string]int{},
	}

	if err := client.refreshModels(); err != nil {
		t.Fatalf("refresh models: %v", err)
	}
	models := client.modelsSnapshot()
	if len(models) != 1 || models[0].Name != "selected" {
		t.Fatalf("registered models: got %+v, want selected only", models)
	}

	client.cfg.RegisteredModels = nil
	if err := client.refreshModels(); err != nil {
		t.Fatalf("clear models: %v", err)
	}
	if got := len(client.modelsSnapshot()); got != 0 {
		t.Fatalf("registered models after clearing selection: got %d, want 0", got)
	}
}

func TestRefreshModelsSerializesConcurrentDiscovery(t *testing.T) {
	engine := &concurrentFakeEngine{fakeEngine: fakeEngine{
		name: "openai", models: []*public.Model{{Name: "selected", Engine: "openai"}},
	}}
	client := &Client{
		ctx: context.Background(), cfg: &config.Config{
			RegisteredModels: []string{"selected"}, ModelPrices: map[string]config.ModelPrice{},
		},
		engines: []inference.Engine{engine}, routingRR: map[string]int{},
	}

	var waitGroup sync.WaitGroup
	for index := 0; index < 4; index++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			if err := client.refreshModels(); err != nil {
				t.Errorf("refresh models: %v", err)
			}
		}()
	}
	waitGroup.Wait()

	engine.mu.Lock()
	defer engine.mu.Unlock()
	if engine.maxActive != 1 {
		t.Fatalf("concurrent discoveries: got %d, want 1", engine.maxActive)
	}
}

func TestApplyProxyBackendsSkipsUnchangedConfiguration(t *testing.T) {
	backends := []config.ProxyBackend{{
		Name: "primary", BaseURL: "https://example.test/v1", APIKey: "secret", Enabled: true,
	}}
	client := &Client{
		ctx: context.Background(), cfg: &config.Config{ProxyBackends: append([]config.ProxyBackend(nil), backends...)},
	}

	if err := client.applyProxyBackends(backends); err != nil {
		t.Fatalf("apply unchanged backends: %v", err)
	}
	if len(client.engines) != 0 {
		t.Fatalf("unchanged configuration rebuilt engines: %+v", client.engines)
	}
}
