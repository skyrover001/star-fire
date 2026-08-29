package public

import "encoding/json"

// 本文件定义多格式 API 转换的 Canonical（无损中间表示）。
//
// 三种上游/用户格式（OpenAI Chat / Anthropic Messages / OpenAI Responses）
// 都先转换到 Canonical，再由 Canonical 转换到目标格式。Canonical 的 content
// 统一为 block 数组，这是无损转换的关键；任何无法映射到标准字段的原始字段
// 都放入 Extra 兜底，转换回原格式时原样还原，确保不丢字段。
//
// 详见 docs/multi-format-api-design.md 第二章。

// CanonicalRequest 是三种格式请求的无损中间表示。
type CanonicalRequest struct {
	Model       string                     `json:"model"`
	Stream      bool                       `json:"stream"`
	System      []CanonicalContent         `json:"system"` // 系统提示（Anthropic system / OpenAI system message / Responses instructions）
	Messages    []CanonicalMessage         `json:"messages"`
	Tools       []CanonicalTool            `json:"tools"`
	MaxTokens   *int                       `json:"max_tokens"`
	Temperature *float64                   `json:"temperature"`
	TopP        *float64                   `json:"top_p"`
	Thinking    json.RawMessage            `json:"thinking,omitempty"` // 透传 thinking/reasoning 参数
	Extra       map[string]json.RawMessage `json:"extra,omitempty"`    // 无法映射的原始字段透传（保底无损）
}

// CanonicalMessage 是统一后的消息。
type CanonicalMessage struct {
	Role       string              `json:"role"` // system | user | assistant | tool
	Content    []CanonicalContent  `json:"content"`
	ToolCalls  []CanonicalToolCall `json:"tool_calls,omitempty"`   // assistant 的工具调用
	ToolCallID string              `json:"tool_call_id,omitempty"` // tool 消息关联的 tool_use id
	Name       string              `json:"name,omitempty"`
}

// CanonicalContent 是统一后的内容块。
type CanonicalContent struct {
	Type     string `json:"type"` // text | image | video | tool_use | tool_result | thinking
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	// VideoURL 承载视频输入（data URL 或 http(s) URL）。
	// 对应 OpenAI Chat 的 video_url / Responses 的 input_video / Anthropic 的 video block。
	VideoURL string `json:"video_url,omitempty"`
	// tool_use
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
	// ToolType 保留工具调用的原始类型（function | web_search | tool_search | custom | ...）。
	// 用于响应方向按标准 Responses item 类型映射（function_call / web_search_call / custom_tool_call）。
	ToolType string `json:"tool_type,omitempty"`
	// tool_result
	Content []CanonicalContent `json:"content,omitempty"` // 嵌套内容
	IsError bool               `json:"is_error,omitempty"`
	// Extra 承载块级附加字段（如 Anthropic thinking 的 signature），保底无损。
	Extra map[string]json.RawMessage `json:"extra,omitempty"`
}

// CanonicalTool 是统一后的工具定义。
type CanonicalTool struct {
	Type        string          `json:"type"` // function | web_search | tool_search | ...
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"` // JSON Schema
	// Raw 保留非 function 工具的原始 JSON（如 web_search / tool_search），
	// 供下游转换时原样透传，避免丢失工具类型。
	Raw json.RawMessage `json:"raw,omitempty"`
}

// CanonicalToolCall 是统一后的工具调用。
type CanonicalToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"` // 参数（原始 JSON 字符串）
}

// CanonicalResponse 是三种格式响应的无损中间表示。
type CanonicalResponse struct {
	ID           string             `json:"id"`
	Content      []CanonicalContent `json:"content"`
	Usage        CanonicalUsage     `json:"usage"`
	FinishReason string             `json:"finish_reason,omitempty"`
}

// CanonicalUsage 是统一后的 token 用量。
type CanonicalUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
	CachedTokens int `json:"cached_tokens"` // 缓存命中输入 tokens
}

// CanonicalStreamEvent 是统一后的流式事件。
type CanonicalStreamEvent struct {
	Type         string             `json:"type"` // text_delta | tool_call_delta | done | error
	Text         string             `json:"text,omitempty"`
	ToolCall     *CanonicalToolCall `json:"tool_call,omitempty"`
	Usage        *CanonicalUsage    `json:"usage,omitempty"`
	FinishReason string             `json:"finish_reason,omitempty"`
}

// 流式事件类型常量。
const (
	StreamEventTextDelta     = "text_delta"
	StreamEventToolCallDelta = "tool_call_delta"
	StreamEventThinkingDelta = "thinking_delta"
	StreamEventDone          = "done"
	StreamEventError         = "error"
)
