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
```

### 各格式转换要点

**OpenAI Chat ↔ Canonical**
- `messages[].content`：string 或 `[{type:text|image_url}]` → 统一 block 数组。
- `tool_calls` → `CanonicalToolCall`。
- `role=tool` 的 `tool_call_id` → `CanonicalMessage.ToolCallID`。
- 响应 `choices[0].message` → `CanonicalResponse.Content`。
- 流式 `choices[0].delta` → `CanonicalStreamEvent`。
- usage：`prompt_tokens`/`completion_tokens`/`total_tokens`/`prompt_tokens_details.cached_tokens`。

**Anthropic ↔ Canonical**
- `system`（string 或 block 数组）→ `CanonicalRequest.System`。
- `messages[].content`（block 数组：text/image/tool_use/tool_result/thinking）→ 直接映射 block。
- `max_tokens` 必填 → 若无则给默认值（如 4096）。
- 响应 `content[]`（text/tool_use/thinking）→ `CanonicalResponse.Content`。
- 流式事件：`content_block_start`/`content_block_delta`/`content_block_stop`/`message_delta`/`message_stop` → `CanonicalStreamEvent`。
- usage：`input_tokens`/`output_tokens`/`cache_creation_input_tokens`/`cache_read_input_tokens`。

**Responses ↔ Canonical**
- `input[]`（`{type:message}` / `{type:function_call}` / `{type:function_call_output}`）→ `CanonicalMessage`。
- `instructions` → `CanonicalRequest.System`。
- 响应 `output[]`（`message`/`function_call`/`reasoning`）→ `CanonicalResponse.Content`。
- 流式事件：`response.output_text.delta`/`response.function_call_arguments.delta`/`response.completed` → `CanonicalStreamEvent`。
- usage：`input_tokens`/`output_tokens`/`total_tokens`。

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
| **P2** | client 端 anthropic 引擎（anthropic-sdk-go）+ responses 引擎（openai-go）+ 配置 `Format` + 上报 `UpstreamFormat` | client 支持三种上游 |
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
