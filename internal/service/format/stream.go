package format

import "star-fire/pkg/public"

// StreamAccumulator 处理上游 SSE 流，累积跨事件所需的必要状态（usage、
// finish_reason、工具参数），产出 Canonical 流式事件。
//
// 每个流必须使用一个独立实例，以保证多路流/并发流之间的累积状态互不串扰
// （对应设计文档 3.4 节「状态机隔离」用例）。无状态的 OpenAI Chat 格式仍可用
// Converter.ParseUpstreamStreamEvent 逐帧处理，但为统一适配层接口，三种格式
// 都应提供 NewStreamAccumulator。
type StreamAccumulator interface {
	// Feed 处理一个上游 SSE data 帧（JSON payload，不含 `data: ` 前缀），
	// 产出 0 或多个 Canonical 流式事件。
	Feed(data []byte) ([]*public.CanonicalStreamEvent, error)
	// Flush 在流结束时调用，产出收尾事件（如 done）。若流已正常结束，
	// 返回 nil。它兜底处理异常中断（上游未发终止帧）的场景。
	Flush() (*public.CanonicalStreamEvent, error)
}

// AccumulatingConverter 表示需要跨事件累积状态的转换器（Anthropic Messages /
// OpenAI Responses）。适配层通过 NewStreamAccumulator 为每个流创建独立累积器。
type AccumulatingConverter interface {
	Converter
	NewStreamAccumulator() StreamAccumulator
}

// UserStreamWriter 负责把 Canonical 流式事件转成用户格式的 SSE 事件序列。
// 某些格式（如 OpenAI Responses）要求输出一整套结构性事件（response.created、
// output_item.added、content_part.added、...、response.completed），这些事件
// 需要跨 delta 累积状态。适配层为每个流创建一个独立 writer。
type UserStreamWriter interface {
	// Write 处理一个 Canonical 流式事件，返回 0 或多个用户格式 SSE 事件 JSON。
	// 每个返回的 JSON 对应一行 `data: <json>`。
	Write(ev *public.CanonicalStreamEvent) ([][]byte, error)
	// Flush 在流结束时调用，产出收尾事件（如 response.completed）。若无需收尾返回 nil。
	Flush() ([][]byte, error)
}

// UserStreamWriterConverter 表示需要状态化用户流式输出的转换器（OpenAI Responses）。
type UserStreamWriterConverter interface {
	Converter
	NewUserStreamWriter() UserStreamWriter
}
