package format

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"star-fire/pkg/public"
)

// 本测试文件依据 docs/multi-format-api-design.md 3.5 节的黄金样本编写。
// 测试原则：已知输入 → 已知输出（黄金样本）+ 双向 round-trip（Parse 后 Build 再 Parse）。

const goldenRequestOpenAI = `{
  "model": "gpt-4o",
  "stream": false,
  "messages": [
    { "role": "system", "content": "你是天气预报助手" },
    { "role": "user", "content": "北京今天天气怎么样？" },
    {
      "role": "assistant",
      "content": null,
      "tool_calls": [
        { "id": "call_1", "type": "function", "function": { "name": "get_weather", "arguments": "{\"city\":\"北京\"}" } }
      ]
    },
    { "role": "tool", "tool_call_id": "call_1", "content": "北京今天晴，气温 25°C" }
  ],
  "tools": [
    {
      "type": "function",
      "function": {
        "name": "get_weather",
        "description": "获取指定城市的天气",
        "parameters": { "type": "object", "properties": { "city": { "type": "string" } }, "required": ["city"] }
      }
    }
  ],
  "max_tokens": 1024,
  "temperature": 0.7
}`

const goldenResponseOpenAI = `{
  "id": "chatcmpl-123",
  "object": "chat.completion",
  "choices": [
    { "index": 0, "message": { "role": "assistant", "content": "北京今天晴，气温 25°C。" }, "finish_reason": "stop" }
  ],
  "usage": {
    "prompt_tokens": 120,
    "completion_tokens": 15,
    "total_tokens": 135,
    "prompt_tokens_details": { "cached_tokens": 40 }
  }
}`

func intPtr(v int) *int { return &v }

func TestOpenAIConverter_ParseRequest(t *testing.T) {
	c := &OpenAIConverter{}
	got, err := c.ParseRequest([]byte(goldenRequestOpenAI))
	if err != nil {
		t.Fatalf("ParseRequest error: %v", err)
	}

	if got.Model != "gpt-4o" {
		t.Errorf("Model = %q, want %q", got.Model, "gpt-4o")
	}
	if got.Stream {
		t.Errorf("Stream = true, want false")
	}
	if !reflect.DeepEqual(got.MaxTokens, intPtr(1024)) {
		t.Errorf("MaxTokens = %v, want 1024", got.MaxTokens)
	}
	// float32(0.7) 转 float64 存在精度差异，用近似比较。
	if got.Temperature == nil || *got.Temperature < 0.699 || *got.Temperature > 0.701 {
		t.Errorf("Temperature = %v, want ~0.7", got.Temperature)
	}

	// system 提取
	if len(got.System) != 1 || got.System[0].Type != "text" || got.System[0].Text != "你是天气预报助手" {
		t.Errorf("System = %+v, want single text block", got.System)
	}

	// messages：user / assistant / tool
	if len(got.Messages) != 3 {
		t.Fatalf("len(Messages) = %d, want 3", len(got.Messages))
	}

	user := got.Messages[0]
	if user.Role != "user" || len(user.Content) != 1 || user.Content[0].Text != "北京今天天气怎么样？" {
		t.Errorf("user message = %+v", user)
	}

	assistant := got.Messages[1]
	if assistant.Role != "assistant" {
		t.Fatalf("assistant.Role = %q", assistant.Role)
	}
	if len(assistant.ToolCalls) != 1 {
		t.Fatalf("len(assistant.ToolCalls) = %d, want 1", len(assistant.ToolCalls))
	}
	if rawToString(assistant.ToolCalls[0].Arguments) != `{"city":"北京"}` {
		t.Errorf("ToolCalls[0].Arguments = %q", rawToString(assistant.ToolCalls[0].Arguments))
	}
	if len(assistant.Content) != 1 || assistant.Content[0].Type != "tool_use" {
		t.Fatalf("assistant.Content = %+v, want single tool_use block", assistant.Content)
	}
	if assistant.Content[0].ID != "call_1" || assistant.Content[0].Name != "get_weather" {
		t.Errorf("tool_use block = %+v", assistant.Content[0])
	}
	if string(assistant.Content[0].Input) != `{"city":"北京"}` {
		t.Errorf("tool_use Input = %s", assistant.Content[0].Input)
	}

	tool := got.Messages[2]
	if tool.Role != "tool" || tool.ToolCallID != "call_1" {
		t.Fatalf("tool message = %+v", tool)
	}
	if len(tool.Content) != 1 || tool.Content[0].Type != "tool_result" {
		t.Fatalf("tool.Content = %+v, want single tool_result block", tool.Content)
	}
	if tool.Content[0].ID != "call_1" || len(tool.Content[0].Content) != 1 || tool.Content[0].Content[0].Text != "北京今天晴，气温 25°C" {
		t.Errorf("tool_result block = %+v", tool.Content[0])
	}

	// tools
	if len(got.Tools) != 1 || got.Tools[0].Name != "get_weather" || got.Tools[0].Description != "获取指定城市的天气" {
		t.Fatalf("Tools = %+v", got.Tools)
	}
	if len(got.Tools[0].Parameters) == 0 {
		t.Error("Tools[0].Parameters is empty")
	}
}

// TestOpenAIConverter_RequestRoundTrip 验证 Parse → Build → Parse 后关键字段等价。
func TestOpenAIConverter_RequestRoundTrip(t *testing.T) {
	c := &OpenAIConverter{}
	first, err := c.ParseRequest([]byte(goldenRequestOpenAI))
	if err != nil {
		t.Fatalf("first ParseRequest error: %v", err)
	}

	built, err := c.BuildUpstreamRequest(first)
	if err != nil {
		t.Fatalf("BuildUpstreamRequest error: %v", err)
	}

	second, err := c.ParseRequest(built)
	if err != nil {
		t.Fatalf("second ParseRequest error: %v", err)
	}

	if second.Model != first.Model {
		t.Errorf("Model round-trip: %q != %q", second.Model, first.Model)
	}
	if len(second.Messages) != len(first.Messages) {
		t.Fatalf("len(Messages) round-trip: %d != %d", len(second.Messages), len(first.Messages))
	}
	// 逐条比对 role 与 tool_calls。
	for i := range first.Messages {
		if second.Messages[i].Role != first.Messages[i].Role {
			t.Errorf("Messages[%d].Role round-trip: %q != %q", i, second.Messages[i].Role, first.Messages[i].Role)
		}
		if !reflect.DeepEqual(second.Messages[i].ToolCalls, first.Messages[i].ToolCalls) {
			t.Errorf("Messages[%d].ToolCalls round-trip mismatch", i)
		}
	}
	// system 与 tools 等价。
	if !reflect.DeepEqual(second.System, first.System) {
		t.Errorf("System round-trip mismatch")
	}
	if !reflect.DeepEqual(second.Tools, first.Tools) {
		t.Errorf("Tools round-trip mismatch")
	}
}

func TestOpenAIConverter_ParseUpstreamResponse(t *testing.T) {
	c := &OpenAIConverter{}
	got, err := c.ParseUpstreamResponse([]byte(goldenResponseOpenAI))
	if err != nil {
		t.Fatalf("ParseUpstreamResponse error: %v", err)
	}

	if got.ID != "chatcmpl-123" {
		t.Errorf("ID = %q", got.ID)
	}
	if got.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want stop", got.FinishReason)
	}
	if len(got.Content) != 1 || got.Content[0].Type != "text" || got.Content[0].Text != "北京今天晴，气温 25°C。" {
		t.Errorf("Content = %+v", got.Content)
	}
	wantUsage := public.CanonicalUsage{InputTokens: 120, OutputTokens: 15, TotalTokens: 135, CachedTokens: 40}
	if got.Usage != wantUsage {
		t.Errorf("Usage = %+v, want %+v", got.Usage, wantUsage)
	}
}

// TestOpenAIConverter_ResponseRoundTrip 验证响应 Parse → Build → Parse 等价。
func TestOpenAIConverter_ResponseRoundTrip(t *testing.T) {
	c := &OpenAIConverter{}
	first, err := c.ParseUpstreamResponse([]byte(goldenResponseOpenAI))
	if err != nil {
		t.Fatalf("ParseUpstreamResponse error: %v", err)
	}

	built, err := c.BuildResponse(first)
	if err != nil {
		t.Fatalf("BuildResponse error: %v", err)
	}

	second, err := c.ParseUpstreamResponse(built)
	if err != nil {
		t.Fatalf("re-Parse error: %v", err)
	}

	if !reflect.DeepEqual(first, second) {
		t.Errorf("response round-trip mismatch:\n first=%+v\nsecond=%+v", first, second)
	}
}

// TestOpenAIConverter_ParseUpstreamStreamEvent_Text 黄金样本 (3)：文本分片。
func TestOpenAIConverter_ParseUpstreamStreamEvent_Text(t *testing.T) {
	c := &OpenAIConverter{}

	lines := []string{
		`{"choices":[{"index":0,"delta":{"content":"北京"},"finish_reason":null}]}`,
		`{"choices":[{"index":0,"delta":{"content":"今天"},"finish_reason":null}]}`,
		`{"choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":120,"completion_tokens":15,"total_tokens":135}}`,
		`[DONE]`,
	}

	var events []*public.CanonicalStreamEvent
	for _, line := range lines {
		ev, err := c.ParseUpstreamStreamEvent([]byte(line))
		if err != nil {
			t.Fatalf("line %q error: %v", line, err)
		}
		if ev != nil {
			events = append(events, ev)
		}
	}

	if len(events) != 3 {
		t.Fatalf("len(events) = %d, want 3", len(events))
	}
	if events[0].Type != public.StreamEventTextDelta || events[0].Text != "北京" {
		t.Errorf("events[0] = %+v", events[0])
	}
	if events[1].Type != public.StreamEventTextDelta || events[1].Text != "今天" {
		t.Errorf("events[1] = %+v", events[1])
	}
	if events[2].Type != public.StreamEventDone || events[2].FinishReason != "stop" {
		t.Fatalf("events[2] = %+v", events[2])
	}
	if events[2].Usage == nil || events[2].Usage.TotalTokens != 135 {
		t.Errorf("events[2].Usage = %+v", events[2].Usage)
	}
}

// TestOpenAIConverter_ParseUpstreamStreamEvent_ToolCalls 黄金样本 (4)：工具调用分片直出。
func TestOpenAIConverter_ParseUpstreamStreamEvent_ToolCalls(t *testing.T) {
	c := &OpenAIConverter{}

	lines := []string{
		`{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"get_weather","arguments":""}}]},"finish_reason":null}]}`,
		`{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"city\":"}}]},"finish_reason":null}]}`,
		`{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"北京\"}"}}]},"finish_reason":null}]}`,
		`{"choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`,
		`[DONE]`,
	}

	var events []*public.CanonicalStreamEvent
	for _, line := range lines {
		ev, err := c.ParseUpstreamStreamEvent([]byte(line))
		if err != nil {
			t.Fatalf("line %q error: %v", line, err)
		}
		if ev != nil {
			events = append(events, ev)
		}
	}

	if len(events) != 4 {
		t.Fatalf("len(events) = %d, want 4", len(events))
	}
	// 首帧：带 id/name，arguments 为空。
	if events[0].Type != public.StreamEventToolCallDelta {
		t.Fatalf("events[0].Type = %q", events[0].Type)
	}
	if events[0].ToolCall == nil || events[0].ToolCall.ID != "call_1" || events[0].ToolCall.Name != "get_weather" {
		t.Errorf("events[0].ToolCall = %+v", events[0].ToolCall)
	}
	// 后续帧：仅 arguments 分片，id/name 为空。
	if events[1].ToolCall == nil || rawToString(events[1].ToolCall.Arguments) != `{"city":` {
		t.Errorf("events[1].ToolCall = %+v", events[1].ToolCall)
	}
	if events[1].ToolCall.ID != "" || events[1].ToolCall.Name != "" {
		t.Errorf("events[1].ToolCall 应无 id/name: %+v", events[1].ToolCall)
	}
	if events[2].ToolCall == nil || rawToString(events[2].ToolCall.Arguments) != `"北京"}` {
		t.Errorf("events[2].ToolCall = %+v", events[2].ToolCall)
	}
	// 分片拼接后 arguments 完整。
	joined := rawToString(events[0].ToolCall.Arguments) + rawToString(events[1].ToolCall.Arguments) + rawToString(events[2].ToolCall.Arguments)
	if joined != `{"city":"北京"}` {
		t.Errorf("joined arguments = %q, want %q", joined, `{"city":"北京"}`)
	}
	// 结束事件：finish_reason=tool_calls，无 usage。
	if events[3].Type != public.StreamEventDone || events[3].FinishReason != "tool_calls" {
		t.Errorf("events[3] = %+v", events[3])
	}
}

// TestOpenAIConverter_BuildUserStreamEvent 验证 Canonical 事件 → OpenAI chunk 的逆向。
func TestOpenAIConverter_BuildUserStreamEvent(t *testing.T) {
	c := &OpenAIConverter{}

	// text_delta
	b, err := c.BuildUserStreamEvent(&public.CanonicalStreamEvent{Type: public.StreamEventTextDelta, Text: "北京"})
	if err != nil {
		t.Fatal(err)
	}
	ev, err := c.ParseUpstreamStreamEvent(b)
	if err != nil {
		t.Fatal(err)
	}
	if ev == nil || ev.Type != public.StreamEventTextDelta || ev.Text != "北京" {
		t.Errorf("text_delta round-trip = %+v", ev)
	}

	// done with usage
	usage := public.CanonicalUsage{InputTokens: 120, OutputTokens: 15, TotalTokens: 135, CachedTokens: 40}
	b, err = c.BuildUserStreamEvent(&public.CanonicalStreamEvent{Type: public.StreamEventDone, FinishReason: "stop", Usage: &usage})
	if err != nil {
		t.Fatal(err)
	}
	ev, err = c.ParseUpstreamStreamEvent(b)
	if err != nil {
		t.Fatal(err)
	}
	if ev == nil || ev.Type != public.StreamEventDone || ev.FinishReason != "stop" {
		t.Fatalf("done round-trip = %+v", ev)
	}
	if ev.Usage == nil || ev.Usage.CachedTokens != 40 {
		t.Errorf("done usage round-trip = %+v", ev.Usage)
	}
}

// TestOpenAIConverter_EmptyContent 验证 content 为 null/空时不 panic 且正确标记。
func TestOpenAIConverter_EmptyContent(t *testing.T) {
	c := &OpenAIConverter{}
	body := `{"model":"gpt-4o","messages":[{"role":"user","content":null}]}`
	got, err := c.ParseRequest([]byte(body))
	if err != nil {
		t.Fatalf("ParseRequest error: %v", err)
	}
	if len(got.Messages) != 1 {
		t.Fatalf("len(Messages) = %d", len(got.Messages))
	}
	if len(got.Messages[0].Content) != 0 {
		t.Errorf("Content = %+v, want empty", got.Messages[0].Content)
	}
}

// TestOpenAIConverter_Multimodal 验证多模态（text+image）content 数组解析。
func TestOpenAIConverter_Multimodal(t *testing.T) {
	c := &OpenAIConverter{}
	body := `{
		"model": "gpt-4o",
		"messages": [{
			"role": "user",
			"content": [
				{ "type": "text", "text": "这张图里有什么？" },
				{ "type": "image_url", "image_url": { "url": "data:image/png;base64,AAAA", "detail": "high" } }
			]
		}]
	}`
	got, err := c.ParseRequest([]byte(body))
	if err != nil {
		t.Fatalf("ParseRequest error: %v", err)
	}
	if len(got.Messages) != 1 || len(got.Messages[0].Content) != 2 {
		t.Fatalf("Messages[0].Content = %+v", got.Messages[0].Content)
	}
	text := got.Messages[0].Content[0]
	img := got.Messages[0].Content[1]
	if text.Type != "text" || text.Text != "这张图里有什么？" {
		t.Errorf("text block = %+v", text)
	}
	if img.Type != "image" || img.ImageURL != "data:image/png;base64,AAAA" {
		t.Errorf("image block = %+v", img)
	}
	if img.Extra == nil || string(img.Extra["detail"]) != `"high"` {
		t.Errorf("image detail Extra = %+v", img.Extra)
	}

	// round-trip：detail 还原。
	built, err := c.BuildUpstreamRequest(got)
	if err != nil {
		t.Fatal(err)
	}
	var obj struct {
		Messages []struct {
			Content []struct {
				Type     string `json:"type"`
				ImageURL struct {
					URL    string `json:"url"`
					Detail string `json:"detail"`
				} `json:"image_url"`
			} `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(built, &obj); err != nil {
		t.Fatal(err)
	}
	if obj.Messages[0].Content[1].ImageURL.Detail != "high" {
		t.Errorf("detail round-trip = %q, want high", obj.Messages[0].Content[1].ImageURL.Detail)
	}
}

// TestOpenAIConverter_InvalidInput 验证空/非法输入返回明确 error。
func TestOpenAIConverter_InvalidInput(t *testing.T) {
	c := &OpenAIConverter{}
	cases := []string{
		``,
		`{invalid json`,
		`{"messages":[]}`,    // 缺 model
		`{"model":"gpt-4o"}`, // 缺 messages
		`{"model":"gpt-4o","messages":"bad"}`,
	}
	for _, in := range cases {
		if _, err := c.ParseRequest([]byte(in)); err == nil {
			t.Errorf("ParseRequest(%q) 应返回 error", in)
		}
	}
}

// TestOpenAIConverter_DeveloperRole 验证 developer 角色归一化：
// Parse 时 developer 视同 system 提取到 System；Build 时 developer 映射为 system。
func TestOpenAIConverter_DeveloperRole(t *testing.T) {
	c := &OpenAIConverter{}

	// Parse：developer 消息 → System
	got, err := c.ParseRequest([]byte(`{"model":"gpt-4o","messages":[{"role":"developer","content":"你是助手"}]}`))
	if err != nil {
		t.Fatalf("ParseRequest error: %v", err)
	}
	if len(got.System) != 1 || got.System[0].Text != "你是助手" {
		t.Errorf("System = %+v, want developer content extracted", got.System)
	}

	// Build：Canonical 中 developer 消息 → system 角色
	cr := &public.CanonicalRequest{
		Model: "gpt-4o",
		Messages: []public.CanonicalMessage{
			{Role: "developer", Content: []public.CanonicalContent{{Type: "text", Text: "你是助手"}}},
		},
	}
	built, err := c.BuildUpstreamRequest(cr)
	if err != nil {
		t.Fatalf("BuildUpstreamRequest error: %v", err)
	}
	var obj struct {
		Messages []struct {
			Role string `json:"role"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(built, &obj); err != nil {
		t.Fatal(err)
	}
	if len(obj.Messages) != 1 || obj.Messages[0].Role != "system" {
		t.Errorf("built messages role = %+v, want system", obj.Messages)
	}
}

// TestOpenAIConverter_FreeformSchemaInjection 验证 FREEFORM 工具（apply_patch）
// 的空 schema 被注入 input 字符串属性 schema，引导模型把 patch 内容放进 input 字段。
func TestOpenAIConverter_FreeformSchemaInjection(t *testing.T) {
	c := &OpenAIConverter{}
	cr := &public.CanonicalRequest{
		Model: "gpt-4o",
		Messages: []public.CanonicalMessage{
			{Role: "user", Content: []public.CanonicalContent{{Type: "text", Text: "fix the bug"}}},
		},
		Tools: []public.CanonicalTool{
			// apply_patch with empty schema (FREEFORM tool)
			{Type: "function", Name: "apply_patch", Description: "Apply a patch. This is a FREEFORM tool.", Parameters: json.RawMessage(`{"properties":{},"type":"object"}`)},
			// exec_command with real schema (should NOT be injected)
			{Type: "function", Name: "exec_command", Description: "Run a command.", Parameters: json.RawMessage(`{"type":"object","properties":{"cmd":{"type":"string"}},"required":["cmd"]}`)},
		},
	}
	built, err := c.BuildUpstreamRequest(cr)
	if err != nil {
		t.Fatalf("BuildUpstreamRequest error: %v", err)
	}
	var obj struct {
		Tools []struct {
			Type     string `json:"type"`
			Function struct {
				Name       string          `json:"name"`
				Parameters json.RawMessage `json:"parameters"`
			} `json:"function"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(built, &obj); err != nil {
		t.Fatal(err)
	}
	if len(obj.Tools) != 2 {
		t.Fatalf("len(tools) = %d, want 2", len(obj.Tools))
	}
	// apply_patch: schema should have "input" property
	var schema1 struct {
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	if err := json.Unmarshal(obj.Tools[0].Function.Parameters, &schema1); err != nil {
		t.Fatal(err)
	}
	if _, ok := schema1.Properties["input"]; !ok {
		t.Errorf("apply_patch schema missing 'input' property: %s", string(obj.Tools[0].Function.Parameters))
	}
	if len(schema1.Required) != 1 || schema1.Required[0] != "input" {
		t.Errorf("apply_patch schema required = %v, want [input]", schema1.Required)
	}
	// exec_command: schema should be unchanged (has "cmd", no "input")
	var schema2 struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(obj.Tools[1].Function.Parameters, &schema2); err != nil {
		t.Fatal(err)
	}
	if _, ok := schema2.Properties["input"]; ok {
		t.Errorf("exec_command schema should NOT have 'input' property")
	}
	if _, ok := schema2.Properties["cmd"]; !ok {
		t.Errorf("exec_command schema missing 'cmd' property")
	}
}

// TestOpenAIConverter_FreeformArgumentsNormalization 验证 FREEFORM 工具
// （apply_patch）的自由文本 arguments 在转成 Chat Completions 上游请求时，
// 被归一化为 {"input":"<原文>"}，避免上游因「arguments 非 JSON」返回 400。
func TestOpenAIConverter_FreeformArgumentsNormalization(t *testing.T) {
	c := &OpenAIConverter{}
	patchText := "*** Add File: fig_v3/engine_v3.py ***\n# -*- coding: utf-8 -*-\n"

	cr := &public.CanonicalRequest{
		Model: "gpt-4o",
		Messages: []public.CanonicalMessage{
			{Role: "user", Content: []public.CanonicalContent{{Type: "text", Text: "继续"}}},
			{
				Role: "assistant",
				Content: []public.CanonicalContent{
					{Type: "tool_use", ID: "chatcmpl-tool-bfbedd9cd35b1746", Name: "apply_patch", Input: marshalString(patchText)},
				},
				ToolCalls: []public.CanonicalToolCall{
					{ID: "chatcmpl-tool-bfbedd9cd35b1746", Name: "apply_patch", Arguments: marshalString(patchText)},
				},
			},
		},
		Tools: []public.CanonicalTool{
			{Type: "function", Name: "apply_patch", Description: "Apply a patch. FREEFORM tool.", Parameters: json.RawMessage(`{"properties":{},"type":"object"}`)},
		},
	}

	built, err := c.BuildUpstreamRequest(cr)
	if err != nil {
		t.Fatalf("BuildUpstreamRequest error: %v", err)
	}

	var obj struct {
		Messages []struct {
			Role      string `json:"role"`
			ToolCalls []struct {
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(built, &obj); err != nil {
		t.Fatalf("unmarshal built request: %v", err)
	}

	// 找到 assistant 消息的 tool_calls。
	var args string
	for _, m := range obj.Messages {
		if m.Role == "assistant" && len(m.ToolCalls) > 0 {
			args = m.ToolCalls[0].Function.Arguments
		}
	}
	if args == "" {
		t.Fatalf("no assistant tool_call found in built request: %s", string(built))
	}

	// arguments 必须是合法 JSON 对象，且 input 字段还原出原始 patch 文本。
	var argObj map[string]string
	if err := json.Unmarshal([]byte(args), &argObj); err != nil {
		t.Fatalf("arguments is not valid JSON: %q (err=%v)", args, err)
	}
	if argObj["input"] != patchText {
		t.Errorf("arguments.input = %q, want %q", argObj["input"], patchText)
	}
}

// TestOpenAIConverter_FreeformArgumentsAlreadyJSON 验证已经是合法 JSON 对象的
// arguments（如 {"input":"..."}）不会被二次包装。
func TestOpenAIConverter_FreeformArgumentsAlreadyJSON(t *testing.T) {
	c := &OpenAIConverter{}
	cr := &public.CanonicalRequest{
		Model: "gpt-4o",
		Messages: []public.CanonicalMessage{
			{Role: "user", Content: []public.CanonicalContent{{Type: "text", Text: "hi"}}},
			{
				Role:      "assistant",
				ToolCalls: []public.CanonicalToolCall{{ID: "call_1", Name: "apply_patch", Arguments: json.RawMessage(`"{\"input\":\"x\"}"`)}},
			},
		},
		Tools: []public.CanonicalTool{
			{Type: "function", Name: "apply_patch", Description: "FREEFORM tool.", Parameters: json.RawMessage(`{"properties":{},"type":"object"}`)},
		},
	}
	built, err := c.BuildUpstreamRequest(cr)
	if err != nil {
		t.Fatalf("BuildUpstreamRequest error: %v", err)
	}
	var obj struct {
		Messages []struct {
			Role      string `json:"role"`
			ToolCalls []struct {
				Function struct {
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(built, &obj); err != nil {
		t.Fatal(err)
	}
	args := obj.Messages[1].ToolCalls[0].Function.Arguments
	if args != `{"input":"x"}` {
		t.Errorf("arguments = %q, want %q (no double wrapping)", args, `{"input":"x"}`)
	}
}

// TestOpenAIConverter_MalformedToolArgumentsNormalization 验证上游模型生成的
// 畸形工具调用——非 FREEFORM 工具（如 exec_command）却带自由文本（patch 内容）
// 作为 arguments——也会被归一化为 {"input":"<原文>"}，避免上游 json.loads 报 400
// （"Expecting value: line 1 column 2"）。
func TestOpenAIConverter_MalformedToolArgumentsNormalization(t *testing.T) {
	c := &OpenAIConverter{}
	// 复现 server.log 中的畸形调用：exec_command 却携带 patch 文本。
	malformed := " Begin Patch\n*** Update File: fig_v3/engine.py\n def emit(...):\n*** End Patch\n"

	cr := &public.CanonicalRequest{
		Model: "gpt-4o",
		Messages: []public.CanonicalMessage{
			{Role: "user", Content: []public.CanonicalContent{{Type: "text", Text: "fix emit"}}},
			{
				Role:      "assistant",
				ToolCalls: []public.CanonicalToolCall{{ID: "fc_5cd7142d1a1e0195", Name: "exec_command", Arguments: marshalString(malformed)}},
			},
		},
		Tools: []public.CanonicalTool{
			{Type: "function", Name: "exec_command", Description: "Run a command.", Parameters: json.RawMessage(`{"type":"object","properties":{"cmd":{"type":"string"}},"required":["cmd"]}`)},
		},
	}
	built, err := c.BuildUpstreamRequest(cr)
	if err != nil {
		t.Fatalf("BuildUpstreamRequest error: %v", err)
	}
	var obj struct {
		Messages []struct {
			Role      string `json:"role"`
			ToolCalls []struct {
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(built, &obj); err != nil {
		t.Fatal(err)
	}
	args := obj.Messages[1].ToolCalls[0].Function.Arguments
	var argObj map[string]string
	if err := json.Unmarshal([]byte(args), &argObj); err != nil {
		t.Fatalf("arguments is not valid JSON: %q (err=%v)", args, err)
	}
	if argObj["input"] != malformed {
		t.Errorf("arguments.input = %q, want %q", argObj["input"], malformed)
	}
}

// TestOpenAIConverter_FreeformResponseExtraction 验证非流式响应中，
// 模型返回的 {"input":"<patch>"} 被提取为原始 patch 内容字符串。
func TestOpenAIConverter_FreeformResponseExtraction(t *testing.T) {
	c := &OpenAIConverter{}
	patchContent := "*** Begin Patch\n*** Update File: main.go\n@@ -1,3 +1,3 @@\n-old\n+new\n*** End Patch"
	// 模型按注入 schema 返回 {"input":"<patch content>"}
	// 用 json.Marshal 构造合法的 arguments JSON（patchContent 中的换行会被转义）
	argsObj := map[string]string{"input": patchContent}
	argsBytes, _ := json.Marshal(argsObj)
	argsJSON := string(argsBytes)
	respBody := `{
		"id": "chatcmpl-1",
		"object": "chat.completion",
		"choices": [{
			"index": 0,
			"message": {
				"role": "assistant",
				"content": null,
				"tool_calls": [{
					"id": "call_1",
					"type": "function",
					"function": {"name": "apply_patch", "arguments": ` + strconv.Quote(argsJSON) + `}
				}]
			},
			"finish_reason": "tool_calls"
		}]
	}`
	got, err := c.ParseUpstreamResponse([]byte(respBody))
	if err != nil {
		t.Fatalf("ParseUpstreamResponse error: %v", err)
	}
	// 找到 tool_use 块
	var toolUse *public.CanonicalContent
	for i := range got.Content {
		if got.Content[i].Type == "tool_use" {
			toolUse = &got.Content[i]
			break
		}
	}
	if toolUse == nil {
		t.Fatalf("no tool_use block in response")
	}
	if toolUse.Name != "apply_patch" {
		t.Errorf("tool_use name = %q, want apply_patch", toolUse.Name)
	}
	// Input should be the raw patch content (as JSON string literal)
	var extracted string
	if err := json.Unmarshal(toolUse.Input, &extracted); err != nil {
		t.Fatalf("failed to unmarshal Input as string: %v, raw=%s", err, string(toolUse.Input))
	}
	if extracted != patchContent {
		t.Errorf("extracted input = %q, want %q", extracted, patchContent)
	}
}

// TestOpenAIConverter_FreeformStreamExtraction 验证流式响应中，
// 模型分片返回的 {"input":"<patch>"} 被缓冲并提取为原始 patch 内容。
func TestOpenAIConverter_FreeformStreamExtraction(t *testing.T) {
	// 这个测试验证 responsesUserWriter 的 FREEFORM 缓冲逻辑。
	patchContent := "*** Begin Patch\n*** End Patch"
	// 构造合法的 arguments JSON
	argsObj := map[string]string{"input": patchContent}
	argsBytes, _ := json.Marshal(argsObj)
	fullArgs := string(argsBytes)

	// 构造 OpenAI 流式 chunks
	chunk1 := `{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"apply_patch","arguments":""}}]},"finish_reason":null}]}`
	// 发送完整 arguments（JSON 字符串形式，需转义）
	chunk2 := `{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":` + strconv.Quote(fullArgs) + `}}]},"finish_reason":null}]}`
	chunk3 := `{"choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`
	chunk4 := `[DONE]`

	oc := &OpenAIConverter{}
	// Parse upstream stream events (OpenAI format → Canonical)
	var canonicalEvents []*public.CanonicalStreamEvent
	for _, line := range []string{chunk1, chunk2, chunk3, chunk4} {
		ev, err := oc.ParseUpstreamStreamEvent([]byte(line))
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if ev != nil {
			canonicalEvents = append(canonicalEvents, ev)
		}
	}

	// Feed canonical events into responsesUserWriter (Canonical → Responses stream)
	w := (&ResponsesConverter{}).NewUserStreamWriter()
	if setter, ok := w.(interface{ SetModel(string) }); ok {
		setter.SetModel("gpt-4o")
	}
	var outputFrames [][]byte
	for _, ev := range canonicalEvents {
		frames, err := w.Write(ev)
		if err != nil {
			t.Fatalf("write error: %v", err)
		}
		outputFrames = append(outputFrames, frames...)
	}

	// Find the custom_tool_call_input.delta event that contains the extracted patch
	var foundPatchDelta bool
	for _, frame := range outputFrames {
		var ev struct {
			Type  string `json:"type"`
			Delta string `json:"delta"`
		}
		if json.Unmarshal(frame, &ev) == nil && ev.Type == "response.custom_tool_call_input.delta" {
			if ev.Delta == patchContent {
				foundPatchDelta = true
			}
		}
	}
	if !foundPatchDelta {
		t.Errorf("did not find custom_tool_call_input.delta with extracted patch content %q", patchContent)
		// Log all delta events for debugging
		for _, frame := range outputFrames {
			var ev struct {
				Type  string `json:"type"`
				Delta string `json:"delta"`
			}
			if json.Unmarshal(frame, &ev) == nil && strings.Contains(ev.Type, "delta") {
				t.Logf("  delta event: type=%s delta=%q", ev.Type, ev.Delta)
			}
		}
	}

	// Also verify the output_item.done has the raw patch content as input (custom_tool_call field)
	var foundDoneWithPatch bool
	for _, frame := range outputFrames {
		var ev struct {
			Type string `json:"type"`
			Item struct {
				Input string `json:"input"`
			} `json:"item"`
		}
		if json.Unmarshal(frame, &ev) == nil && ev.Type == "response.output_item.done" {
			if ev.Item.Input == patchContent {
				foundDoneWithPatch = true
			}
		}
	}
	if !foundDoneWithPatch {
		t.Errorf("did not find output_item.done with raw patch content as input")
	}
}

// TestOpenAIConverter_AssistantMessageHasContent 验证当 assistant 消息只有
// tool_calls 而没有 text content 时，序列化后的 JSON 仍然包含 "content" 字段
// （空字符串）。vLLM 等后端要求 assistant 消息必须带 content 字段，否则返回 400。
func TestOpenAIConverter_AssistantMessageHasContent(t *testing.T) {
	c := &OpenAIConverter{}
	cr := &public.CanonicalRequest{
		Model: "gpt-4o",
		Messages: []public.CanonicalMessage{
			{Role: "user", Content: []public.CanonicalContent{{Type: "text", Text: "list files"}}},
			// assistant 消息只有 tool_use，没有 text content
			{Role: "assistant", Content: []public.CanonicalContent{}, ToolCalls: []public.CanonicalToolCall{
				{ID: "call_123", Name: "exec_command", Arguments: json.RawMessage(`{"cmd":"ls"}`)},
			}},
			{Role: "tool", ToolCallID: "call_123", Content: []public.CanonicalContent{
				{Type: "tool_result", Content: []public.CanonicalContent{{Type: "text", Text: "file1.txt"}}},
			}},
		},
	}
	built, err := c.BuildUpstreamRequest(cr)
	if err != nil {
		t.Fatalf("BuildUpstreamRequest error: %v", err)
	}
	var obj struct {
		Messages []map[string]json.RawMessage `json:"messages"`
	}
	if err := json.Unmarshal(built, &obj); err != nil {
		t.Fatalf("unmarshal built request error: %v", err)
	}
	if len(obj.Messages) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(obj.Messages))
	}
	// messages[1] 是 assistant 消息
	assistantMsg := obj.Messages[1]
	var role string
	_ = json.Unmarshal(assistantMsg["role"], &role)
	if role != "assistant" {
		t.Fatalf("expected messages[1] to be assistant, got %s", role)
	}
	// 检查是否有 content 字段
	contentRaw, hasContent := assistantMsg["content"]
	if !hasContent {
		t.Fatalf("assistant message missing 'content' field - vLLM will reject with 400")
	}
	var content string
	if err := json.Unmarshal(contentRaw, &content); err != nil {
		t.Fatalf("content field is not a string: %v", err)
	}
	if content != "" {
		t.Fatalf("expected empty content string, got %q", content)
	}
}
