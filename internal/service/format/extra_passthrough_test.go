package format

import (
	"encoding/json"
	"strings"
	"testing"

	"star-fire/pkg/public"
)

// 本测试文件覆盖 Extra 透传（chat_template_kwargs 等）与新版 vLLM
// reasoning 字段解析。背景：推理模型（如 DeepSeek-V4-Flash）在思考模式下
// 会把全部输出 token 消耗在隐藏思考阶段，stop_sequences/max_tokens 提前
// 截断时返回空 content。用户通过 chat_template_kwargs:{"thinking":false}
// 关闭思考模式可拿到部分文本，该字段必须无损透传给上游。

// ---- Anthropic ParseRequest：Extra 捕获 ----

func TestAnthropicConverter_ParseRequest_ExtraChatTemplateKwargs(t *testing.T) {
	c := &AnthropicConverter{}
	body := `{
		"model": "DeepSeek-V4",
		"max_tokens": 100,
		"stop_sequences": ["STOP"],
		"messages": [{ "role": "user", "content": "say RED STOP BLUE" }],
		"chat_template_kwargs": { "thinking": false }
	}`
	got, err := c.ParseRequest([]byte(body))
	if err != nil {
		t.Fatalf("ParseRequest error: %v", err)
	}
	if got.Extra == nil {
		t.Fatal("Extra is nil, want chat_template_kwargs captured")
	}
	v, ok := got.Extra["chat_template_kwargs"]
	if !ok {
		t.Fatal("Extra[chat_template_kwargs] missing")
	}
	var kw map[string]interface{}
	if err := json.Unmarshal(v, &kw); err != nil {
		t.Fatalf("chat_template_kwargs not valid JSON: %v", err)
	}
	if enabled, ok := kw["thinking"].(bool); !ok || enabled {
		t.Errorf("chat_template_kwargs.thinking = %v, want false", kw["thinking"])
	}
}

func TestAnthropicConverter_ParseRequest_NoExtraWhenAbsent(t *testing.T) {
	c := &AnthropicConverter{}
	body := `{
		"model": "DeepSeek-V4",
		"max_tokens": 100,
		"messages": [{ "role": "user", "content": "hi" }]
	}`
	got, err := c.ParseRequest([]byte(body))
	if err != nil {
		t.Fatalf("ParseRequest error: %v", err)
	}
	if got.Extra != nil {
		t.Errorf("Extra = %v, want nil when no extra fields present", got.Extra)
	}
}

func TestAnthropicConverter_ParseRequest_ExtraNullIgnored(t *testing.T) {
	c := &AnthropicConverter{}
	body := `{
		"model": "DeepSeek-V4",
		"max_tokens": 100,
		"messages": [{ "role": "user", "content": "hi" }],
		"chat_template_kwargs": null
	}`
	got, err := c.ParseRequest([]byte(body))
	if err != nil {
		t.Fatalf("ParseRequest error: %v", err)
	}
	if got.Extra != nil {
		t.Errorf("Extra = %v, want nil for null chat_template_kwargs", got.Extra)
	}
}

// ---- Anthropic → OpenAI 转发：Extra 注入上游请求 ----

func TestOpenAIConverter_BuildUpstreamRequest_InjectsExtra(t *testing.T) {
	c := &OpenAIConverter{}
	cr := &public.CanonicalRequest{
		Model:         "DeepSeek-V4",
		MaxTokens:     intPtr(100),
		StopSequences: []string{"STOP"},
		Messages: []public.CanonicalMessage{{
			Role:    "user",
			Content: []public.CanonicalContent{{Type: "text", Text: "say RED STOP BLUE"}},
		}},
		Extra: map[string]json.RawMessage{
			"chat_template_kwargs": json.RawMessage(`{"thinking":false}`),
		},
	}
	raw, err := c.BuildUpstreamRequest(cr)
	if err != nil {
		t.Fatalf("BuildUpstreamRequest error: %v", err)
	}
	var req map[string]json.RawMessage
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatalf("upstream request not valid JSON: %v", err)
	}
	kwRaw, ok := req["chat_template_kwargs"]
	if !ok {
		t.Fatalf("upstream request missing chat_template_kwargs: %s", raw)
	}
	var kw map[string]interface{}
	if err := json.Unmarshal(kwRaw, &kw); err != nil {
		t.Fatalf("chat_template_kwargs not valid JSON: %v", err)
	}
	if enabled, ok := kw["thinking"].(bool); !ok || enabled {
		t.Errorf("chat_template_kwargs.thinking = %v, want false", kw["thinking"])
	}
	// 标准字段不受影响。
	if _, ok := req["model"]; !ok {
		t.Error("upstream request missing model")
	}
	if _, ok := req["stop"]; !ok {
		t.Error("upstream request missing stop")
	}
}

func TestOpenAIConverter_BuildUpstreamRequest_NoExtraWhenEmpty(t *testing.T) {
	c := &OpenAIConverter{}
	cr := &public.CanonicalRequest{
		Model:    "DeepSeek-V4",
		Messages: []public.CanonicalMessage{{Role: "user", Content: []public.CanonicalContent{{Type: "text", Text: "hi"}}}},
	}
	raw, err := c.BuildUpstreamRequest(cr)
	if err != nil {
		t.Fatalf("BuildUpstreamRequest error: %v", err)
	}
	if strings.Contains(string(raw), "chat_template_kwargs") {
		t.Errorf("upstream request should not contain chat_template_kwargs: %s", raw)
	}
}

// ---- Anthropic BuildUpstreamRequest：Extra 注入（Anthropic→Anthropic 往返）----

func TestAnthropicConverter_BuildUpstreamRequest_InjectsExtra(t *testing.T) {
	c := &AnthropicConverter{}
	cr := &public.CanonicalRequest{
		Model:     "DeepSeek-V4",
		MaxTokens: intPtr(100),
		Messages: []public.CanonicalMessage{{
			Role:    "user",
			Content: []public.CanonicalContent{{Type: "text", Text: "hi"}},
		}},
		Extra: map[string]json.RawMessage{
			"chat_template_kwargs": json.RawMessage(`{"thinking":false}`),
		},
	}
	raw, err := c.BuildUpstreamRequest(cr)
	if err != nil {
		t.Fatalf("BuildUpstreamRequest error: %v", err)
	}
	var req map[string]json.RawMessage
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatalf("upstream request not valid JSON: %v", err)
	}
	if _, ok := req["chat_template_kwargs"]; !ok {
		t.Fatalf("upstream request missing chat_template_kwargs: %s", raw)
	}
}

// ---- OpenAI ParseRequest：chat_template_kwargs 捕获（SDK 无该字段）----

func TestOpenAIConverter_ParseRequest_CapturesChatTemplateKwargs(t *testing.T) {
	c := &OpenAIConverter{}
	body := `{
		"model": "DeepSeek-V4",
		"max_tokens": 100,
		"stop": ["STOP"],
		"messages": [{ "role": "user", "content": "say RED STOP BLUE" }],
		"chat_template_kwargs": { "thinking": false }
	}`
	got, err := c.ParseRequest([]byte(body))
	if err != nil {
		t.Fatalf("ParseRequest error: %v", err)
	}
	if got.Extra == nil {
		t.Fatal("Extra is nil, want chat_template_kwargs captured")
	}
	v, ok := got.Extra["chat_template_kwargs"]
	if !ok {
		t.Fatal("Extra[chat_template_kwargs] missing")
	}
	var kw map[string]interface{}
	if err := json.Unmarshal(v, &kw); err != nil {
		t.Fatalf("chat_template_kwargs not valid JSON: %v", err)
	}
	if enabled, ok := kw["thinking"].(bool); !ok || enabled {
		t.Errorf("chat_template_kwargs.thinking = %v, want false", kw["thinking"])
	}
}

// ---- 端到端：Anthropic 请求 → Canonical → OpenAI 上游请求 ----

func TestExtraPassthrough_AnthropicToOpenAI(t *testing.T) {
	anthropicConv := &AnthropicConverter{}
	openaiConv := &OpenAIConverter{}

	userBody := `{
		"model": "DeepSeek-V4",
		"max_tokens": 100,
		"stop_sequences": ["STOP"],
		"messages": [{ "role": "user", "content": "say RED STOP BLUE" }],
		"chat_template_kwargs": { "thinking": false }
	}`
	canonical, err := anthropicConv.ParseRequest([]byte(userBody))
	if err != nil {
		t.Fatalf("ParseRequest error: %v", err)
	}
	upstreamBody, err := openaiConv.BuildUpstreamRequest(canonical)
	if err != nil {
		t.Fatalf("BuildUpstreamRequest error: %v", err)
	}

	var req map[string]json.RawMessage
	if err := json.Unmarshal(upstreamBody, &req); err != nil {
		t.Fatalf("upstream body not valid JSON: %v", err)
	}
	// chat_template_kwargs 必须出现在上游请求中。
	kwRaw, ok := req["chat_template_kwargs"]
	if !ok {
		t.Fatalf("upstream body missing chat_template_kwargs: %s", upstreamBody)
	}
	var kw map[string]interface{}
	if err := json.Unmarshal(kwRaw, &kw); err != nil {
		t.Fatalf("chat_template_kwargs not valid JSON: %v", err)
	}
	if enabled, ok := kw["thinking"].(bool); !ok || enabled {
		t.Errorf("chat_template_kwargs.thinking = %v, want false", kw["thinking"])
	}
	// stop_sequences → stop。
	if _, ok := req["stop"]; !ok {
		t.Error("upstream body missing stop (from stop_sequences)")
	}
}

// ---- 新版 vLLM reasoning 字段解析（非流式）----

func TestOpenAIConverter_ParseUpstreamResponse_ReasoningField(t *testing.T) {
	c := &OpenAIConverter{}
	// 新版 vLLM 后端返回 reasoning 字段（非 reasoning_content），go-openai 不解析。
	data := `{
		"id": "chatcmpl-123",
		"object": "chat.completion",
		"choices": [{
			"index": 0,
			"finish_reason": "stop",
			"message": {
				"role": "assistant",
				"content": null,
				"reasoning": "We need answer only \"RED"
			},
			"stop_reason": "STOP"
		}],
		"usage": {
			"prompt_tokens": 10,
			"completion_tokens": 11,
			"completion_tokens_details": { "reasoning_tokens": 11 }
		}
	}`
	got, err := c.ParseUpstreamResponse([]byte(data))
	if err != nil {
		t.Fatalf("ParseUpstreamResponse error: %v", err)
	}
	var thinkingText string
	for _, b := range got.Content {
		if b.Type == "thinking" {
			thinkingText = b.Text
			break
		}
	}
	if thinkingText == "" {
		t.Fatalf("thinking block missing, content = %+v", got.Content)
	}
	if thinkingText != `We need answer only "RED` {
		t.Errorf("thinking text = %q, want %q", thinkingText, `We need answer only "RED`)
	}
}

func TestOpenAIConverter_ParseUpstreamResponse_ReasoningContentPreferred(t *testing.T) {
	c := &OpenAIConverter{}
	// 旧格式 reasoning_content 优先，不应重复添加 thinking 块。
	data := `{
		"id": "chatcmpl-123",
		"choices": [{
			"finish_reason": "stop",
			"message": {
				"role": "assistant",
				"content": "RED",
				"reasoning_content": "old style reasoning",
				"reasoning": "new style reasoning"
			}
		}]
	}`
	got, err := c.ParseUpstreamResponse([]byte(data))
	if err != nil {
		t.Fatalf("ParseUpstreamResponse error: %v", err)
	}
	thinkingCount := 0
	for _, b := range got.Content {
		if b.Type == "thinking" {
			thinkingCount++
			if b.Text != "old style reasoning" {
				t.Errorf("thinking text = %q, want old style reasoning", b.Text)
			}
		}
	}
	if thinkingCount != 1 {
		t.Errorf("thinking block count = %d, want 1", thinkingCount)
	}
}

// ---- 新版 vLLM reasoning 字段解析（流式）----

func TestOpenAIConverter_ParseUpstreamStreamEvent_ReasoningDelta(t *testing.T) {
	c := &OpenAIConverter{}
	line := `{
		"id": "chatcmpl-123",
		"object": "chat.completion.chunk",
		"choices": [{
			"index": 0,
			"delta": { "role": "assistant", "reasoning": "We need" }
		}]
	}`
	ev, err := c.ParseUpstreamStreamEvent([]byte(line))
	if err != nil {
		t.Fatalf("ParseUpstreamStreamEvent error: %v", err)
	}
	if ev == nil {
		t.Fatal("event is nil, want thinking delta")
	}
	if ev.Type != public.StreamEventThinkingDelta {
		t.Errorf("event type = %q, want %q", ev.Type, public.StreamEventThinkingDelta)
	}
	if ev.Text != "We need" {
		t.Errorf("event text = %q, want %q", ev.Text, "We need")
	}
}

func TestOpenAIConverter_ParseUpstreamStreamEvent_ReasoningContentPreferred(t *testing.T) {
	c := &OpenAIConverter{}
	// 旧格式 reasoning_content 优先。
	line := `{
		"choices": [{
			"delta": { "reasoning_content": "old", "reasoning": "new" }
		}]
	}`
	ev, err := c.ParseUpstreamStreamEvent([]byte(line))
	if err != nil {
		t.Fatalf("ParseUpstreamStreamEvent error: %v", err)
	}
	if ev == nil || ev.Type != public.StreamEventThinkingDelta {
		t.Fatalf("event = %+v, want thinking delta", ev)
	}
	if ev.Text != "old" {
		t.Errorf("event text = %q, want old", ev.Text)
	}
}

// ---- 辅助函数单测 ----

func TestSortedExtraKeys(t *testing.T) {
	extra := map[string]json.RawMessage{
		"zebra":  json.RawMessage(`1`),
		"apple":  json.RawMessage(`2`),
		"middle": json.RawMessage(`3`),
	}
	keys := sortedExtraKeys(extra)
	if len(keys) != 3 {
		t.Fatalf("keys = %v, want 3 keys", keys)
	}
	if keys[0] != "apple" || keys[1] != "middle" || keys[2] != "zebra" {
		t.Errorf("keys = %v, want sorted [apple middle zebra]", keys)
	}
	if sortedExtraKeys(nil) != nil {
		t.Error("sortedExtraKeys(nil) should be nil")
	}
}

func TestInjectExtraFields(t *testing.T) {
	raw := []byte(`{"model":"m","messages":[]}`)
	extra := map[string]json.RawMessage{
		"chat_template_kwargs": json.RawMessage(`{"thinking":false}`),
	}
	got := injectExtraFields(raw, extra, []string{"chat_template_kwargs"})
	var req map[string]json.RawMessage
	if err := json.Unmarshal(got, &req); err != nil {
		t.Fatalf("result not valid JSON: %v", err)
	}
	if _, ok := req["chat_template_kwargs"]; !ok {
		t.Errorf("chat_template_kwargs not injected: %s", got)
	}
	if _, ok := req["model"]; !ok {
		t.Errorf("model lost: %s", got)
	}
	// null 值不注入。
	extraNull := map[string]json.RawMessage{"k": json.RawMessage(`null`)}
	got2 := injectExtraFields(raw, extraNull, []string{"k"})
	if strings.Contains(string(got2), `"k"`) {
		t.Errorf("null value should not be injected: %s", got2)
	}
}

func TestExtractRawTopLevelFields(t *testing.T) {
	body := []byte(`{"model":"m","chat_template_kwargs":{"thinking":false},"other":1}`)
	got := extractRawTopLevelFields(body, "chat_template_kwargs")
	if got == nil || len(got) != 1 {
		t.Fatalf("got = %v, want 1 field", got)
	}
	if _, ok := got["chat_template_kwargs"]; !ok {
		t.Error("chat_template_kwargs missing")
	}
	// 无匹配字段返回 nil。
	if extractRawTopLevelFields(body, "nonexistent") != nil {
		t.Error("want nil for nonexistent key")
	}
	// 无效 JSON 返回 nil。
	if extractRawTopLevelFields([]byte(`not json`), "k") != nil {
		t.Error("want nil for invalid JSON")
	}
}
