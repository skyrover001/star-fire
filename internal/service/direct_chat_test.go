package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	configs "star-fire/config"
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

func TestAnthropicSSEEventNames(t *testing.T) {
	for _, scenario := range []struct {
		name   string
		event  *public.CanonicalStreamEvent
		failed bool
	}{
		{name: "empty"},
		{name: "text", event: &public.CanonicalStreamEvent{Type: public.StreamEventTextDelta, Text: "hello"}},
		{name: "tool", event: &public.CanonicalStreamEvent{Type: public.StreamEventToolCallDelta, ToolCall: &public.CanonicalToolCall{ID: "call_1", Name: "weather"}}},
		{name: "error", failed: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			ctx, recorder := newDirectGin()
			ctx.Writer.Header().Set("Content-Type", "text/event-stream")
			converter := &format.AnthropicConverter{}
			writer := newUserStreamWriter(ctx, converter, "test-model")
			if scenario.event != nil {
				writeUserStreamEvent(ctx, nil, "", "", 0, 0, 0, "test-model", scenario.event, converter, nil, writer)
			}
			finishUserStream(ctx, converter, writer, scenario.failed)
			body := recorder.Body.String()
			if strings.Contains(body, "[DONE]") {
				t.Fatal("Anthropic stream must not contain [DONE]")
			}
			for _, frame := range strings.Split(strings.TrimSpace(body), "\n\n") {
				lines := strings.Split(frame, "\n")
				if len(lines) != 2 || !strings.HasPrefix(lines[0], "event: ") || !strings.HasPrefix(lines[1], "data: ") {
					t.Fatalf("expected named SSE event, got %q", frame)
				}
				var payload struct {
					Type string `json:"type"`
				}
				if err := json.Unmarshal([]byte(strings.TrimPrefix(lines[1], "data: ")), &payload); err != nil {
					t.Fatal(err)
				}
				if lines[0] != "event: "+payload.Type {
					t.Fatalf("SSE event name does not match payload: %q", frame)
				}
			}
			if scenario.failed {
				if !strings.HasPrefix(body, "event: error\n") {
					t.Fatal(body)
				}
			} else if !strings.Contains(body, "event: message_delta\n") || !strings.Contains(body, `"stop_reason":"end_turn"`) || !strings.HasSuffix(body, "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n") {
				t.Fatalf("missing named termination events: %s", body)
			}
		})
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
	// 成功应采样可靠性 1，EMA 从 0.5 上升
	if ema := b.GetReliabilityEMA(); ema <= 0.5 {
		t.Fatalf("reliability EMA = %v, want > 0.5 after success", ema)
	}
}

func TestDirectChatAnthropicStream(t *testing.T) {
	for _, scenario := range []struct {
		finishReason string
		stopReason   string
	}{
		{finishReason: "stop", stopReason: "end_turn"},
		{finishReason: "length", stopReason: "max_tokens"},
		{finishReason: "tool_calls", stopReason: "tool_use"},
	} {
		t.Run(scenario.finishReason, func(t *testing.T) {
			upstream := mockDirectBackend(t, func(writer http.ResponseWriter, request *http.Request) {
				writer.Header().Set("Content-Type", "text/event-stream")
				if scenario.finishReason == "tool_calls" {
					fmt.Fprint(writer, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"weather\",\"arguments\":\"{}\"}}]}}]}\n\n")
				} else {
					fmt.Fprint(writer, "data: {\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\n")
				}
				fmt.Fprintf(writer, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":%q}]}\n\ndata: {\"choices\":[],\"usage\":{\"prompt_tokens\":2,\"completion_tokens\":1,\"total_tokens\":3}}\n\ndata: [DONE]\n\n", scenario.finishReason)
			})
			server := newDirectTestServer(t)
			backend := &models.DirectBackend{ID: "anthropic-test", BaseURL: upstream.URL, Format: "openai", Enabled: true, MaxConns: 4}
			ctx, recorder := newDirectGin()
			request := public.ExtendedChatRequest{}
			request.Model = "test-model"
			request.Stream = true
			request.Messages = []openai.ChatCompletionMessage{{Role: "user", Content: "hi"}}
			if !handleDirectChat(ctx, server, backend, request, "u1", &format.AnthropicConverter{}) {
				t.Fatal("handleDirectChat returned false")
			}
			body := recorder.Body.String()
			if !strings.HasPrefix(body, "event: message_start\n") || !strings.Contains(body, "event: message_delta\n") || !strings.HasSuffix(body, "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n") {
				t.Fatalf("missing named lifecycle events: %s", body)
			}
			if !strings.Contains(body, `"stop_reason":"`+scenario.stopReason+`"`) || strings.Contains(body, "[DONE]") {
				t.Fatalf("invalid Anthropic termination: %s", body)
			}
		})
	}
}

func TestDirectStreamBilling(t *testing.T) {
	for _, userFormat := range []string{public.FormatOpenAI, public.FormatAnthropic, public.FormatResponses} {
		t.Run(userFormat, func(t *testing.T) {
			upstream := mockDirectBackend(t, func(writer http.ResponseWriter, request *http.Request) {
				var body struct {
					StreamOptions struct {
						IncludeUsage bool `json:"include_usage"`
					} `json:"stream_options"`
				}
				if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				writer.Header().Set("Content-Type", "text/event-stream")
				fmt.Fprint(writer, "data: {\"choices\":[{\"delta\":{\"content\":\"hi\"}}]}\n\ndata: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
				if body.StreamOptions.IncludeUsage {
					fmt.Fprint(writer, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":2,\"total_tokens\":12,\"prompt_tokens_details\":{\"cached_tokens\":4}}}\n\n")
				}
				fmt.Fprint(writer, "data: [DONE]\n\n")
			})
			server := newDirectTestServer(t)
			backend := &models.DirectBackend{ID: "billing", BaseURL: upstream.URL, Models: []*public.Model{{Name: "test-model", IPPM: 2, OPPM: 6, CIPPM: 0.5}}}
			ctx, _ := newDirectGin()
			request := public.ExtendedChatRequest{}
			request.Model = "test-model"
			request.Stream = true
			request.Messages = []openai.ChatCompletionMessage{{Role: "user", Content: "hi"}}
			var converter format.Converter
			if userFormat != public.FormatOpenAI {
				var err error
				converter, err = format.GetConverter(userFormat)
				if err != nil {
					t.Fatal(err)
				}
			}
			if !handleDirectChat(ctx, server, backend, request, "u1", converter) {
				t.Fatal("request failed")
			}
			usages, _, err := server.TokenUsageDB.GetUserTokenUsagePaged("u1", time.Now().Add(-time.Hour), time.Now().Add(time.Hour), 1, 10)
			if err != nil || len(usages) != 1 {
				t.Fatalf("want exactly one usage row, got %d: %v", len(usages), err)
			}
			if usages[0].ClientID != "direct:billing" || usages[0].CachedTokens != 4 || usages[0].TotalTokens != 12 {
				t.Fatalf("incorrect usage: %+v", usages[0])
			}
			const expectedCost = 26.0 / 1000000
			if math.Abs(usages[0].Cost-expectedCost) > 1e-12 {
				t.Fatalf("cost = %v, want %v", usages[0].Cost, expectedCost)
			}
			balance, spent, err := server.UserDB.GetBalance("u1")
			if err != nil || math.Abs(spent-expectedCost) > 1e-9 || math.Abs(balance-(1000-expectedCost)) > 1e-9 {
				t.Fatalf("incorrect deduction: balance=%v spent=%v err=%v", balance, spent, err)
			}
			if backend.GetCacheHitEMA() != 0.4 {
				t.Fatalf("cache feedback = %v, want 0.4", backend.GetCacheHitEMA())
			}
		})
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
	// 流式成功应采样可靠性 1，EMA 从 0.5 上升
	if ema := b.GetReliabilityEMA(); ema <= 0.5 {
		t.Fatalf("reliability EMA = %v, want > 0.5 after stream success", ema)
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

func TestDirectNonStreamReasoning(t *testing.T) {
	upstream := mockDirectBackend(t, func(writer http.ResponseWriter, request *http.Request) {
		fmt.Fprint(writer, `{"choices":[{"message":{"role":"assistant","reasoning_content":"plan","tool_calls":[{"id":"call_1","type":"function","function":{"name":"weather","arguments":"{}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":2,"completion_tokens":1,"total_tokens":3}}`)
	})
	server := newDirectTestServer(t)
	backend := &models.DirectBackend{ID: "reasoning", BaseURL: upstream.URL}
	ctx, _ := newDirectGin()
	ctx.Set("reasoning_conv_key", "conversation")
	request := public.ExtendedChatRequest{}
	request.Model = "test-model"
	if !handleDirectChat(ctx, server, backend, request, "u1", &format.AnthropicConverter{}) {
		t.Fatal("request failed")
	}
	if got := server.GetReasoning("conversation")["call_1"]; got != "plan" {
		t.Fatalf("saved reasoning = %q, want plan", got)
	}
}

type failingDirectConverter struct {
	format.AnthropicConverter
}

func (converter *failingDirectConverter) BuildResponse(response *public.CanonicalResponse) ([]byte, error) {
	return nil, errors.New("local conversion failed")
}

func TestDirectAdapterErrorDoesNotCooldown(t *testing.T) {
	upstream := mockDirectBackend(t, func(writer http.ResponseWriter, request *http.Request) {
		fmt.Fprint(writer, `{"choices":[{"message":{"role":"assistant","content":"hi"},"finish_reason":"stop"}]}`)
	})
	server := newDirectTestServer(t)
	backend := &models.DirectBackend{ID: "adapter", BaseURL: upstream.URL}
	ctx, _ := newDirectGin()
	if handleDirectChat(ctx, server, backend, public.ExtendedChatRequest{}, "u1", &failingDirectConverter{}) {
		t.Fatal("conversion failure should not report success")
	}
	if backend.GetFailures() != 0 || backend.InCooldown() || backend.GetReliabilityEMA() != 0.5 {
		t.Fatal("local conversion error must not penalize backend")
	}
}

func TestDirectInvalidNonStreamResponse(t *testing.T) {
	upstream := mockDirectBackend(t, func(writer http.ResponseWriter, request *http.Request) {
		fmt.Fprint(writer, "<html>not JSON</html>")
	})
	server := newDirectTestServer(t)
	backend := &models.DirectBackend{ID: "invalid", BaseURL: upstream.URL}
	ctx, recorder := newDirectGin()
	if handleDirectChat(ctx, server, backend, public.ExtendedChatRequest{}, "u1") {
		t.Fatal("invalid upstream JSON must not be returned as success")
	}
	if recorder.Body.Len() != 0 || backend.GetFailures() != 1 {
		t.Fatal("invalid upstream response should be retryable and count as backend failure")
	}
}

func TestDirectStreamFailureTermination(t *testing.T) {
	for _, userFormat := range []string{public.FormatAnthropic, public.FormatResponses} {
		for _, failure := range []string{"invalid", "eof", "cut"} {
			t.Run(userFormat+"/"+failure, func(t *testing.T) {
				upstream := mockDirectBackend(t, func(writer http.ResponseWriter, request *http.Request) {
					writer.Header().Set("Content-Type", "text/event-stream")
					fmt.Fprint(writer, "data: {\"choices\":[{\"delta\":{\"content\":\"hi\"}}]}\n\n")
					writer.(http.Flusher).Flush()
					if failure == "invalid" {
						fmt.Fprint(writer, "data: not-json\n\n")
					} else if failure == "cut" {
						panic("cut")
					}
				})
				server := newDirectTestServer(t)
				backend := &models.DirectBackend{ID: "failure", BaseURL: upstream.URL}
				ctx, recorder := newDirectGin()
				request := public.ExtendedChatRequest{}
				request.Model = "test-model"
				request.Stream = true
				converter, err := format.GetConverter(userFormat)
				if err != nil {
					t.Fatal(err)
				}
				if !handleDirectChat(ctx, server, backend, request, "u1", converter) {
					t.Fatal("must not retry after response started")
				}
				body := recorder.Body.String()
				marker := `"type":"response.failed"`
				if userFormat == public.FormatAnthropic {
					marker = "event: error\n"
				}
				if !strings.Contains(body, marker) || (userFormat == public.FormatAnthropic && strings.Contains(body, "[DONE]")) {
					t.Fatalf("missing format-specific error termination: %s", body)
				}
				if backend.GetFailures() != 1 || backend.GetReliabilityEMA() >= 0.5 {
					t.Fatal("invalid/truncated upstream must count as backend failure")
				}
			})
		}
	}
}

func TestDirectChatPreservesUnknownFields(t *testing.T) {
	previous := configs.Config.DirectBackendsEnabled
	configs.Config.DirectBackendsEnabled = true
	t.Cleanup(func() { configs.Config.DirectBackendsEnabled = previous })
	for _, stream := range []bool{false, true} {
		t.Run(fmt.Sprintf("stream=%v", stream), func(t *testing.T) {
			upstream := mockDirectBackend(t, func(writer http.ResponseWriter, request *http.Request) {
				var fields map[string]json.RawMessage
				if err := json.NewDecoder(request.Body).Decode(&fields); err != nil {
					t.Error(err)
				}
				if string(fields["vendor_option"]) != `{"value":1234567890123456789}` {
					t.Errorf("unknown field lost or changed: %s", fields["vendor_option"])
				}
				if _, exists := fields["routing"]; exists {
					t.Error("gateway-only routing must not reach upstream")
				}
				if stream {
					writer.Header().Set("Content-Type", "text/event-stream")
					fmt.Fprint(writer, "data: {\"choices\":[{\"delta\":{\"content\":\"hi\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
				} else {
					fmt.Fprint(writer, `{"choices":[{"message":{"role":"assistant","content":"hi"},"finish_reason":"stop"}]}`)
				}
			})
			server := newDirectTestServer(t)
			backend := &models.DirectBackend{ID: "raw", BaseURL: upstream.URL, Format: public.FormatOpenAI, Enabled: true, MaxConns: 4, Models: []*public.Model{{Name: "test-model", IPPM: 2, OPPM: 6}}}
			registerDirectBackend(t, server, backend)
			ctx, _ := newDirectGin()
			body := fmt.Sprintf(`{"model":"test-model","stream":%v,"routing":"stability","messages":[{"role":"user","content":"hi"}],"stream_options": null ,"vendor_option":{"value":1234567890123456789}}`, stream)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
			ctx.Request.Header.Set("Content-Type", "application/json")
			HandleChatRequest(ctx, server)
		})
	}
}

type failingDirectWriter struct {
	format.UserStreamWriter
	failOn string
}

func (writer *failingDirectWriter) Write(event *public.CanonicalStreamEvent) ([][]byte, error) {
	if event.Type == writer.failOn {
		return nil, errors.New("local stream conversion failed")
	}
	return writer.UserStreamWriter.Write(event)
}

type failingDirectStreamConverter struct {
	format.ResponsesConverter
	failOn string
}

func (converter *failingDirectStreamConverter) NewUserStreamWriter() format.UserStreamWriter {
	return &failingDirectWriter{UserStreamWriter: converter.ResponsesConverter.NewUserStreamWriter(), failOn: converter.failOn}
}

func TestDirectStreamAdapterError(t *testing.T) {
	for _, eventType := range []string{public.StreamEventTextDelta, public.StreamEventDone} {
		t.Run(eventType, func(t *testing.T) {
			upstream := mockDirectBackend(t, func(writer http.ResponseWriter, request *http.Request) {
				fmt.Fprint(writer, "data: {\"choices\":[{\"delta\":{\"content\":\"hi\"}}]}\n\ndata: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
			})
			server := newDirectTestServer(t)
			backend := &models.DirectBackend{ID: "adapter", BaseURL: upstream.URL}
			ctx, recorder := newDirectGin()
			request := public.ExtendedChatRequest{}
			request.Stream = true
			if !handleDirectChat(ctx, server, backend, request, "u1", &failingDirectStreamConverter{failOn: eventType}) {
				t.Fatal("must not retry after writing error response")
			}
			if !strings.Contains(recorder.Body.String(), `"type":"response.failed"`) {
				t.Fatalf("missing error termination: %s", recorder.Body.String())
			}
			if backend.GetFailures() != 0 || backend.InCooldown() || backend.GetReliabilityEMA() != 0.5 {
				t.Fatal("local stream conversion error must not change backend reliability or cooldown")
			}
		})
	}
}

func TestDirectRawStreamOptions(t *testing.T) {
	for _, options := range []string{`null`, `{}`, `{"include_usage":false,"vendor_option":1234567890123456789}`} {
		t.Run(options, func(t *testing.T) {
			upstream := mockDirectBackend(t, func(writer http.ResponseWriter, request *http.Request) {
				var body struct {
					StreamOptions map[string]json.RawMessage `json:"stream_options"`
				}
				if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if string(body.StreamOptions["include_usage"]) != "true" {
					t.Error("include_usage must be true")
				}
				if strings.Contains(options, "vendor_option") && string(body.StreamOptions["vendor_option"]) != "1234567890123456789" {
					t.Error("existing stream options must be preserved exactly")
				}
				fmt.Fprint(writer, "data: [DONE]\n\n")
			})
			server := newDirectTestServer(t)
			ctx, _ := newDirectGin()
			request := public.ExtendedChatRequest{}
			request.Stream = true
			request.RawBody = json.RawMessage(`{"model":"test-model","stream":true,"stream_options":` + options + `}`)
			if !handleDirectChat(ctx, server, &models.DirectBackend{ID: "raw-options", BaseURL: upstream.URL}, request, "u1") {
				t.Fatal("request failed")
			}
		})
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
	// 4xx 属于请求问题，不采样可靠性，EMA 保持中性 0.5
	if ema := b.GetReliabilityEMA(); ema != 0.5 {
		t.Fatalf("reliability EMA = %v, want 0.5 (no sampling on 4xx)", ema)
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
	// 5xx 属于后端过错，应采样可靠性 0，EMA 从 0.5 下降
	if ema := b.GetReliabilityEMA(); ema >= 0.5 {
		t.Fatalf("reliability EMA = %v, want < 0.5 after 5xx", ema)
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
	// 网络错误属于后端过错，应采样可靠性 0，EMA 从 0.5 下降
	if ema := b.GetReliabilityEMA(); ema >= 0.5 {
		t.Fatalf("reliability EMA = %v, want < 0.5 after network error", ema)
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
	// 断流属于后端过错，应采样可靠性 0，EMA 从 0.5 下降
	if ema := b.GetReliabilityEMA(); ema >= 0.5 {
		t.Fatalf("reliability EMA = %v, want < 0.5 after mid-stream cut", ema)
	}
}
