# Codex 接入本地网关：最终测试报告（第三轮确认）

- **日期**: 2026-08-25
- **Codex CLI**: `0.149.0-alpha.4.3`
- **本地网关**: `http://localhost:8080/v1`（OpenAI 兼容，底层 vLLM 0.26.0）
- **接入方式**: `~/.codex/config.toml` `wire_api = "responses"`

---

## 一、本轮结论一句话

网关经过多轮修改后，**非思考模型已完全可用，但思考模型（deepseek-v4-pro / deepseek-v4-flash）的工具调用仍有根本性缺陷**。同时 web_search 的**搜索结果内容**在回传时被吞掉（模型看到空结果）。

---

## 二、全面测试矩阵（5 模型 × 4 能力）

| 能力 | DeepSeek-V4 | GLM-5.2 | Qwen3.5-397B | deepseek-v4-pro | deepseek-v4-flash |
|---|---|---|---|---|---|
| **基础对话** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **web_search 工具定义+调用决策** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **function 工具回传**（2+2=4） | ✅ | ✅ | ✅ | ❌ | ❌ |
| **web_search 结果回传** | ⚠️ 内容被吞 | ⚠️ 内容被吞 | ⚠️ 内容被吞 | ❌ | ❌ |

✅ 正常　⚠️ 部分（能返回但不透传内容）　❌ 失败

---

## 三、已确认修复/正常的能力

对比上一轮，本轮网关修改后：

1. ✅ **基础对话 5/5 全部通过** —— 之前 Qwen 的 `System message must be at the beginning` 已修复。
2. ✅ **function 工具回传（2+2=4）** —— 非思考模型（V4/GLM/Qwen）全部成功。
3. ✅ **web_search 工具定义 + 返回标准 `web_search_call`** —— 全部模型正常。
4. ✅ 非思考模型能接受 `web_search_call` input item（不再 400）。

---

## 四、剩余问题（两个）

### 问题 A：web_search 搜索结果内容被吞（影响所有模型）

**表现**：给模型提供搜索"结果"（`web_search_result` 含 text），模型回答 **"search result came back empty"**（搜索结果是空的）。

> 实测：input 给了 `output: [{type: web_search_result, text: "Today in Beijing: sunny, 25 degrees"}]`，模型答："I don't have any weather information ... The search result came back **empty**, so there is no data to report."

**根因**：网关接受了 `web_search_call` item（不再报 400），但**没有把 WebSearchResult 的 `output`/`text` 内容透传给下游模型**，模型拿到的搜索结果是空。属于"半透传"——结构接受、内容丢失。

**对比**：function 工具结果的 `function_call_output` 内容（`output: "4"`）能被模型正确读取（2+2=4 证明透传 OK），所以**只有 web_search 的内容透传有缺陷**。

### 问题 B：思考模型（pro/flash）工具调用失败

**表现**：pro/flash 三个函数调用链全部 HTTP 400。

**实测报错（三种输入方式）**：
| 输入方式 | 报错 |
|---|---|
| Responses `function_call` → `function_call_output` | `The reasoning_content in the thinking mode must be passed back to the API` |
| `assistant(tool_calls)` → `tool`(结果) | `messages[2]: missing field 'tool_call_id'` |
| `web_search_call` 结果回传 | `Messages with role 'tool' must be a response to a preceding message with 'tool_calls'` |

**触发条件**：仅**工具结果回传**触发；纯对话、纯多轮（无工具）均正常（200）。

**根因**：pro/flash 是**思考型模型**，DeepSeek 思考模型机制要求：
- 续写（收到工具结果后继续生成）时必须把上一轮模型的 **`reasoning_content`（思维链）** 回传给 API；
- 工具结果回传消息必须正确携带 **`tool_call_id`**、并紧跟其 `tool_calls` 消息。

网关在把 Codex 的工具循环（function_call → function_call_output → 继续）转成下游 Chat 时，**既未回传 reasoning_content，也未正确构造 tool 消息结构**，导致思考模型报错。非思考模型无此机制，故正常。

---

## 五、与 DeepSeek 官方机制对照

DeepSeek 思维链模型（如 deepseek-reasoner / thinking 模式）的官方要求：
- 推理内容 `reasoning_content` 在对话继续时**必须回传**（官方文档"思考模式"章节）。
- 工具调用在多轮中需要正确的 `tool_calls` ↔ `tool` 消息配对。

你的 pro/flash 报错正是这两条未满足。网关需对**思考模型的工具续写**做专门的状态回传处理。

---

## 六、结论与修复建议

### 修复优先级
1. **问题 B（pro/flash 思考模型工具调用）** —— 功能缺失根因。网关需实现思考模型工具循环的：
   - `reasoning_content` 回传
   - 正确的 `tool_call_id` / `tool_calls`↔`tool` 消息配对
2. **问题 A（web_search 内容透传）** —— 影响所有模型搜索结果利用。网关需把 `web_search_result` 的内容（text/title/url）透传给下游模型。

### 修复后预期
- 所有 5 模型基础对话 ✅（已达成）
- 所有 5 模型 function 工具调用 ✅
- 所有 5 模型 web_search 能真正利用搜索结果 ❌→✅

---

## 附：相关文件
- `E:\Study\test\codex-full-test-report.md` — 第二轮全面测试矩阵
- `E:\Study\test\codex-tool-issues-deep.md` — 三类问题深度定位
- `E:\Study\test\codex-tool-transparency.md` — 工具透传问题
- 测试脚本：`C:\Users\skyro\AppData\Local\Temp\opencode\full_test.ps1`、`tool_deep.ps1`、`thinking_test.ps1`、`confirm_ws.py`、`pro_tool_chain.py`
