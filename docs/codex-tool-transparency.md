# Codex 工具调用失败分析：网关双向透传丢失/篡改工具语义

- **日期**: 2026-08-25
- **Codex CLI**: `0.149.0-alpha.4.3`
- **本地网关**: `http://localhost:8080/v1`（OpenAI 兼容，底层 vLLM 0.26.0）
- **接入方式**: `~/.codex/config.toml` `wire_api = "responses"`
- **测试方法**: 在网关前置捕获代理（`127.0.0.1:8081 → 8080`），完整记录 Codex 发出请求与网关返回响应，逐段对比。

---

## 一、核心结论（一句话）

> 网关在 Responses 协议上是**有损转发**：请求方向，它没有把 Codex 的工具定义按原语义完整交给下游模型；返回方向，它把模型的工具调用用错误的 Response 事件类型（`function_call`）返回，而不是标准类型（`web_search_call` 等）。Codex 据此报 `unsupported call`，模型也以为没有搜索/工具能力。

**根本定位**：网关不能丢/改任何请求内容和回复内容——现在它在工具语义上双向都丢了。这导致 web_search、tool_search 等工具调用不了（与"服务端执行"无关，网关职责是**如实透传调用决策**）。

---

## 二、Codex 发出的请求（正确，作为基准）

`POST /v1/responses`，含 15 个工具。**非 function 类型工具**（Codex 专有/标准能力）：

| 索引 | type | name | 用途 |
|---|---|---|---|
| 7 | `custom` | `apply_patch` | Codex 代码编辑 |
| 9 | `namespace` | `collaboration` | Codex 多代理协作 |
| 13 | `tool_search` | —（无 name） | Codex 工具发现/延迟加载 |
| 14 | `web_search` | —（无 name） | 网页搜索 |

```jsonc
"tools": [
  { "type": "function", "name": "exec_command", ... },
  { "type": "function", "name": "write_stdin", ... },
  { "type": "function", "name": "list_mcp_resources", ... },
  // ...
  { "type": "custom",    "name": "apply_patch", ... },
  { "type": "namespace", "name": "collaboration", "tools": [...] },
  { "type": "tool_search", "execution": "client", "parameters": {...} },
  { "type": "web_search", "external_web_access": false }
],
```

这是 Codex 的标准 Responses 请求，工具类型语义正确、完整。

---

## 三、网关返回的响应（问题所在——丢失/篡改）

### 现象 A：工具调用被错误包装成 `function_call`

模型决定搜索时，网关把它返回成 **`function_call name="tool_search"` / `name="web_search"`**，而不是标准 Response 输出类型：

```jsonc
// 网关返回（第 1 轮，name=tool_search）
{ "type": "function_call", "id": "fc_...", "name": "tool_search",
  "arguments": "{ \"query\": \"web_search search engine query news\" }" }

// 网关返回（另有场景 name=web_search）
{ "type": "function_call", "name": "web_search", "arguments": "{}" }
```

而 OpenAI Responses 规范里，网页搜索应该是 **`web_search_call`** 类型，Codex 也据此识别。

### 现象 B：Codex 无法识别 → 报错

```
ERROR codex_core::tools::router: error=unsupported call: web_search
ERROR: tool_search handler received unsupported
ERROR: Fatal error: tool_search handler received unsupported
```

### 现象 C：模型被误导，认为没有搜索工具

模型反复尝试后妥协，最终返回：
```
"I don't have a working web search tool available in this environment.
 The web search capability came back as unsupported, and neither
 browsing nor search tools are currently accessible."
```

---

## 四、发送 vs 返回 对照表（证据一目了然）

| 环节 | Codex / 模型的工具 | 网关 | 结果 |
|---|---|---|---|
| **请求：Codex → 网关 → 下游模型** | 传 `web_search`（type=web_search） | 需无损坏传给模型 | 模型未正确获得该工具语义 |
| **请求：同上** | 传 `tool_search`（专有） | 需无损传给模型 | 模型获得的是被改动的东西 |
| **返回：模型 → 网关 → Codex** | 模型决定调用 web_search | 应返回 `web_search_call` | 返回了 `function_call name=web_search` |
| **返回：同上** | 模型决定调用 tool_search | 应返回正确类型 | 返回了 `function_call name=tool_search` |
| **终端效果** | — | — | Codex 报 `unsupported call`；模型称"无搜索工具" |

---

## 五、根因分析

### 1. 网关做了"篡改"而非"透传"
- 把 `web_search`、`tool_search` 这些非 function 工具，在下游转换时**改成了 function 语义**（或降级处理）。
- 返回时，无论模型genuinely调用了哪个工具，网关都用统一 `function_call` 包装，**丢失了工具类型信息**。

### 2. 网关对"不认识的工具类型"处理策略错误
- Codex 的 `custom` / `namespace` / `tool_search` / `web_search` 都是合法 Responses 工具类型。
- 正确做法：**原样透传**给模型（模型理解这些类型），返回时**按类型映射到标准 Response 输出 item**（`function_call` / `custom_tool_call` / `web_search_call`）。
- 错误做法（当前）：统称为 `function` 并强加 `name`，丢弃类型。

### 3. 网关不能丢内容
- 请求的工具定义、模型的调用决策、工具名、参数，都必须无损。
- 当前返回 `arguments: "{}"`（web_search）说明参数还可能被清空。

---

## 六、正确做法（修复方向）

网关定位是**透明转发 + 无损双向映射**，原则：

1. **请求方向**：Codex 传来的 `tools[]` 各 item **原样透传**给下游模型（`function`、`custom apply_patch`、`namespace`、`tool_search`、`web_search` 保持各自 type）。若下游后端不支持某类型，应**映射为下游等价的表达**，而不是粗暴统一成 function。

2. **返回方向**：模型输出的工具调用，按 OpenAI Responses **标准 item 类型**返回：
   - function 调用 → `function_call`（含合法 name+arguments）
   - `apply_patch` custom 工具 → `custom_tool_call`
   - 网页搜索 → `web_search_call`
   - Codex 专有（如 tool_search）→ 按其应有语义返回，不强行转 function

3. **不丢失字段**：工具的 `name`、`type`、`arguments`/`parameters`、`call_id` 等必须完整保留。

4. **验证**：修完后，Codex 执行 `Search the web ...` 应能真正触发搜索并拿到结果，不再报 `unsupported call`；`apply_patch` 编辑、`tool_search` 工具发现也应可用。

---

## 七、结论摘要

- **基础对话**：DeepSeek-V4 / GLM-5.2 / deepseek-v4-pro / deepseek-v4-flash 可用；Qwen 另报 `System message must be at the beginning`（独立小问题）。
- **工具调用**：**web_search / tool_search / apply_patch / collaboration 均无法正常使用**。
- **根因**：网关对 Responses 工具做了**有损/篡改式转换**，双向丢失工具类型与语义，导致 Codex `unsupported call`、模型认为无搜索能力。
- **修复原则**：网关必须**无损透传 + 按标准类型映射**，不丢任何请求/回复内容。

---

## 附：相关文件
- `E:\Study\test\codex-developer-role-issue.md` — 早期 developer role 分析（已过时）
- `E:\Study\test\codex-local-gateway-report.md` — 接入总体报告
- `E:\Study\test\responses-stream-spec-fix.md` — 流式事件序列规范
