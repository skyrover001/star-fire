package format

import "star-fire/pkg/public"

// ReasoningProvider 由状态化用户流式 writer（如 responsesUserWriter）实现，暴露本流
// 累积到的 reasoning_content（keyed by tool_call_id），供适配层在流结束时保存到会话状态，
// 以便后续工具循环续接请求回传 reasoning_content。
type ReasoningProvider interface {
	ReasoningMap() map[string]string
}

// ExtractToolCallReasoning 从 Canonical 响应中提取每个工具调用对应的 reasoning_content。
//
// 思考模型（deepseek-v4-pro/flash 等）在调用工具时，先输出 reasoning_content，再输出
// tool_calls。reasoning_content 必须随该工具调用一起回传（否则后端报
// "reasoning_content in thinking mode must be passed back"）。本函数把 reasoning_content
// 关联到其后出现的每一个 tool_use 块，返回 map[toolCallID]reasoningText。
//
// 一个 assistant 回合可能包含多个 tool_use，它们共享同一段 reasoning_content。
func ExtractToolCallReasoning(cr *public.CanonicalResponse) map[string]string {
	if cr == nil {
		return nil
	}
	result := map[string]string{}
	var reasoning string
	for _, b := range cr.Content {
		switch b.Type {
		case "thinking":
			reasoning += b.Text
		case "tool_use":
			if b.ID != "" && reasoning != "" {
				result[b.ID] = reasoning
			}
		}
	}
	return result
}

// InjectReasoningContent 把上一轮保存的 reasoning_content 注入到请求中携带 tool_calls 的
// assistant 消息（作为 thinking 块置于消息开头）。
//
// 当 Codex 在工具循环的续接请求里只回传 function_call/web_search_call（而思考内容仅以
// 加密形式存在于 include 字段，无法使用）时，网关必须从自身状态还原 reasoning_content，
// 否则思考模型后端会拒绝。注入后，OpenAIConverter.BuildUpstreamRequest 会把 thinking 块
// 映射回下游 Chat 请求的 reasoning_content 字段。
func InjectReasoningContent(cr *public.CanonicalRequest, reasoning map[string]string) {
	if cr == nil || len(reasoning) == 0 {
		return
	}
	for i := range cr.Messages {
		m := &cr.Messages[i]
		if m.Role != "assistant" || len(m.ToolCalls) == 0 || hasThinkingBlock(m.Content) {
			continue
		}
		if r, ok := reasoning[m.ToolCalls[0].ID]; ok && r != "" {
			m.Content = append([]public.CanonicalContent{{Type: "thinking", Text: r}}, m.Content...)
		}
	}
}
