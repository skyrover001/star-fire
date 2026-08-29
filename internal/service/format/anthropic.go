package format

import (
	"encoding/json"
	"errors"

	"star-fire/pkg/public"
)

// AnthropicConverter 实现 Anthropic Messages API ↔ Canonical 的双向转换。
//
// 详见 docs/multi-format-api-design.md 3.2 节。
type AnthropicConverter struct{}

func init() {
	Register(public.FormatAnthropic, &AnthropicConverter{})
}

// ---- Anthropic 请求/响应局部结构（用 json.RawMessage 保底无损）----

type anthropicRequest struct {
	Model         string             `json:"model"`
	MaxTokens     int                `json:"max_tokens"`
	System        json.RawMessage    `json:"system,omitempty"` // string 或 block 数组
	Messages      []anthropicMessage `json:"messages"`
	Tools         []anthropicTool    `json:"tools,omitempty"`
	Temperature   *float64           `json:"temperature,omitempty"`
	TopP          *float64           `json:"top_p,omitempty"`
	Thinking      json.RawMessage    `json:"thinking,omitempty"`
	StopSequences []string           `json:"stop_sequences,omitempty"`
	Metadata      json.RawMessage    `json:"metadata,omitempty"`
	Stream        bool               `json:"stream,omitempty"`
}

type anthropicMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"` // string 或 block 数组
}

type anthropicTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema,omitempty"`
}

type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	// image
	Source json.RawMessage `json:"source,omitempty"`
	// tool_use
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
	// tool_result
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
	// thinking
	Thinking  string `json:"thinking,omitempty"`
	Signature string `json:"signature,omitempty"`
}

type anthropicResponse struct {
	ID         string                  `json:"id"`
	Type       string                  `json:"type"`
	Role       string                  `json:"role"`
	Content    []anthropicContentBlock `json:"content"`
	StopReason string                  `json:"stop_reason"`
	Usage      anthropicUsage          `json:"usage"`
}

type anthropicUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
}

// ---- 默认 max_tokens ----

const defaultAnthropicMaxTokens = 4096

// ---- 请求解析 ----

func (c *AnthropicConverter) ParseRequest(body []byte) (*public.CanonicalRequest, error) {
	var req anthropicRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}
	if req.Model == "" {
		return nil, errors.New("anthropic: missing required field 'model'")
	}
	if len(req.Messages) == 0 {
		return nil, errors.New("anthropic: missing required field 'messages'")
	}

	cr := &public.CanonicalRequest{
		Model:       req.Model,
		Stream:      req.Stream,
		Temperature: req.Temperature,
		TopP:        req.TopP,
	}
	if req.MaxTokens > 0 {
		v := req.MaxTokens
		cr.MaxTokens = &v
	}
	if len(req.System) > 0 {
		cr.System = parseAnthropicSystem(req.System)
	}
	if len(req.Thinking) > 0 {
		cr.Thinking = req.Thinking
	}

	for _, t := range req.Tools {
		ct := public.CanonicalTool{Type: "function", Name: t.Name, Description: t.Description}
		if len(t.InputSchema) > 0 {
			ct.Parameters = t.InputSchema
		}
		cr.Tools = append(cr.Tools, ct)
	}

	for _, m := range req.Messages {
		blocks := parseAnthropicContent(m.Content)

		// Anthropic 的 tool_result 出现在 role=user 消息中。含 tool_result 的
		// 消息转成 role=tool 消息（与 OpenAI/Responses 语义对齐）。
		if hasToolResult(blocks) {
			for _, b := range blocks {
				if b.Type != "tool_result" {
					continue
				}
				cr.Messages = append(cr.Messages, public.CanonicalMessage{
					Role:       "tool",
					ToolCallID: b.ToolUseID,
					Content: []public.CanonicalContent{{
						Type:    "tool_result",
						ID:      b.ToolUseID,
						Content: anthropicToolResultToCanonical(b.Content),
						IsError: b.IsError,
					}},
				})
			}
			continue
		}

		cm := public.CanonicalMessage{Role: m.Role}
		for _, b := range blocks {
			if b.Type == "tool_use" {
				cm.Content = append(cm.Content, public.CanonicalContent{
					Type:  "tool_use",
					ID:    b.ID,
					Name:  b.Name,
					Input: compactJSON(b.Input),
				})
				cm.ToolCalls = append(cm.ToolCalls, public.CanonicalToolCall{
					ID:        b.ID,
					Name:      b.Name,
					Arguments: marshalString(rawToArgsString(b.Input)),
				})
				continue
			}
			cm.Content = append(cm.Content, anthropicBlockToCanonical(b))
		}
		cr.Messages = append(cr.Messages, cm)
	}

	return cr, nil
}

// BuildUpstreamRequest 将 Canonical 请求 → Anthropic MessageCreateParams。
func (c *AnthropicConverter) BuildUpstreamRequest(cr *public.CanonicalRequest) ([]byte, error) {
	req := anthropicRequest{
		Model:       cr.Model,
		MaxTokens:   defaultAnthropicMaxTokens,
		Stream:      cr.Stream,
		Temperature: cr.Temperature,
		TopP:        cr.TopP,
	}
	if cr.MaxTokens != nil {
		req.MaxTokens = *cr.MaxTokens
	}
	if len(cr.Thinking) > 0 {
		req.Thinking = cr.Thinking
	}

	// system：单文本块用 string 形式，否则用 block 数组。
	if len(cr.System) > 0 {
		if len(cr.System) == 1 && cr.System[0].Type == "text" && len(cr.System[0].Extra) == 0 {
			req.System = marshalString(cr.System[0].Text)
		} else {
			req.System = marshalAnthropicBlocks(canonicalToAnthropicBlocks(cr.System))
		}
	}

	for _, t := range cr.Tools {
		req.Tools = append(req.Tools, anthropicTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.Parameters,
		})
	}

	for _, cm := range cr.Messages {
		if cm.Role == "tool" {
			blocks := []anthropicContentBlock{}
			for _, b := range cm.Content {
				if b.Type != "tool_result" {
					continue
				}
				blocks = append(blocks, anthropicContentBlock{
					Type:      "tool_result",
					ToolUseID: firstNonEmpty(b.ID, cm.ToolCallID),
					Content:   canonicalToolResultToAnthropic(b.Content),
					IsError:   b.IsError,
				})
			}
			if len(blocks) == 0 {
				blocks = append(blocks, anthropicContentBlock{
					Type:      "tool_result",
					ToolUseID: cm.ToolCallID,
					Content:   marshalString(blocksToText(cm.Content)),
				})
			}
			req.Messages = append(req.Messages, anthropicMessage{
				Role:    "user",
				Content: marshalAnthropicBlocks(blocks),
			})
			continue
		}

		blocks := canonicalToAnthropicBlocks(cm.Content)
		if len(blocks) == 0 {
			// Content 为空但有 ToolCalls 时，从 ToolCalls 重建 tool_use 块。
			for _, tc := range cm.ToolCalls {
				blocks = append(blocks, anthropicContentBlock{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Name,
					Input: parseArgsToInput(rawToString(tc.Arguments)),
				})
			}
		}
		// Anthropic 只接受 user/assistant 角色；developer/system 归一为 user
		// （system 已提取到顶层 System 字段，这里兜底处理残留的 developer）。
		role := cm.Role
		if role == "developer" || role == "system" {
			role = "user"
		}
		req.Messages = append(req.Messages, anthropicMessage{
			Role:    role,
			Content: marshalAnthropicBlocks(blocks),
		})
	}

	return json.Marshal(req)
}

// ParseUpstreamResponse 解析 Anthropic Message → Canonical。
func (c *AnthropicConverter) ParseUpstreamResponse(data []byte) (*public.CanonicalResponse, error) {
	var resp anthropicResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	cr := &public.CanonicalResponse{
		ID:           resp.ID,
		FinishReason: anthropicStopReasonToCanonical(resp.StopReason),
	}
	for _, b := range resp.Content {
		if b.Type == "tool_use" {
			cr.Content = append(cr.Content, public.CanonicalContent{
				Type:  "tool_use",
				ID:    b.ID,
				Name:  b.Name,
				Input: compactJSON(b.Input),
			})
			continue
		}
		cr.Content = append(cr.Content, anthropicBlockToCanonical(b))
	}
	cr.Usage = anthropicUsageToCanonical(resp.Usage)
	return cr, nil
}

// BuildResponse 将 Canonical 响应 → Anthropic Message。
func (c *AnthropicConverter) BuildResponse(cr *public.CanonicalResponse) ([]byte, error) {
	resp := anthropicResponse{
		ID:         cr.ID,
		Type:       "message",
		Role:       "assistant",
		Content:    canonicalToAnthropicBlocks(cr.Content),
		StopReason: canonicalFinishToAnthropicStopReason(cr.FinishReason),
		Usage:      canonicalToAnthropicUsage(cr.Usage),
	}
	return json.Marshal(resp)
}

// ---- 流式 ----

func (c *AnthropicConverter) NewStreamAccumulator() StreamAccumulator {
	return &anthropicAccumulator{}
}

// ParseUpstreamStreamEvent 是单帧便捷方法（无状态），完整流式解析请使用
// NewStreamAccumulator。
func (c *AnthropicConverter) ParseUpstreamStreamEvent(line []byte) (*public.CanonicalStreamEvent, error) {
	acc := c.NewStreamAccumulator()
	evs, err := acc.Feed(line)
	if err != nil {
		return nil, err
	}
	if len(evs) > 0 {
		return evs[0], nil
	}
	return acc.Flush()
}

// BuildUserStreamEvent 将 Canonical 流式事件 → Anthropic 流式事件 JSON。
func (c *AnthropicConverter) BuildUserStreamEvent(ev *public.CanonicalStreamEvent) ([]byte, error) {
	switch ev.Type {
	case public.StreamEventTextDelta:
		return json.Marshal(map[string]any{
			"type":  "content_block_delta",
			"index": 0,
			"delta": map[string]any{"type": "text_delta", "text": ev.Text},
		})
	case public.StreamEventToolCallDelta:
		if ev.ToolCall == nil {
			return json.Marshal(map[string]any{"type": "content_block_stop", "index": 0})
		}
		return json.Marshal(map[string]any{
			"type":  "content_block_delta",
			"index": 0,
			"delta": map[string]any{"type": "input_json_delta", "partial_json": rawToString(ev.ToolCall.Arguments)},
		})
	case public.StreamEventDone:
		return json.Marshal(map[string]any{
			"type": "message_delta",
			"delta": map[string]any{
				"stop_reason": canonicalFinishToAnthropicStopReason(ev.FinishReason),
			},
		})
	default:
		return json.Marshal(map[string]any{"type": "error"})
	}
}

// anthropicAccumulator 是有状态的上游 SSE 解析器。
type anthropicAccumulator struct {
	inputTokens  int
	outputTokens int
	cachedTokens int
	stopReason   string
	done         bool
}

func (a *anthropicAccumulator) Feed(data []byte) ([]*public.CanonicalStreamEvent, error) {
	var ev struct {
		Type         string                 `json:"type"`
		Delta        json.RawMessage        `json:"delta"`
		Usage        *anthropicUsage        `json:"usage"`
		ContentBlock *anthropicContentBlock `json:"content_block"`
		Message      *struct {
			Usage *anthropicUsage `json:"usage"`
		} `json:"message"`
		Error *struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(data, &ev); err != nil {
		return nil, err
	}

	switch ev.Type {
	case "message_start":
		// message_start 的 usage 嵌套在 message.usage 内。
		if ev.Message != nil && ev.Message.Usage != nil {
			a.inputTokens = ev.Message.Usage.InputTokens
			a.cachedTokens = ev.Message.Usage.CacheReadInputTokens + ev.Message.Usage.CacheCreationInputTokens
		}
		return nil, nil

	case "content_block_start":
		if ev.ContentBlock != nil && ev.ContentBlock.Type == "tool_use" {
			return []*public.CanonicalStreamEvent{{
				Type: public.StreamEventToolCallDelta,
				ToolCall: &public.CanonicalToolCall{
					ID:        ev.ContentBlock.ID,
					Name:      ev.ContentBlock.Name,
					Arguments: marshalString(""),
				},
			}}, nil
		}
		return nil, nil

	case "content_block_delta":
		var delta struct {
			Type        string `json:"type"`
			Text        string `json:"text"`
			PartialJSON string `json:"partial_json"`
		}
		if err := json.Unmarshal(ev.Delta, &delta); err != nil {
			return nil, err
		}
		switch delta.Type {
		case "text_delta":
			return []*public.CanonicalStreamEvent{{Type: public.StreamEventTextDelta, Text: delta.Text}}, nil
		case "input_json_delta":
			return []*public.CanonicalStreamEvent{{
				Type: public.StreamEventToolCallDelta,
				ToolCall: &public.CanonicalToolCall{
					Arguments: marshalString(delta.PartialJSON),
				},
			}}, nil
		}
		return nil, nil

	case "message_delta":
		var delta struct {
			StopReason string `json:"stop_reason"`
		}
		if err := json.Unmarshal(ev.Delta, &delta); err != nil {
			return nil, err
		}
		a.stopReason = delta.StopReason
		if ev.Usage != nil {
			a.outputTokens = ev.Usage.OutputTokens
		}
		return nil, nil

	case "message_stop":
		a.done = true
		u := a.buildUsage()
		return []*public.CanonicalStreamEvent{{
			Type:         public.StreamEventDone,
			FinishReason: anthropicStopReasonToCanonical(a.stopReason),
			Usage:        &u,
		}}, nil

	case "error":
		a.done = true
		msg := ""
		if ev.Error != nil {
			msg = ev.Error.Message
		}
		return []*public.CanonicalStreamEvent{{Type: public.StreamEventError, Text: msg}}, nil
	}
	return nil, nil
}

func (a *anthropicAccumulator) Flush() (*public.CanonicalStreamEvent, error) {
	if a.done {
		return nil, nil
	}
	u := a.buildUsage()
	return &public.CanonicalStreamEvent{
		Type:         public.StreamEventDone,
		FinishReason: anthropicStopReasonToCanonical(a.stopReason),
		Usage:        &u,
	}, nil
}

func (a *anthropicAccumulator) buildUsage() public.CanonicalUsage {
	return public.CanonicalUsage{
		InputTokens:  a.inputTokens,
		OutputTokens: a.outputTokens,
		TotalTokens:  a.inputTokens + a.outputTokens,
		CachedTokens: a.cachedTokens,
	}
}

// ---- 块映射辅助 ----

func hasToolResult(blocks []anthropicContentBlock) bool {
	for _, b := range blocks {
		if b.Type == "tool_result" {
			return true
		}
	}
	return false
}

// parseAnthropicSystem 解析 system 字段（string 或 block 数组）。
func parseAnthropicSystem(raw json.RawMessage) []public.CanonicalContent {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return []public.CanonicalContent{{Type: "text", Text: s}}
	}
	var blocks []anthropicContentBlock
	if json.Unmarshal(raw, &blocks) == nil {
		return anthropicBlocksToCanonical(blocks)
	}
	return nil
}

// parseAnthropicContent 解析 message.content（string 或 block 数组）。
func parseAnthropicContent(raw json.RawMessage) []anthropicContentBlock {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return []anthropicContentBlock{{Type: "text", Text: s}}
	}
	var blocks []anthropicContentBlock
	if json.Unmarshal(raw, &blocks) == nil {
		return blocks
	}
	return nil
}

func anthropicBlockToCanonical(b anthropicContentBlock) public.CanonicalContent {
	switch b.Type {
	case "text":
		return public.CanonicalContent{Type: "text", Text: b.Text}
	case "image":
		return public.CanonicalContent{Type: "image", ImageURL: sourceToImageURL(b.Source)}
	case "video":
		return public.CanonicalContent{Type: "video", VideoURL: sourceToImageURL(b.Source)}
	case "thinking":
		cc := public.CanonicalContent{Type: "thinking", Text: b.Thinking}
		if b.Signature != "" {
			cc.Extra = map[string]json.RawMessage{"signature": marshalString(b.Signature)}
		}
		return cc
	default:
		// 未知块类型保底：类型名保留 + 原文进 Extra 兜底。
		cc := public.CanonicalContent{Type: b.Type, Text: b.Text}
		return cc
	}
}

func canonicalToAnthropicBlocks(blocks []public.CanonicalContent) []anthropicContentBlock {
	out := make([]anthropicContentBlock, 0, len(blocks))
	for _, b := range blocks {
		out = append(out, canonicalToAnthropicBlock(b))
	}
	return out
}

func anthropicBlocksToCanonical(blocks []anthropicContentBlock) []public.CanonicalContent {
	out := make([]public.CanonicalContent, 0, len(blocks))
	for _, b := range blocks {
		out = append(out, anthropicBlockToCanonical(b))
	}
	return out
}

func canonicalToAnthropicBlock(b public.CanonicalContent) anthropicContentBlock {
	switch b.Type {
	case "text":
		return anthropicContentBlock{Type: "text", Text: b.Text}
	case "image":
		return anthropicContentBlock{Type: "image", Source: imageURLToSource(b.ImageURL)}
	case "video":
		return anthropicContentBlock{Type: "video", Source: imageURLToSource(b.VideoURL)}
	case "tool_use":
		return anthropicContentBlock{Type: "tool_use", ID: b.ID, Name: b.Name, Input: b.Input}
	case "thinking":
		ab := anthropicContentBlock{Type: "thinking", Thinking: b.Text}
		if b.Extra != nil {
			if sig, ok := b.Extra["signature"]; ok {
				ab.Signature = rawToString(sig)
			}
		}
		return ab
	default:
		return anthropicContentBlock{Type: b.Type, Text: b.Text}
	}
}

func anthropicToolResultToCanonical(raw json.RawMessage) []public.CanonicalContent {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return []public.CanonicalContent{{Type: "text", Text: s}}
	}
	var blocks []anthropicContentBlock
	if json.Unmarshal(raw, &blocks) == nil {
		return anthropicBlocksToCanonical(blocks)
	}
	return nil
}

func canonicalToolResultToAnthropic(blocks []public.CanonicalContent) json.RawMessage {
	if len(blocks) == 1 && blocks[0].Type == "text" && len(blocks[0].Extra) == 0 {
		return marshalString(blocks[0].Text)
	}
	return marshalAnthropicBlocks(canonicalToAnthropicBlocks(blocks))
}

func marshalAnthropicBlocks(blocks []anthropicContentBlock) json.RawMessage {
	b, _ := json.Marshal(blocks)
	return b
}

// ---- 语义映射 ----

func anthropicStopReasonToCanonical(sr string) string {
	switch sr {
	case "end_turn", "stop_sequence", "":
		return "stop"
	case "max_tokens":
		return "length"
	case "tool_use":
		return "tool_calls"
	case "refusal":
		return "content_filter"
	default:
		return sr
	}
}

func canonicalFinishToAnthropicStopReason(fr string) string {
	switch fr {
	case "stop", "":
		return "end_turn"
	case "length":
		return "max_tokens"
	case "tool_calls":
		return "tool_use"
	case "content_filter":
		return "refusal"
	default:
		return fr
	}
}

func anthropicUsageToCanonical(u anthropicUsage) public.CanonicalUsage {
	return public.CanonicalUsage{
		InputTokens:  u.InputTokens,
		OutputTokens: u.OutputTokens,
		TotalTokens:  u.InputTokens + u.OutputTokens,
		CachedTokens: u.CacheReadInputTokens + u.CacheCreationInputTokens,
	}
}

func canonicalToAnthropicUsage(cu public.CanonicalUsage) anthropicUsage {
	return anthropicUsage{
		InputTokens:          cu.InputTokens,
		OutputTokens:         cu.OutputTokens,
		CacheReadInputTokens: cu.CachedTokens,
	}
}

// firstNonEmpty 返回第一个非空字符串。
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
