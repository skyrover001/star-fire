# 多格式 API 支持设计方案

> 目标：不管 client 接入的上游模型服务是 **OpenAI Chat** / **Anthropic Messages** / **OpenAI Responses** 三种格式中的哪一种，server 端都对外提供这三种格式的调用方式。

## 一、总体架构

```
用户调用方
  ├─ POST /v1/chat/completions  (OpenAI Chat)
  ├─ POST /v1/messages          (Anthropic Messages)
  └─ POST /v1/responses         (OpenAI Responses)
        │
        ▼
┌──────────────────────────────────────────────┐
│ Server 端「格式适配层」                        │
│  ParseUserRequest(userFormat, body)          │
│      → CanonicalRequest                      │
│  鉴权/余额/限流/计费（基于 Canonical）          │
│  负载均衡选 client（读 client.UpstreamFormat） │
│  ConvertCanonicalToUpstream(upstreamFormat)   │
│      → 上游格式请求                            │
│  收到上游响应 → ConvertUpstreamToCanonical    │
│      → ConvertCanonicalToUser(userFormat)     │
│      → 返回给用户                              │
└──────────────────────────────────────────────┘
        │  WSMessage{ Content: <上游格式请求>, Format: <client上游格式> }
        ▼
Client 引擎层（每引擎一种上游格式，用官方 SDK）
  ├─ openai/     → sashabaranov/go-openai
  ├─ anthropic/  → anthropics/anthropic-sdk-go
  └─ responses/  → openai/openai-go
        │
        ▼
  上游模型服务
```

**核心原则**：server 与 client 之间传输的是**该 client 上游格式的请求**（不是 Canonical）。转换全部在 server 端完成。client 只负责"上游格式请求 → 上游调用 → 上游格式响应"。

> 这样 client 端几乎零转换逻辑，只做"格式透传 + 流式转发"，最简洁。

## 二、Canonical 模型（`pkg/public/canonical.go`）

三种格式的无损中间表示。**content 统一为 block 数组**，这是无损转换的关键。

```go
// 请求侧
type CanonicalRequest struct {
    Model       string
    Stream      bool
    System      []CanonicalContent      // 系统提示（Anthropic system / OpenAI system message）
    Messages    []CanonicalMessage
    Tools       []CanonicalTool
    MaxTokens   *int
    Temperature *float64
    TopP        *float64
    Thinking    json.RawMessage         // 透传 thinking 参数
    Extra       map[string]json.RawMessage // 无法映射的原始字段透传（保底无损）
}

type CanonicalMessage struct {
    Role       string               // system | user | assistant | tool
    Content    []CanonicalContent   // 统一 block 数组
    ToolCalls  []CanonicalToolCall  // assistant 的工具调用
    ToolCallID string               // tool 消息关联的 tool_use id
    Name       string
}

type CanonicalContent struct {
    Type      string               // text | image | tool_use | tool_result | thinking
    Text      string
    ImageURL  string
    // tool_use
    ID        string
    Name      string
    Input     json.RawMessage
    // tool_result
    Content   []CanonicalContent   // 嵌套内容
    IsError   bool
}

type CanonicalTool struct {
    Type        string             // function
    Name        string
    Description string
    Parameters  json.RawMessage    // JSON Schema
}

type CanonicalToolCall struct {
    ID        string
    Name      string
    Arguments json.RawMessage
}

// 响应侧
type CanonicalResponse struct {
    ID           string
    Content      []CanonicalContent
    Usage        CanonicalUsage
    FinishReason string
}

type CanonicalUsage struct {
    InputTokens  int
    OutputTokens int
    TotalTokens  int
    CachedTokens int
}

// 流式事件
type CanonicalStreamEvent struct {
    Type         string   // text_delta | tool_call_delta | done | error
    Text         string
    ToolCall     *CanonicalToolCall
    Usage        *CanonicalUsage
    FinishReason string
}
```

**无损保证**：`Extra map[string]json.RawMessage` 兜底——任何三种格式中无法映射到 Canonical 标准字段的原始字段，都原样塞进 `Extra`，转换回原格式时原样还原，确保不丢字段。

## 三、转换器设计（server 端 `internal/service/format/`）

每个转换器实现两个方向：

```go
// 接口
type Converter interface {
    // 用户格式 → Canonical
    ParseRequest(body []byte) (*CanonicalRequest, error)
    // Canonical → 用户格式（用于回传响应）
    BuildResponse(canonical *CanonicalResponse) ([]byte, error)
    // Canonical → 上游格式（用于发给 client）
    BuildUpstreamRequest(canonical *CanonicalRequest) ([]byte, error)
    // 上游格式响应 → Canonical
    ParseUpstreamResponse(data []byte) (*CanonicalResponse, error)
    // 流式：上游 SSE 事件 → Canonical 事件
    ParseUpstreamStreamEvent(line []byte) (*CanonicalStreamEvent, error)
    // 流式：Canonical 事件 → 用户格式 SSE 事件
    BuildUserStreamEvent(ev *CanonicalStreamEvent) ([]byte, error)
}
```

### 文件划分

```
internal/service/format/
  canonical.go      // Canonical 结构体
  converter.go      // Converter 接口 + 注册表
  openai.go         // OpenAI Chat ↔ Canonical
  anthropic.go      // Anthropic ↔ Canonical
  responses.go      // Responses ↔ Canonical
  adapter.go        // 统一入口（编排转换 + 转发）
  openai_test.go    // OpenAI 转换器单测（见 3.4）
  anthropic_test.go // Anthropic 转换器单测（见 3.4）
  responses_test.go // Responses 转换器单测（见 3.4）
```

### 各转换器方法与字段映射（详细）

> 三个转换器都实现同一 `Converter` 接口的 6 个方法。下面逐转换器给出**具体方法签名**、**逐字段映射表**、以及**必须通过的单元测试用例**。测试通过是进入下一阶段的门禁（见第十八章）。

#### 3.1 OpenAI Chat 转换器（`openai.go`）

```go
type OpenAIConverter struct{}

// 用户/上游请求 → Canonical（解析 openai.ChatCompletionRequest）
func (c *OpenAIConverter) ParseRequest(body []byte) (*CanonicalRequest, error)

// Canonical → 上游请求（生成 openai.ChatCompletionRequest）
func (c *OpenAIConverter) BuildUpstreamRequest(cr *CanonicalRequest) ([]byte, error)

// 上游响应 → Canonical（解析 openai.ChatCompletionResponse）
func (c *OpenAIConverter) ParseUpstreamResponse(data []byte) (*CanonicalResponse, error)

// Canonical → 用户响应（生成 openai.ChatCompletionResponse）
func (c *OpenAIConverter) BuildResponse(cr *CanonicalResponse) ([]byte, error)

// 上游流式 chunk → Canonical 事件（解析 ChatCompletionStreamResponse，含 `[DONE]`）
func (c *OpenAIConverter) ParseUpstreamStreamEvent(line []byte) (*CanonicalStreamEvent, error)

// Canonical 事件 → 用户流式 chunk（生成 ChatCompletionStreamResponse）
func (c *OpenAIConverter) BuildUserStreamEvent(ev *CanonicalStreamEvent) ([]byte, error)
```

**请求字段映射（ChatCompletionRequest → CanonicalRequest）**

| 用户/上游字段 | Canonical 字段 | 说明 |
|---|---|---|
| `model` | `Model` | 直接复制 |
| `stream` | `Stream` | 直接复制 |
| `messages[].role` | `Messages[].Role` | `system`/`user`/`assistant`/`tool` 原样 |
| `messages[].content`（string） | `Messages[].Content` = `[{type:text,text:...}]` | **string → 单元素 text block** |
| `messages[].content`（array） | `Messages[].Content` | `{type:text}` → text block；`{type:image_url}` → image block（`image_url.url` → `ImageURL`） |
| `messages[].tool_calls` | `Messages[].ToolCalls` | `id`→`ID`、`function.name`→`Name`、`function.arguments`→`Arguments`（保留原始 JSON 字符串） |
| `messages[].tool_call_id`（`role=tool`） | `Messages[].ToolCallID` | 关联 assistant 的工具调用 |
| `messages[].name` | `Messages[].Name` | 函数名 |
| `tools[]` | `Tools[]` | `function.name`→`Name`、`function.description`→`Description`、`function.parameters`→`Parameters` |
| `max_tokens` | `MaxTokens` | `*int` |
| `temperature` / `top_p` | `Temperature` / `TopP` | `*float64` |
| `reasoning_effort` | `Thinking`（或 `Extra`） | 推理强度，走 `Extra` 保底无损 |
| `stop` / `n` / `seed` / `logprobs` 等 | `Extra` | 无法映射字段 → `Extra` 透传 |

**响应字段映射（ChatCompletionResponse ↔ CanonicalResponse）**

| 上游字段 | Canonical 字段 | 说明 |
|---|---|---|
| `id` | `ID` | 直接复制 |
| `choices[0].message.content` | `Content` | string → text block |
| `choices[0].message.tool_calls` | `Content`（`tool_use` block）+ `ToolCalls` | 工具调用转为 content 块 |
| `choices[0].finish_reason` | `FinishReason` | `stop`/`length`/`tool_calls` 原样 |
| `usage` | `Usage` | 见下方 usage 映射 |
| `error` | error | 非 2xx 时返回错误，不进 Canonical |

**usage 映射**：`prompt_tokens`→`InputTokens`、`completion_tokens`→`OutputTokens`、`total_tokens`→`TotalTokens`、`prompt_tokens_details.cached_tokens`→`CachedTokens`。

**流式事件映射**

| 上游 chunk | Canonical 事件 | 说明 |
|---|---|---|
| `choices[0].delta.content` | `text_delta`（`Text`） | 文本增量 |
| `choices[0].delta.tool_calls` | `tool_call_delta`（`ToolCall`） | 参数分片需**累积拼接**（见状态机） |
| 末尾 `usage` chunk | `done`（`Usage`） | `stream_options.include_usage=true` 时 |
| 字面量 `[DONE]` | `done` | 流终止 |
| `choices[0].finish_reason` | `done`（`FinishReason`） | 结束标记 |

**流式状态机**：OpenAI 的 `tool_calls` 参数在多个 chunk 中**分片返回**（`function.arguments` 逐段追加），转换器需维护 `map[index]*CanonicalToolCall` 按 `tool_calls[].index` 累积，`finish_reason` 出现时清空状态。

#### 3.2 Anthropic 转换器（`anthropic.go`）

```go
type AnthropicConverter struct{}

func (c *AnthropicConverter) ParseRequest(body []byte) (*CanonicalRequest, error)         // 解析 anthropic.MessageCreateParams
func (c *AnthropicConverter) BuildUpstreamRequest(cr *CanonicalRequest) ([]byte, error)    // 生成 MessageCreateParams
func (c *AnthropicConverter) ParseUpstreamResponse(data []byte) (*CanonicalResponse, error) // 解析 anthropic.Message
func (c *AnthropicConverter) BuildResponse(cr *CanonicalResponse) ([]byte, error)          // 生成 anthropic.Message
func (c *AnthropicConverter) ParseUpstreamStreamEvent(line []byte) (*CanonicalStreamEvent, error) // 解析 anthropic 流式事件
func (c *AnthropicConverter) BuildUserStreamEvent(ev *CanonicalStreamEvent) ([]byte, error) // 生成 anthropic 流式事件
```

**请求字段映射（MessageCreateParams → CanonicalRequest）**

| 上游字段 | Canonical 字段 | 说明 |
|---|---|---|
| `model` | `Model` | 直接复制 |
| `stream` | `Stream` | 直接复制 |
| `system`（string） | `System` = `[{type:text,text:...}]` | string → 单元素 text block |
| `system`（array） | `System` | block 数组直接映射 |
| `messages[].role` | `Messages[].Role` | `user`/`assistant` 原样（Anthropic 无 system/tool 顶层 role） |
| `messages[].content[]` | `Messages[].Content` | `text`/`image`/`tool_use`/`tool_result`/`thinking` 逐块映射 |
| content block `tool_use` | `Content` 块（`type=tool_use`）+ `ToolCalls` | `id`→`ID`、`name`→`Name`、`input`→`Input` |
| content block `tool_result` | `Content` 块（`type=tool_result`） | `tool_use_id`→`ID`、`content`→`Content`、`is_error`→`IsError` |
| `max_tokens`（必填） | `MaxTokens` | **缺失时给默认值 4096**（见风险 1） |
| `temperature` / `top_p` | `Temperature` / `TopP` | `*float64` |
| `thinking` | `Thinking` | 透传 |
| `tools[]` | `Tools[]` | `name`/`description`/`input_schema`→`Parameters` |
| `stop_sequences` / `metadata` | `Extra` | 透传 |

**响应字段映射（anthropic.Message ↔ CanonicalResponse）**

| 上游字段 | Canonical 字段 | 说明 |
|---|---|---|
| `id` | `ID` | 直接复制 |
| `content[]`（`text`） | `Content`（text block） | |
| `content[]`（`tool_use`） | `Content`（tool_use block）+ `ToolCalls` | |
| `content[]`（`thinking`） | `Content`（thinking block） | |
| `stop_reason` | `FinishReason` | `end_turn`/`max_tokens`/`tool_use`/`stop_sequence` |
| `usage` | `Usage` | 见下方 usage 映射 |

**usage 映射**：`input_tokens`→`InputTokens`、`output_tokens`→`OutputTokens`、`cache_read_input_tokens`→`CachedTokens`、`cache_creation_input_tokens`→`CachedTokens`（写入缓存，计入 `CachedTokens` 或单独字段）。

**流式事件映射（需维护 block 索引状态）**

| 上游事件 | Canonical 事件 | 说明 |
|---|---|---|
| `message_start` | （记录 `usage.input_tokens`） | 携带输入 usage |
| `content_block_start` | （记录 block `index`+`type`） | 不直接产出事件，仅建状态 |
| `content_block_delta`（`text_delta`） | `text_delta` | `text`→`Text` |
| `content_block_delta`（`input_json_delta`） | `tool_call_delta` | tool 参数分片，按 index 累积 |
| `content_block_stop` | （结束当前 block） | 清理对应 index 状态 |
| `message_delta` | （记录 `usage.output_tokens` + `stop_reason`） | 输出 usage + 结束原因 |
| `message_stop` | `done` | 汇总 usage → `done`（`Usage`） |
| `error` | `error` | 错误事件 |

**流式状态机**：Anthropic 流式按 `content_block_start`（含 `index`）→ 多个 `content_block_delta` → `content_block_stop` 组织，转换器需维护 `map[index]{type, 累积内容}`，`message_stop` 时输出 `done` 并携带完整 usage。

#### 3.3 Responses 转换器（`responses.go`）

```go
type ResponsesConverter struct{}

func (c *ResponsesConverter) ParseRequest(body []byte) (*CanonicalRequest, error)         // 解析 openai-go ResponsesNewParams
func (c *ResponsesConverter) BuildUpstreamRequest(cr *CanonicalRequest) ([]byte, error)    // 生成 ResponsesNewParams
func (c *ResponsesConverter) ParseUpstreamResponse(data []byte) (*CanonicalResponse, error) // 解析 openai-go Response
func (c *ResponsesConverter) BuildResponse(cr *CanonicalResponse) ([]byte, error)          // 生成 Response
func (c *ResponsesConverter) ParseUpstreamStreamEvent(line []byte) (*CanonicalStreamEvent, error) // 解析 responses 流式事件
func (c *ResponsesConverter) BuildUserStreamEvent(ev *CanonicalStreamEvent) ([]byte, error) // 生成 responses 流式事件
```

**请求字段映射（ResponsesNewParams → CanonicalRequest）**

| 上游字段 | Canonical 字段 | 说明 |
|---|---|---|
| `model` | `Model` | 直接复制 |
| `stream` | `Stream` | 直接复制 |
| `input[]`（`{type:message}`） | `Messages[]` | `role` + `content`（`input_text`/`input_image`） |
| `input[]`（`{type:function_call}`） | `Messages[].ToolCalls` | assistant 侧工具调用 |
| `input[]`（`{type:function_call_output}`） | `Messages[]`（`role=tool`） | `call_id`→`ToolCallID`、`output`→`Content` |
| `instructions` | `System` | 顶层系统指令 → text block |
| `tools[]` | `Tools[]` | `name`/`description`/`parameters` |
| `max_output_tokens` | `MaxTokens` | |
| `temperature` / `top_p` | `Temperature` / `TopP` | |
| `reasoning` | `Thinking`（或 `Extra`） | 推理配置透传 |
| `metadata` / `truncation` 等 | `Extra` | 透传 |

**响应字段映射（openai-go Response ↔ CanonicalResponse）**

| 上游字段 | Canonical 字段 | 说明 |
|---|---|---|
| `id` | `ID` | 直接复制 |
| `output[]`（`{type:message}`） | `Content`（text block） | `content[].text` → text |
| `output[]`（`{type:function_call}`） | `Content`（tool_use block）+ `ToolCalls` | |
| `output[]`（`{type:reasoning}`） | `Content`（thinking block） | |
| `status` | `FinishReason` | `completed`→`stop`、`incomplete`→`length` |
| `usage` | `Usage` | 见下方 usage 映射 |

**usage 映射**：`input_tokens`→`InputTokens`、`output_tokens`→`OutputTokens`、`total_tokens`→`TotalTokens`、`input_tokens_details.cached_tokens`→`CachedTokens`。

**流式事件映射**

| 上游事件 | Canonical 事件 | 说明 |
|---|---|---|
| `response.output_text.delta` | `text_delta` | `delta`→`Text` |
| `response.function_call_arguments.delta` | `tool_call_delta` | 参数分片，按 `item_id` 累积 |
| `response.completed` | `done`（含完整 `response`） | 从 `response.usage` 提取 usage |
| `response.failed` | `error` | 失败事件 |
| `response.output_item.done` | （辅助结束 tool block） | 按 `item_id` 清理状态 |

**流式状态机**：Responses 的 function_call 参数按 `item_id` 分片（`response.function_call_arguments.delta`），转换器维护 `map[item_id]*CanonicalToolCall` 累积，`response.completed` 时输出 `done` 并从事件内嵌的 `response` 对象提取 usage。

#### 3.4 转换器单元测试（必须通过）

> 每个转换器独立成表驱动测试文件（`openai_test.go` / `anthropic_test.go` / `responses_test.go`），下述用例**全绿**才视为该转换器完成。测试原则：**双向 round-trip**（`Parse` 后 `Build` 再 `Parse`，断言字段等价）与**已知输入 → 已知输出**（黄金样本）结合。

| 类别 | 测试用例 | 断言要点 |
|---|---|---|
| **基础请求** | 纯文本单轮 / 多轮 / 空 content / content 为 null | Role、Content 块类型与数量、空 content 不 panic 且正确标记 |
| **content 形态** | content 为 string / content 为 block 数组 / 多模态（text+image） | string 与数组产出等价 block；image 的 URL 正确映射 |
| **system 提取** | OpenAI `messages[0].role=system` / Anthropic 顶层 `system` / Responses `instructions` | 三者都能正确落到 `CanonicalRequest.System` |
| **工具调用（请求）** | 各格式 tool 定义 → `Tools[]`；assistant `tool_calls` → `ToolCalls`；`role=tool`/`tool_result` → `ToolCallID` | Name/Description/Parameters（JSON Schema）无损；ID 关联正确 |
| **工具调用（响应）** | 各格式响应中的 tool_use/function_call → `Content` 块 + `ToolCalls` | 参数 JSON 字符串/对象双向一致 |
| **参数传递** | `max_tokens`/`max_output_tokens`、`temperature`、`top_p` 的 nil 与非 nil | 指针 nil 时字段缺省；非 nil 时值一致 |
| **Anthropic `max_tokens` 补全** | Canonical 无 `MaxTokens` → Build anthropic 请求 | 自动给默认 4096，不 panic |
| **Extra 无损** | 注入未知字段（如 `stop`、`seed`、`metadata`） | round-trip 后未知字段仍在 `Extra` 且可还原 |
| **响应 usage** | 各格式 usage（含 `cached_tokens`/`cache_read_input_tokens`/`input_tokens_details.cached_tokens`） | 四类 token 计数正确、缓存命中识别正确 |
| **非流式响应** | 各格式完整响应 → Canonical → 各格式 | ID/Content/FinishReason/Usage 全字段等价 |
| **流式文本** | 文本分片序列 → `text_delta` 事件序列 | 文本按序拼接完整、无丢失/重复 |
| **流式工具调用** | tool 参数分片（OpenAI `index` / Anthropic `index` / Responses `item_id`） | 分片按 id 正确累积，最终 arguments 完整 |
| **流式终止** | `[DONE]` / `message_stop` / `response.completed` | 都映射为 `done`，usage 与 finish_reason 正确提取 |
| **流式错误** | 上游 `error` 事件 / `response.failed` | 映射为 `error`，不 panic |
| **空/异常输入** | 空 body、非法 JSON、缺 `model`、缺 `messages`/`input` | 返回明确 error，不 panic、不产生半成品 Canonical |
| **状态机隔离** | 同一转换器并发/顺序处理多路流 | 每个流的累积状态互不串扰（用独立状态对象或 map 按 id 隔离） |

**测试通过判定**：上表每一行至少一个用例通过；`go test ./internal/service/format/...` 全绿；覆盖率对 6 个方法核心分支 ≥ 80%。这些单测是第十八章 P1/P2 门禁的组成部分。

#### 3.5 黄金样本（Golden Samples）

> 统一用「天气查询工具调用」这一场景，给出三种格式的请求/响应/流式 JSON，以及它们转换后得到的**同一份** Canonical。测试代码直接把这些 JSON 作为"已知输入 → 已知输出"的断言依据（表驱动用例的 `input` 与 `want`）。

##### (1) 请求黄金样本：三种格式 → 同一 `CanonicalRequest`

场景：系统提示「你是天气预报助手」，用户问「北京今天天气怎么样？」，模型调用 `get_weather(city="北京")`，用户回传工具结果「北京今天晴，气温 25°C」。

**① OpenAI Chat 请求：**

```json
{
  "model": "gpt-4o",
  "stream": false,
  "messages": [
    { "role": "system", "content": "你是天气预报助手" },
    { "role": "user", "content": "北京今天天气怎么样？" },
    {
      "role": "assistant",
      "content": null,
      "tool_calls": [
        { "id": "call_1", "type": "function", "function": { "name": "get_weather", "arguments": "{\"city\":\"北京\"}" } }
      ]
    },
    { "role": "tool", "tool_call_id": "call_1", "content": "北京今天晴，气温 25°C" }
  ],
  "tools": [
    {
      "type": "function",
      "function": {
        "name": "get_weather",
        "description": "获取指定城市的天气",
        "parameters": { "type": "object", "properties": { "city": { "type": "string" } }, "required": ["city"] }
      }
    }
  ],
  "max_tokens": 1024,
  "temperature": 0.7
}
```

**② Anthropic Messages 请求：**

```json
{
  "model": "claude-3-5-sonnet",
  "max_tokens": 1024,
  "system": "你是天气预报助手",
  "messages": [
    { "role": "user", "content": [ { "type": "text", "text": "北京今天天气怎么样？" } ] },
    {
      "role": "assistant",
      "content": [ { "type": "tool_use", "id": "toolu_1", "name": "get_weather", "input": { "city": "北京" } } ]
    },
    {
      "role": "user",
      "content": [ { "type": "tool_result", "tool_use_id": "toolu_1", "content": "北京今天晴，气温 25°C" } ]
    }
  ],
  "tools": [
    {
      "name": "get_weather",
      "description": "获取指定城市的天气",
      "input_schema": { "type": "object", "properties": { "city": { "type": "string" } }, "required": ["city"] }
    }
  ]
}
```

**③ OpenAI Responses 请求：**

```json
{
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
}
```

**④ 三者转换后得到同一份 `CanonicalRequest`（断言目标）：**

```json
{
  "model": "gpt-4o",
  "stream": false,
  "system": [ { "type": "text", "text": "你是天气预报助手" } ],
  "messages": [
    { "role": "user", "content": [ { "type": "text", "text": "北京今天天气怎么样？" } ] },
    {
      "role": "assistant",
      "content": [ { "type": "tool_use", "id": "call_1", "name": "get_weather", "input": { "city": "北京" } } ],
      "tool_calls": [ { "id": "call_1", "name": "get_weather", "arguments": "{\"city\":\"北京\"}" } ]
    },
    {
      "role": "tool",
      "content": [ { "type": "tool_result", "id": "call_1", "content": [ { "type": "text", "text": "北京今天晴，气温 25°C" } ] } ],
      "tool_call_id": "call_1"
    }
  ],
  "tools": [
    {
      "type": "function",
      "name": "get_weather",
      "description": "获取指定城市的天气",
      "parameters": { "type": "object", "properties": { "city": { "type": "string" } }, "required": ["city"] }
    }
  ],
  "max_tokens": 1024,
  "temperature": 0.7
}
```

> 关键差异被 Canonical 归一：
> - **工具调用 ID**：OpenAI `call_1` / Anthropic `toolu_1` / Responses `call_1` → `Content.ID` 与 `ToolCalls.ID`。
> - **工具参数**：OpenAI/Responses 的字符串 `arguments` 与 Anthropic 的对象 `input` **同时保留**（`ToolCalls.Arguments` 存字符串、`Content.Input` 存对象），无损且可互转。
> - **工具结果**：OpenAI `role=tool`+`tool_call_id` / Anthropic `role=user`+`tool_result` / Responses `function_call_output` → `role=tool` + `ToolCallID` + `tool_result` 块。

##### (2) 响应黄金样本：三种格式 → 同一 `CanonicalResponse`

**① OpenAI Chat 响应：**

```json
{
  "id": "chatcmpl-123",
  "object": "chat.completion",
  "choices": [
    { "index": 0, "message": { "role": "assistant", "content": "北京今天晴，气温 25°C。" }, "finish_reason": "stop" }
  ],
  "usage": {
    "prompt_tokens": 120,
    "completion_tokens": 15,
    "total_tokens": 135,
    "prompt_tokens_details": { "cached_tokens": 40 }
  }
}
```

**② Anthropic Messages 响应：**

```json
{
  "id": "msg_123",
  "type": "message",
  "role": "assistant",
  "content": [ { "type": "text", "text": "北京今天晴，气温 25°C。" } ],
  "stop_reason": "end_turn",
  "usage": { "input_tokens": 120, "output_tokens": 15, "cache_read_input_tokens": 40 }
}
```

**③ OpenAI Responses 响应：**

```json
{
  "id": "resp_123",
  "object": "response",
  "status": "completed",
  "output": [
    { "type": "message", "role": "assistant", "content": [ { "type": "output_text", "text": "北京今天晴，气温 25°C。" } ] }
  ],
  "usage": { "input_tokens": 120, "output_tokens": 15, "total_tokens": 135, "input_tokens_details": { "cached_tokens": 40 } }
}
```

**④ 三者转换后得到同一份 `CanonicalResponse`（断言目标）：**

```json
{
  "id": "chatcmpl-123",
  "content": [ { "type": "text", "text": "北京今天晴，气温 25°C。" } ],
  "finish_reason": "stop",
  "usage": { "input_tokens": 120, "output_tokens": 15, "total_tokens": 135, "cached_tokens": 40 }
}
```

> 注意：`stop_reason=end_turn`（Anthropic）与 `status=completed`（Responses）都归一为 `FinishReason=stop`；缓存命中字段三种写法都归一为 `CachedTokens=40`。

##### (3) 流式黄金样本：三种格式 SSE → 同一 `CanonicalStreamEvent` 序列

文本分片「北京」「今天」，结束带 usage。

**① OpenAI Chat 流式（每行一个 `data:` 帧）：**

```
data: {"choices":[{"index":0,"delta":{"content":"北京"},"finish_reason":null}]}
data: {"choices":[{"index":0,"delta":{"content":"今天"},"finish_reason":null}]}
data: {"choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":120,"completion_tokens":15,"total_tokens":135}}
data: [DONE]
```

**② Anthropic Messages 流式（`event:` + `data:` 帧）：**

```
event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"北京"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"今天"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":15}}

event: message_stop
data: {"type":"message_stop"}
```

**③ OpenAI Responses 流式：**

```
event: response.output_text.delta
data: {"type":"response.output_text.delta","item_id":"msg_123","output_index":0,"content_index":0,"delta":"北京"}

event: response.output_text.delta
data: {"type":"response.output_text.delta","item_id":"msg_123","output_index":0,"content_index":0,"delta":"今天"}

event: response.completed
data: {"type":"response.completed","response":{"id":"resp_123","status":"completed","usage":{"input_tokens":120,"output_tokens":15,"total_tokens":135}}}
```

**④ 三者转换后得到同一份 `CanonicalStreamEvent` 序列（断言目标）：**

```json
{ "type": "text_delta", "text": "北京" }
{ "type": "text_delta", "text": "今天" }
{ "type": "done", "finish_reason": "stop", "usage": { "input_tokens": 120, "output_tokens": 15, "total_tokens": 135, "cached_tokens": 0 } }
```

> 注意：OpenAI 的 usage 在 `finish_reason` chunk、Anthropic 在 `message_delta`、Responses 在 `response.completed`，三种位置的 usage 最终都汇总进同一个 `done` 事件。单测断言：三个转换器对各自输入产出的 `CanonicalStreamEvent` 序列完全相等。

##### (4) 流式工具调用分片黄金样本：参数分片累积

场景：模型流式调用 `get_weather`，参数 `{"city":"北京"}` 被拆成多片逐段返回。这是**流式转换最易错**的部分（需要按 `index`/`item_id` 累积拼接），必须独立黄金样本。

**① OpenAI Chat 工具调用分片（按 `tool_calls[].index` 累积）：**

```
data: {"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"get_weather","arguments":""}}]},"finish_reason":null}]}
data: {"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"city\":"}}]},"finish_reason":null}]}
data: {"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"北京\"}"}}]},"finish_reason":null}]}
data: {"choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}
data: [DONE]
```

> 首帧带 `id`/`name`，后续帧只带 `arguments` 分片，按 `index=0` 累积。转换器需维护 `map[0]*CanonicalToolCall`。

**② Anthropic 工具调用分片（按 content block `index` 累积）：**

```
event: content_block_start
data: {"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_1","name":"get_weather","input":{}}}

event: content_block_delta
data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"city\":"}}

event: content_block_delta
data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"\"北京\"}"}}

event: content_block_stop
data: {"type":"content_block_stop","index":1}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":12}}

event: message_stop
data: {"type":"message_stop"}
```

> `content_block_start` 建状态（含 `id`/`name`），`input_json_delta` 的 `partial_json` 逐段累积，`content_block_stop` 结束该 block。

**③ OpenAI Responses 工具调用分片（按 `item_id` 累积）：**

```
event: response.output_item.added
data: {"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","id":"fc_1","call_id":"call_1","name":"get_weather","arguments":""}}

event: response.function_call_arguments.delta
data: {"type":"response.function_call_arguments.delta","item_id":"fc_1","output_index":0,"delta":"{\"city\":"}

event: response.function_call_arguments.delta
data: {"type":"response.function_call_arguments.delta","item_id":"fc_1","output_index":0,"delta":"\"北京\"}"}

event: response.function_call_arguments.done
data: {"type":"response.function_call_arguments.done","item_id":"fc_1","output_index":0,"arguments":"{\"city\":\"北京\"}"}

event: response.completed
data: {"type":"response.completed","response":{"id":"resp_123","status":"completed","usage":{"input_tokens":80,"output_tokens":12,"total_tokens":92}}}
```

> `response.output_item.added` 建状态，`function_call_arguments.delta` 按 `item_id` 累积，`function_call_arguments.done` 给出完整 arguments（可做一致性校验）。

**④ 三者转换后得到同一份 `CanonicalStreamEvent` 序列（断言目标）：**

```json
{ "type": "tool_call_delta", "tool_call": { "id": "call_1", "name": "get_weather", "arguments": "" } }
{ "type": "tool_call_delta", "tool_call": { "id": "", "name": "", "arguments": "{\"city\":" } }
{ "type": "tool_call_delta", "tool_call": { "id": "", "name": "", "arguments": "\"北京\"}" } }
{ "type": "done", "finish_reason": "tool_calls", "usage": { "input_tokens": 80, "output_tokens": 12, "total_tokens": 92, "cached_tokens": 0 } }
```

> 关键断言：
> - **首帧带全量元信息**（`id`/`name`），后续帧只带 `arguments` 分片（`id`/`name` 为空）。
> - 三份输入的累积结果一致：拼接后 `arguments` = `{"city":"北京"}`。
> - **finish_reason 归一**：OpenAI `tool_calls` / Anthropic `tool_use` / Responses `status=completed`（含 tool 输出）都归一为 `tool_calls`。
> - 额外单测：只给分片不给首帧（缺 `id`/`name`）→ 仍能累积但不产出孤儿工具调用；只给首帧不给分片 → arguments 为空字符串；**多工具并行**（`index=0` 与 `index=1` 交错）→ 两路独立累积互不串扰。

##### (5) 错误响应黄金样本：三种格式 error → 统一错误结构

上游错误也要做格式转换（第十章），需独立黄金样本保证错误字段不丢。

**① OpenAI Chat 错误：**

```json
{ "error": { "message": "模型负载过高，请重试", "type": "server_error", "code": "overloaded" } }
```

**② Anthropic 错误：**

```json
{ "type": "error", "error": { "type": "overloaded_error", "message": "模型负载过高，请重试" } }
```

**③ OpenAI Responses 错误：**

```json
{ "error": { "code": "overloaded", "message": "模型负载过高，请重试", "type": "server_error" } }
```

**④ 三者转换后得到统一错误结构（断言目标，可复用现有 `isClientRequestError` 逻辑）：**

```json
{ "type": "error", "code": "overloaded", "message": "模型负载过高，请重试", "retryable": true }
```

> 关键断言：
> - **code 归一**：OpenAI `code` / Anthropic `error.type` / Responses `code` → 统一 `code`（`overloaded`/`overloaded_error` 归一为 `overloaded`）。
> - **message 归一**：三种格式的 `message` 字段都落到 `message`。
> - **retryable 判定**：`server_error` / `overloaded` / 5xx 类 → `retryable=true`；`invalid_request_error` / 4xx 类 → `retryable=false`（对应现有 `isClientRequestError`）。
> - 错误**不进入** `CanonicalResponse`，而是单独的错误通道，避免与正常响应混淆。

##### (6) 多模态（图片）黄金样本：三种格式 image block → 同一 Canonical

场景：用户上传一张图片并问「这张图里有什么？」，图片以 base64 data URL 传入。

**① OpenAI Chat 请求（`image_url` block）：**

```json
{
  "model": "gpt-4o",
  "messages": [
    {
      "role": "user",
      "content": [
        { "type": "text", "text": "这张图里有什么？" },
        { "type": "image_url", "image_url": { "url": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==", "detail": "high" } }
      ]
    }
  ]
}
```

**② Anthropic 请求（`image` block，base64 需拆分 `media_type` 与 `data`）：**

```json
{
  "model": "claude-3-5-sonnet",
  "max_tokens": 1024,
  "messages": [
    {
      "role": "user",
      "content": [
        { "type": "text", "text": "这张图里有什么？" },
        {
          "type": "image",
          "source": { "type": "base64", "media_type": "image/png", "data": "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==" }
        }
      ]
    }
  ]
}
```

**③ OpenAI Responses 请求（`input_image` block）：**

```json
{
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
}
```

**④ 三者转换后得到同一份 `CanonicalRequest`（断言目标）：**

```json
{
  "model": "gpt-4o",
  "stream": false,
  "messages": [
    {
      "role": "user",
      "content": [
        { "type": "text", "text": "这张图里有什么？" },
        { "type": "image", "image_url": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==" }
      ]
    }
  ]
}
```

> 关键断言：
> - **URL 归一**：OpenAI `image_url.url`（含 `detail`）/ Responses `image_url` 字符串 → `ImageURL`；Anthropic 的 `source` 对象（`media_type` + `data`）**重新拼回** data URL（`data:{media_type};base64,{data}`）。
> - **`detail` 字段**：OpenAI 的 `detail:"high"` 属 OpenAI 特有，无法映射到 Anthropic/Responses，应放入 `Extra`（`image_detail`）保底；转回 OpenAI 时还原。
> - **base64 原样透传**：图片数据不解析、不裁剪、不重编码，字符串完全一致（对比图片字节级无损）。
> - 边界单测：`image_url` 为 http(s) URL（非 data URL）→ 原样透传不拆；Anthropic `source.type` 非 `base64`（如 `url`）→ 走 `Extra`。

##### (7) thinking / 推理内容黄金样本：三种格式推理 → 同一 Canonical

场景：模型先输出推理过程，再输出最终答案。三种格式对「推理内容」的表达差异最大（风险 4），需独立黄金样本。

**① OpenAI Chat 请求（`reasoning_effort`）+ 响应（无独立推理块，靠 `reasoning_effort` 控制）：**

请求：
```json
{ "model": "o3", "messages": [ { "role": "user", "content": "1+1=?" } ], "reasoning_effort": "high" }
```

响应（部分后端会额外返回 `reasoning` 字段，需透传）：
```json
{
  "id": "chatcmpl-456",
  "choices": [
    { "index": 0, "message": { "role": "assistant", "content": "答案是 2", "reasoning": "先计算 1+1，得到 2" }, "finish_reason": "stop" }
  ],
  "usage": { "prompt_tokens": 10, "completion_tokens": 30, "total_tokens": 40 }
}
```

**② Anthropic 请求（`thinking` block）+ 响应（`thinking` content block）：**

请求：
```json
{
  "model": "claude-3-7-sonnet",
  "max_tokens": 1024,
  "thinking": { "type": "enabled", "budget_tokens": 2048 },
  "messages": [ { "role": "user", "content": "1+1=?" } ]
}
```

响应（`thinking` block 在 `content[]` 内，且带 `signature`）：
```json
{
  "id": "msg_456",
  "content": [
    { "type": "thinking", "thinking": "先计算 1+1，得到 2", "signature": "EjY9..." },
    { "type": "text", "text": "答案是 2" }
  ],
  "stop_reason": "end_turn",
  "usage": { "input_tokens": 10, "output_tokens": 30 }
}
```

**③ OpenAI Responses 请求（`reasoning`）+ 响应（`reasoning` output item）：**

请求：
```json
{
  "model": "o3",
  "input": [ { "role": "user", "content": [ { "type": "input_text", "text": "1+1=?" } ] } ],
  "reasoning": { "effort": "high" }
}
```

响应（`reasoning` 作为独立 output item）：
```json
{
  "id": "resp_456",
  "status": "completed",
  "output": [
    { "type": "reasoning", "summary": [ { "type": "summary_text", "text": "先计算 1+1，得到 2" } ] },
    { "type": "message", "role": "assistant", "content": [ { "type": "output_text", "text": "答案是 2" } ] }
  ],
  "usage": { "input_tokens": 10, "output_tokens": 30, "total_tokens": 40 }
}
```

**④ 三者转换后得到同一份 `CanonicalRequest`/`CanonicalResponse`（断言目标）：**

请求（推理参数统一存 `Thinking`，无法映射的细节存 `Extra`）：
```json
{
  "model": "o3",
  "stream": false,
  "messages": [ { "role": "user", "content": [ { "type": "text", "text": "1+1=?" } ] } ],
  "thinking": { "type": "enabled", "budget_tokens": 2048 },
  "extra": { "reasoning_effort": "high", "responses_reasoning": { "effort": "high" } }
}
```

响应（推理内容归一为 `thinking` block，附加元信息存 `Extra`）：
```json
{
  "id": "chatcmpl-456",
  "content": [
    { "type": "thinking", "text": "先计算 1+1，得到 2", "extra": { "signature": "EjY9..." } },
    { "type": "text", "text": "答案是 2" }
  ],
  "finish_reason": "stop",
  "usage": { "input_tokens": 10, "output_tokens": 30, "total_tokens": 40, "cached_tokens": 0 }
}
```

> 关键断言：
> - **请求侧推理参数**：`reasoning_effort`（OpenAI）/ `thinking`（Anthropic）/ `reasoning`（Responses）→ 统一存 `CanonicalRequest.Thinking`（用 Anthropic 的 `{type, budget_tokens}` 为基准），其余细节（`effort` 值）进 `Extra`。
> - **响应侧推理内容**：OpenAI `message.reasoning`（非标准字段）/ Anthropic `thinking` block / Responses `reasoning` item → 统一 `Content` 中的 `type=thinking` 块。
> - **`signature` 保留**：Anthropic 的 `thinking.signature` 是**强制回传字段**（若被丢弃，Claude 会拒绝后续请求），必须进 `Extra` 并 round-trip 还原。
> - **推理内容不计入正文**：`thinking` 块与 `text` 块严格分离，正文拼接只取 `text`，避免推理泄漏到最终答案。
> - 边界单测：上游**不支持** thinking（如普通 openai 模型）→ 转换时静默丢弃 thinking 块或降级（只保留最终 text）；`thinking` 参数缺失 `budget_tokens` → 用默认值。

## 四、Server 端适配层（`internal/service/format/adapter.go`）

```go
func HandleChatRequest(c *gin.Context, server *models.Server, userFormat string) {
    // 1. 按 userFormat 解析
    conv := GetConverter(userFormat)
    canonical, err := conv.ParseRequest(readBody(c))

    // 2. 复用现有鉴权/余额/限流（基于 canonical.Model / EstimateTokens(canonical)）
    //    handleChatWithRetry 泛化为基于 CanonicalRequest

    // 3. 负载均衡选 client（按 canonical.Model）
    client := server.LoadBalanceExcluding(canonical.Model, userID, failedClients)
    upstreamFormat := client.UpstreamFormat   // 关键：读 client 上游格式

    // 4. Canonical → 上游格式
    upstreamConv := GetConverter(upstreamFormat)
    upstreamBody := upstreamConv.BuildUpstreamRequest(canonical)

    // 5. 发给 client（携带上游格式）
    WSMessage{ Type: MESSAGE, Content: upstreamBody, Format: upstreamFormat }

    // 6. 收到上游响应 → Canonical → 用户格式
    //    非流式：ParseUpstreamResponse → BuildResponse
    //    流式：逐事件 ParseUpstreamStreamEvent → BuildUserStreamEvent → SSE 写出
}
```

**关键**：`handleChatWithRetry` 目前强依赖 `request.Model`/`request.Messages`（`EstimateTokens`）。需要泛化为基于 `CanonicalRequest`，或让 `CanonicalRequest` 提供 `GetModel()`/`EstimateTokens()` 方法。

## 五、上游格式声明机制（两者结合）

### 1. Client 配置声明（`client/internal/config/config.go`）

```go
type ProxyBackend struct {
    Name    string `json:"name"`
    BaseURL string `json:"base_url"`
    APIKey  string `json:"api_key"`
    Format  string `json:"format"` // openai | anthropic | responses，默认 openai
    Enabled bool   `json:"enabled"`
}
```

`buildProxyEngines` 按 `Format` 创建对应引擎（openai/anthropic/responses）。

### 2. 上报模型带格式（`pkg/public/openai.go` 的 `Model`）

```go
type Model struct {
    ...
    UpstreamFormat string `json:"upstream_format"` // openai | anthropic | responses
}
```

client 在 `ListModels` 上报时，把 `ProxyBackend.Format` 填进每个模型的 `UpstreamFormat`。server 端 `LoadBalance` 选到 client 后，从该 client 的模型信息读取 `UpstreamFormat`。

**两者结合的意义**：
- client 配置 `Format` 决定**用哪个引擎**调上游。
- 上报的 `UpstreamFormat` 决定 **server 端把用户请求转成什么格式**发给 client。
- 二者必须一致（client 上报时从配置读取），server 端以 `UpstreamFormat` 为准做转换。

## 六、Client 端引擎（每引擎一种上游格式）

### 新增引擎

```
client/internal/inference/
  engine.go        // 接口不变
  openai/          // 现有，不动
  anthropic/       // 新增：用 anthropic-sdk-go
  responses/       // 新增：用 openai-go
```

### 引擎职责（大幅简化）

因为 server 已经把请求转成**上游格式**了，client 引擎只需：
1. 接收上游格式请求（`WSMessage.Content` 已是上游格式 JSON）。
2. 用官方 SDK 调用上游。
3. 把上游响应（含流式 SSE）原样通过 WS 返回。

```go
// anthropic 引擎 HandleChat
func (e *Engine) HandleChat(ctx, fingerprint, request *public.AnthropicRequest, responseConn) error {
    // 用 anthropic-sdk-go 调用
    // 流式：逐事件转发 content_block_delta / message_delta / message_stop
    // 非流式：返回完整 AnthropicResponse
}
```

> 注意：client 端不再需要 Canonical 转换，因为 server 已转好。client 只是"格式透传 + 上游调用"。这大大简化 client 端。

## 七、9 条转换路径

| 用户格式 \ 上游格式 | openai | anthropic | responses |
|---|---|---|---|
| **openai** | 直通（现有，无转换） | openai→canonical→anthropic | openai→canonical→responses |
| **anthropic** | anthropic→canonical→openai | 直通 | anthropic→canonical→responses |
| **responses** | responses→canonical→openai | responses→canonical→anthropic | 直通 |

每层 3 个转换器（openai/anthropic/responses），组合出 9 条路径。直通路径（对角线）零转换，性能最优。

## 八、流式处理

流式是最大难点，因为三种格式的 SSE 事件结构完全不同：

| 格式 | 流式事件 |
|---|---|
| OpenAI Chat | `data: {"choices":[{"delta":{"content":"..."}}]}` |
| Anthropic | `event: content_block_delta` + `data: {"delta":{"type":"text_delta","text":"..."}}` |
| Responses | `event: response.output_text.delta` + `data: {"delta":"..."}` |

**方案**：server 端做**事件级转换**。
- 上游流式事件 → `CanonicalStreamEvent`（统一为 `text_delta`/`tool_call_delta`/`done`/`error`）。
- `CanonicalStreamEvent` → 用户格式的 SSE 事件。

这样流式转换也是 3×3，但每层只有 3 个事件转换器。

**流式终止**：各格式的结束事件（OpenAI `[DONE]`、Anthropic `message_stop`、Responses `response.completed`）统一映射为 `done` 事件，再转成用户格式的结束标记。

## 九、计费

三种格式的 usage 字段不同，统一到 `CanonicalUsage` 后计费：

| 格式 | 输入 | 输出 | 缓存 |
|---|---|---|---|
| OpenAI | `prompt_tokens` | `completion_tokens` | `prompt_tokens_details.cached_tokens` |
| Anthropic | `input_tokens` | `output_tokens` | `cache_read_input_tokens` |
| Responses | `input_tokens` | `output_tokens` | `input_tokens_details.cached_tokens` |

server 端 `recordTokenUsage` 改为接收 `CanonicalUsage`，统一计费。

## 十、错误处理

- 上游 4xx（请求错误）→ 转成用户格式的错误响应返回（`isClientRequestError` 逻辑复用）。
- 上游 5xx → 重试（现有 `handleChatWithRetry` 逻辑复用）。
- 错误信息也要做格式转换：Anthropic 的 `error.type`/`error.message`、Responses 的 `error.code`/`error.message` → 用户格式的错误结构。

## 十一、分阶段实施计划

| 阶段 | 内容 | 交付物 |
|---|---|---|
| **P0** | Canonical 结构体 + Converter 接口 + 传输层 `Format` 字段 | `pkg/public/canonical.go`、`protocol.go` 加字段 |
| **P1** | OpenAI Chat ↔ Canonical 转换器 + server 适配层重构（泛化 handleChatWithRetry） | 打通现有链路，验证架构 |
| **P2** | client 端 anthropic 引擎（anthropic-sdk-go）+ responses 引擎（openai-go）+ 后端自动发现（远程 Proxy + 本地 Ollama/llama.cpp）+ 配置 `Format` + 上报 `UpstreamFormat` | client 支持三种上游 + 本地引擎自动发现 |
| **P3** | server 端 anthropic / responses 用户格式解析与回传（含流式事件转换） | 三种用户入口 |
| **P4** | 9 条路径完整测试 + 计费 + 错误处理 | 全链路验证 |

## 十二、风险与注意点

1. **`max_tokens` 必填**：Anthropic 要求 `max_tokens`，从 OpenAI/Responses 转过来时若无，需给默认值。
2. **`system` 位置**：Anthropic 的 `system` 是顶层字段，OpenAI 是 `messages[0]`，转换时需正确提取/放置。
3. **工具调用差异**：三种格式的 tool 结构差异大（OpenAI `tool_calls`、Anthropic `tool_use` block、Responses `function_call`），是转换最复杂的部分，需重点测试。
4. **thinking 块**：Anthropic 的 `thinking` block 和 OpenAI 的 `reasoning_effort` 需要映射，且部分后端不支持，需透传或降级。
5. **老 client 兼容**：传输层加 `Format` 字段后，老 client 不认识会走默认 openai，需确认 client 统一升级。

## 十三、关于 opentrans 的结论

**确认用官方 SDK 后，opentrans 用不到了。**

原因：
- opentrans 的价值是**用一个 SDK 统一调用多种上游格式**（OpenAI/Anthropic/Gemini），减少多 SDK 维护成本。
- 但本架构里，**client 端每个引擎只调一种上游格式**（openai 引擎只调 OpenAI Chat，anthropic 引擎只调 Anthropic，responses 引擎只调 Responses），不存在"一个引擎要同时调多种格式"的场景。
- 需要**无损转换**，而 opentrans 为了统一抽象，往往对各家**特有新特性**（Anthropic 的 thinking/tool_use/content block、Responses 的 function_call input）支持滞后或丢失细节。官方 SDK 对各自格式支持最完整、类型最安全、文档最全。
- 引入 opentrans 反而多一层抽象，且它内部可能还是封装官方 SDK，等于白加一层。

**什么情况下才需要 opentrans**：如果未来某个 client 引擎要"一个后端同时兼容三种格式"（比如一个 vLLM 网关同时暴露三种协议），那 opentrans 才有价值。当前按"每引擎一种格式"设计，不需要。

---

## 十四、本地引擎的 OpenAI Chat 兼容性（Ollama / llama.cpp）

结论先行：**Ollama 和 llama.cpp 都已经原生支持 OpenAI Chat API，天然属于 `openai` 上游格式，且都纳入统一的后端自动发现与接入体系（第十五章）。**

### 1. Ollama

- Ollama 官方内置 OpenAI 兼容层，原生提供 `POST /v1/chat/completions`、`GET /v1/models`、`POST /v1/embeddings`。
- 当前 star-fire 用 `github.com/ollama/ollama/api` SDK 调用（走 Ollama 原生接口，非 OpenAI 格式），但引擎内部已通过 `convertToOpenAIStreamResponse` 把 Ollama 的 `api.ChatResponse` 统一转成 `openai.ChatCompletionStreamResponse` 再回传 server。
- **多格式视角**：Ollama 上游格式 = `openai`。它的模型在 `public.Model.UpstreamFormat` 中应标记为 `openai`。

### 2. llama.cpp

- `llama-server` 原生提供 OpenAI 兼容 API（`/v1/chat/completions`、`/v1/models`、`/v1/embeddings`），通过 `--api-key` 与 OpenAI 风格参数使用。
- 当前 star-fire **没有 llama.cpp 专用引擎**，通过统一后端自动发现（第十五章）把 `llama-server` 作为本地 OpenAI 兼容后端接入。

### 3. 结论与影响

| 本地引擎 | 上游格式 | 处理方式 |
|---|---|---|
| Ollama | openai | 保留现有 ollama 引擎（输出已转 OpenAI），标记 `UpstreamFormat=openai`；同时支持通过自动发现走 OpenAI 兼容端点 |
| llama.cpp | openai | 通过统一后端自动发现，复用 openai 引擎，`Format=openai` |

- 本地引擎天然走"openai 用户格式 ↔ openai 上游格式"的**直通路径**（对角线），零转换开销。
- 只有 **Anthropic Messages** 与 **OpenAI Responses** 两种上游才需要新增引擎（第六章）。

---

## 十五、统一后端自动发现与格式适配（Proxy / Ollama / llama.cpp）

**目标**：client 接入任意后端（远程 Proxy、本地 Ollama、本地 llama.cpp）时，自动探测其协议格式并接入，无需用户手动指定格式。

### 1. 统一后端模型

把 `ProxyBackend` 泛化为 `Backend`，统一覆盖远程与本地后端：

```go
type Backend struct {
    Name    string `json:"name"`
    BaseURL string `json:"base_url"`
    APIKey  string `json:"api_key"`
    Format  string `json:"format"` // openai | anthropic | responses，默认 openai
    Local   bool   `json:"local"`  // 是否为本地后端（本地后端通常无需 APIKey）
    Enabled bool   `json:"enabled"`
}
```

- 远程 Proxy 与本地引擎（Ollama/llama.cpp）统一用 `Backend` 表示，格式探测与引擎选择走同一套逻辑。
- `Local=true` 表示本地后端：无需 APIKey、默认地址可预测、探测失败静默跳过。

### 2. 三类后端与默认探测地址

| 后端类型 | 默认地址 | 默认格式 | 探测方式 |
|---|---|---|---|
| 远程 Proxy | 用户配置 `base_url` | 自动探测（三级降级） | 显式 > 探测 > openai |
| 本地 Ollama | `http://localhost:11434` | openai | 探测 `GET /v1/models` |
| 本地 llama.cpp | `http://localhost:8080` | openai | 探测 `GET /v1/models` |

> Ollama 与 llama.cpp 的 OpenAI 兼容端点都挂在 `/v1` 下，因此探测地址分别是 `http://localhost:11434/v1` 与 `http://localhost:8080/v1`。

### 3. 本地后端自动发现

Ollama 和 llama.cpp 无需用户手动配置，client 启动/刷新时自动探测默认地址：

```go
// 本地后端候选（按优先级探测）
var localBackendCandidates = []Backend{
    {Name: "ollama",   BaseURL: "http://localhost:11434/v1", Local: true},
    {Name: "llamacpp", BaseURL: "http://localhost:8080/v1",  Local: true},
}

func discoverLocalBackends(ctx context.Context) []Backend {
    // 对每个候选地址 GET /v1/models：
    //   返回 200 且 JSON 含 data[] → 判定为 openai 格式，纳入接入
    //   连接失败/超时 → 静默跳过（后端未启动）
}
```

- 探测到即接入，探测失败（后端未启动）静默跳过，不报错。
- 探测到的格式标记为 `openai`，走 openai 引擎。
- 用户可通过配置/环境变量覆盖默认地址（如 `OLLAMA_HOST`、`LLAMACPP_HOST`）。

### 4. Ollama 的双模式接入

Ollama 有两种接入方式，均支持，按场景选择：

| 模式 | 说明 | 优势 | 劣势 |
|---|---|---|---|
| 专用 SDK（现有） | `github.com/ollama/ollama/api` | 拿到模型详细信息（参数规模、量化、上下文长度等） | 多一套 SDK，非 OpenAI 原生格式 |
| OpenAI 兼容端点 | 走 openai 引擎，`BaseURL=localhost:11434/v1` | 统一架构，纳入自动发现 | 元信息较少 |

**建议**：默认保留专用 SDK（现有代码不动、元信息丰富），同时支持通过自动发现走 OpenAI 兼容端点。二者 `UpstreamFormat` 都标记为 `openai`，server 端无需感知差异。

### 5. llama.cpp 新增接入

llama.cpp 当前无专门引擎，通过**统一后端自动发现**接入：

- `llama-server` 启动后默认监听 `http://localhost:8080`（可用 `--port` 自定义）。
- 提供 OpenAI 兼容端点 `/v1/chat/completions`、`/v1/models`、`/v1/embeddings`。
- 通过自动发现探测到后，用 openai 引擎接入，格式 = `openai`。
- **无需新增引擎代码**，只需在本地后端候选列表加入 llama.cpp 地址即可。

### 6. 自动发现策略（三级降级，统一适用）

```
1. 显式配置优先：Backend.Format 显式指定 → 直接用
2. 主动探测：探测特征端点/特征行为判断格式
3. 默认兜底：探测失败则按 openai 处理（业界最普遍，本地引擎默认就是 openai）
```

### 7. 主动探测方法

通过探测各格式的**特征端点/特征行为**来判断（按顺序，命中即停）：

| 探测目标 | 判断依据 | 结论 |
|---|---|---|
| `GET /v1/models` 返回 200 且 JSON 含 `data[]` | OpenAI 标准模型列表 | 支持 OpenAI 兼容（本地引擎命中此条即结束） |
| `POST /v1/messages` 返回 Anthropic 风格错误（如 `{"type":"error","error":{"type":"invalid_request_error"}}`）| Anthropic 错误结构 | 支持 Anthropic Messages |
| `POST /v1/responses` 返回 Responses 风格错误（如 `{"error":{"code":...}}`）| Responses 错误结构 | 支持 Responses |

> 更稳妥的方式：发一个**最小合法请求**（如 `POST /v1/chat/completions` 带空 model）看返回的**错误格式**特征，通过错误结构反推协议类型。避免真实推理产生费用。

### 8. 统一适配流程

```go
// client 初始化/刷新时，统一发现所有后端（远程 + 本地）
func discoverBackends(ctx context.Context, cfg *config.Config) []Backend {
    backends := []Backend{}
    backends = append(backends, cfg.ProxyBackends...)        // 远程 proxy
    backends = append(backends, discoverLocalBackends(ctx)...) // 本地 ollama/llama.cpp
    for i := range backends {
        backends[i].Format = detectBackendFormat(backends[i]) // 统一探测/回填格式
    }
    return backends
}

// 探测单个后端格式
func detectBackendFormat(backend Backend) string {
    if backend.Format != "" {
        return backend.Format  // 显式配置优先
    }
    if f := probeFormat(backend.BaseURL); f != "" {
        return f
    }
    return "openai"  // 兜底
}
```

- 探测结果缓存，避免每次请求都探测。
- 后端重建（`applyProxyBackends`）或本地引擎变化时重新探测。
- 探测到的格式写入 `public.Model.UpstreamFormat`，随模型上报给 server。

### 9. 与配置声明的结合

- **`Backend.Format`**：显式指定时**跳过探测**（用户最清楚后端类型，也避免探测请求副作用）。
- **自动探测**：未指定时启用，探测结果回填 `UpstreamFormat`。
- **两者关系**：显式配置 > 自动探测 > 默认 openai。
- **本地引擎**：默认就是 openai，探测主要为兼容未来 Ollama/llama.cpp 可能新增的 Anthropic/Responses 端点。

---

## 十六、计费全支持

### 1. 计费数据流（统一到 CanonicalUsage）

三种格式的 usage 字段在 server 端统一到 `CanonicalUsage`，再由 `recordTokenUsage` 统一计费（缓存命中输入按 `CIPPM`，普通输入按 `IPPM`，输出按 `OPPM`）：

```
上游响应 usage → ConvertXxxToCanonical 提取 → CanonicalUsage
    → recordTokenUsage(server, canonicalUsage)
        ├─ 输入 tokens（未命中缓存）→ IPPM
        ├─ 缓存命中 tokens             → CIPPM
        └─ 输出 tokens                 → OPPM
```

### 2. 各格式 usage 字段映射（含缓存命中）

| 格式 | 输入 | 输出 | 缓存命中输入 |
|---|---|---|---|
| OpenAI Chat | `usage.prompt_tokens` | `usage.completion_tokens` | `usage.prompt_tokens_details.cached_tokens` |
| Anthropic Messages | `usage.input_tokens` | `usage.output_tokens` | `usage.cache_read_input_tokens`（+ `cache_creation_input_tokens` 写入缓存） |
| OpenAI Responses | `usage.input_tokens` | `usage.output_tokens` | `usage.input_tokens_details.cached_tokens` |

### 3. 流式计费的特殊处理

- 流式请求的 usage 通常在**最后一个事件**中才出现：
  - OpenAI：`stream_options.include_usage=true` 时，最后一个 chunk 带 `usage`。
  - Anthropic：`message_delta` 事件带 `usage.output_tokens`，`message_start` 带 `usage.input_tokens`。
  - Responses：`response.completed` 事件带完整 `usage`。
- **server 端计费逻辑**（复用现有 `handleStreamChatResponse` 的 usage 提取模式）：统一在收到"结束事件"时提取 usage 计费，中间文本事件不触发计费。

### 4. 计费完整性要求

- 非流式：三种格式都从最终响应提取 usage。
- 流式：三种格式都从结束事件提取 usage，缺失时按 0 计费但需告警日志。
- 缓存命中：三种格式都要正确识别 `CIPPM`，避免缓存命中的输入被按普通 `IPPM` 高估收费。
- 错误/中断：请求失败或中断不产生计费（现有逻辑已保证：只有拿到 usage 才 `recordTokenUsage`）。

---

## 十七、前端一键接入：Codex / Claude Code

**目标**：前端 UI 对任意模型生成 Codex / Claude Code 的接入配置，用户复制即用，实现"一键接入任意 client 端模型"。

### 1. 三种接入方式与格式对应

| 接入工具 | 使用协议 | 对应端点 | 前端生成物 |
|---|---|---|---|
| **Codex CLI**（chat）| OpenAI Chat | `POST /v1/chat/completions` | `~/.codex/config.toml`（`wire_api="chat"`）|
| **Codex CLI**（responses）| OpenAI Responses | `POST /v1/responses` | `~/.codex/config.toml`（`wire_api="responses"`）|
| **Claude Code** | Anthropic Messages | `POST /v1/messages` | 环境变量（`ANTHROPIC_BASE_URL` + token）|

### 2. Codex CLI 接入（chat / responses 双模式）

Codex 通过 `~/.codex/config.toml` 的 `model_providers` 配置自定义后端，`wire_api` 决定用 chat 还是 responses：

```toml
model = "<模型名>"
model_provider = "star-fire"

[model_providers.star-fire]
name = "star-fire"
base_url = "http://<server>/v1"
wire_api = "chat"        # chat 模式：/v1/chat/completions
# wire_api = "responses" # responses 模式：/v1/responses
requires_openai_auth = false
```

- 前端对每个模型生成上述 TOML，`wire_api` 提供 `chat` 和 `responses` 两个选项（对应用户需求）。
- 鉴权：Codex 支持通过环境变量 `OPENAI_API_KEY` 传 star-fire 的 API key；`requires_openai_auth=false` 表示不走 OpenAI 官方鉴权头。

### 3. Claude Code 接入（messages）

Claude Code 通过环境变量接入，SDK 自动拼 `/v1/messages`：

```bash
export ANTHROPIC_BASE_URL="http://<server>"       # 不含 /v1，SDK 自动拼 /v1/messages
export ANTHROPIC_AUTH_TOKEN="<star-fire api key>"
# 或 export ANTHROPIC_API_KEY="<star-fire api key>"
```

- 前端生成这段环境变量命令，用户复制到终端执行即可。
- Claude Code 使用的模型名在启动时通过参数/配置指定，前端提供模型名复制。

### 4. 前端交互设计

```
模型详情页（任意模型）
  └─ [接入方式] 分栏
      ├─ Codex (chat)      → 生成 config.toml 片段 + 复制按钮
      ├─ Codex (responses) → 生成 config.toml 片段 + 复制按钮
      └─ Claude Code       → 生成环境变量命令 + 复制按钮
```

- 前端从 `/api/market/models` 拿到模型列表（含 `upstream_format`、`name`）。
- 一键接入**不关心**模型上游格式（openai/anthropic/responses 都可接入），因为 server 端已做格式转换——这正是多格式适配层的价值。

### 5. 关键注意点

- `base_url` 必须是 server 的对外地址（含鉴权，用户 API key）。
- Anthropic 的 `max_tokens` 必填：Claude Code 会自带，但若第三方工具没带，server 适配层需给默认值（见第十二章风险 1）。
- Codex `responses` 模式需 server 的 `/v1/responses` 端点完整可用（含流式 `response.output_text.delta`）。

---

## 十八、测试策略与验收标准

> **核心原则**：每个功能模块开发完成后，**先通过该模块的专项测试，再进入下一个模块**；全部模块完成后做端到端全链路测试。

### 1. 测试分层

| 层级 | 范围 | 通过标准 |
|---|---|---|
| **单元测试** | 6 个转换器（3 格式 × 请求/响应/流式） | 每个转换器独立通过（用例详见 3.4） |
| **模块测试** | 每个引擎（openai/anthropic/responses） | 各自调通真实/模拟上游 |
| **集成测试** | server↔client 传输（Format 字段透传） | 格式正确透传、不丢字段 |
| **端到端测试** | 9 条转换路径全链路 | 每条路径全通过 |

### 2. 分阶段测试门禁（对应第十一章）

| 阶段 | 完成后必须通过的测试 | 未通过则 |
|---|---|---|
| **P0** | Canonical 结构体序列化/反序列化单测、传输层 Format 字段兼容单测 | 不进入 P1 |
| **P1** | OpenAI↔Canonical 转换器单测（含流式）、现有 chat 链路回归测试 | 不进入 P2 |
| **P2** | anthropic 引擎模块测试、responses 引擎模块测试、后端自动发现测试（远程 Proxy + 本地 Ollama/llama.cpp） | 不进入 P3 |
| **P3** | anthropic/responses 用户入口集成测试（含流式）、计费提取测试 | 不进入 P4 |
| **P4** | 9 条路径端到端测试 + 计费 + 错误处理 + Codex/Claude Code 接入 | 全绿才合入 |

### 3. 必测清单（全链路）

**三种用户 API 请求全面验证**：
- ✅ `POST /v1/chat/completions`（OpenAI Chat）— 非流式 + 流式
- ✅ `POST /v1/messages`（Anthropic Messages）— 非流式 + 流式（`content_block_delta`/`message_stop`）
- ✅ `POST /v1/responses`（OpenAI Responses）— 非流式 + 流式（`response.output_text.delta`/`response.completed`）

**9 条转换路径**（第七章矩阵的每一条）：
- 纯文本、多轮对话、系统提示、工具调用、多模态（图片）、thinking/推理、缓存命中。

**计费验证**（第十六章）：
- 三种格式的输入/输出/缓存命中 tokens 提取正确，计费金额与期望一致。

**工具调用链路**：
- OpenAI `tool_calls` ↔ Anthropic `tool_use`/`tool_result` ↔ Responses `function_call` 三种格式交叉转换，多轮 tool 循环正确。

**错误处理**（第十章）：
- 上游 4xx → 用户格式错误响应；上游 5xx → 重试；三种格式错误结构正确转换。

**后端自动发现验证**（第十五章）：
- ✅ 远程 Proxy 自动探测格式（openai / anthropic / responses 各测一个）。
- ✅ 本地 Ollama 自动发现（`localhost:11434`）并接入。
- ✅ 本地 llama.cpp 自动发现（`localhost:8080`）并接入。
- ✅ Ollama 双模式：专用 SDK 与 OpenAI 兼容端点都能正常对话。
- ✅ 后端未启动时静默跳过，不影响其他引擎。

**前端接入验证**：
- Codex（chat）→ 接入 → 正常对话
- Codex（responses）→ 接入 → 正常对话
- Claude Code → 接入 → 正常对话
- 三者都验证到任意 client 端模型（openai / anthropic / responses 上游各测一次）。

### 4. 测试工具建议

- **单元测试**：Go 原生 `testing` + 表驱动，覆盖 6 个转换器所有字段映射。
- **模块测试**：真实上游或本地 mock（如 `vLLM`、`Ollama`、`llama-server` 本地起服务）。
- **端到端**：`curl` 脚本 + 三种官方 CLI（Codex、Claude Code）实际跑通对话。
- **回归**：现有 chat 链路测试（`internal/service`、`client/internal`）必须保持全绿，确保重构不破坏现有功能。

### 5. 完成定义（DoD）

整个多格式功能「验收通过」需同时满足：
1. 三种用户 API 请求（chat / messages / responses）非流式、流式均正常。
2. 9 条转换路径全通，字段无损（含工具调用、多模态、thinking）。
3. 三种上游格式（openai / anthropic / responses）的 token 计费全部正确（含缓存命中）。
4. 后端自动发现与格式适配工作正常（远程 Proxy + 本地 Ollama + 本地 llama.cpp）。
5. Codex（chat + responses）与 Claude Code 一键接入任意 client 端模型，实际对话通过。
6. 现有 OpenAI Chat 链路回归无退化。
