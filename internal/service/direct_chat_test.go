package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"star-fire/internal/models"
	"star-fire/internal/service/format"
	"star-fire/pkg/public"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/sashabaranov/go-openai"
	"gorm.io/gorm"
)

// newDirectTestServer 构造一个带 :memory: sqlite 的 Server，含计费所需 DB。
func newDirectTestServer(t *testing.T) *models.Server {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	server := &models.Server{
		DirectBackendDB: models.NewDirectBackendDB(db),
		TokenUsageDB:    models.NewTokenUsageDB(db),
		UserDB:          models.NewUserDB(db),
		UserPriceCapDB:  models.NewUserPriceCapDB(db),
	}
	// 预置一个用户，余额充足
	user := &models.User{ID: "u1", Username: "u1", Password: "x", Balance: 1000}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	return server
}

// newDirectGin 构造 gin 测试上下文，并注入 user_id。
func newDirectGin() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Set("user_id", "u1")
	return c, w
}

// mockDirectBackend 起一个 httptest 后端，按 handler 处理 /chat/completions。
func mockDirectBackend(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			http.NotFound(w, r)
			return
		}
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// registerDirectBackend 保存后端到 DB 并载入注册表。
func registerDirectBackend(t *testing.T, server *models.Server, b *models.DirectBackend) {
	t.Helper()
	if err := server.DirectBackendDB.Save(b); err != nil {
		t.Fatalf("save backend: %v", err)
	}
	if err := server.LoadDirectBackends(); err != nil {
		t.Fatalf("load backends: %v", err)
	}
}

func TestDirectChatNonStream(t *testing.T) {
	srv := mockDirectBackend(t, func(w http.ResponseWriter, r *http.Request) {
		// 校验鉴权头
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(openai.ChatCompletionResponse{
			Usage: openai.Usage{
				PromptTokens:     10,
				CompletionTokens: 20,
				TotalTokens:      30,
				PromptTokensDetails: &openai.PromptTokensDetails{
					CachedTokens: 4,
				},
			},
		})
	})

	server := newDirectTestServer(t)
	b := &models.DirectBackend{
		ID: "b1", Name: "b1", BaseURL: srv.URL, Format: "openai",
		Enabled: true, MaxConns: 4, Priority: 1, APIKey: "sk-test",
		Models: []*public.Model{{Name: "qwen3-32b", IPPM: 2.0, OPPM: 6.0, CIPPM: 0.5}},
	}
	registerDirectBackend(t, server, b)

	c, w := newDirectGin()
	req := public.ExtendedChatRequest{}
	req.Model = "qwen3-32b"
	req.Messages = []openai.ChatCompletionMessage{{Role: "user", Content: "hi"}}

	if done := handleDirectChat(c, server, b, req, "u1"); !done {
		t.Fatalf("handleDirectChat done=false, want true")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	// 计费应写入 TokenUsage
	var resp openai.ChatCompletionResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Usage.PromptTokens != 10 {
		t.Fatalf("usage prompt = %d, want 10", resp.Usage.PromptTokens)
	}
	// 验证 TokenUsage 落库
	usages, _, err := server.TokenUsageDB.GetUserTokenUsagePaged("u1", time.Now().Add(-time.Hour), time.Now().Add(time.Hour), 1, 10)
	if err != nil || len(usages) == 0 {
		t.Fatalf("no token usage recorded: %v, len=%d", err, len(usages))
	}
	if usages[0].ClientID != "direct:b1" {
		t.Fatalf("clientID = %s, want direct:b1", usages[0].ClientID)
	}
	if usages[0].CachedTokens != 4 {
		t.Fatalf("cached = %d, want 4", usages[0].CachedTokens)
	}
}

func TestDirectChatStream(t *testing.T) {
	srv := mockDirectBackend(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fl, _ := w.(http.Flusher)
		fmt.Fprint(w, "data: {\"id\":\"1\",\"choices\":[{\"delta\":{\"content\":\"hi\"}}]}\n\n")
		fl.Flush()
		fmt.Fprint(w, "data: {\"id\":\"1\",\"choices\":[{\"delta\":{\"content\":\" there\"},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":5,\"completion_tokens\":3,\"total_tokens\":8}}\n\n")
		fl.Flush()
		fmt.Fprint(w, "data: [DONE]\n\n")
		fl.Flush()
	})

	server := newDirectTestServer(t)
	b := &models.DirectBackend{
		ID: "b1", Name: "b1", BaseURL: srv.URL, Format: "openai",
		Enabled: true, MaxConns: 4, Priority: 1,
		Models: []*public.Model{{Name: "qwen3-32b", IPPM: 2.0, OPPM: 6.0, CIPPM: 0.5}},
	}
	registerDirectBackend(t, server, b)

	c, w := newDirectGin()
	req := public.ExtendedChatRequest{}
	req.Model = "qwen3-32b"
	req.Stream = true
	req.Messages = []openai.ChatCompletionMessage{{Role: "user", Content: "hi"}}

	if done := handleDirectChat(c, server, b, req, "u1"); !done {
		t.Fatalf("handleDirectChat done=false, want true")
	}
	body := w.Body.String()
	if !strings.Contains(body, "data: [DONE]") {
		t.Fatalf("stream missing [DONE]: %s", body)
	}
	usages, _, err := server.TokenUsageDB.GetUserTokenUsagePaged("u1", time.Now().Add(-time.Hour), time.Now().Add(time.Hour), 1, 10)
	if err != nil || len(usages) == 0 {
		t.Fatalf("no token usage recorded: %v, len=%d", err, len(usages))
	}
	if usages[0].TotalTokens != 8 {
		t.Fatalf("total = %d, want 8", usages[0].TotalTokens)
	}
}

func TestDirectChatConvertsResponseToResponsesFormat(t *testing.T) {
	srv := mockDirectBackend(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(openai.ChatCompletionResponse{
			ID: "chatcmpl-1", Model: "qwen3-32b",
			Choices: []openai.ChatCompletionChoice{{
				Message:      openai.ChatCompletionMessage{Role: "assistant", Content: "hello"},
				FinishReason: "stop",
			}},
			Usage: openai.Usage{PromptTokens: 2, CompletionTokens: 1, TotalTokens: 3},
		})
	})
	server := newDirectTestServer(t)
	b := &models.DirectBackend{
		ID: "b1", Name: "b1", BaseURL: srv.URL, Format: "openai", Enabled: true, MaxConns: 4,
		Models: []*public.Model{{Name: "qwen3-32b", IPPM: 2, OPPM: 6}},
	}
	registerDirectBackend(t, server, b)
	responsesConv, err := format.GetConverter(public.FormatResponses)
	if err != nil {
		t.Fatalf("get Responses converter: %v", err)
	}
	c, w := newDirectGin()
	req := public.ExtendedChatRequest{}
	req.Model = "qwen3-32b"
	req.Messages = []openai.ChatCompletionMessage{{Role: "user", Content: "hi"}}
	if done := handleDirectChat(c, server, b, req, "u1", responsesConv); !done {
		t.Fatal("handleDirectChat done=false, want true")
	}
	var response struct {
		Output []struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode Responses body: %v", err)
	}
	if len(response.Output) != 1 || len(response.Output[0].Content) != 1 || response.Output[0].Content[0].Text != "hello" {
		t.Fatalf("unexpected converted response: %s", w.Body.String())
	}
}

func TestDirectChat4xx(t *testing.T) {
	srv := mockDirectBackend(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad request"}`))
	})
	server := newDirectTestServer(t)
	b := &models.DirectBackend{
		ID: "b1", Name: "b1", BaseURL: srv.URL, Format: "openai",
		Enabled: true, MaxConns: 4, Priority: 1,
		Models: []*public.Model{{Name: "qwen3-32b", IPPM: 2.0, OPPM: 6.0, CIPPM: 0.5}},
	}
	registerDirectBackend(t, server, b)

	c, w := newDirectGin()
	req := public.ExtendedChatRequest{}
	req.Model = "qwen3-32b"
	req.Messages = []openai.ChatCompletionMessage{{Role: "user", Content: "hi"}}

	if done := handleDirectChat(c, server, b, req, "u1"); !done {
		t.Fatalf("4xx should be done=true (passthrough)")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	// 4xx 不记失败
	if got := b.GetFailures(); got != 0 {
		t.Fatalf("failures = %d, want 0", got)
	}
}

func TestDirectChat5xx(t *testing.T) {
	srv := mockDirectBackend(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	})
	server := newDirectTestServer(t)
	b := &models.DirectBackend{
		ID: "b1", Name: "b1", BaseURL: srv.URL, Format: "openai",
		Enabled: true, MaxConns: 4, Priority: 1,
		Models: []*public.Model{{Name: "qwen3-32b", IPPM: 2.0, OPPM: 6.0, CIPPM: 0.5}},
	}
	registerDirectBackend(t, server, b)

	c, w := newDirectGin()
	req := public.ExtendedChatRequest{}
	req.Model = "qwen3-32b"
	req.Messages = []openai.ChatCompletionMessage{{Role: "user", Content: "hi"}}

	if done := handleDirectChat(c, server, b, req, "u1"); done {
		t.Fatalf("5xx should be done=false (retryable)")
	}
	if got := b.GetFailures(); got != 1 {
		t.Fatalf("failures = %d, want 1", got)
	}
	if !b.InCooldown() {
		t.Fatalf("5xx should trip cooldown")
	}
	_ = w
}

func TestDirectChatNetworkError(t *testing.T) {
	// 关闭的 server → 连接错误
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	server := newDirectTestServer(t)
	b := &models.DirectBackend{
		ID: "b1", Name: "b1", BaseURL: url, Format: "openai",
		Enabled: true, MaxConns: 4, Priority: 1,
		Models: []*public.Model{{Name: "qwen3-32b", IPPM: 2.0, OPPM: 6.0, CIPPM: 0.5}},
	}
	registerDirectBackend(t, server, b)

	c, _ := newDirectGin()
	req := public.ExtendedChatRequest{}
	req.Model = "qwen3-32b"
	req.Messages = []openai.ChatCompletionMessage{{Role: "user", Content: "hi"}}

	if done := handleDirectChat(c, server, b, req, "u1"); done {
		t.Fatalf("network error should be done=false")
	}
	if got := b.GetFailures(); got != 1 {
		t.Fatalf("failures = %d, want 1", got)
	}
}

func TestDirectChatMidStreamCut(t *testing.T) {
	srv := mockDirectBackend(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fl, _ := w.(http.Flusher)
		fmt.Fprint(w, "data: {\"id\":\"1\",\"choices\":[{\"delta\":{\"content\":\"hi\"}}]}\n\n")
		fl.Flush()
		// 直接关闭连接，模拟断流
		panic("cut")
	})

	server := newDirectTestServer(t)
	b := &models.DirectBackend{
		ID: "b1", Name: "b1", BaseURL: srv.URL, Format: "openai",
		Enabled: true, MaxConns: 4, Priority: 1,
		Models: []*public.Model{{Name: "qwen3-32b", IPPM: 2.0, OPPM: 6.0, CIPPM: 0.5}},
	}
	registerDirectBackend(t, server, b)

	c, w := newDirectGin()
	req := public.ExtendedChatRequest{}
	req.Model = "qwen3-32b"
	req.Stream = true
	req.Messages = []openai.ChatCompletionMessage{{Role: "user", Content: "hi"}}

	if done := handleDirectChat(c, server, b, req, "u1"); !done {
		t.Fatalf("mid-stream cut after chunk should be done=true (not retryable)")
	}
	if !strings.Contains(w.Body.String(), "data: [DONE]") {
		t.Fatalf("mid-stream cut should write [DONE]: %s", w.Body.String())
	}
	// 断流记失败
	if got := b.GetFailures(); got != 1 {
		t.Fatalf("failures = %d, want 1", got)
	}
}
