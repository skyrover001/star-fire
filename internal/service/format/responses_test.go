package format

import (
	"encoding/json"
	"strings"
	"testing"

	"star-fire/pkg/public"
)

// 本测试文件依据 docs/multi-format-api-design.md 3.5 节黄金样本编写。

const goldenRequestResponses = `{
  "model": "gpt-4o",
  "instructions": "你是天气预报助手",
  "input": [
    { "role": "user", "content": [ { "type": "input_text", "text": "北京今天天气怎么样？" } ] },
    { "type": "function_call", "call_id": "call_1", "name": "get_weather", "arguments": "{\"city\":\"北京\"}" },
    { "type": "function_call_output", "call_id": "call_1", "output": "北京今天晴，气温 25°C" }
  ],
  "tools": [
    {
      "type": "function",
      "name": "get_weather",
      "description": "获取指定城市的天气",
      "parameters": { "type": "object", "properties": { "city": { "type": "string" } }, "required": ["city"] }
    }
  ]
}`

const goldenResponseResponses = `{
  "id": "resp_123",
  "object": "response",
  "status": "completed",
  "output": [
    { "type": "message", "role": "assistant", "content": [ { "type": "output_text", "text": "北京今天晴，气温 25°C。" } ] }
  ],
  "usage": { "input_tokens": 120, "output_tokens": 15, "total_tokens": 135, "input_tokens_details": { "cached_tokens": 40 } }
}`

func TestResponsesConverter_ParseRequest(t *testing.T) {
	c := &ResponsesConverter{}
	got, err := c.ParseRequest([]byte(goldenRequestResponses))
	if err != nil {
		t.Fatalf("ParseRequest error: %v", err)
	}

	if got.Model != "gpt-4o" {
		t.Errorf("Model = %q, want gpt-4o", got.Model)
	}

	// instructions → System
	if len(got.System) != 1 || got.System[0].Type != "text" || got.System[0].Text != "你是天气预报助手" {
		t.Errorf("System = %+v, want single text block", got.System)
	}

	if len(got.Messages) != 3 {
		t.Fatalf("len(Messages) = %d, want 3", len(got.Messages))
	}

	// user 文本（input 中 type 缺省视作 message）
	user := got.Messages[0]
	if user.Role != "user" || len(user.Content) != 1 || user.Content[0].Type != "text" || user.Content[0].Text != "北京今天天气怎么样？" {
		t.Errorf("user message = %+v", user)
	}

	// function_call → assistant tool_use + tool_calls
	assistant := got.Messages[1]
	if assistant.Role != "assistant" {
		t.Fatalf("assistant.Role = %q", assistant.Role)
	}
	if len(assistant.ToolCalls) != 1 {
		t.Fatalf("len(assistant.ToolCalls) = %d, want 1", len(assistant.ToolCalls))
	}
	if assistant.ToolCalls[0].ID != "call_1" || assistant.ToolCalls[0].Name != "get_weather" {
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

	// function_call_output → role=tool
	tool := got.Messages[2]
	if tool.Role != "tool" || tool.ToolCallID != "call_1" {
		t.Fatalf("tool message = %+v", tool)
	}
	if len(tool.Content) != 1 || tool.Content[0].Type != "tool_result" || tool.Content[0].ID != "call_1" {
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

func TestResponsesConverter_ParseRequest_MissingModel(t *testing.T) {
	c := &ResponsesConverter{}
	_, err := c.ParseRequest([]byte(`{"input":[{"role":"user","content":"hi"}]}`))
	if err == nil {
		t.Fatal("expected error for missing model")
	}
}

func TestResponsesConverter_RequestRoundTrip(t *testing.T) {
	c := &ResponsesConverter{}
	got1, err := c.ParseRequest([]byte(goldenRequestResponses))
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
		t.Errorf("System round-trip mismatch")
	}
	if len(got2.Messages) != len(got1.Messages) {
		t.Fatalf("Messages length round-trip: %d vs %d", len(got2.Messages), len(got1.Messages))
	}
	assistant2 := got2.Messages[1]
	if rawToString(assistant2.ToolCalls[0].Arguments) != `{"city":"北京"}` {
		t.Errorf("arguments round-trip: %q", rawToString(assistant2.ToolCalls[0].Arguments))
	}
}

func TestResponsesConverter_ParseUpstreamResponse(t *testing.T) {
	c := &ResponsesConverter{}
	got, err := c.ParseUpstreamResponse([]byte(goldenResponseResponses))
	if err != nil {
		t.Fatalf("ParseUpstreamResponse error: %v", err)
	}

	if got.ID != "resp_123" {
		t.Errorf("ID = %q, want resp_123", got.ID)
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

func TestResponsesConverter_ResponseRoundTrip(t *testing.T) {
	c := &ResponsesConverter{}
	got1, err := c.ParseUpstreamResponse([]byte(goldenResponseResponses))
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

// ---- 流式：文本（黄金样本 (3) ③）----

func TestResponsesConverter_StreamText(t *testing.T) {
	acc := (&ResponsesConverter{}).NewStreamAccumulator()
	frames := []string{
		`{"type":"response.output_text.delta","item_id":"msg_123","output_index":0,"content_index":0,"delta":"北京"}`,
		`{"type":"response.output_text.delta","item_id":"msg_123","output_index":0,"content_index":0,"delta":"今天"}`,
		`{"type":"response.completed","response":{"id":"resp_123","status":"completed","usage":{"input_tokens":120,"output_tokens":15,"total_tokens":135}}}`,
	}
	events := feedAll(t, acc, frames)

	if len(events) != 3 {
		t.Fatalf("len(events) = %d, want 3", len(events))
	}
	assertTextDelta(t, events[0], "北京")
	assertTextDelta(t, events[1], "今天")
	if events[2].Type != public.StreamEventDone || events[2].FinishReason != "stop" {
		t.Fatalf("events[2] = %+v, want done/stop", events[2])
	}
	u := events[2].Usage
	if u == nil || u.InputTokens != 120 || u.OutputTokens != 15 || u.TotalTokens != 135 || u.CachedTokens != 0 {
		t.Errorf("done usage = %+v, want 120/15/135/0", u)
	}
}

// ---- 流式：工具调用分片（黄金样本 (4) ③）----

func TestResponsesConverter_StreamToolCall(t *testing.T) {
	acc := (&ResponsesConverter{}).NewStreamAccumulator()
	frames := []string{
		`{"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","id":"fc_1","call_id":"call_1","name":"get_weather","arguments":""}}`,
		`{"type":"response.function_call_arguments.delta","item_id":"fc_1","output_index":0,"delta":"{\"city\":"}`,
		`{"type":"response.function_call_arguments.delta","item_id":"fc_1","output_index":0,"delta":"\"北京\"}"}`,
		`{"type":"response.function_call_arguments.done","item_id":"fc_1","output_index":0,"arguments":"{\"city\":\"北京\"}"}`,
		`{"type":"response.completed","response":{"id":"resp_123","status":"completed","usage":{"input_tokens":80,"output_tokens":12,"total_tokens":92}}}`,
	}
	events := feedAll(t, acc, frames)

	if len(events) != 4 {
		t.Fatalf("len(events) = %d, want 4", len(events))
	}

	// 首帧带 id/name，arguments 为空。
	if events[0].Type != public.StreamEventToolCallDelta || events[0].ToolCall == nil {
		t.Fatalf("events[0] = %+v, want tool_call_delta with id/name", events[0])
	}
	if events[0].ToolCall.ID != "call_1" || events[0].ToolCall.Name != "get_weather" {
		t.Errorf("first frame tool_call = %+v", events[0].ToolCall)
	}
	if rawToString(events[0].ToolCall.Arguments) != "" {
		t.Errorf("first frame arguments = %q, want empty", rawToString(events[0].ToolCall.Arguments))
	}

	// 后续帧只带 arguments 分片。
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
	sb := rawToString(events[1].ToolCall.Arguments) + rawToString(events[2].ToolCall.Arguments)
	if sb != `{"city":"北京"}` {
		t.Errorf("concatenated arguments = %q", sb)
	}

	// finish_reason 归一为 tool_calls（流中出现过 function_call）。
	if events[3].Type != public.StreamEventDone || events[3].FinishReason != "tool_calls" {
		t.Fatalf("events[3] = %+v, want done/tool_calls", events[3])
	}
	u := events[3].Usage
	if u == nil || u.InputTokens != 80 || u.OutputTokens != 12 || u.TotalTokens != 92 {
		t.Errorf("done usage = %+v, want 80/12/92", u)
	}
}

func TestResponsesConverter_StreamError(t *testing.T) {
	acc := (&ResponsesConverter{}).NewStreamAccumulator()
	events := feedAll(t, acc, []string{`{"type":"response.failed","delta":"模型负载过高，请重试"}`})
	if len(events) != 1 || events[0].Type != public.StreamEventError {
		t.Fatalf("events = %+v, want single error event", events)
	}
	if events[0].Text != "模型负载过高，请重试" {
		t.Errorf("error text = %q", events[0].Text)
	}
}

func TestResponsesConverter_StreamCustomToolAndOutOfOrderArguments(t *testing.T) {
	acc := (&ResponsesConverter{}).NewStreamAccumulator()
	frames := []string{
		`{"type":"response.custom_tool_call_input.delta","output_index":1,"item_id":"ct_1","delta":"patch"}`,
		`{"type":"response.output_item.added","output_index":1,"item":{"type":"custom_tool_call","id":"ct_1","call_id":"call_patch","name":"apply_patch","input":""}}`,
		`{"type":"response.custom_tool_call_input.done","output_index":1,"item_id":"ct_1","arguments":" body"}`,
		`{"type":"response.completed","response":{"status":"completed"}}`,
	}
	events := feedAll(t, acc, frames)
	if len(events) != 3 {
		t.Fatalf("events = %d, want 3", len(events))
	}
	if events[0].ToolCall.ID != "call_patch" || events[0].ToolCall.Name != "apply_patch" || rawToString(events[0].ToolCall.Arguments) != "patch" {
		t.Errorf("tool metadata/pending args = %+v", events[0].ToolCall)
	}
	if rawToString(events[1].ToolCall.Arguments) != " body" {
		t.Errorf("done arguments = %q", rawToString(events[1].ToolCall.Arguments))
	}
	if events[2].FinishReason != "tool_calls" {
		t.Errorf("finish reason = %q, want tool_calls", events[2].FinishReason)
	}
}

func TestResponsesConverter_StreamIncompleteContentFilter(t *testing.T) {
	acc := (&ResponsesConverter{}).NewStreamAccumulator()
	events := feedAll(t, acc, []string{`{"type":"response.incomplete","response":{"status":"incomplete","incomplete_details":{"reason":"content_filter"}}}`})
	if len(events) != 1 || events[0].Type != public.StreamEventDone || events[0].FinishReason != "content_filter" {
		t.Fatalf("events = %+v, want done/content_filter", events)
	}
}

// ---- 状态机隔离 ----

func TestResponsesConverter_StreamIsolation(t *testing.T) {
	acc1 := (&ResponsesConverter{}).NewStreamAccumulator()
	acc2 := (&ResponsesConverter{}).NewStreamAccumulator()

	feedNoErr(t, acc1, `{"type":"response.output_text.delta","delta":"A"}`)
	feedNoErr(t, acc2, `{"type":"response.output_text.delta","delta":"B"}`)
	feedNoErr(t, acc1, `{"type":"response.completed","response":{"status":"completed","usage":{"input_tokens":100,"output_tokens":5,"total_tokens":105}}}`)
	feedNoErr(t, acc2, `{"type":"response.completed","response":{"status":"completed","usage":{"input_tokens":200,"output_tokens":5,"total_tokens":205}}}`)

	// 已经 completed，Flush 返回 nil。
	e1, _ := acc1.Flush()
	e2, _ := acc2.Flush()
	if e1 != nil || e2 != nil {
		t.Errorf("Flush after completed should be nil, got %+v / %+v", e1, e2)
	}
}

// ---- 多模态（黄金样本 (6) ③）----

func TestResponsesConverter_Multimodal(t *testing.T) {
	c := &ResponsesConverter{}
	const img = `{
  "model": "gpt-4o",
  "input": [
    {
      "role": "user",
      "content": [
        { "type": "input_text", "text": "这张图里有什么？" },
        { "type": "input_image", "image_url": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==" }
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

// ---- thinking（黄金样本 (7) ③）----

func TestResponsesConverter_Thinking(t *testing.T) {
	c := &ResponsesConverter{}
	const resp = `{
  "id": "resp_456",
  "status": "completed",
  "output": [
    { "type": "reasoning", "summary": [ { "type": "summary_text", "text": "先计算 1+1，得到 2" } ] },
    { "type": "message", "role": "assistant", "content": [ { "type": "output_text", "text": "答案是 2" } ] }
  ],
  "usage": { "input_tokens": 10, "output_tokens": 30, "total_tokens": 40 }
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
	if got.Content[1].Type != "text" || got.Content[1].Text != "答案是 2" {
		t.Errorf("text block = %+v", got.Content[1])
	}
}

// ---- max_output_tokens 映射 ----

func TestResponsesConverter_MaxOutputTokens(t *testing.T) {
	c := &ResponsesConverter{}
	// 请求带 max_output_tokens → MaxTokens。
	got, err := c.ParseRequest([]byte(`{"model":"gpt-4o","max_output_tokens":2048,"input":[{"role":"user","content":"hi"}]}`))
	if err != nil {
		t.Fatalf("ParseRequest error: %v", err)
	}
	if got.MaxTokens == nil || *got.MaxTokens != 2048 {
		t.Errorf("MaxTokens = %v, want 2048", got.MaxTokens)
	}

	// 请求不带 max_output_tokens → MaxTokens 为 nil（Build 时补默认值）。
	got2, err := c.ParseRequest([]byte(`{"model":"gpt-4o","input":[{"role":"user","content":"hi"}]}`))
	if err != nil {
		t.Fatalf("ParseRequest error: %v", err)
	}
	if got2.MaxTokens != nil {
		t.Errorf("MaxTokens = %v, want nil", got2.MaxTokens)
	}

	// Build 时无 MaxTokens → 默认 4096。
	out, err := c.BuildUpstreamRequest(got2)
	if err != nil {
		t.Fatalf("BuildUpstreamRequest error: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if v, ok := m["max_output_tokens"].(float64); !ok || int(v) != 4096 {
		t.Errorf("max_output_tokens = %v, want 4096", m["max_output_tokens"])
	}
}

// ---- 用户流式 writer（Codex CLI 兼容）----

// TestResponsesUserWriter_TextThenToolCall 验证“文本 + 工具调用”混合输出：
// 模型先输出文本、再调用工具时，writer 必须输出两个独立 item（message + function_call），
// 而不是把工具调用 delta 挂到 message 上、最终丢弃工具调用（曾导致 Codex 一轮后退出）。
func TestResponsesUserWriter_TextThenToolCall(t *testing.T) {
	w := (&ResponsesConverter{}).NewUserStreamWriter()
	if setter, ok := w.(interface{ SetModel(string) }); ok {
		setter.SetModel("deepseek-v4-pro")
	}

	var all []map[string]any
	write := func(ev *public.CanonicalStreamEvent) {
		t.Helper()
		evs, err := w.Write(ev)
		if err != nil {
			t.Fatalf("Write error: %v", err)
		}
		for _, e := range evs {
			var m map[string]any
			if err := json.Unmarshal(e, &m); err != nil {
				t.Fatalf("unmarshal event %s: %v", string(e), err)
			}
			all = append(all, m)
		}
	}

	write(&public.CanonicalStreamEvent{Type: public.StreamEventTextDelta, Text: "I'll search the web. "})
	write(&public.CanonicalStreamEvent{Type: public.StreamEventToolCallDelta, ToolCall: &public.CanonicalToolCall{
		ID: "call_1", Name: "tool_search", Arguments: marshalString(`{"query":"today"}`),
	}})
	write(&public.CanonicalStreamEvent{Type: public.StreamEventDone, FinishReason: "tool_calls"})

	// 统计 output_item.added 数量（应为 2：message + function_call）。
	var itemAddedTypes []string
	for _, m := range all {
		if m["type"] != "response.output_item.added" {
			continue
		}
		item := m["item"].(map[string]any)
		itemAddedTypes = append(itemAddedTypes, item["type"].(string))
	}
	if len(itemAddedTypes) != 2 {
		t.Fatalf("output_item.added count = %d (%v), want 2 (message + function_call)", len(itemAddedTypes), itemAddedTypes)
	}
	if itemAddedTypes[0] != "message" || itemAddedTypes[1] != "function_call" {
		t.Errorf("item types = %v, want [message function_call]", itemAddedTypes)
	}

	// 校验 function_call delta 的 item_id 是 fc_ 前缀（不是 msg_）。
	sawFuncDelta := false
	for _, m := range all {
		if m["type"] == "response.function_call_arguments.delta" {
			sawFuncDelta = true
			id, _ := m["item_id"].(string)
			if !strings.HasPrefix(id, "fc_") {
				t.Errorf("function_call delta item_id = %q, want fc_ prefix", id)
			}
			if idx, _ := m["output_index"].(float64); idx != 1 {
				t.Errorf("function_call delta output_index = %v, want 1", m["output_index"])
			}
		}
	}
	if !sawFuncDelta {
		t.Errorf("no function_call_arguments.delta emitted")
	}

	// 校验 response.completed 的 output 包含 2 个 item，且第二个是 function_call 带 call_id。
	var completed map[string]any
	for _, m := range all {
		if m["type"] == "response.completed" {
			completed = m
			break
		}
	}
	if completed == nil {
		t.Fatalf("no response.completed emitted")
	}
	resp := completed["response"].(map[string]any)
	out, _ := resp["output"].([]any)
	if len(out) != 2 {
		t.Fatalf("completed output length = %d, want 2", len(out))
	}
	fc := out[1].(map[string]any)
	if fc["type"] != "function_call" {
		t.Errorf("output[1].type = %v, want function_call", fc["type"])
	}
	if fc["call_id"] != "call_1" {
		t.Errorf("output[1].call_id = %v, want call_1", fc["call_id"])
	}
	if fc["name"] != "tool_search" {
		t.Errorf("output[1].name = %v, want tool_search", fc["name"])
	}
}

func TestResponsesUserWriter_EmitsStructuralEvents(t *testing.T) {
	w := (&ResponsesConverter{}).NewUserStreamWriter()
	if setter, ok := w.(interface{ SetModel(string) }); ok {
		setter.SetModel("DeepSeek-V4")
	}

	// 首个文本增量 → 应合成 created + item.added + part.added + delta
	evs, err := w.Write(&public.CanonicalStreamEvent{Type: public.StreamEventTextDelta, Text: "Hello "})
	if err != nil {
		t.Fatalf("Write delta error: %v", err)
	}
	if len(evs) != 4 {
		t.Fatalf("first delta events = %d, want 4 (created+item+part+delta)", len(evs))
	}
	var first map[string]any
	_ = json.Unmarshal(evs[0], &first)
	if first["type"] != "response.created" {
		t.Errorf("event[0] type = %v, want response.created", first["type"])
	}
	var second map[string]any
	_ = json.Unmarshal(evs[1], &second)
	if second["type"] != "response.output_item.added" {
		t.Errorf("event[1] type = %v, want response.output_item.added", second["type"])
	}
	var third map[string]any
	_ = json.Unmarshal(evs[2], &third)
	if third["type"] != "response.content_part.added" {
		t.Errorf("event[2] type = %v, want response.content_part.added", third["type"])
	}
	var fourth map[string]any
	_ = json.Unmarshal(evs[3], &fourth)
	if fourth["type"] != "response.output_text.delta" || fourth["delta"] != "Hello " {
		t.Errorf("event[3] = %v, want output_text.delta Hello ", fourth)
	}

	// 第二个增量 → 只有 delta
	evs2, err := w.Write(&public.CanonicalStreamEvent{Type: public.StreamEventTextDelta, Text: "world"})
	if err != nil {
		t.Fatalf("Write delta2 error: %v", err)
	}
	if len(evs2) != 1 {
		t.Fatalf("second delta events = %d, want 1", len(evs2))
	}

	// done → 应合成 output_text.done + content_part.done + output_item.done + response.completed
	evs3, err := w.Write(&public.CanonicalStreamEvent{
		Type:         public.StreamEventDone,
		FinishReason: "stop",
		Usage:        &public.CanonicalUsage{InputTokens: 7, OutputTokens: 5, TotalTokens: 12},
	})
	if err != nil {
		t.Fatalf("Write done error: %v", err)
	}
	if len(evs3) != 4 {
		t.Fatalf("done events = %d, want 4", len(evs3))
	}
	var done map[string]any
	_ = json.Unmarshal(evs3[0], &done)
	if done["type"] != "response.output_text.done" {
		t.Errorf("done[0] type = %v, want response.output_text.done", done["type"])
	}
	var completed map[string]any
	_ = json.Unmarshal(evs3[3], &completed)
	if completed["type"] != "response.completed" {
		t.Errorf("done[3] type = %v, want response.completed", completed["type"])
	}
	resp := completed["response"].(map[string]any)
	if resp["status"] != "completed" {
		t.Errorf("completed status = %v, want completed", resp["status"])
	}
	if resp["model"] != "DeepSeek-V4" {
		t.Errorf("completed model = %v, want DeepSeek-V4", resp["model"])
	}
	usage := resp["usage"].(map[string]any)
	if usage["total_tokens"].(float64) != 12 {
		t.Errorf("completed usage total = %v, want 12", usage["total_tokens"])
	}
}

// TestResponsesConverter_NonFunctionTools 验证非 function 工具（web_search/tool_search）
// 在下游 Chat 转换时的处理：web_search 被剥离（后端不支持联网搜索），
// tool_search 归一化为 function 类型且 name 非空（后端只接受 function）。
func TestResponsesConverter_NonFunctionTools(t *testing.T) {
	c := &ResponsesConverter{}
	body := `{
		"model": "gpt-4o",
		"input": [{"role":"user","content":"hi"}],
		"tools": [
			{"type":"function","name":"exec_command","description":"run","parameters":{"type":"object"}},
			{"type":"web_search"},
			{"type":"tool_search","max_results":5}
		]
	}`
	got, err := c.ParseRequest([]byte(body))
	if err != nil {
		t.Fatalf("ParseRequest error: %v", err)
	}
	if len(got.Tools) != 3 {
		t.Fatalf("len(Tools) = %d, want 3", len(got.Tools))
	}
	// function 工具：Type=function，Name 保留
	if got.Tools[0].Type != "function" || got.Tools[0].Name != "exec_command" {
		t.Errorf("Tools[0] = %+v", got.Tools[0])
	}
	// web_search：Type=web_search，保留 Raw
	if got.Tools[1].Type != "web_search" || len(got.Tools[1].Raw) == 0 {
		t.Errorf("Tools[1] = %+v, want web_search with Raw", got.Tools[1])
	}
	// tool_search：Type=tool_search，保留 Raw
	if got.Tools[2].Type != "tool_search" || len(got.Tools[2].Raw) == 0 {
		t.Errorf("Tools[2] = %+v, want tool_search with Raw", got.Tools[2])
	}

	// 下游 Chat 转换：web_search 被剥离（后端不支持联网搜索），
	// tool_search 归一化为 function，name 非空
	oc := &OpenAIConverter{}
	built, err := oc.BuildUpstreamRequest(got)
	if err != nil {
		t.Fatalf("BuildUpstreamRequest error: %v", err)
	}
	var obj struct {
		Tools []struct {
			Type     string `json:"type"`
			Function struct {
				Name string `json:"name"`
			} `json:"function"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(built, &obj); err != nil {
		t.Fatal(err)
	}
	// web_search 被剥离 → 只剩 exec_command + tool_search 两个工具
	if len(obj.Tools) != 2 {
		t.Fatalf("built tools = %d, want 2 (web_search stripped)", len(obj.Tools))
	}
	// 剩余工具都应是 function 类型且 name 非空
	for i, tool := range obj.Tools {
		if tool.Type != "function" {
			t.Errorf("tools[%d] type = %v, want function", i, tool.Type)
		}
		if tool.Function.Name == "" {
			t.Errorf("tools[%d] function.name is empty", i)
		}
	}
	// 第一个工具是 exec_command（function 保留）
	if obj.Tools[0].Function.Name != "exec_command" {
		t.Errorf("tools[0] function.name = %q, want exec_command", obj.Tools[0].Function.Name)
	}
	// tool_search → function name = tool_search
	if obj.Tools[1].Function.Name != "tool_search" {
		t.Errorf("tools[1] function.name = %q, want tool_search", obj.Tools[1].Function.Name)
	}
}

// TestResponsesConverter_ToolCallTypeMapping 验证响应方向工具类型映射：
// 模型调用 web_search → 返回 web_search_call；custom → custom_tool_call；function → function_call。
func TestResponsesConverter_ToolCallTypeMapping(t *testing.T) {
	c := &ResponsesConverter{}

	// 构造 Canonical 响应，含不同工具类型
	cr := &public.CanonicalResponse{
		ID: "resp_1",
		Content: []public.CanonicalContent{
			{Type: "tool_use", ID: "fc_1", Name: "exec_command", Input: json.RawMessage(`{"cmd":"ls"}`), ToolType: "function"},
			{Type: "tool_use", ID: "fc_2", Name: "web_search", Input: json.RawMessage(`{"query":"news"}`), ToolType: "web_search"},
			{Type: "tool_use", ID: "fc_3", Name: "apply_patch", Input: json.RawMessage(`{"patch":"x"}`), ToolType: "custom"},
		},
	}

	built, err := c.BuildResponse(cr)
	if err != nil {
		t.Fatalf("BuildResponse error: %v", err)
	}
	var obj struct {
		Output []struct {
			Type string `json:"type"`
			Name string `json:"name"`
		} `json:"output"`
	}
	if err := json.Unmarshal(built, &obj); err != nil {
		t.Fatal(err)
	}
	if len(obj.Output) != 3 {
		t.Fatalf("output = %d, want 3", len(obj.Output))
	}
	if obj.Output[0].Type != "function_call" || obj.Output[0].Name != "exec_command" {
		t.Errorf("output[0] = %+v, want function_call exec_command", obj.Output[0])
	}
	if obj.Output[1].Type != "web_search_call" || obj.Output[1].Name != "web_search" {
		t.Errorf("output[1] = %+v, want web_search_call web_search", obj.Output[1])
	}
	if obj.Output[2].Type != "custom_tool_call" || obj.Output[2].Name != "apply_patch" {
		t.Errorf("output[2] = %+v, want custom_tool_call apply_patch", obj.Output[2])
	}

	// round-trip：ParseUpstreamResponse 应还原 ToolType
	parsed, err := c.ParseUpstreamResponse(built)
	if err != nil {
		t.Fatalf("ParseUpstreamResponse error: %v", err)
	}
	if len(parsed.Content) != 3 {
		t.Fatalf("parsed content = %d, want 3", len(parsed.Content))
	}
	if parsed.Content[1].ToolType != "web_search" {
		t.Errorf("parsed[1].ToolType = %q, want web_search", parsed.Content[1].ToolType)
	}
	if parsed.Content[2].ToolType != "custom" {
		t.Errorf("parsed[2].ToolType = %q, want custom", parsed.Content[2].ToolType)
	}
}

// TestResponsesToChat_ThinkingAndToolCalls 验证 Responses 请求（含 assistant 思考 + 工具调用）
// 转成下游 Chat 请求后 JSON 合法，且 reasoning_content 与 tool_calls 都保留。
func TestResponsesToChat_ThinkingAndToolCalls(t *testing.T) {
	rc := &ResponsesConverter{}
	body := `{
		"model": "deepseek-v4-pro",
		"input": [
			{"role":"user","content":[{"type":"input_text","text":"帮我搜索AI新闻"}]},
			{"type":"message","role":"assistant","content":[
				{"type":"output_text","text":"我来搜索"},
				{"type":"reasoning","summary":[{"type":"summary_text","text":"先搜索再回答"}]}
			]},
			{"type":"function_call","call_id":"fc_1","name":"web_search","arguments":"{\"query\":\"AI news\"}"},
			{"type":"function_call_output","call_id":"fc_1","output":"结果"}
		],
		"tools":[
			{"type":"function","name":"exec_command","description":"run","parameters":{"type":"object"}},
			{"type":"web_search"}
		]
	}`
	cr, err := rc.ParseRequest([]byte(body))
	if err != nil {
		t.Fatalf("ParseRequest error: %v", err)
	}

	// 转成下游 Chat 请求
	oc := &OpenAIConverter{}
	chatBody, err := oc.BuildUpstreamRequest(cr)
	if err != nil {
		t.Fatalf("BuildUpstreamRequest error: %v", err)
	}

	// 验证 JSON 合法（无 Extra data）
	var obj map[string]any
	if err := json.Unmarshal(chatBody, &obj); err != nil {
		t.Fatalf("chat body is invalid JSON: %v\nbody=%s", err, string(chatBody))
	}

	// 验证 assistant 消息保留 reasoning_content 和 tool_calls
	msgs := obj["messages"].([]any)
	var assistantMsg map[string]any
	for _, m := range msgs {
		mm := m.(map[string]any)
		if mm["role"] == "assistant" {
			assistantMsg = mm
			break
		}
	}
	if assistantMsg == nil {
		t.Fatal("no assistant message found")
	}
	if rc2, ok := assistantMsg["reasoning_content"].(string); !ok || rc2 == "" {
		t.Errorf("assistant reasoning_content missing: %+v", assistantMsg)
	}
	if tcs, ok := assistantMsg["tool_calls"].([]any); !ok || len(tcs) == 0 {
		t.Errorf("assistant tool_calls missing: %+v", assistantMsg)
	}
}

// TestResponsesToChat_EmptyFunctionArguments verifies that an interrupted
// Codex function call still produces a valid Chat Completions tool call.
func TestResponsesToChat_EmptyFunctionArguments(t *testing.T) {
	rc := &ResponsesConverter{}
	cr, err := rc.ParseRequest([]byte(`{
		"model":"gpt-4o",
		"input":[
			{"role":"user","content":"continue"},
			{"type":"function_call","call_id":"chatcmpl-tool-interrupted","name":"exec_command","arguments":""}
		]
	}`))
	if err != nil {
		t.Fatalf("ParseRequest error: %v", err)
	}

	chatBody, err := (&OpenAIConverter{}).BuildUpstreamRequest(cr)
	if err != nil {
		t.Fatalf("BuildUpstreamRequest error: %v", err)
	}
	var payload struct {
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
	if err := json.Unmarshal(chatBody, &payload); err != nil {
		t.Fatalf("invalid Chat request JSON: %v", err)
	}
	for _, message := range payload.Messages {
		if message.Role == "assistant" && len(message.ToolCalls) == 1 {
			if got := message.ToolCalls[0].Function.Arguments; got != "{}" {
				t.Fatalf("function.arguments = %q, want {}", got)
			}
			return
		}
	}
	t.Fatal("assistant tool call missing from Chat request")
}

// TestResponsesToChat_WebSearchCallPassback 验证 web_search_call 输入 item（含搜索结果
// output）被转成下游 tool 消息（带 content），而非 content:null 的畸形消息。
func TestResponsesToChat_WebSearchCallPassback(t *testing.T) {
	rc := &ResponsesConverter{}
	body := `{
		"model": "gpt-4o",
		"input": [
			{"role":"user","content":[{"type":"input_text","text":"搜索AI新闻"}]},
			{"type":"function_call","call_id":"fc_1","name":"web_search","arguments":"{\"query\":\"AI news\"}"},
			{"type":"web_search_call","id":"ws_1","action":{"type":"search","query":"AI news"},"status":"completed","output":"AI 新闻结果..."}
		]
	}`
	cr, err := rc.ParseRequest([]byte(body))
	if err != nil {
		t.Fatalf("ParseRequest error: %v", err)
	}

	oc := &OpenAIConverter{}
	chatBody, err := oc.BuildUpstreamRequest(cr)
	if err != nil {
		t.Fatalf("BuildUpstreamRequest error: %v", err)
	}
	var obj map[string]any
	if err := json.Unmarshal(chatBody, &obj); err != nil {
		t.Fatalf("chat body is invalid JSON: %v\nbody=%s", err, string(chatBody))
	}

	msgs := obj["messages"].([]any)
	// 找到 tool 消息，验证 content 非空且携带搜索结果。
	var toolMsg map[string]any
	for _, m := range msgs {
		mm := m.(map[string]any)
		if mm["role"] == "tool" {
			toolMsg = mm
			break
		}
	}
	if toolMsg == nil {
		t.Fatal("no tool message found for web_search_call result")
	}
	content, ok := toolMsg["content"].(string)
	if !ok || content == "" {
		t.Errorf("tool content missing/empty: %+v", toolMsg)
	}
	if content != "AI 新闻结果..." {
		t.Errorf("tool content = %q, want search result text", content)
	}
}

// TestResponsesToChat_ReasoningItemPassback 验证独立的 reasoning item（type=reasoning）
// 被合并到携带 function_call 的 assistant 消息，作为 reasoning_content 回传。
func TestResponsesToChat_ReasoningItemPassback(t *testing.T) {
	rc := &ResponsesConverter{}
	body := `{
		"model": "deepseek-v4-pro",
		"input": [
			{"role":"user","content":[{"type":"input_text","text":"计算 2+2"}]},
			{"type":"reasoning","summary":[{"type":"summary_text","text":"先算 2+2 得到 4"}]},
			{"type":"function_call","call_id":"fc_1","name":"add","arguments":"{\"a\":2,\"b\":2}"},
			{"type":"function_call_output","call_id":"fc_1","output":"4"}
		]
	}`
	cr, err := rc.ParseRequest([]byte(body))
	if err != nil {
		t.Fatalf("ParseRequest error: %v", err)
	}

	oc := &OpenAIConverter{}
	chatBody, err := oc.BuildUpstreamRequest(cr)
	if err != nil {
		t.Fatalf("BuildUpstreamRequest error: %v", err)
	}
	var obj map[string]any
	if err := json.Unmarshal(chatBody, &obj); err != nil {
		t.Fatalf("chat body is invalid JSON: %v\nbody=%s", err, string(chatBody))
	}

	msgs := obj["messages"].([]any)
	// 应只有一条 assistant 消息，且同时含 reasoning_content 与 tool_calls。
	var assistantMsgs []map[string]any
	for _, m := range msgs {
		mm := m.(map[string]any)
		if mm["role"] == "assistant" {
			assistantMsgs = append(assistantMsgs, mm)
		}
	}
	if len(assistantMsgs) != 1 {
		t.Fatalf("assistant messages = %d, want 1", len(assistantMsgs))
	}
	am := assistantMsgs[0]
	if rc2, ok := am["reasoning_content"].(string); !ok || rc2 != "先算 2+2 得到 4" {
		t.Errorf("reasoning_content = %v, want 先算 2+2 得到 4", am["reasoning_content"])
	}
	if tcs, ok := am["tool_calls"].([]any); !ok || len(tcs) == 0 {
		t.Errorf("tool_calls missing: %+v", am)
	}
}

// TestResponsesToChat_DeveloperMerge 验证多条 developer/system 消息被合并为单条 system，
// 避免非首位的 system 被 Qwen 等后端拒绝。
func TestResponsesToChat_DeveloperMerge(t *testing.T) {
	rc := &ResponsesConverter{}
	body := `{
		"model": "qwen",
		"instructions": "你是主系统提示",
		"input": [
			{"type":"message","role":"developer","content":[{"type":"input_text","text":"技能指令"}]},
			{"type":"message","role":"developer","content":[{"type":"input_text","text":"权限指令"}]},
			{"type":"message","role":"user","content":[{"type":"input_text","text":"你好"}]}
		]
	}`
	cr, err := rc.ParseRequest([]byte(body))
	if err != nil {
		t.Fatalf("ParseRequest error: %v", err)
	}

	oc := &OpenAIConverter{}
	chatBody, err := oc.BuildUpstreamRequest(cr)
	if err != nil {
		t.Fatalf("BuildUpstreamRequest error: %v", err)
	}
	var obj map[string]any
	if err := json.Unmarshal(chatBody, &obj); err != nil {
		t.Fatalf("chat body is invalid JSON: %v\nbody=%s", err, string(chatBody))
	}

	msgs := obj["messages"].([]any)
	if len(msgs) != 2 {
		t.Fatalf("messages = %d, want 2 (system + user)", len(msgs))
	}
	first := msgs[0].(map[string]any)
	if first["role"] != "system" {
		t.Errorf("first message role = %v, want system", first["role"])
	}
	second := msgs[1].(map[string]any)
	if second["role"] != "user" {
		t.Errorf("second message role = %v, want user", second["role"])
	}
}

// TestResponsesToChat_WebSearchResultOutput 验证 web_search_call 的 output 为
// web_search_result 数组时，标题/链接/正文被透传到下游 tool 消息（而非被吞掉）。
func TestResponsesToChat_WebSearchResultOutput(t *testing.T) {
	rc := &ResponsesConverter{}
	body := `{
		"model": "gpt-4o",
		"input": [
			{"role":"user","content":[{"type":"input_text","text":"搜索AI新闻"}]},
			{"type":"web_search_call","call_id":"ws_1","action":{"type":"search","query":"AI news"},"status":"completed","output":[{"type":"web_search_result","title":"AI新闻","url":"https://example.com/a","text":"AI领域最新进展"}]}
		]
	}`
	cr, err := rc.ParseRequest([]byte(body))
	if err != nil {
		t.Fatalf("ParseRequest error: %v", err)
	}

	oc := &OpenAIConverter{}
	chatBody, err := oc.BuildUpstreamRequest(cr)
	if err != nil {
		t.Fatalf("BuildUpstreamRequest error: %v", err)
	}
	var obj map[string]any
	if err := json.Unmarshal(chatBody, &obj); err != nil {
		t.Fatalf("chat body is invalid JSON: %v\nbody=%s", err, string(chatBody))
	}

	msgs := obj["messages"].([]any)
	var toolMsg map[string]any
	for _, m := range msgs {
		mm := m.(map[string]any)
		if mm["role"] == "tool" {
			toolMsg = mm
			break
		}
	}
	if toolMsg == nil {
		t.Fatal("no tool message found for web_search_result")
	}
	content, ok := toolMsg["content"].(string)
	if !ok || content == "" {
		t.Fatalf("tool content missing/empty: %+v", toolMsg)
	}
	if !strings.Contains(content, "AI新闻") || !strings.Contains(content, "https://example.com/a") || !strings.Contains(content, "AI领域最新进展") {
		t.Errorf("tool content = %q, want title/url/text all present", content)
	}
}

// TestResponsesToChat_WebSearchCallAlone 验证单独的 web_search_call（无前置 function_call）
// 被拆成 assistant(tool_calls) + tool(result) 两条消息，满足下游 Chat 的配对约束。
func TestResponsesToChat_WebSearchCallAlone(t *testing.T) {
	rc := &ResponsesConverter{}
	body := `{
		"model": "gpt-4o",
		"input": [
			{"role":"user","content":[{"type":"input_text","text":"搜索AI新闻"}]},
			{"type":"web_search_call","call_id":"ws_1","action":{"type":"search","query":"AI news"},"status":"completed","output":"AI 新闻结果"}
		]
	}`
	cr, err := rc.ParseRequest([]byte(body))
	if err != nil {
		t.Fatalf("ParseRequest error: %v", err)
	}

	oc := &OpenAIConverter{}
	chatBody, err := oc.BuildUpstreamRequest(cr)
	if err != nil {
		t.Fatalf("BuildUpstreamRequest error: %v", err)
	}
	var obj map[string]any
	if err := json.Unmarshal(chatBody, &obj); err != nil {
		t.Fatalf("chat body is invalid JSON: %v\nbody=%s", err, string(chatBody))
	}

	msgs := obj["messages"].([]any)
	// 期望：user, assistant(tool_calls), tool
	if len(msgs) != 3 {
		t.Fatalf("messages = %d, want 3 (user, assistant, tool)", len(msgs))
	}
	assistantMsg := msgs[1].(map[string]any)
	if assistantMsg["role"] != "assistant" {
		t.Errorf("msgs[1].role = %v, want assistant", assistantMsg["role"])
	}
	tcs, ok := assistantMsg["tool_calls"].([]any)
	if !ok || len(tcs) != 1 {
		t.Fatalf("assistant tool_calls missing: %+v", assistantMsg)
	}
	tc0 := tcs[0].(map[string]any)
	if tc0["id"] != "ws_1" {
		t.Errorf("tool_call id = %v, want ws_1", tc0["id"])
	}

	toolMsg := msgs[2].(map[string]any)
	if toolMsg["role"] != "tool" {
		t.Errorf("msgs[2].role = %v, want tool", toolMsg["role"])
	}
	if toolMsg["tool_call_id"] != "ws_1" {
		t.Errorf("tool_call_id = %v, want ws_1", toolMsg["tool_call_id"])
	}
}

// TestInjectReasoningContent 验证 reasoning_content 被注入到携带 tool_calls 的 assistant
// 消息开头（作为 thinking 块）。
func TestInjectReasoningContent(t *testing.T) {
	cr := &public.CanonicalRequest{
		Messages: []public.CanonicalMessage{
			{
				Role: "assistant",
				Content: []public.CanonicalContent{
					{Type: "tool_use", ID: "fc_1", Name: "web_search"},
				},
				ToolCalls: []public.CanonicalToolCall{{ID: "fc_1", Name: "web_search", Arguments: json.RawMessage(`{}`)}},
			},
		},
	}
	InjectReasoningContent(cr, map[string]string{"fc_1": "我先搜索一下"})

	if len(cr.Messages) != 1 {
		t.Fatalf("messages = %d, want 1", len(cr.Messages))
	}
	content := cr.Messages[0].Content
	if len(content) == 0 || content[0].Type != "thinking" {
		t.Fatalf("first content block = %+v, want thinking", content)
	}
	if content[0].Text != "我先搜索一下" {
		t.Errorf("thinking text = %q, want 我先搜索一下", content[0].Text)
	}
}

// TestResponsesToChat_ChatStyleToolCalls 验证 Chat 风格的 assistant tool_calls 与
// role=tool + tool_call_id 消息在 Responses input 内被正确转换（修复
// "missing field 'tool_call_id'" 与 "role 'tool' must be a response..." 报错）。
func TestResponsesToChat_ChatStyleToolCalls(t *testing.T) {
	rc := &ResponsesConverter{}
	body := `{
		"model": "deepseek-v4-pro",
		"input": [
			{"role":"user","content":[{"type":"input_text","text":"查天气"}]},
			{"role":"assistant","content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"get_weather","arguments":"{\"city\":\"北京\"}"}}]},
			{"role":"tool","tool_call_id":"call_1","content":"晴，25°C"}
		]
	}`
	cr, err := rc.ParseRequest([]byte(body))
	if err != nil {
		t.Fatalf("ParseRequest error: %v", err)
	}

	oc := &OpenAIConverter{}
	chatBody, err := oc.BuildUpstreamRequest(cr)
	if err != nil {
		t.Fatalf("BuildUpstreamRequest error: %v", err)
	}
	var obj map[string]any
	if err := json.Unmarshal(chatBody, &obj); err != nil {
		t.Fatalf("chat body is invalid JSON: %v\nbody=%s", err, string(chatBody))
	}

	msgs := obj["messages"].([]any)
	if len(msgs) != 3 {
		t.Fatalf("messages = %d, want 3 (user, assistant, tool)", len(msgs))
	}
	assistantMsg := msgs[1].(map[string]any)
	if assistantMsg["role"] != "assistant" {
		t.Errorf("msgs[1].role = %v, want assistant", assistantMsg["role"])
	}
	tcs, ok := assistantMsg["tool_calls"].([]any)
	if !ok || len(tcs) != 1 {
		t.Fatalf("assistant tool_calls missing: %+v", assistantMsg)
	}

	toolMsg := msgs[2].(map[string]any)
	if toolMsg["role"] != "tool" {
		t.Errorf("msgs[2].role = %v, want tool", toolMsg["role"])
	}
	if toolMsg["tool_call_id"] != "call_1" {
		t.Errorf("tool_call_id = %v, want call_1", toolMsg["tool_call_id"])
	}
	if toolMsg["content"] != "晴，25°C" {
		t.Errorf("tool content = %v, want 晴，25°C", toolMsg["content"])
	}
}

// TestResponsesConverter_CustomToolCallInputField 验证非流式响应中，
// custom_tool_call item 使用 input 字段（而非 arguments）传递工具参数。
// Codex 期望 custom_tool_call 的参数在 input 字段中，使用 arguments 会导致
// Codex 无法解析工具调用结果，从而在一轮后退出。
func TestResponsesConverter_CustomToolCallInputField(t *testing.T) {
	c := &ResponsesConverter{}

	patchContent := "*** Begin Patch\n*** End Patch"
	// 模拟 FREEFORM 工具调用：input 是 {"input":"<patch>"} 形式的 JSON
	// 使用 json.Marshal 确保生成合法 JSON（换行符等特殊字符正确转义）
	argsObj := map[string]string{"input": patchContent}
	argsBytes, _ := json.Marshal(argsObj)
	cr := &public.CanonicalResponse{
		ID: "resp_1",
		Content: []public.CanonicalContent{
			{
				Type:     "tool_use",
				ID:       "fc_1",
				Name:     "apply_patch",
				Input:    json.RawMessage(argsBytes),
				ToolType: "custom",
			},
		},
	}

	built, err := c.BuildResponse(cr)
	if err != nil {
		t.Fatalf("BuildResponse error: %v", err)
	}

	var obj struct {
		Output []struct {
			Type      string `json:"type"`
			Name      string `json:"name"`
			Input     string `json:"input"`
			Arguments string `json:"arguments"`
		} `json:"output"`
	}
	if err := json.Unmarshal(built, &obj); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(obj.Output) != 1 {
		t.Fatalf("output length = %d, want 1", len(obj.Output))
	}
	out := obj.Output[0]
	if out.Type != "custom_tool_call" {
		t.Errorf("output[0].type = %q, want custom_tool_call", out.Type)
	}
	if out.Name != "apply_patch" {
		t.Errorf("output[0].name = %q, want apply_patch", out.Name)
	}
	// custom_tool_call 必须用 input 字段，且内容是提取后的原始 patch
	if out.Input != patchContent {
		t.Errorf("output[0].input = %q, want %q", out.Input, patchContent)
	}
	// custom_tool_call 不应有 arguments 字段
	if out.Arguments != "" {
		t.Errorf("output[0].arguments = %q, want empty (custom_tool_call uses input)", out.Arguments)
	}
}

// TestResponsesConverter_CustomToolCallStreamInputField 验证流式响应中，
// custom_tool_call item 使用 input 字段和 response.custom_tool_call_input.delta 事件类型。
// Codex 期望 custom_tool_call 的流式增量使用 response.custom_tool_call_input.delta，
// 使用 response.function_call_arguments.delta 会导致 Codex 无法接收工具调用增量。
func TestResponsesConverter_CustomToolCallStreamInputField(t *testing.T) {
	w := (&ResponsesConverter{}).NewUserStreamWriter()
	if setter, ok := w.(interface{ SetModel(string) }); ok {
		setter.SetModel("gpt-4o")
	}

	patchContent := "*** Begin Patch\n*** End Patch"
	// 构造 FREEFORM 工具调用的 arguments：{"input":"<patch>"}
	argsObj := map[string]string{"input": patchContent}
	argsBytes, _ := json.Marshal(argsObj)
	fullArgs := string(argsBytes)

	var all []map[string]any
	write := func(ev *public.CanonicalStreamEvent) {
		t.Helper()
		evs, err := w.Write(ev)
		if err != nil {
			t.Fatalf("Write error: %v", err)
		}
		for _, e := range evs {
			var m map[string]any
			if err := json.Unmarshal(e, &m); err != nil {
				t.Fatalf("unmarshal event %s: %v", string(e), err)
			}
			all = append(all, m)
		}
	}

	// 工具调用开始（首帧带 id/name）
	write(&public.CanonicalStreamEvent{
		Type: public.StreamEventToolCallDelta,
		ToolCall: &public.CanonicalToolCall{
			ID:        "call_1",
			Name:      "apply_patch",
			Arguments: json.RawMessage(`""`),
		},
	})
	// 工具调用参数（完整 arguments）
	write(&public.CanonicalStreamEvent{
		Type: public.StreamEventToolCallDelta,
		ToolCall: &public.CanonicalToolCall{
			ID:        "call_1",
			Name:      "apply_patch",
			Arguments: json.RawMessage(marshalString(fullArgs)),
		},
	})
	// 结束
	write(&public.CanonicalStreamEvent{
		Type:         public.StreamEventDone,
		FinishReason: "tool_calls",
	})

	// 1. 验证 output_item.added 使用 custom_tool_call 类型
	var addedItem map[string]any
	for _, m := range all {
		if m["type"] == "response.output_item.added" {
			addedItem = m["item"].(map[string]any)
			break
		}
	}
	if addedItem == nil {
		t.Fatal("no output_item.added emitted")
	}
	if addedItem["type"] != "custom_tool_call" {
		t.Errorf("added item type = %v, want custom_tool_call", addedItem["type"])
	}
	// added 时 input 应为空字符串
	if addedItem["input"] != "" {
		t.Errorf("added item input = %v, want empty string", addedItem["input"])
	}
	// added 时不应有 arguments 字段
	if _, hasArgs := addedItem["arguments"]; hasArgs {
		t.Errorf("added item has arguments field, want only input for custom_tool_call")
	}

	// 2. 验证 delta 事件使用 response.custom_tool_call_input.delta（不是 function_call_arguments.delta）
	var customDeltaFound bool
	var funcDeltaFound bool
	for _, m := range all {
		switch m["type"] {
		case "response.custom_tool_call_input.delta":
			customDeltaFound = true
			if m["delta"] != patchContent {
				t.Errorf("custom_tool_call_input.delta = %v, want %q", m["delta"], patchContent)
			}
		case "response.function_call_arguments.delta":
			funcDeltaFound = true
		}
	}
	if !customDeltaFound {
		t.Error("no response.custom_tool_call_input.delta emitted")
	}
	if funcDeltaFound {
		t.Error("response.function_call_arguments.delta emitted for custom_tool_call, should use custom_tool_call_input.delta")
	}

	// 3. 验证 output_item.done 的 item 使用 input 字段（不是 arguments）
	var doneItem map[string]any
	for _, m := range all {
		if m["type"] == "response.output_item.done" {
			doneItem = m["item"].(map[string]any)
			break
		}
	}
	if doneItem == nil {
		t.Fatal("no output_item.done emitted")
	}
	if doneItem["type"] != "custom_tool_call" {
		t.Errorf("done item type = %v, want custom_tool_call", doneItem["type"])
	}
	if doneItem["input"] != patchContent {
		t.Errorf("done item input = %v, want %q", doneItem["input"], patchContent)
	}
	if _, hasArgs := doneItem["arguments"]; hasArgs {
		t.Errorf("done item has arguments field, want only input for custom_tool_call")
	}

	// 4. 验证 response.completed 的 output 中 custom_tool_call 使用 input 字段
	var completed map[string]any
	for _, m := range all {
		if m["type"] == "response.completed" {
			completed = m
			break
		}
	}
	if completed == nil {
		t.Fatal("no response.completed emitted")
	}
	resp := completed["response"].(map[string]any)
	out, _ := resp["output"].([]any)
	if len(out) != 1 {
		t.Fatalf("completed output length = %d, want 1", len(out))
	}
	finalItem := out[0].(map[string]any)
	if finalItem["type"] != "custom_tool_call" {
		t.Errorf("completed output[0].type = %v, want custom_tool_call", finalItem["type"])
	}
	if finalItem["input"] != patchContent {
		t.Errorf("completed output[0].input = %v, want %q", finalItem["input"], patchContent)
	}
	if _, hasArgs := finalItem["arguments"]; hasArgs {
		t.Errorf("completed output[0] has arguments field, want only input for custom_tool_call")
	}
}

// TestResponsesToChat_CustomToolCallOutput 验证 custom_tool_call_output item 被正确
// 转换为 tool 结果消息，而不是落入 default 分支产生 {"role":"user","content":null}。
// 复现 server.log 中观察到的 400 错误场景：Codex 发送 custom_tool_call +
// custom_tool_call_output，后者若无专门 case 会因无 role/content 被误转成空 user 消息。
func TestResponsesToChat_CustomToolCallOutput(t *testing.T) {
	rc := &ResponsesConverter{}
	body := `{
		"model": "GLM-5.2",
		"input": [
			{"role":"user","content":[{"type":"input_text","text":"应用补丁"}]},
			{"type":"custom_tool_call","id":"fc_1","call_id":"chatcmpl-tool-1","name":"apply_patch","input":"*** Begin Patch\n*** End Patch\n","status":"completed"},
			{"type":"custom_tool_call_output","id":"ctco_1","call_id":"chatcmpl-tool-1","output":"apply_patch verification failed: invalid hunk"}
		]
	}`
	cr, err := rc.ParseRequest([]byte(body))
	if err != nil {
		t.Fatalf("ParseRequest error: %v", err)
	}

	// canonical 层：不应有空 content 的 user 消息。
	for i, m := range cr.Messages {
		if m.Role == "user" && len(m.Content) == 0 {
			t.Errorf("messages[%d] is empty-content user message (content:null bug)", i)
		}
	}

	oc := &OpenAIConverter{}
	chatBody, err := oc.BuildUpstreamRequest(cr)
	if err != nil {
		t.Fatalf("BuildUpstreamRequest error: %v", err)
	}
	var obj map[string]any
	if err := json.Unmarshal(chatBody, &obj); err != nil {
		t.Fatalf("chat body is invalid JSON: %v\nbody=%s", err, string(chatBody))
	}

	msgs := obj["messages"].([]any)
	// 不应出现 content:null 的消息。
	for i, m := range msgs {
		mm := m.(map[string]any)
		if mm["role"] == "user" {
			if c, ok := mm["content"]; !ok || c == nil {
				t.Errorf("messages[%d] user content is null/missing: %+v", i, mm)
			}
		}
	}
	// 应有一条 tool 消息携带 custom_tool_call_output 的结果。
	var toolMsg map[string]any
	for _, m := range msgs {
		mm := m.(map[string]any)
		if mm["role"] == "tool" {
			toolMsg = mm
			break
		}
	}
	if toolMsg == nil {
		t.Fatal("no tool message found for custom_tool_call_output")
	}
	content, _ := toolMsg["content"].(string)
	if !strings.Contains(content, "apply_patch verification failed") {
		t.Errorf("tool content = %q, want custom_tool_call_output text", content)
	}
	if toolMsg["tool_call_id"] != "chatcmpl-tool-1" {
		t.Errorf("tool_call_id = %v, want chatcmpl-tool-1", toolMsg["tool_call_id"])
	}
}

// TestResponsesToChat_EmptyContentUserFiltered 验证 default 分支的防护：
// 空 content 的 user 消息 item 应被过滤，不进入 messages。
func TestResponsesToChat_EmptyContentUserFiltered(t *testing.T) {
	rc := &ResponsesConverter{}
	body := `{
		"model": "GLM-5.2",
		"input": [
			{"role":"user","content":[{"type":"input_text","text":"你好"}]},
			{"role":"user","content":null}
		]
	}`
	cr, err := rc.ParseRequest([]byte(body))
	if err != nil {
		t.Fatalf("ParseRequest error: %v", err)
	}
	for i, m := range cr.Messages {
		if m.Role == "user" && len(m.Content) == 0 {
			t.Errorf("messages[%d] is empty-content user message, should be filtered", i)
		}
	}
}

// TestResponsesToChat_ToolMessageMissingToolCallID 验证当 tool 消息缺少
// tool_call_id 时（如 Codex CLI 对不支持的调用生成的 "unsupported call" 结果），
// BuildUpstreamRequest 能从前面的 assistant 消息中找到未匹配的 tool_call.id
// 来关联，避免 vLLM 因 tool 消息缺少 tool_call_id 而返回 400。
func TestResponsesToChat_ToolMessageMissingToolCallID(t *testing.T) {
	rc := &ResponsesConverter{}
	// 模拟 Codex 发送的内嵌 Chat 风格 tool 消息，缺少 tool_call_id。
	body := `{
		"model": "GLM-5.2",
		"input": [
			{"role":"user","content":[{"type":"input_text","text":"查看图片"}]},
			{"role":"assistant","tool_calls":[{"id":"fc_abc","type":"function","function":{"name":"view_image","arguments":"{\"path\":\"test.png\"}"}}]},
			{"role":"tool","content":"unsupported call: "}
		]
	}`
	cr, err := rc.ParseRequest([]byte(body))
	if err != nil {
		t.Fatalf("ParseRequest error: %v", err)
	}

	// canonical 层：tool 消息应存在，ToolCallID 为空。
	var toolMsg *public.CanonicalMessage
	for i := range cr.Messages {
		if cr.Messages[i].Role == "tool" {
			toolMsg = &cr.Messages[i]
			break
		}
	}
	if toolMsg == nil {
		t.Fatal("no tool message found in canonical")
	}
	if toolMsg.ToolCallID != "" {
		t.Errorf("ToolCallID = %q, want empty (missing in source)", toolMsg.ToolCallID)
	}

	oc := &OpenAIConverter{}
	chatBody, err := oc.BuildUpstreamRequest(cr)
	if err != nil {
		t.Fatalf("BuildUpstreamRequest error: %v", err)
	}
	var obj map[string]any
	if err := json.Unmarshal(chatBody, &obj); err != nil {
		t.Fatalf("chat body is invalid JSON: %v\nbody=%s", err, string(chatBody))
	}

	msgs := obj["messages"].([]any)
	// 找到 tool 消息，验证 tool_call_id 已被兜底填充。
	var chatToolMsg map[string]any
	for _, m := range msgs {
		mm := m.(map[string]any)
		if mm["role"] == "tool" {
			chatToolMsg = mm
			break
		}
	}
	if chatToolMsg == nil {
		t.Fatal("no tool message in chat request")
	}
	tcid, ok := chatToolMsg["tool_call_id"]
	if !ok || tcid == "" {
		t.Errorf("tool_call_id missing or empty in chat request: %+v", chatToolMsg)
	}
	if tcid != "fc_abc" {
		t.Errorf("tool_call_id = %v, want fc_abc (from preceding assistant)", tcid)
	}
}

// TestResponsesConverter_NonFunctionToolsUpstream 验证非 function 工具
// （web_search/tool_search）在 Responses 上游请求中保留原始 JSON 透传，
// 而不是被强制转成 function 类型（这是 Codex WebSearch 续接失败的关键修复）。
func TestResponsesConverter_NonFunctionToolsUpstream(t *testing.T) {
	c := &ResponsesConverter{}
	body := `{
		"model": "GLM-5.3-Flash",
		"input": [{"role":"user","content":"hi"}],
		"tools": [
			{"type":"function","name":"exec_command","description":"run","parameters":{"type":"object"}},
			{"type":"web_search","search_context_size":"high"},
			{"type":"tool_search","max_results":5}
		]
	}`
	cr, err := c.ParseRequest([]byte(body))
	if err != nil {
		t.Fatalf("ParseRequest error: %v", err)
	}

	out, err := c.BuildUpstreamRequest(cr)
	if err != nil {
		t.Fatalf("BuildUpstreamRequest error: %v", err)
	}

	var obj struct {
		Tools []json.RawMessage `json:"tools"`
	}
	if err := json.Unmarshal(out, &obj); err != nil {
		t.Fatalf("invalid JSON: %v\nbody=%s", err, string(out))
	}
	if len(obj.Tools) != 3 {
		t.Fatalf("len(tools) = %d, want 3", len(obj.Tools))
	}

	// function 工具：正常 function 结构
	var fn struct {
		Type string `json:"type"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(obj.Tools[0], &fn); err != nil {
		t.Fatal(err)
	}
	if fn.Type != "function" || fn.Name != "exec_command" {
		t.Errorf("tools[0] = %s, want function exec_command", string(obj.Tools[0]))
	}

	// web_search：保留原始 JSON（type=web_search + search_context_size）
	var ws struct {
		Type              string `json:"type"`
		SearchContextSize string `json:"search_context_size"`
	}
	if err := json.Unmarshal(obj.Tools[1], &ws); err != nil {
		t.Fatal(err)
	}
	if ws.Type != "web_search" {
		t.Errorf("tools[1] type = %q, want web_search (was forced to function)", ws.Type)
	}
	if ws.SearchContextSize != "high" {
		t.Errorf("tools[1] search_context_size = %q, want high (raw JSON lost)", ws.SearchContextSize)
	}

	// tool_search：保留原始 JSON（type=tool_search + max_results）
	var ts struct {
		Type       string `json:"type"`
		MaxResults int    `json:"max_results"`
	}
	if err := json.Unmarshal(obj.Tools[2], &ts); err != nil {
		t.Fatal(err)
	}
	if ts.Type != "tool_search" {
		t.Errorf("tools[2] type = %q, want tool_search (was forced to function)", ts.Type)
	}
	if ts.MaxResults != 5 {
		t.Errorf("tools[2] max_results = %d, want 5 (raw JSON lost)", ts.MaxResults)
	}
}
