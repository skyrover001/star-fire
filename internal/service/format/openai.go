package format

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/sashabaranov/go-openai"

	"star-fire/pkg/public"
)

// OpenAIConverter 实现 OpenAI Chat Completions ↔ Canonical 的双向转换。
//
// 详见 docs/multi-format-api-design.md 3.1 节。
type OpenAIConverter struct{}

func init() {
	Register(public.FormatOpenAI, &OpenAIConverter{})
}

// ParseRequest 解析 OpenAI Chat 请求 → Canonical。
func (c *OpenAIConverter) ParseRequest(body []byte) (*public.CanonicalRequest, error) {
	var req openai.ChatCompletionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}
	if req.Model == "" {
		return nil, errors.New("openai: missing required field 'model'")
	}
	if len(req.Messages) == 0 {
		return nil, errors.New("openai: missing required field 'messages'")
	}

	cr := &public.CanonicalRequest{
		Model:  req.Model,
		Stream: req.Stream,
	}

	// max_tokens：优先 max_tokens，兼容 max_completion_tokens（o1 系列）。
	if req.MaxTokens > 0 {
		v := req.MaxTokens
		cr.MaxTokens = &v
	} else if req.MaxCompletionTokens > 0 {
		v := req.MaxCompletionTokens
		cr.MaxTokens = &v
	}
	if req.Temperature != 0 {
		v := float64(req.Temperature)
		cr.Temperature = &v
	}
	if req.TopP != 0 {
		v := float64(req.TopP)
		cr.TopP = &v
	}
	// reasoning_effort → Thinking（保留为字符串字面量）。
	if req.ReasoningEffort != "" {
		cr.Thinking = marshalString(req.ReasoningEffort)
	}

	for _, t := range req.Tools {
		ct := public.CanonicalTool{Type: string(t.Type)}
		if t.Function != nil {
			ct.Name = t.Function.Name
			ct.Description = t.Function.Description
			if t.Function.Parameters != nil {
				ct.Parameters, _ = json.Marshal(t.Function.Parameters)
			}
		}
		cr.Tools = append(cr.Tools, ct)
	}

	for _, m := range req.Messages {
		// developer 视同 system（OpenAI Responses 的 developer 角色承载系统指令）。
		if m.Role == openai.ChatMessageRoleSystem || m.Role == "developer" {
			// 系统提示提取到顶层 System。
			cr.System = append(cr.System, messageToBlocks(m)...)
			continue
		}

		cm := public.CanonicalMessage{Role: m.Role, Name: m.Name}

		if m.Role == openai.ChatMessageRoleTool {
			// tool 消息：content 包成 tool_result 块，关联 tool_use id。
			cm.ToolCallID = m.ToolCallID
			cm.Content = []public.CanonicalContent{{
				Type:    "tool_result",
				ID:      m.ToolCallID,
				Content: messageToBlocks(m),
			}}
		} else {
			cm.Content = messageToBlocks(m)
			// assistant 的工具调用：同时填充 Content 的 tool_use 块与 ToolCalls。
			for _, tc := range m.ToolCalls {
				cm.ToolCalls = append(cm.ToolCalls, public.CanonicalToolCall{
					ID:        tc.ID,
					Name:      tc.Function.Name,
					Arguments: marshalString(tc.Function.Arguments),
				})
				cm.Content = append(cm.Content, public.CanonicalContent{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Function.Name,
					Input: parseArgsToInput(tc.Function.Arguments),
				})
			}
		}

		cr.Messages = append(cr.Messages, cm)
	}

	return cr, nil
}

// BuildUpstreamRequest 将 Canonical 请求 → OpenAI Chat 请求。
func (c *OpenAIConverter) BuildUpstreamRequest(cr *public.CanonicalRequest) ([]byte, error) {
	req := openai.ChatCompletionRequest{
		Model:  cr.Model,
		Stream: cr.Stream,
	}
	if cr.MaxTokens != nil {
		req.MaxTokens = *cr.MaxTokens
	}
	if cr.Temperature != nil {
		req.Temperature = float32(*cr.Temperature)
	}
	if cr.TopP != nil {
		req.TopP = float32(*cr.TopP)
	}

	// 系统提示还原为 system 消息。
	if len(cr.System) > 0 {
		req.Messages = append(req.Messages, openai.ChatCompletionMessage{
			Role:         openai.ChatMessageRoleSystem,
			MultiContent: blocksToParts(cr.System),
		})
	}

	for _, t := range cr.Tools {
		// 非 function 工具（web_search/tool_search/custom/namespace 等）：
		// 下游 vLLM 后端只接受 function 类型工具，因此把非 function 工具
		// 归一化为 function 类型，name 用工具类型名（保证非空合法），
		// 参数用原始 JSON 的其余字段。这样后端能接受，Codex 也能继续调用。
		if t.Type != "" && t.Type != "function" {
			tool := openai.Tool{Type: openai.ToolTypeFunction}
			fd := &openai.FunctionDefinition{Name: t.Name, Description: t.Description}
			if fd.Name == "" {
				fd.Name = t.Type // 用工具类型作为 name，保证非空
			}
			if len(t.Parameters) > 0 {
				fd.Parameters = json.RawMessage(t.Parameters)
			} else if len(t.Raw) > 0 {
				// 从原始 JSON 提取 parameters（若有），否则用空对象。
				var raw map[string]json.RawMessage
				if json.Unmarshal(t.Raw, &raw) == nil {
					if p, ok := raw["parameters"]; ok {
						fd.Parameters = json.RawMessage(p)
					}
				}
				if fd.Parameters == nil {
					fd.Parameters = json.RawMessage(`{"type":"object","properties":{}}`)
				}
			}
			// FREEFORM 工具（如 apply_patch）的 parameters 为空对象 schema，
			// 模型不理解 FREEFORM 语义会生成 {} 空参数。注入 input 字符串属性
			// schema，引导模型把自由格式内容放进 input 字段。
			if isFreeformTool(fd.Name, fd.Parameters) {
				fd.Parameters = json.RawMessage(freeformInputSchema)
			}
			tool.Function = fd
			req.Tools = append(req.Tools, tool)
			continue
		}
		tool := openai.Tool{Type: openai.ToolTypeFunction}
		fd := &openai.FunctionDefinition{Name: t.Name, Description: t.Description}
		if len(t.Parameters) > 0 {
			fd.Parameters = json.RawMessage(t.Parameters)
		}
		// FREEFORM 工具（如 apply_patch）即使是 function 类型，也可能带空 schema。
		if isFreeformTool(fd.Name, fd.Parameters) {
			fd.Parameters = json.RawMessage(freeformInputSchema)
		}
		tool.Function = fd
		req.Tools = append(req.Tools, tool)
	}

	// videoParts 记录需要注入到序列化 JSON 中的视频块。
	// go-openai 的 ChatMessagePart 只支持 text/image，无法承载 video_url，
	// 因此这里先收集，序列化后再通过 ensureVideoParts 注入原始 JSON。
	var videoInjects []videoInject

	for _, cm := range cr.Messages {
		// 归一化角色：developer 视同 system（OpenAI Responses 的 developer 角色
		// 承载系统/技能/权限指令，但部分后端 Chat API 不接受 developer，只认
		// system/user/assistant/tool）。统一映射为 system，避免 400 unknown variant。
		role := cm.Role
		if role == "developer" {
			role = openai.ChatMessageRoleSystem
		}
		m := openai.ChatCompletionMessage{Role: role, Name: cm.Name}

		if cm.Role == openai.ChatMessageRoleTool {
			m.ToolCallID = cm.ToolCallID
			// 兜底：tool 消息缺少 tool_call_id 时（如 Codex CLI 对不支持的
			// 工具调用生成的 "unsupported call" 结果），从前面的 assistant
			// 消息中找一个尚未被 tool 消息匹配的 tool_call.id 来关联。
			// vLLM 等后端要求 tool 消息必须带 tool_call_id，否则返回 400。
			if m.ToolCallID == "" {
				m.ToolCallID = findUnmatchedToolCallID(req.Messages, cm.ToolCallID)
			}
			// 从 tool_result 块还原 content 字符串。
			if len(cm.Content) > 0 && cm.Content[0].Type == "tool_result" {
				m.Content = blocksToText(cm.Content[0].Content)
			} else {
				m.Content = blocksToText(cm.Content)
			}
		} else {
			// text/image 块 → MultiContent；tool_use 块由 ToolCalls 承载；
			// thinking 块 → ReasoningContent（reasoning 模型要求回传 reasoning_content）；
			// video 块 → 记录到 videoInjects，序列化后注入原始 JSON。
			var parts []openai.ChatMessagePart
			for _, b := range cm.Content {
				switch b.Type {
				case "text", "image":
					parts = append(parts, blocksToParts([]public.CanonicalContent{b})...)
				case "video":
					videoInjects = append(videoInjects, videoInject{msgIdx: len(req.Messages), url: b.VideoURL})
				case "thinking":
					m.ReasoningContent = b.Text
				}
			}
			if len(parts) > 0 {
				m.MultiContent = parts
			}
			for _, tc := range cm.ToolCalls {
				// 兜底：确保 tool_call.id 非空。
				// go-openai 的 ToolCall.ID 带 omitempty，空值会被序列化时丢弃，
				// 导致后端（如 vLLM Pydantic 校验）报 400 "field required: id"。
				// 这里作为最后一道防线合成稳定 ID，避免请求被后端拒绝。
				// 正常路径下 responses.go 已保证非空。
				callID := tc.ID
				if callID == "" {
					callID = "call_" + randSuffix()
				}
				// 任何工具调用的 arguments 都必须是 JSON。上游模型可能生成畸形调用，
				// 把自由文本（patch 内容等）塞进 arguments（甚至塞错工具名）。这里
				// 对非 JSON 的 arguments 归一化为 {"input":"<原文>"}，避免上游 400。
				m.ToolCalls = append(m.ToolCalls, openai.ToolCall{
					ID:       callID,
					Type:     openai.ToolTypeFunction,
					Function: openai.FunctionCall{Name: tc.Name, Arguments: normalizeToolArguments(rawToString(tc.Arguments))},
				})
			}
		}

		req.Messages = append(req.Messages, m)
	}

	raw, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	// 后处理：go-openai 的 ChatCompletionMessage.MarshalJSON 对空 Content
	// 使用 omitempty，会省略 "content" 字段。但 vLLM 等后端的 Pydantic
	// 校验要求 assistant 消息必须带 content 字段（即使是空字符串），
	// 否则返回 400 "Expecting value: line 1 column 1 (char 0)"。
	// 这里在序列化后的 JSON 中为缺失 content 的 assistant 消息补上
	// "content":""，确保后端能接受。
	raw = ensureAssistantContent(raw)
	// 注入视频块：go-openai 无法承载 video_url，这里把收集到的视频块
	// 以 {"type":"video_url","video_url":{"url":...}} 形式追加到对应
	// 消息的 content 数组中。
	if len(videoInjects) > 0 {
		raw = ensureVideoParts(raw, videoInjects)
	}
	return raw, nil
}

// findUnmatchedToolCallID 在已构建的消息列表中查找一个 assistant 消息的
// tool_call.id，该 id 尚未被后续的 tool 消息（tool_call_id）匹配过。
// 用于兜底处理 Codex CLI 产生的缺少 tool_call_id 的 tool 消息
// （如 "unsupported call" 结果）。
func findUnmatchedToolCallID(messages []openai.ChatCompletionMessage, _ string) string {
	// 收集所有已被 tool 消息匹配的 tool_call_id。
	matched := make(map[string]bool)
	for _, m := range messages {
		if m.Role == openai.ChatMessageRoleTool && m.ToolCallID != "" {
			matched[m.ToolCallID] = true
		}
	}
	// 从后往前找最近的 assistant 消息中未匹配的 tool_call.id。
	for i := len(messages) - 1; i >= 0; i-- {
		m := messages[i]
		if m.Role != openai.ChatMessageRoleAssistant {
			continue
		}
		for _, tc := range m.ToolCalls {
			if tc.ID != "" && !matched[tc.ID] {
				return tc.ID
			}
		}
	}
	return ""
}

// ensureAssistantContent 在序列化后的 Chat 请求 JSON 中，为缺失 "content"
// 字段的 assistant 消息补上 "content":""。
//
// go-openai 的 ChatCompletionMessage.MarshalJSON 对空 Content 使用 omitempty，
// 会完全省略 "content" 字段。但 vLLM 等后端的 Pydantic 校验要求 assistant
// 消息必须带 content 字段，否则返回 400。
// 这里解析 JSON 数组，逐条检查 role=assistant 且没有 content 键的消息，
// 插入 "content":""。
func ensureAssistantContent(raw []byte) []byte {
	var req map[string]json.RawMessage
	if json.Unmarshal(raw, &req) != nil {
		return raw // 解析失败，原样返回
	}
	msgsRaw, ok := req["messages"]
	if !ok {
		return raw
	}
	var msgs []json.RawMessage
	if json.Unmarshal(msgsRaw, &msgs) != nil {
		return raw
	}
	modified := false
	for i, mRaw := range msgs {
		var m map[string]json.RawMessage
		if json.Unmarshal(mRaw, &m) != nil {
			continue
		}
		role := ""
		if r, ok := m["role"]; ok {
			_ = json.Unmarshal(r, &role)
		}
		if role != "assistant" {
			continue
		}
		// 检查是否已有 content 字段。
		if _, hasContent := m["content"]; hasContent {
			continue
		}
		// 插入 "content":""。
		m["content"] = json.RawMessage(`""`)
		newRaw, err := json.Marshal(m)
		if err != nil {
			continue
		}
		msgs[i] = newRaw
		modified = true
	}
	if !modified {
		return raw
	}
	newMsgs, err := json.Marshal(msgs)
	if err != nil {
		return raw
	}
	req["messages"] = newMsgs
	result, err := json.Marshal(req)
	if err != nil {
		return raw
	}
	return result
}

// videoInject 描述一个需要注入到序列化 Chat 请求 JSON 中的视频块。
type videoInject struct {
	msgIdx int    // 目标消息在 messages 数组中的下标
	url    string // 视频 URL（data URI 或 http(s) URL）
}

// ensureVideoParts 把视频块注入到序列化后的 Chat 请求 JSON 中。
//
// go-openai 的 ChatMessagePart 只支持 text/image，无法承载 video_url，
// 因此视频块在 BuildUpstreamRequest 中先被收集，序列化后再以
// {"type":"video_url","video_url":{"url":...}} 形式追加到对应消息的
// content 数组中。这样 vLLM 等支持视频输入的后端能正确接收视频。
func ensureVideoParts(raw []byte, injects []videoInject) []byte {
	var req map[string]json.RawMessage
	if json.Unmarshal(raw, &req) != nil {
		return raw
	}
	msgsRaw, ok := req["messages"]
	if !ok {
		return raw
	}
	var msgs []json.RawMessage
	if json.Unmarshal(msgsRaw, &msgs) != nil {
		return raw
	}
	modified := false
	for _, inj := range injects {
		if inj.msgIdx < 0 || inj.msgIdx >= len(msgs) {
			continue
		}
		var m map[string]json.RawMessage
		if json.Unmarshal(msgs[inj.msgIdx], &m) != nil {
			continue
		}
		// 构造视频 part。
		videoPart := map[string]interface{}{
			"type": "video_url",
			"video_url": map[string]interface{}{
				"url": inj.url,
			},
		}
		vpRaw, err := json.Marshal(videoPart)
		if err != nil {
			continue
		}
		// 追加到 content 数组。
		var content []json.RawMessage
		if cRaw, has := m["content"]; has {
			_ = json.Unmarshal(cRaw, &content)
		}
		content = append(content, vpRaw)
		cArr, err := json.Marshal(content)
		if err != nil {
			continue
		}
		m["content"] = cArr
		newRaw, err := json.Marshal(m)
		if err != nil {
			continue
		}
		msgs[inj.msgIdx] = newRaw
		modified = true
	}
	if !modified {
		return raw
	}
	newMsgs, err := json.Marshal(msgs)
	if err != nil {
		return raw
	}
	req["messages"] = newMsgs
	result, err := json.Marshal(req)
	if err != nil {
		return raw
	}
	return result
}

// ParseUpstreamResponse 解析 OpenAI Chat 响应 → Canonical。
func (c *OpenAIConverter) ParseUpstreamResponse(data []byte) (*public.CanonicalResponse, error) {
	var resp openai.ChatCompletionResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	cr := &public.CanonicalResponse{ID: resp.ID}
	if len(resp.Choices) > 0 {
		msg := resp.Choices[0].Message
		cr.FinishReason = string(resp.Choices[0].FinishReason)
		// reasoning_content（deepseek 等）→ thinking 块，置于正文之前。
		if msg.ReasoningContent != "" {
			cr.Content = append(cr.Content, public.CanonicalContent{Type: "thinking", Text: msg.ReasoningContent})
		}
		cr.Content = append(cr.Content, messageToBlocks(msg)...)
		for _, tc := range msg.ToolCalls {
			// FREEFORM 工具（如 apply_patch）：模型按注入的 schema 生成
			// {"input":"<patch content>"}，这里提取 input 字段并还原为原始
			// 内容字符串，让 Codex 收到的是自由格式文本而非 JSON 包裹。
			args := tc.Function.Arguments
			if isFreeformTool(tc.Function.Name, nil) {
				if input := extractFreeformInput(args); input != "" {
					args = input
				}
			}
			cr.Content = append(cr.Content, public.CanonicalContent{
				Type:     "tool_use",
				ID:       tc.ID,
				Name:     tc.Function.Name,
				Input:    parseArgsToInput(args),
				ToolType: inferToolType(tc.Function.Name),
			})
		}
	}
	cr.Usage = usageToCanonical(resp.Usage)
	return cr, nil
}

// BuildResponse 将 Canonical 响应 → OpenAI Chat 响应。
func (c *OpenAIConverter) BuildResponse(cr *public.CanonicalResponse) ([]byte, error) {
	resp := openai.ChatCompletionResponse{
		ID:     cr.ID,
		Object: "chat.completion",
	}

	msg := openai.ChatCompletionMessage{Role: openai.ChatMessageRoleAssistant}
	var parts []openai.ChatMessagePart
	var toolCalls []openai.ToolCall
	for _, b := range cr.Content {
		switch b.Type {
		case "text":
			parts = append(parts, openai.ChatMessagePart{Type: openai.ChatMessagePartTypeText, Text: b.Text})
		case "image":
			parts = append(parts, blocksToParts([]public.CanonicalContent{b})...)
		case "tool_use":
			toolCalls = append(toolCalls, openai.ToolCall{
				ID:       b.ID,
				Type:     openai.ToolTypeFunction,
				Function: openai.FunctionCall{Name: b.Name, Arguments: rawToArgsString(b.Input)},
			})
		case "thinking":
			msg.ReasoningContent = b.Text
		}
	}
	if len(parts) > 0 {
		msg.MultiContent = parts
	}
	msg.ToolCalls = toolCalls

	resp.Choices = []openai.ChatCompletionChoice{{
		Index:        0,
		Message:      msg,
		FinishReason: openai.FinishReason(cr.FinishReason),
	}}
	resp.Usage = canonicalToUsage(cr.Usage)
	return json.Marshal(resp)
}

// ParseUpstreamStreamEvent 解析单个 OpenAI 流式事件 → Canonical 流式事件。
//
// 输入 line 是单个 SSE 事件的 JSON payload（不含 `data: ` 前缀）或终止标记 `[DONE]`。
// 本方法无状态：每个 chunk 独立转换，tool call 参数分片直出（不跨事件累积），
// 累积拼接由消费方（适配层）完成。`[DONE]` 不产出事件（终止信息已由 usage/finish_reason
// chunk 表达）。
func (c *OpenAIConverter) ParseUpstreamStreamEvent(line []byte) (*public.CanonicalStreamEvent, error) {
	s := strings.TrimSpace(string(line))
	if s == "[DONE]" {
		return nil, nil
	}

	var chunk openai.ChatCompletionStreamResponse
	if err := json.Unmarshal(line, &chunk); err != nil {
		return nil, err
	}

	// 带 usage 的结束 chunk（stream_options.include_usage=true）。
	if chunk.Usage != nil {
		u := usageToCanonical(*chunk.Usage)
		ev := &public.CanonicalStreamEvent{Type: public.StreamEventDone, Usage: &u}
		if len(chunk.Choices) > 0 {
			ev.FinishReason = string(chunk.Choices[0].FinishReason)
		}
		return ev, nil
	}

	if len(chunk.Choices) == 0 {
		return nil, nil
	}

	delta := chunk.Choices[0].Delta
	finishReason := string(chunk.Choices[0].FinishReason)

	if delta.ReasoningContent != "" {
		return &public.CanonicalStreamEvent{Type: public.StreamEventThinkingDelta, Text: delta.ReasoningContent}, nil
	}
	if delta.Content != "" {
		return &public.CanonicalStreamEvent{Type: public.StreamEventTextDelta, Text: delta.Content}, nil
	}
	if len(delta.ToolCalls) > 0 {
		tc := delta.ToolCalls[0]
		return &public.CanonicalStreamEvent{
			Type: public.StreamEventToolCallDelta,
			ToolCall: &public.CanonicalToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: marshalString(tc.Function.Arguments),
			},
		}, nil
	}
	if finishReason != "" {
		return &public.CanonicalStreamEvent{Type: public.StreamEventDone, FinishReason: finishReason}, nil
	}
	return nil, nil
}

// BuildUserStreamEvent 将 Canonical 流式事件 → OpenAI 流式 chunk。
func (c *OpenAIConverter) BuildUserStreamEvent(ev *public.CanonicalStreamEvent) ([]byte, error) {
	chunk := openai.ChatCompletionStreamResponse{
		ID:     "chatcmpl-stream",
		Object: "chat.completion.chunk",
	}

	switch ev.Type {
	case public.StreamEventTextDelta:
		chunk.Choices = []openai.ChatCompletionStreamChoice{{
			Index: 0,
			Delta: openai.ChatCompletionStreamChoiceDelta{Content: ev.Text},
		}}
	case public.StreamEventThinkingDelta:
		chunk.Choices = []openai.ChatCompletionStreamChoice{{
			Index: 0,
			Delta: openai.ChatCompletionStreamChoiceDelta{ReasoningContent: ev.Text},
		}}
	case public.StreamEventToolCallDelta:
		if ev.ToolCall != nil {
			chunk.Choices = []openai.ChatCompletionStreamChoice{{
				Index: 0,
				Delta: openai.ChatCompletionStreamChoiceDelta{
					ToolCalls: []openai.ToolCall{{
						ID:       ev.ToolCall.ID,
						Type:     openai.ToolTypeFunction,
						Function: openai.FunctionCall{Name: ev.ToolCall.Name, Arguments: rawToString(ev.ToolCall.Arguments)},
					}},
				},
			}}
		}
	case public.StreamEventDone:
		chunk.Choices = []openai.ChatCompletionStreamChoice{{
			Index:        0,
			FinishReason: openai.FinishReason(ev.FinishReason),
		}}
		if ev.Usage != nil {
			u := canonicalToUsage(*ev.Usage)
			chunk.Usage = &u
		}
	default:
		// error 等类型：返回空 chunk。
	}
	return json.Marshal(chunk)
}

// ---- 以下为包内共享的映射辅助函数 ----

// messageToBlocks 把 OpenAI 消息的 content（string 或 MultiContent 数组）转为 block 数组。
func messageToBlocks(m openai.ChatCompletionMessage) []public.CanonicalContent {
	if len(m.MultiContent) > 0 {
		blocks := make([]public.CanonicalContent, 0, len(m.MultiContent))
		for _, part := range m.MultiContent {
			switch part.Type {
			case openai.ChatMessagePartTypeText:
				blocks = append(blocks, public.CanonicalContent{Type: "text", Text: part.Text})
			case openai.ChatMessagePartTypeImageURL:
				b := public.CanonicalContent{Type: "image"}
				if part.ImageURL != nil {
					b.ImageURL = part.ImageURL.URL
					if part.ImageURL.Detail != "" {
						b.Extra = map[string]json.RawMessage{"detail": marshalString(string(part.ImageURL.Detail))}
					}
				}
				blocks = append(blocks, b)
			}
		}
		return blocks
	}
	if m.Content != "" {
		return []public.CanonicalContent{{Type: "text", Text: m.Content}}
	}
	return nil
}

// blocksToParts 把 text/image block 数组转为 OpenAI 的 ChatMessagePart 数组。
func blocksToParts(blocks []public.CanonicalContent) []openai.ChatMessagePart {
	var parts []openai.ChatMessagePart
	for _, b := range blocks {
		switch b.Type {
		case "text":
			parts = append(parts, openai.ChatMessagePart{Type: openai.ChatMessagePartTypeText, Text: b.Text})
		case "image":
			part := openai.ChatMessagePart{Type: openai.ChatMessagePartTypeImageURL, ImageURL: &openai.ChatMessageImageURL{URL: b.ImageURL}}
			if b.Extra != nil {
				if d, ok := b.Extra["detail"]; ok {
					part.ImageURL.Detail = openai.ImageURLDetail(rawToString(d))
				}
			}
			parts = append(parts, part)
		}
	}
	return parts
}

// blocksToText 把 block 数组中的所有 text 块拼接为纯文本。
func blocksToText(blocks []public.CanonicalContent) string {
	var sb strings.Builder
	for _, b := range blocks {
		if b.Type == "text" {
			sb.WriteString(b.Text)
		}
	}
	return sb.String()
}

// usageToCanonical 把 OpenAI Usage → CanonicalUsage。
func usageToCanonical(u openai.Usage) public.CanonicalUsage {
	cu := public.CanonicalUsage{
		InputTokens:  u.PromptTokens,
		OutputTokens: u.CompletionTokens,
		TotalTokens:  u.TotalTokens,
	}
	if u.PromptTokensDetails != nil {
		cu.CachedTokens = u.PromptTokensDetails.CachedTokens
	}
	return cu
}

// canonicalToUsage 把 CanonicalUsage → OpenAI Usage。
func canonicalToUsage(cu public.CanonicalUsage) openai.Usage {
	return openai.Usage{
		PromptTokens:     cu.InputTokens,
		CompletionTokens: cu.OutputTokens,
		TotalTokens:      cu.TotalTokens,
		PromptTokensDetails: &openai.PromptTokensDetails{
			CachedTokens: cu.CachedTokens,
		},
	}
}
