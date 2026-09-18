package format

import (
	"encoding/json"
	"errors"
	"strings"

	"star-fire/pkg/public"
)

// AnthropicConverter 实现 Anthropic Messages API ↔ Canonical 的双向转换。
//
// 详见 docs/multi-format-api-design.md 3.2 节。
type AnthropicConverter struct{}

func init() {
	Register(public.FormatAnthropic, &AnthropicConverter{})
}

// IsAnthropic 判断 converter 是否为 Anthropic Messages 格式。
// 适配层据此决定错误体格式与流式结尾标志（Anthropic 用 message_stop，而非 [DONE]）。
func IsAnthropic(c Converter) bool {
	_, ok := c.(*AnthropicConverter)
	return ok
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

// anthropicExtraKeys 是需要透传到上游的非标准顶层字段。
// 典型场景：vLLM 后端通过 chat_template_kwargs 控制推理模板行为
// （如 {"thinking":false} 关闭思考模式）。Anthropic 标准没有该字段，
// 但用户在 /v1/messages 请求中携带时必须无损转发给 OpenAI 上游，
// 否则推理模型会把全部输出 token 消耗在隐藏思考阶段，提前截断时
// 返回空 content（无法恢复从未生成的文本）。
var anthropicExtraKeys = []string{"chat_template_kwargs"}

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
	if req.MaxTokens <= 0 {
		return nil, errors.New("anthropic: missing required field 'max_tokens'")
	}

	cr := &public.CanonicalRequest{
		Model:       req.Model,
		Stream:      req.Stream,
		Temperature: req.Temperature,
		TopP:        req.TopP,
	}
	{
		v := req.MaxTokens
		cr.MaxTokens = &v
	}
	if len(req.StopSequences) > 0 {
		cr.StopSequences = req.StopSequences
	}
	if len(req.System) > 0 {
		cr.System = parseAnthropicSystem(req.System)
	}
	if len(req.Thinking) > 0 {
		cr.Thinking = req.Thinking
	}
	// 非标准顶层字段（chat_template_kwargs 等）→ Extra，供下游透传。
	cr.Extra = extractAnthropicExtra(body)

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

// extractAnthropicExtra 从原始请求 JSON 中提取需要透传的非标准顶层字段。
// 只提取 anthropicExtraKeys 中列出的键，避免把 Anthropic 标准字段
// （model/messages/max_tokens 等）重复塞进 Extra。
func extractAnthropicExtra(body []byte) map[string]json.RawMessage {
	return extractRawTopLevelFields(body, anthropicExtraKeys...)
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
	if len(cr.StopSequences) > 0 {
		req.StopSequences = cr.StopSequences
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

	// Extra 透传：canonical.Extra 中的非标准字段（chat_template_kwargs 等）
	// 序列化后追加到请求 JSON 顶层，保证 Anthropic→Anthropic 往返不丢字段。
	// （Anthropic→OpenAI 方向由 OpenAIConverter.BuildUpstreamRequest 处理。）
	extraKeys := sortedExtraKeys(cr.Extra)

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

	// Extra 透传：注入到序列化后的请求 JSON 顶层。
	if len(extraKeys) > 0 {
		raw, err := json.Marshal(req)
		if err != nil {
			return nil, err
		}
		return injectExtraFields(raw, cr.Extra, extraKeys), nil
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
		ID:         anthropicMessageID(cr.ID),
		Type:       "message",
		Role:       "assistant",
		Content:    canonicalToAnthropicBlocks(cr.Content),
		StopReason: canonicalFinishToAnthropicStopReasonWithStops(cr.FinishReason, cr.StopSequences, hasTextContent(cr.Content)),
		Usage:      canonicalToAnthropicUsage(cr.Usage),
	}
	return json.Marshal(resp)
}

// hasTextContent 判断响应中是否有非空文本块（用于停止序列启发式）。
func hasTextContent(content []public.CanonicalContent) bool {
	for _, b := range content {
		if b.Type == "text" && b.Text != "" {
			return true
		}
	}
	return false
}

// anthropicMessageID 保证返回的 message id 以 msg_ 前缀开头（符合 Anthropic 标准）。
// 上游可能是 OpenAI 的 chatcmpl-... 或 Anthropic 的 msg_...；前者需重写为 msg_，
// 后者直接透传。
func anthropicMessageID(id string) string {
	if strings.HasPrefix(id, "msg_") {
		return id
	}
	return "msg_" + randSuffix()
}

// anthropicToolUseID 保证 tool_use id 以 toolu_ 前缀开头（符合 Anthropic 标准）。
func anthropicToolUseID(id string) string {
	if strings.HasPrefix(id, "toolu_") {
		return id
	}
	return "toolu_" + randSuffix()
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

// NewUserStreamWriter 返回状态化的 Anthropic 用户流式 writer。
//
// Anthropic Messages 流式要求输出一整套结构性事件，且严格有序：
//
//	message_start → content_block_start → content_block_delta×N →
//	content_block_stop → [更多 block] → message_delta → message_stop
//
// 而 Canonical 流式事件（text_delta / tool_call_delta / done）是扁平化的增量。
// 本 writer 在首个 delta 前合成 message_start，在 block 切换时合成
// content_block_start/stop，在 done 时合成 message_delta + message_stop。
// Anthropic 流式以 message_stop 结尾，不使用 OpenAI 的 `data: [DONE]`。
func (c *AnthropicConverter) NewUserStreamWriter() UserStreamWriter {
	return &anthropicUserWriter{}
}

// SetModel 设置 writer 的模型名（用于 message_start 的 message.model 字段）。
func (w *anthropicUserWriter) SetModel(model string) {
	w.model = model
}

// SetStopSequences 设置请求中的停止序列（用于 done 事件的 stop_reason 启发式判断）。
func (w *anthropicUserWriter) SetStopSequences(stops []string) {
	w.stopSequences = stops
}

// anthropicUserWriter 累积 Anthropic 用户流式输出的状态。
type anthropicUserWriter struct {
	started      bool
	finished     bool // message_stop 是否已发（防止异常路径重复收尾）
	model        string
	messageID    string
	usage        public.CanonicalUsage
	finishReason string

	// stopSequences/hasText 用于停止序列启发式（见
	// canonicalFinishToAnthropicStopReasonWithStops）：请求带 stop_sequences、
	// 上游 finish_reason=stop 且全程无文本 delta 时，映射为 stop_sequence。
	stopSequences []string
	hasText       bool

	blockIndex   int
	blockStarted bool
	blockType    string // "text" | "tool_use"
	toolID       string
	toolName     string
}

func (w *anthropicUserWriter) Write(ev *public.CanonicalStreamEvent) ([][]byte, error) {
	if ev.Usage != nil {
		w.usage = *ev.Usage
	}
	if ev.FinishReason != "" {
		w.finishReason = ev.FinishReason
	}

	switch ev.Type {
	case public.StreamEventTextDelta:
		if ev.Text != "" {
			w.hasText = true
		}
		if !w.started {
			return w.startAndText(ev.Text)
		}
		var out [][]byte
		// 若当前 block 不是 text（例如之前是 tool_use），先关闭并开启新的 text block。
		if closeEvt := w.ensureBlock("text", "", ""); closeEvt != nil {
			out = append(out, closeEvt...)
		}
		out = append(out, marshalStream(map[string]any{
			"type":  "content_block_delta",
			"index": w.blockIndex,
			"delta": map[string]any{"type": "text_delta", "text": ev.Text},
		}))
		return out, nil

	case public.StreamEventToolCallDelta:
		if ev.ToolCall == nil {
			return nil, nil
		}
		if !w.started {
			return w.startAndTool(ev.ToolCall)
		}
		var out [][]byte
		// 新工具调用（携带 id）→ 开启新的 tool_use block。
		if ev.ToolCall.ID != "" {
			if closeEvt := w.ensureBlock("tool_use", ev.ToolCall.ID, ev.ToolCall.Name); closeEvt != nil {
				out = append(out, closeEvt...)
			}
		}
		out = append(out, marshalStream(map[string]any{
			"type":  "content_block_delta",
			"index": w.blockIndex,
			"delta": map[string]any{"type": "input_json_delta", "partial_json": rawToString(ev.ToolCall.Arguments)},
		}))
		return out, nil

	case public.StreamEventDone:
		if w.finished {
			return nil, nil
		}
		w.finished = true
		var out [][]byte
		if w.blockStarted {
			out = append(out, marshalStream(map[string]any{
				"type":  "content_block_stop",
				"index": w.blockIndex,
			}))
			w.blockStarted = false
		}
		stopReason := canonicalFinishToAnthropicStopReasonWithStops(w.finishReason, w.stopSequences, w.hasText)
		out = append(out,
			marshalStream(map[string]any{
				"type": "message_delta",
				"delta": map[string]any{
					"stop_reason":   stopReason,
					"stop_sequence": nil,
				},
				"usage": map[string]any{"output_tokens": w.usage.OutputTokens},
			}),
			marshalStream(map[string]any{"type": "message_stop"}),
		)
		return out, nil

	case public.StreamEventError:
		return [][]byte{marshalStream(map[string]any{
			"type": "error",
			"error": map[string]any{
				"type":    "api_error",
				"message": ev.Text,
			},
		})}, nil

	default:
		return nil, nil
	}
}

func (w *anthropicUserWriter) Flush() ([][]byte, error) {
	if w.started {
		return w.Write(&public.CanonicalStreamEvent{Type: public.StreamEventDone, FinishReason: w.finishReason})
	}
	return nil, nil
}

// startAndText 在尚未发送 message_start 时，先合成 message_start + text block。
func (w *anthropicUserWriter) startAndText(text string) ([][]byte, error) {
	out := w.startMessage()
	w.blockIndex = 0
	w.blockStarted = true
	w.blockType = "text"
	out = append(out,
		marshalStream(map[string]any{
			"type":          "content_block_start",
			"index":         0,
			"content_block": map[string]any{"type": "text", "text": ""},
		}),
		marshalStream(map[string]any{
			"type":  "content_block_delta",
			"index": 0,
			"delta": map[string]any{"type": "text_delta", "text": text},
		}),
	)
	return out, nil
}

// startAndTool 在尚未发送 message_start 时，先合成 message_start + tool_use block。
func (w *anthropicUserWriter) startAndTool(tc *public.CanonicalToolCall) ([][]byte, error) {
	out := w.startMessage()
	w.blockIndex = 0
	w.blockStarted = true
	w.blockType = "tool_use"
	w.toolID = tc.ID
	w.toolName = tc.Name
	out = append(out, marshalStream(map[string]any{
		"type":  "content_block_start",
		"index": 0,
		"content_block": map[string]any{
			"type":  "tool_use",
			"id":    tc.ID,
			"name":  tc.Name,
			"input": map[string]any{},
		},
	}))
	if len(tc.Arguments) > 0 {
		out = append(out, marshalStream(map[string]any{
			"type":  "content_block_delta",
			"index": 0,
			"delta": map[string]any{"type": "input_json_delta", "partial_json": rawToString(tc.Arguments)},
		}))
	}
	return out, nil
}

// startMessage 合成 message_start 事件，返回单元素切片。
func (w *anthropicUserWriter) startMessage() [][]byte {
	w.started = true
	if w.messageID == "" {
		w.messageID = "msg_" + randSuffix()
	}
	return [][]byte{marshalStream(map[string]any{
		"type": "message_start",
		"message": map[string]any{
			"id":            w.messageID,
			"type":          "message",
			"role":          "assistant",
			"content":       []any{},
			"model":         w.model,
			"stop_reason":   nil,
			"stop_sequence": nil,
			"usage": map[string]any{
				"input_tokens":  w.usage.InputTokens,
				"output_tokens": w.usage.OutputTokens,
			},
		},
	})}
}

// ensureBlock 确保当前 content block 与期望类型一致；不一致时关闭旧 block 并
// 开启新 block。返回需要下发的合成事件（含可能的 content_block_stop + start）。
func (w *anthropicUserWriter) ensureBlock(blockType, toolID, toolName string) [][]byte {
	// 同类型（且同工具）→ 无需切换。
	sameBlock := w.blockStarted && w.blockType == blockType
	if blockType == "tool_use" {
		sameBlock = sameBlock && w.toolID == toolID
	}
	if sameBlock {
		return nil
	}

	var out [][]byte
	if w.blockStarted {
		out = append(out, marshalStream(map[string]any{
			"type":  "content_block_stop",
			"index": w.blockIndex,
		}))
	}
	w.blockIndex++
	w.blockStarted = true
	w.blockType = blockType
	w.toolID = toolID
	w.toolName = toolName

	if blockType == "tool_use" {
		out = append(out, marshalStream(map[string]any{
			"type":  "content_block_start",
			"index": w.blockIndex,
			"content_block": map[string]any{
				"type":  "tool_use",
				"id":    toolID,
				"name":  toolName,
				"input": map[string]any{},
			},
		}))
	} else {
		out = append(out, marshalStream(map[string]any{
			"type":          "content_block_start",
			"index":         w.blockIndex,
			"content_block": map[string]any{"type": "text", "text": ""},
		}))
	}
	return out
}

// marshalStream 将事件 map 序列化为 JSON，忽略序列化错误（map 内容恒可序列化）。
func marshalStream(m map[string]any) []byte {
	b, _ := json.Marshal(m)
	return b
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
		return anthropicContentBlock{Type: "tool_use", ID: anthropicToolUseID(b.ID), Name: b.Name, Input: b.Input}
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

// canonicalFinishToAnthropicStopReasonWithStops 在 canonicalFinishToAnthropicStopReason
// 基础上加入停止序列启发式：请求带 stop_sequences、上游 finish_reason=stop 且无文本
// 内容时，判定为命中停止序列。背景：推理模型（如 DeepSeek-V4-Flash）在思考阶段被
// stop/max_tokens 截断时不产出正文，上游只返回 finish_reason=stop，与自然结束无法
// 区分；此时按 Anthropic 语义映射为 stop_sequence 更符合请求意图。
func canonicalFinishToAnthropicStopReasonWithStops(fr string, stops []string, hasText bool) string {
	if len(stops) > 0 && fr == "stop" && !hasText {
		return "stop_sequence"
	}
	return canonicalFinishToAnthropicStopReason(fr)
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
