// Package format 实现三种 API 格式（OpenAI Chat / Anthropic Messages /
// OpenAI Responses）与 Canonical 中间表示之间的双向转换。
//
// 核心原则：转换全部在 server 端完成。client 只负责「上游格式请求 → 上游
// 调用 → 上游格式响应」的透传。详见 docs/multi-format-api-design.md。
package format

import (
	"fmt"

	"star-fire/pkg/public"
)

// Converter 是三种 API 格式与 Canonical 中间表示之间双向转换的统一接口。
// 每个格式（openai / anthropic / responses）实现一个 Converter。
type Converter interface {
	// ParseRequest 解析用户/上游请求 → Canonical。
	ParseRequest(body []byte) (*public.CanonicalRequest, error)
	// BuildResponse 将 Canonical 响应 → 用户格式响应（用于回传用户）。
	BuildResponse(canonical *public.CanonicalResponse) ([]byte, error)
	// BuildUpstreamRequest 将 Canonical 请求 → 上游格式请求（用于发给 client）。
	BuildUpstreamRequest(canonical *public.CanonicalRequest) ([]byte, error)
	// ParseUpstreamResponse 解析上游格式响应 → Canonical。
	ParseUpstreamResponse(data []byte) (*public.CanonicalResponse, error)
	// ParseUpstreamStreamEvent 解析上游 SSE 事件 → Canonical 流式事件。
	ParseUpstreamStreamEvent(line []byte) (*public.CanonicalStreamEvent, error)
	// BuildUserStreamEvent 将 Canonical 流式事件 → 用户格式 SSE 事件。
	BuildUserStreamEvent(ev *public.CanonicalStreamEvent) ([]byte, error)
}

// registry 保存已注册的转换器，按格式名索引。
var registry = map[string]Converter{}

// Register 注册一个格式转换器（P1/P2 阶段分别注册 openai/anthropic/responses）。
func Register(format string, c Converter) {
	registry[format] = c
}

// GetConverter 按格式名返回对应转换器。
func GetConverter(format string) (Converter, error) {
	c, ok := registry[format]
	if !ok {
		return nil, fmt.Errorf("unsupported format: %q", format)
	}
	return c, nil
}
