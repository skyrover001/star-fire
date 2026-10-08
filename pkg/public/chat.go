package public

import (
	"encoding/json"

	"github.com/sashabaranov/go-openai"
)

// ExtendedChatRequest 在 go-openai 标准请求之上，额外承载 SDK 尚未覆盖的
// 思考相关参数。标准字段（含 reasoning_effort、chat_template_kwargs）仍由
// go-openai 的 ChatCompletionRequest 负责，因此升级 SDK 不受影响；这里只
// 补充 SDK 缺失的字段，做到“外包一层”。
type ExtendedChatRequest struct {
	openai.ChatCompletionRequest
	// Thinking 原样透传给后端（例如 Anthropic 风格 {"type":"enabled"} 等），
	// 使用 json.RawMessage 不做结构解析，后端要什么形状就带什么形状。
	Thinking json.RawMessage `json:"thinking,omitempty"`
	// EnableThinking 使用指针以区分“未传”与“显式传 false”。
	EnableThinking *bool `json:"enable_thinking,omitempty"`
	// RawBody 保留原始请求体 JSON。当请求含视频等多模态输入（go-openai 无法承载
	// video part，反序列化时会丢弃）时，server 用它原样透传给 client，client 据此
	// 走原始 JSON 直连上游，避免视频 part 丢失。仅视频请求时填充，普通请求为空。
	RawBody json.RawMessage `json:"raw_body,omitempty"`

	// P1-M2: 路由偏好 + 容忍度（仅 server 消费，不透传给后端）
	// Routing 路由偏好：stability|cost|balanced，空则回退 header/APIKey/config。
	Routing string `json:"routing,omitempty"`
	// MaxLatencyMs 容忍度：最大可接受延迟（ms），0 表示不限制。
	MaxLatencyMs int `json:"max_latency_ms,omitempty"`
	// MinStability 容忍度：最低稳定性分 [0,1]，0 表示不限制。
	MinStability float64 `json:"min_stability,omitempty"`
}

// ExtraFields 返回 go-openai 未覆盖、需要额外合并进请求体的字段。
func (r *ExtendedChatRequest) ExtraFields() map[string]json.RawMessage {
	extra := make(map[string]json.RawMessage)
	if len(r.Thinking) > 0 {
		extra["thinking"] = r.Thinking
	}
	if r.EnableThinking != nil {
		if b, err := json.Marshal(*r.EnableThinking); err == nil {
			extra["enable_thinking"] = b
		}
	}
	return extra
}

// BuildRequestBody 先用 go-openai 序列化标准请求，再叠加扩展字段，
// 生成最终发给后端的请求体 JSON。当没有扩展字段时，等价于直接序列化
// 标准请求。
func (r *ExtendedChatRequest) BuildRequestBody() ([]byte, error) {
	base, err := json.Marshal(r.ChatCompletionRequest)
	if err != nil {
		return nil, err
	}

	extra := r.ExtraFields()
	if len(extra) == 0 {
		return base, nil
	}

	var merged map[string]json.RawMessage
	if err := json.Unmarshal(base, &merged); err != nil {
		return nil, err
	}
	for k, v := range extra {
		merged[k] = v
	}
	return json.Marshal(merged)
}
