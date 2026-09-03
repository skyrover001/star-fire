package format

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"
)

// 本文件存放三个转换器共享的 JSON 处理辅助函数。

// marshalString 把 Go 字符串编码为 JSON 字符串字面量（含引号），存入 RawMessage。
// 例如 `{"city":"北京"}` → `"{\"city\":\"北京\"}"`。
func marshalString(s string) json.RawMessage {
	b, _ := json.Marshal(s)
	return json.RawMessage(b)
}

// rawToString 把 json.RawMessage（JSON 字符串字面量或原始 JSON）转回 Go 字符串。
// 优先按字符串字面量解析；解析失败则返回原始内容。
func rawToString(r json.RawMessage) string {
	if len(r) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(r, &s) == nil {
		return s
	}
	return string(r)
}

// randSuffix 生成一个随机十六进制后缀（用于合成 response.id / item.id）。
func randSuffix() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// timeNowUnix 返回当前 Unix 时间戳（秒）。
func timeNowUnix() int64 {
	return time.Now().Unix()
}

// inferToolType 根据工具名推断工具类型（用于响应方向映射标准 Responses item 类型）。
// 下游 Chat 只返回 function 工具名，网关据此还原 Codex 期望的工具类型。
func inferToolType(name string) string {
	switch name {
	case "web_search":
		return "web_search"
	case "tool_search":
		return "tool_search"
	case "apply_patch":
		return "custom"
	case "collaboration":
		return "namespace"
	default:
		return "function"
	}
}

// isFreeformTool 判断工具是否为 FREEFORM（自由格式）工具。
// Codex 的 apply_patch 等工具声明 "This is a FREEFORM tool, so do not wrap the
// patch in JSON"，其 parameters schema 为空对象 {"properties":{},"type":"object"}。
// 下游模型（如 GLM-5.2）不理解 FREEFORM 语义，看到空 schema 就生成 {} 作为参数，
// 导致工具调用无实际内容。网关需为这类工具注入一个 input 字符串属性 schema，
// 引导模型把自由格式内容放进 input 字段。
// parameters 接受 json.RawMessage、[]byte、string 或 nil；其他类型视为非空 schema。
func isFreeformTool(name string, parameters any) bool {
	if name != "apply_patch" {
		return false
	}
	var raw json.RawMessage
	switch p := parameters.(type) {
	case nil:
		return true
	case json.RawMessage:
		raw = p
	case []byte:
		raw = p
	case string:
		raw = json.RawMessage(p)
	default:
		// 非 JSON 原始类型（如 map/struct）视为有内容 schema，不注入。
		return false
	}
	if len(raw) == 0 {
		return true
	}
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if json.Unmarshal(raw, &schema) != nil {
		return false
	}
	return len(schema.Properties) == 0
}

// freeformInputSchema 是注入给 FREEFORM 工具的参数 schema。
// 让模型把自由格式内容（如 patch 文本）放进 input 字段。
const freeformInputSchema = `{"type":"object","properties":{"input":{"type":"string","description":"The freeform content (e.g. patch text). Do NOT wrap in JSON; put the raw content directly as the value of this field."}},"required":["input"]}`

// extractFreeformInput 从工具调用参数中提取 FREEFORM 工具的 input 字段值。
// 模型按注入的 schema 生成 {"input":"<content>"}，这里提取 input 字段并返回原始内容字符串。
// 若参数不是 {"input":...} 格式（如模型仍返回 {} 或其他），则返回空字符串。
func extractFreeformInput(arguments string) string {
	if arguments == "" {
		return ""
	}
	var obj struct {
		Input string `json:"input"`
	}
	if json.Unmarshal([]byte(arguments), &obj) == nil && obj.Input != "" {
		return obj.Input
	}
	return ""
}

// extractWebSearchQuery 从 web_search 工具调用参数里提取查询词。
// 模型按 web_search 的 function schema 生成 {"query":"<搜索内容>"}，
// 这里优先提取 query 字段；也兼容 {"input":...} 或裸字符串查询词。
func extractWebSearchQuery(arguments string) string {
	if arguments == "" {
		return ""
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal([]byte(arguments), &obj) == nil {
		// 标准字段优先：query / input。
		if q := rawToString(obj["query"]); q != "" {
			return q
		}
		if in := rawToString(obj["input"]); in != "" {
			return in
		}
		// 兜底：模型可能编造非标准字段（如 {"arg-a":"..."}）。
		// 取第一个非空字符串值作为查询词，避免把整个 JSON 字符串当 query。
		for _, v := range obj {
			if s := rawToString(v); s != "" {
				return s
			}
		}
	}
	var s string
	if json.Unmarshal([]byte(arguments), &s) == nil {
		return s
	}
	return arguments
}

// webSearchActionMap 构建 web_search_call item 的 action 字段。
// 标准 Responses 协议要求 web_search_call 用 action 而非 arguments：
// {"type":"search","query":"..."}。
func webSearchActionMap(arguments string) map[string]any {
	return map[string]any{
		"type":  "search",
		"query": extractWebSearchQuery(arguments),
	}
}

// isWebSearchTool 判断工具是否为 web_search（按 name 或 type 识别）。
func isWebSearchTool(name, toolType string) bool {
	return name == "web_search" || toolType == "web_search"
}

// normalizeToolArguments 把工具调用参数归一化为合法 JSON 字符串，供 Chat Completions
// 上游使用。Chat Completions 协议要求 function.arguments 必须是 JSON 字符串，但上游
// 模型（如 GLM-5.2）可能生成畸形工具调用——把自由文本（patch 内容等）直接塞进
// arguments（甚至塞错工具，如 exec_command 却带 patch 文本）。这些非 JSON 的 arguments
// 原样透传会让上游在 json.loads(arguments) 时报 400（"Expecting value: line 1 column 2"）。
//
// 规则：
//   - 已是合法 JSON：原样返回（避免二次包装/破坏正常调用）。
//   - 非 JSON（自由文本）：包装为 {"input":"<原文>"}，保证协议合法，模型下一轮可恢复。
func normalizeToolArguments(arguments string) string {
	trimmed := strings.TrimSpace(arguments)
	if trimmed == "" {
		return arguments
	}
	if json.Valid([]byte(arguments)) {
		return arguments
	}
	b, _ := json.Marshal(map[string]string{"input": arguments})
	return string(b)
}

// parseArgsToInput 把工具调用参数字符串解析为 JSON 对象存入 RawMessage（供 Anthropic
// 的 input 字段使用）；若参数不是 JSON 对象（如纯文本），则保留为字符串字面量。
func parseArgsToInput(args string) json.RawMessage {
	if args == "" {
		return nil
	}
	trimmed := strings.TrimSpace(args)
	if strings.HasPrefix(trimmed, "{") {
		var obj json.RawMessage
		if json.Unmarshal([]byte(args), &obj) == nil {
			return obj
		}
	}
	return marshalString(args)
}

// rawToArgsString 把 json.RawMessage 转回工具调用参数字符串。
// 若内容是 JSON 字符串字面量则解引用；否则返回压缩后的紧凑 JSON（对象/数组）。
func rawToArgsString(r json.RawMessage) string {
	if len(r) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(r, &s) == nil {
		return s
	}
	return string(compactJSON(r))
}

// compactJSON 把任意 JSON 值压缩为紧凑形式（去除空白），保证跨格式归一。
// 若输入不是合法 JSON，则原样返回。
func compactJSON(raw json.RawMessage) json.RawMessage {
	var buf bytes.Buffer
	if err := json.Compact(&buf, raw); err != nil {
		return raw
	}
	return json.RawMessage(buf.Bytes())
}
