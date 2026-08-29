package format

import (
	"encoding/json"
	"testing"

	"star-fire/pkg/public"
)

// 本测试文件依据 docs/multi-format-api-design.md 3.5 节黄金样本编写。

const goldenRequestAnthropic = `{
  "model": "claude-3-5-sonnet",
  "max_tokens": 1024,
  "system": "你是天气预报助手",
  "messages": [
    { "role": "user", "content": [ { "type": "text", "text": "北京今天天气怎么样？" } ] },
    {
      "role": "assistant",
      "content": [ { "type": "tool_use", "id": "toolu_1", "name": "get_weather", "input": { "city": "北京" } } ]
    },
    {
      "role": "user",
      "content": [ { "type": "tool_result", "tool_use_id": "toolu_1", "content": "北京今天晴，气温 25°C" } ]
    }
  ],
  "tools": [
    {
      "name": "get_weather",
      "description": "获取指定城市的天气",
      "input_schema": { "type": "object", "properties": { "city": { "type": "string" } }, "required": ["city"] }
    }
  ]
}`

const goldenResponseAnthropic = `{
  "id": "msg_123",
  "type": "message",
  "role": "assistant",
  "content": [ { "type": "text", "text": "北京今天晴，气温 25°C。" } ],
  "stop_reason": "end_turn",
  "usage": { "input_tokens": 120, "output_tokens": 15, "cache_read_input_tokens": 40 }
}`

func TestAnthropicConverter_ParseRequest(t *testing.T) {
	c := &AnthropicConverter{}
	got, err := c.ParseRequest([]byte(goldenRequestAnthropic))
	if err != nil {
		t.Fatalf("ParseRequest error: %v", err)
	}

	if got.Model != "claude-3-5-sonnet" {
		t.Errorf("Model = %q, want claude-3-5-sonnet", got.Model)
	}
	if got.MaxTokens == nil || *got.MaxTokens != 1024 {
		t.Errorf("MaxTokens = %v, want 1024", got.MaxTokens)
	}

	// system 提取（顶层 string → 单 text block）
	if len(got.System) != 1 || got.System[0].Type != "text" || got.System[0].Text != "你是天气预报助手" {
		t.Errorf("System = %+v, want single text block", got.System)
	}

	if len(got.Messages) != 3 {
		t.Fatalf("len(Messages) = %d, want 3", len(got.Messages))
	}

	// user 文本
	user := got.Messages[0]
	if user.Role != "user" || len(user.Content) != 1 || user.Content[0].Type != "text" || user.Content[0].Text != "北京今天天气怎么样？" {
		t.Errorf("user message = %+v", user)
	}

	// assistant tool_use
	assistant := got.Messages[1]
	if assistant.Role != "assistant" {
		t.Fatalf("assistant.Role = %q", assistant.Role)
	}
	if len(assistant.ToolCalls) != 1 {
		t.Fatalf("len(assistant.ToolCalls) = %d, want 1", len(assistant.ToolCalls))
	}
	if assistant.ToolCalls[0].ID != "toolu_1" || assistant.ToolCalls[0].Name != "get_weather" {
		t.Errorf("ToolCalls[0] = %+v", assistant.ToolCalls[0])
	}
	if rawToString(assistant.ToolCalls[0].Arguments) != `{"city":"北京"}` {
		t.Errorf("ToolCalls[0].Arguments = %q", rawToString(assistant.ToolCalls[0].Arguments))
	}
	if len(assistant.Content) != 1 || assistant.Content[0].Type != "tool_use" {
		t.Fatalf("assistant.Content = %+v, want single tool_use block", assistant.Content)
	}
	if string(assistant.Content[0].Input) != `{"city":"北京"}` {
		t.Errorf("tool_use Input = %s", assistant.Content[0].Input)
	}

	// tool_result → role=tool
	tool := got.Messages[2]
	if tool.Role != "tool" || tool.ToolCallID != "toolu_1" {
		t.Fatalf("tool message = %+v", tool)
	}
	if len(tool.Content) != 1 || tool.Content[0].Type != "tool_result" || tool.Content[0].ID != "toolu_1" {
		t.Fatalf("tool.Content = %+v, want single tool_result block", tool.Content)
	}
	if len(tool.Content[0].Content) != 1 || tool.Content[0].Content[0].Text != "北京今天晴，气温 25°C" {
		t.Errorf("tool_result nested content = %+v", tool.Content[0].Content)
	}

	// tools
	if len(got.Tools) != 1 {
		t.Fatalf("len(Tools) = %d, want 1", len(got.Tools))
	}
	if got.Tools[0].Name != "get_weather" || got.Tools[0].Description != "获取指定城市的天气" {
		t.Errorf("Tools[0] = %+v", got.Tools[0])
	}
}

func TestAnthropicConverter_ParseRequest_MissingModel(t *testing.T) {
	c := &AnthropicConverter{}
	_, err := c.ParseRequest([]byte(`{"messages":[{"role":"user","content":"hi"}]}`))
	if err == nil {
		t.Fatal("expected error for missing model")
	}
}

func TestAnthropicConverter_ParseRequest_MissingMessages(t *testing.T) {
	c := &AnthropicConverter{}
	_, err := c.ParseRequest([]byte(`{"model":"claude-3-5-sonnet"}`))
	if err == nil {
		t.Fatal("expected error for missing messages")
	}
}

func TestAnthropicConverter_MaxTokensDefault(t *testing.T) {
	c := &AnthropicConverter{}
	// Canonical 无 MaxTokens → Build 时自动补 4096。
	cr := &public.CanonicalRequest{
		Model:    "claude-3-5-sonnet",
		Messages: []public.CanonicalMessage{{Role: "user", Content: []public.CanonicalContent{{Type: "text", Text: "hi"}}}},
	}
	out, err := c.BuildUpstreamRequest(cr)
	if err != nil {
		t.Fatalf("BuildUpstreamRequest error: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if v, ok := m["max_tokens"].(float64); !ok || int(v) != 4096 {
		t.Errorf("max_tokens = %v, want 4096", m["max_tokens"])
	}
}

func TestAnthropicConverter_RequestRoundTrip(t *testing.T) {
	c := &AnthropicConverter{}
	got1, err := c.ParseRequest([]byte(goldenRequestAnthropic))
	if err != nil {
		t.Fatalf("ParseRequest error: %v", err)
	}
	out, err := c.BuildUpstreamRequest(got1)
	if err != nil {
		t.Fatalf("BuildUpstreamRequest error: %v", err)
	}
	got2, err := c.ParseRequest(out)
	if err != nil {
		t.Fatalf("re-ParseRequest error: %v", err)
	}

	if got2.Model != got1.Model {
		t.Errorf("Model round-trip: %q vs %q", got2.Model, got1.Model)
	}
	if len(got2.System) != len(got1.System) || got2.System[0].Text != got1.System[0].Text {
		t.Errorf("System round-trip mismatch: %+v vs %+v", got2.System, got1.System)
	}
	if len(got2.Messages) != len(got1.Messages) {
		t.Fatalf("Messages length round-trip: %d vs %d", len(got2.Messages), len(got1.Messages))
	}
	// 工具调用参数在 Arguments（字符串）与 Input（对象）之间互转后应保持一致。
	assistant1 := got1.Messages[1]
	assistant2 := got2.Messages[1]
	if rawToString(assistant2.ToolCalls[0].Arguments) != rawToString(assistant1.ToolCalls[0].Arguments) {
		t.Errorf("arguments round-trip: %q vs %q", rawToString(assistant2.ToolCalls[0].Arguments), rawToString(assistant1.ToolCalls[0].Arguments))
	}
}

func TestAnthropicConverter_ParseUpstreamResponse(t *testing.T) {
	c := &AnthropicConverter{}
	got, err := c.ParseUpstreamResponse([]byte(goldenResponseAnthropic))
	if err != nil {
		t.Fatalf("ParseUpstreamResponse error: %v", err)
	}

	if got.ID != "msg_123" {
		t.Errorf("ID = %q, want msg_123", got.ID)
	}
	if got.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want stop", got.FinishReason)
	}
	if len(got.Content) != 1 || got.Content[0].Type != "text" || got.Content[0].Text != "北京今天晴，气温 25°C。" {
		t.Errorf("Content = %+v", got.Content)
	}
	u := got.Usage
	if u.InputTokens != 120 || u.OutputTokens != 15 || u.TotalTokens != 135 || u.CachedTokens != 40 {
		t.Errorf("Usage = %+v, want 120/15/135/40", u)
	}
}

func TestAnthropicConverter_ResponseRoundTrip(t *testing.T) {
	c := &AnthropicConverter{}
	got1, err := c.ParseUpstreamResponse([]byte(goldenResponseAnthropic))
	if err != nil {
		t.Fatalf("ParseUpstreamResponse error: %v", err)
	}
	out, err := c.BuildResponse(got1)
	if err != nil {
		t.Fatalf("BuildResponse error: %v", err)
	}
	got2, err := c.ParseUpstreamResponse(out)
	if err != nil {
		t.Fatalf("re-ParseUpstreamResponse error: %v", err)
	}
	if got2.FinishReason != got1.FinishReason {
		t.Errorf("FinishReason round-trip: %q vs %q", got2.FinishReason, got1.FinishReason)
	}
	if len(got2.Content) != len(got1.Content) || got2.Content[0].Text != got1.Content[0].Text {
		t.Errorf("Content round-trip mismatch")
	}
	if got2.Usage != got1.Usage {
		t.Errorf("Usage round-trip: %+v vs %+v", got2.Usage, got1.Usage)
	}
}

// ---- 流式：文本（黄金样本 (3) ②）----

func TestAnthropicConverter_StreamText(t *testing.T) {
	acc := (&AnthropicConverter{}).NewStreamAccumulator()
	frames := []string{
		`{"type":"message_start","message":{"usage":{"input_tokens":120,"cache_read_input_tokens":40}}}`,
		`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"北京"}}`,
		`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"今天"}}`,
		`{"type":"content_block_stop","index":0}`,
		`{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":15}}`,
		`{"type":"message_stop"}`,
	}
	events := feedAll(t, acc, frames)

	if len(events) != 3 {
		t.Fatalf("len(events) = %d, want 3", len(events))
	}
	assertTextDelta(t, events[0], "北京")
	assertTextDelta(t, events[1], "今天")
	if events[2].Type != public.StreamEventDone {
		t.Fatalf("events[2].Type = %q, want done", events[2].Type)
	}
	if events[2].FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want stop", events[2].FinishReason)
	}
	u := events[2].Usage
	if u == nil || u.InputTokens != 120 || u.OutputTokens != 15 || u.TotalTokens != 135 || u.CachedTokens != 40 {
		t.Errorf("done usage = %+v, want 120/15/135/40", u)
	}
}

// ---- 流式：工具调用分片（黄金样本 (4) ②）----

func TestAnthropicConverter_StreamToolCall(t *testing.T) {
	acc := (&AnthropicConverter{}).NewStreamAccumulator()
	frames := []string{
		`{"type":"message_start","message":{"usage":{"input_tokens":80}}}`,
		`{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_1","name":"get_weather","input":{}}}`,
		`{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"city\":"}}`,
		`{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"\"北京\"}"}}`,
		`{"type":"content_block_stop","index":1}`,
		`{"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":12}}`,
		`{"type":"message_stop"}`,
	}
	events := feedAll(t, acc, frames)

	if len(events) != 4 {
		t.Fatalf("len(events) = %d, want 4", len(events))
	}

	// 首帧带 id/name，arguments 为空。
	if events[0].Type != public.StreamEventToolCallDelta || events[0].ToolCall == nil {
		t.Fatalf("events[0] = %+v, want tool_call_delta with id/name", events[0])
	}
	if events[0].ToolCall.ID != "toolu_1" || events[0].ToolCall.Name != "get_weather" {
		t.Errorf("first frame tool_call = %+v", events[0].ToolCall)
	}
	if rawToString(events[0].ToolCall.Arguments) != "" {
		t.Errorf("first frame arguments = %q, want empty", rawToString(events[0].ToolCall.Arguments))
	}

	// 后续帧只带 arguments 分片，id/name 为空。
	if events[1].ToolCall == nil || events[1].ToolCall.ID != "" || events[1].ToolCall.Name != "" {
		t.Errorf("events[1].ToolCall = %+v, want empty id/name", events[1].ToolCall)
	}
	if rawToString(events[1].ToolCall.Arguments) != `{"city":` {
		t.Errorf("events[1] arguments = %q", rawToString(events[1].ToolCall.Arguments))
	}
	if rawToString(events[2].ToolCall.Arguments) != `"北京"}` {
		t.Errorf("events[2] arguments = %q", rawToString(events[2].ToolCall.Arguments))
	}

	// 拼接分片后 arguments 完整。
	var sb []byte
	sb = append(sb, rawToString(events[1].ToolCall.Arguments)...)
	sb = append(sb, rawToString(events[2].ToolCall.Arguments)...)
	if string(sb) != `{"city":"北京"}` {
		t.Errorf("concatenated arguments = %q", string(sb))
	}

	if events[3].Type != public.StreamEventDone || events[3].FinishReason != "tool_calls" {
		t.Fatalf("events[3] = %+v, want done/tool_calls", events[3])
	}
}

func TestAnthropicConverter_StreamError(t *testing.T) {
	acc := (&AnthropicConverter{}).NewStreamAccumulator()
	events := feedAll(t, acc, []string{`{"type":"error","error":{"type":"overloaded_error","message":"模型负载过高，请重试"}}`})
	if len(events) != 1 || events[0].Type != public.StreamEventError {
		t.Fatalf("events = %+v, want single error event", events)
	}
	if events[0].Text != "模型负载过高，请重试" {
		t.Errorf("error text = %q", events[0].Text)
	}
}

// ---- 状态机隔离 ----

func TestAnthropicConverter_StreamIsolation(t *testing.T) {
	acc1 := (&AnthropicConverter{}).NewStreamAccumulator()
	acc2 := (&AnthropicConverter{}).NewStreamAccumulator()

	// 两路流交错喂入，互不串扰。
	feedNoErr(t, acc1, `{"type":"message_start","message":{"usage":{"input_tokens":100}}}`)
	feedNoErr(t, acc2, `{"type":"message_start","message":{"usage":{"input_tokens":200}}}`)
	feedNoErr(t, acc1, `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"A"}}`)
	feedNoErr(t, acc2, `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"B"}}`)

	e1, err := acc1.Flush()
	if err != nil {
		t.Fatalf("acc1.Flush error: %v", err)
	}
	e2, err := acc2.Flush()
	if err != nil {
		t.Fatalf("acc2.Flush error: %v", err)
	}
	if e1.Usage.InputTokens != 100 || e2.Usage.InputTokens != 200 {
		t.Errorf("usage cross-contamination: acc1=%d acc2=%d", e1.Usage.InputTokens, e2.Usage.InputTokens)
	}
}

// ---- 多模态（黄金样本 (6) ②）----

func TestAnthropicConverter_Multimodal(t *testing.T) {
	c := &AnthropicConverter{}
	const img = `{
  "model": "claude-3-5-sonnet",
  "max_tokens": 1024,
  "messages": [
    {
      "role": "user",
      "content": [
        { "type": "text", "text": "这张图里有什么？" },
        {
          "type": "image",
          "source": { "type": "base64", "media_type": "image/png", "data": "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==" }
        }
      ]
    }
  ]
}`
	got, err := c.ParseRequest([]byte(img))
	if err != nil {
		t.Fatalf("ParseRequest error: %v", err)
	}
	m := got.Messages[0]
	if len(m.Content) != 2 {
		t.Fatalf("len(Content) = %d, want 2", len(m.Content))
	}
	if m.Content[0].Type != "text" || m.Content[0].Text != "这张图里有什么？" {
		t.Errorf("text block = %+v", m.Content[0])
	}
	if m.Content[1].Type != "image" {
		t.Fatalf("image block = %+v", m.Content[1])
	}
	want := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="
	if m.Content[1].ImageURL != want {
		t.Errorf("ImageURL = %q, want reassembled data URL", m.Content[1].ImageURL)
	}
}

// ---- thinking（黄金样本 (7) ②）----

func TestAnthropicConverter_Thinking(t *testing.T) {
	c := &AnthropicConverter{}
	const resp = `{
  "id": "msg_456",
  "content": [
    { "type": "thinking", "thinking": "先计算 1+1，得到 2", "signature": "EjY9..." },
    { "type": "text", "text": "答案是 2" }
  ],
  "stop_reason": "end_turn",
  "usage": { "input_tokens": 10, "output_tokens": 30 }
}`
	got, err := c.ParseUpstreamResponse([]byte(resp))
	if err != nil {
		t.Fatalf("ParseUpstreamResponse error: %v", err)
	}
	if len(got.Content) != 2 {
		t.Fatalf("len(Content) = %d, want 2", len(got.Content))
	}
	if got.Content[0].Type != "thinking" || got.Content[0].Text != "先计算 1+1，得到 2" {
		t.Errorf("thinking block = %+v", got.Content[0])
	}
	if rawToString(got.Content[0].Extra["signature"]) != "EjY9..." {
		t.Errorf("signature = %q", rawToString(got.Content[0].Extra["signature"]))
	}
	if got.Content[1].Type != "text" || got.Content[1].Text != "答案是 2" {
		t.Errorf("text block = %+v", got.Content[1])
	}
}

// ---- 辅助 ----

func feedAll(t *testing.T, acc StreamAccumulator, frames []string) []*public.CanonicalStreamEvent {
	t.Helper()
	var out []*public.CanonicalStreamEvent
	for _, f := range frames {
		evs := feedNoErr(t, acc, f)
		out = append(out, evs...)
	}
	if ev, err := acc.Flush(); err != nil {
		t.Fatalf("Flush error: %v", err)
	} else if ev != nil {
		out = append(out, ev)
	}
	return out
}

func feedNoErr(t *testing.T, acc StreamAccumulator, frame string) []*public.CanonicalStreamEvent {
	t.Helper()
	evs, err := acc.Feed([]byte(frame))
	if err != nil {
		t.Fatalf("Feed(%s) error: %v", frame, err)
	}
	return evs
}

func assertTextDelta(t *testing.T, ev *public.CanonicalStreamEvent, want string) {
	t.Helper()
	if ev.Type != public.StreamEventTextDelta {
		t.Fatalf("event type = %q, want text_delta", ev.Type)
	}
	if ev.Text != want {
		t.Errorf("text = %q, want %q", ev.Text, want)
	}
}
